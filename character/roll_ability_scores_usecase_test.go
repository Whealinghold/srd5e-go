package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func rollScoresUseCase(t *testing.T, roller character.Roller, opts ...character.Option) character.RollAbilityScores {
	t.Helper()
	uc, err := character.NewRollAbilityScores(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

// Requirement: doc/specs/character-creation.md FR-057, FR-058 and the table "Random Generation cases" - exactly six 4d6 requests are made in order,
// each total is the highest three dice (ordinary, order-free, all ones, all sixes, a tie at the cut, the lowest die dropped), and
// the totals and dice are returned in roll order.
// Path: per case.
func TestRollAbilityScoresTotals(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(6, 5, 4, 1, 1, 4, 6, 5, 1, 1, 1, 1, 6, 6, 6, 6, 3, 3, 3, 2, 2, 6, 6, 6)
	res, err := rollScoresUseCase(t, roller).Run(context.Background(), character.RollAbilityScoresCommand{})
	got, ok := res.(character.RollAbilityScoresSucceeded)
	if err != nil || !ok {
		t.Fatalf("= %v, %v", res, err)
	}
	if want := []int{15, 15, 3, 18, 9, 18}; !slices.Equal(got.Scores().Totals(), want) {
		t.Errorf("totals = %v, want %v", got.Scores().Totals(), want)
	}
	if first := got.Scores().Scores()[0]; !slices.Equal(first.Dice(), []int{6, 5, 4, 1}) || first.Total() != 15 {
		t.Errorf("first roll = %v, %d", first.Dice(), first.Total())
	}
	want4d6 := dice.MustExpression(4, dice.D6(), 0)
	if reqs := roller.Requests(); len(reqs) != 6 || !reflect.DeepEqual(reqs, slices.Repeat([]dice.Expression{want4d6}, 6)) {
		t.Errorf("requests = %v", reqs)
	}
	got.Scores().Scores()[0] = character.RolledAbilityScore{}
	got.Scores().Totals()[0] = 0
	got.Scores().Scores()[1].Dice()[0] = 0
	if got.Scores().Totals()[0] != 15 || got.Scores().Scores()[1].Dice()[0] != 1 {
		t.Error("a returned slice reached the stored value")
	}
}

// Requirement: doc/specs/character-creation.md FR-058 - with the minimum and maximum rollers every total is 3 and 18.
// Path: unhappy (boundary).
func TestRollAbilityScoresExtremes(t *testing.T) {
	t.Parallel()
	for roller, want := range map[character.Roller]int{dicetest.NewMinimumRoller(): 3, dicetest.NewMaximumRoller(): 18} {
		res, _ := rollScoresUseCase(t, roller).Run(context.Background(), character.RollAbilityScoresCommand{})
		for _, total := range res.(character.RollAbilityScoresSucceeded).Scores().Totals() {
			if total != want {
				t.Errorf("total = %d, want %d", total, want)
			}
		}
	}
}

// Requirement: doc/specs/character-creation.md FR-097, FR-099, FR-100 - a roll is success/rolled, records each of the six totals in the
// ability_score_roll histogram, carries no method attribute, logs nothing and gives the same result as the Noop; a roller failure is
// recorded once with error.type and records no total.
// Path: per case.
func TestRollAbilityScoresSignals(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	roller := dicetest.NewFixedRoller(6, 5, 4, 1, 1, 4, 6, 5, 1, 1, 1, 1, 6, 6, 6, 6, 3, 3, 3, 2, 2, 6, 6, 6)
	res, err := rollScoresUseCase(t, roller, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).Run(context.Background(), character.RollAbilityScoresCommand{})
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := rollScoresUseCase(t, dicetest.NewFixedRoller(6, 5, 4, 1, 1, 4, 6, 5, 1, 1, 1, 1, 6, 6, 6, 6, 3, 3, 3, 2, 2, 6, 6, 6)).Run(context.Background(), character.RollAbilityScoresCommand{})
	if !reflect.DeepEqual(res, plain) {
		t.Error("Noop and a recorder must give equal results")
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Name != character.SpanRollAbilityScores || spans[0].Status != telemetry.StatusUnset {
		t.Fatalf("spans = %+v", spans)
	}
	sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("roll_ability_scores", map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "rolled"}))
	var totals []float64
	for _, m := range rec.Meter.MeasurementsNamed(character.MetricAbilityScoreRoll) {
		totals = append(totals, m.Value)
		if len(m.Attrs) != 0 {
			t.Errorf("ability_score_roll carries attributes %v", m.Attrs)
		}
	}
	if !slices.Equal(totals, []float64{15, 15, 3, 18, 9, 18}) {
		t.Errorf("histogram = %v", totals)
	}
	noLogsOrViolations(t, rec)

	fail := telemetrytest.NewRecorder()
	_, err = rollScoresUseCase(t, dicetest.NewFixedRoller(6, 5, 4, 1), character.WithTelemetry(fail.Telemetry())).Run(context.Background(), character.RollAbilityScoresCommand{})
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, dicetest.ErrExhausted) {
		t.Fatalf("err = %v", err)
	}
	spans = fail.Tracer.Spans()
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
		t.Errorf("spans = %+v", spans)
	}
	sameAttrs(t, "operations", onlyOne(t, fail, character.MetricOperations).Attrs, opAttrs("roll_ability_scores", map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"}))
	if n := len(fail.Meter.MeasurementsNamed(character.MetricAbilityScoreRoll)); n != 0 {
		t.Errorf("%d totals recorded for a failed roll", n)
	}
	noLogsOrViolations(t, fail)
}

// Requirement: doc/specs/character-creation.md FR-008 - a roller that answers another expression is a dice port failure.
// Path: unhappy (collaborator failure).
func TestRollAbilityScoresWrongExpression(t *testing.T) {
	t.Parallel()
	wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D4(), 0))
	})
	res, err := rollScoresUseCase(t, wrong).Run(context.Background(), character.RollAbilityScoresCommand{})
	if res != nil || !errors.Is(err, srderr.ErrContractViolation) {
		t.Errorf("= %v, %v", res, err)
	}
}

// Requirement: doc/specs/character-creation.md section 6 - a nil roller and the zero use case are invalid-input errors and a cancelled
// context is ctx.Err(), none rolling or emitting a signal.
// Path: unhappy (invalid input).
func TestRollAbilityScoresInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := character.NewRollAbilityScores(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller error = %v", err)
	}
	if _, err := (character.RollAbilityScores{}).Run(context.Background(), character.RollAbilityScoresCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller()
	if _, err := rollScoresUseCase(t, roller, character.WithTelemetry(rec.Telemetry())).Run(ctx, character.RollAbilityScoresCommand{}); !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled error = %v", err)
	}
}
