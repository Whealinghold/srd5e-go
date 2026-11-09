package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func trinketUseCase(t *testing.T, roller character.Roller, opts ...character.Option) character.RollTrinket {
	t.Helper()
	uc, err := character.NewRollTrinket(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-094, FR-095, Table TR (SRD 5.2.1, "Trinkets",
// pp. 26-27) - one 1d100 is rolled through the roller and the result holds the roll and the matching entry,
// for the rolls 1, 34, 35, 72, 73, 99 and 100 (the 00 entry), with no Character involved.
// Path: per case.
func TestRollTrinketTableTR(t *testing.T) {
	t.Parallel()
	table := srdTrinketTable(t)
	_, texts := srdTrinkets(t)
	for _, roll := range []int{1, 34, 35, 72, 73, 99, 100} {
		roller := dicetest.NewFixedRoller(roll)
		res, err := trinketUseCase(t, roller).Roll(context.Background(), character.RollTrinketCommand{Table: table})
		r, ok := res.(character.RollTrinketRolled)
		if err != nil || !ok || r.Roll() != roll || r.Trinket().Description() != texts[roll-1] {
			t.Errorf("roll %d: = %#v, %v, want %q", roll, res, err, texts[roll-1])
		}
		if got := roller.Requests(); len(got) != 1 || got[0] != dice.MustExpression(1, dice.D100(), 0) {
			t.Errorf("roll %d: requests = %v, want exactly one 1d100", roll, got)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-096 - a roll of 0 or above 100, a roller error and a
// roller that answers another expression are a *srderr.PortError for dice.Port with the cause reachable, and no
// result is built.
// Path: per case.
func TestRollTrinketRollerFailures(t *testing.T) {
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
		{"roll 101", dicetest.NewFixedRoller(101), dicetest.ErrValueOutOfRange},
		{"roller error", dicetest.NewFixedRoller(), dicetest.ErrExhausted},
		{"another expression", wrong, srderr.ErrContractViolation},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := trinketUseCase(t, c.roller).Roll(context.Background(), character.RollTrinketCommand{Table: srdTrinketTable(t)})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
				t.Errorf("= %v, %v", res, err)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-097, FR-094 - the command and table are not altered, the
// same table and roll give an equal result on every call, and nothing is remembered between calls.
// Path: unhappy (immutability, determinism).
func TestRollTrinketIsDeterministicAndKeepsNoRecord(t *testing.T) {
	t.Parallel()
	table := srdTrinketTable(t)
	uc := trinketUseCase(t, dicetest.NewFixedRoller(35, 35, 12))
	a, _ := uc.Roll(context.Background(), character.RollTrinketCommand{Table: table})
	b, _ := uc.Roll(context.Background(), character.RollTrinketCommand{Table: table})
	c, _ := uc.Roll(context.Background(), character.RollTrinketCommand{Table: table})
	if a != b || a == c || table != srdTrinketTable(t) {
		t.Errorf("results %#v %#v %#v", a, b, c)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-121, FR-122, section 7 - a rolled Trinket is
// success/rolled, a roller failure is recorded once with error.type and no result, the duration carries the
// operation only, nothing is logged and a recorder and the Noop give equal results.
// Path: per case.
func TestRollTrinketSignals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		roller  character.Roller
		attrs   map[string]any
		status  telemetry.Status
		errors  int
		wantErr bool
	}{
		{"rolled", dicetest.NewFixedRoller(7), map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "rolled"}, telemetry.StatusUnset, 0, false},
		{"roller failure", dicetest.NewFixedRoller(), map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"}, telemetry.StatusError, 1, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			cmd := character.RollTrinketCommand{Table: srdTrinketTable(t)}
			res, err := trinketUseCase(t, c.roller, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).Roll(context.Background(), cmd)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v", err)
			}
			plain, _ := trinketUseCase(t, dicetest.NewFixedRoller(7)).Roll(context.Background(), cmd)
			if !c.wantErr && res != plain {
				t.Error("Noop and a recorder must give equal results")
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanAdvancementTrinket || spans[0].Status != c.status || len(spans[0].Errors) != c.errors {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("trinket", c.attrs))
			d := onlyOne(t, rec, character.MetricDuration)
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "trinket"})
			if d.Value != 0.25 {
				t.Errorf("duration = %+v", d)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-123 - the zero table, the zero use case and a nil
// roller are invalid-input errors, a cancelled context is ctx.Err(), and none rolls or emits a signal.
// Path: per case.
func TestRollTrinketInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller(7)
	uc := trinketUseCase(t, roller, character.WithTelemetry(rec.Telemetry()))
	if _, err := uc.Roll(context.Background(), character.RollTrinketCommand{}); !isInvalid(err) {
		t.Errorf("zero table: %v", err)
	}
	if _, err := (character.RollTrinket{}).Roll(context.Background(), character.RollTrinketCommand{Table: srdTrinketTable(t)}); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := character.NewRollTrinket(nil); !isInvalid(err) {
		t.Errorf("nil roller: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Roll(ctx, character.RollTrinketCommand{Table: srdTrinketTable(t)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()) + len(roller.Requests()); n != 0 {
		t.Errorf("%d signals or rolls before validation", n)
	}
}
