package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func moneyUseCase(t *testing.T, roller character.Roller, opts ...character.Option) character.RollHigherLevelStartingMoney {
	t.Helper()
	uc, err := character.NewRollHigherLevelStartingMoney(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func moneyCmd(level int) character.RollHigherLevelStartingMoneyCommand {
	return character.RollHigherLevelStartingMoneyCommand{StartingLevel: character.MustLevel(level)}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-085, FR-086, FR-088, Table SM (SRD 5.2.1, p. 24) -
// a level with rolled gold rolls exactly one 1d10 and returns the allowance, the roll and the total in GP
// (minimum and maximum of every tier); levels 2 to 4 return 0 GP without calling the roller; level 1 is
// NotApplicable without calling the roller.
// Path: per case.
func TestRollHigherLevelStartingMoneyTableSM(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		level    int
		values   []int
		wantRoll int
		wantGP   int
		rolled   bool
		na       bool
	}{
		{"level 3, none rolled", 3, nil, 0, 0, false, false},
		{"level 5, minimum", 5, []int{1}, 1, 525, true, false},
		{"level 10, maximum", 10, []int{10}, 10, 750, true, false},
		{"level 11, minimum", 11, []int{1}, 1, 5250, true, false},
		{"level 16, maximum", 16, []int{10}, 10, 7500, true, false},
		{"level 17, minimum", 17, []int{1}, 1, 20250, true, false},
		{"level 20, maximum", 20, []int{10}, 10, 22500, true, false},
		{"level 18, middle", 18, []int{5}, 5, 21250, true, false},
		{"level 1, not applicable", 1, nil, 0, 0, false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.values...)
			res, err := moneyUseCase(t, roller).Roll(context.Background(), moneyCmd(c.level))
			if err != nil {
				t.Fatal(err)
			}
			if c.na {
				if res != (character.RollHigherLevelStartingMoneyNotApplicable{}) || len(roller.Requests()) != 0 {
					t.Errorf("result = %#v, rolls %d, want NotApplicable and no roll [FR-086]", res, len(roller.Requests()))
				}
				return
			}
			r, ok := res.(character.RollHigherLevelStartingMoneyRolled)
			roll, rolled := r.Roll()
			if !ok || r.Total() != content.MustCost(c.wantGP, content.CoinGP, 0) || roll != c.wantRoll || rolled != c.rolled {
				t.Fatalf("result = %#v, want %d GP from roll %d (rolled %v)", res, c.wantGP, c.wantRoll, c.rolled)
			}
			if r.Allowance() != allowanceAt(t, c.level) {
				t.Errorf("allowance = %#v, want the allowance of level %d [FR-088]", r.Allowance(), c.level)
			}
			wantRequests := 0
			if c.rolled {
				wantRequests = 1
			}
			if got := roller.Requests(); len(got) != wantRequests || (wantRequests == 1 && got[0] != dice.MustExpression(1, dice.D10(), 0)) {
				t.Errorf("requests = %v, want %d request of 1d10 [FR-085]", got, wantRequests)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-087 - a roll off the d10 or a roller error
// is a *srderr.PortError for dice.Port with the cause reachable, a roller that answers another
// expression is a contract violation, and no money is returned.
// Path: per case.
func TestRollHigherLevelStartingMoneyRollerFailures(t *testing.T) {
	t.Parallel()
	wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D4(), 0))
	})
	for _, c := range []struct {
		name   string
		roller character.Roller
		cause  error
	}{
		{"roll 0", dicetest.NewFixedRoller(0), dicetest.ErrValueOutOfRange},
		{"roll 11", dicetest.NewFixedRoller(11), dicetest.ErrValueOutOfRange},
		{"roller error", dicetest.NewFixedRoller(), dicetest.ErrExhausted},
		{"another expression", wrong, srderr.ErrContractViolation},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := moneyUseCase(t, c.roller).Roll(context.Background(), moneyCmd(5))
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) || srderr.ErrorType(err) != "dice_roller_error" {
				t.Errorf("= %v, %v", res, err)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-089, FR-123 - the result compares by value and the
// same level and roll give an equal result.
// Path: unhappy (immutability, determinism).
func TestRollHigherLevelStartingMoneyIsDeterministic(t *testing.T) {
	t.Parallel()
	a, _ := moneyUseCase(t, dicetest.NewFixedRoller(7)).Roll(context.Background(), moneyCmd(12))
	b, _ := moneyUseCase(t, dicetest.NewFixedRoller(7)).Roll(context.Background(), moneyCmd(12))
	if a != b || a == nil {
		t.Errorf("%#v != %#v", a, b)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-121, FR-122, section 7 - a rolled amount and a
// not-applicable level are success (results rolled and not_applicable), a roller failure is recorded
// once with error.type and no result, the duration carries the operation only, nothing is logged, and a
// recorder and the Noop give equal results.
// Path: per case.
func TestRollHigherLevelStartingMoneySignals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		level   int
		roller  character.Roller
		attrs   map[string]any
		status  telemetry.Status
		errors  int
		wantErr bool
	}{
		{"rolled", 5, dicetest.NewFixedRoller(3), map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "rolled"}, telemetry.StatusUnset, 0, false},
		{"levels 2 to 4 roll nothing", 3, dicetest.NewFixedRoller(), map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "rolled"}, telemetry.StatusUnset, 0, false},
		{"not applicable", 1, dicetest.NewFixedRoller(), map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "not_applicable"}, telemetry.StatusUnset, 0, false},
		{"roller failure", 5, dicetest.NewFixedRoller(), map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"}, telemetry.StatusError, 1, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			res, err := moneyUseCase(t, c.roller, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).Roll(context.Background(), moneyCmd(c.level))
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v", err)
			}
			plain, _ := moneyUseCase(t, dicetest.NewFixedRoller(3)).Roll(context.Background(), moneyCmd(c.level))
			if !c.wantErr && res != plain {
				t.Error("Noop and a recorder must give equal results")
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanAdvancementStartingMoney || spans[0].Status != c.status || len(spans[0].Errors) != c.errors {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("starting_money", c.attrs))
			d := onlyOne(t, rec, character.MetricDuration)
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "starting_money"})
			if d.Value != 0.25 {
				t.Errorf("duration = %+v", d)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-087, FR-123 - the zero level, the zero use case
// and a nil roller are invalid-input errors, a cancelled context is ctx.Err(), and none emits a signal.
// Path: per case.
func TestRollHigherLevelStartingMoneyInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	uc := moneyUseCase(t, dicetest.NewFixedRoller(3), character.WithTelemetry(rec.Telemetry()))
	if _, err := uc.Roll(context.Background(), character.RollHigherLevelStartingMoneyCommand{}); !isInvalid(err) {
		t.Errorf("zero level: %v", err)
	}
	if _, err := (character.RollHigherLevelStartingMoney{}).Roll(context.Background(), moneyCmd(5)); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := character.NewRollHigherLevelStartingMoney(nil); !isInvalid(err) {
		t.Errorf("nil roller: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Roll(ctx, moneyCmd(5)); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()); n != 0 {
		t.Errorf("%d signals before validation", n)
	}
}
