package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func useFocus(t *testing.T, roller character.Roller, opts ...character.Option) character.UseFocusFeature {
	t.Helper()
	return must(character.NewUseFocusFeature(mnkProgression(t), roller, opts...))
}

// Requirement: doc/specs/class-monk.md FR-038, FR-088, FR-145 - UseFocusFeature spends the Focus Point and returns the grant; only a paid Patient Defense of
// level 10 or more rolls (exactly two Martial Arts dice, one request) and gains Temporary Hit Points; the new Character holds the expended point, the
// input is untouched and the FocusPointsExpended event names the feature.
// Path: per case.
func TestUseFocusFeatureApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name       string
		level      int
		use        character.MonkFocusUse
		roll       []int
		thp, spent int
	}{
		{"flurry of blows", 2, mnkUse(character.MonkFlurryOfBlows, true), nil, 0, 1},
		{"free patient defense", 10, mnkUse(character.MonkPatientDefense, false), nil, 0, 0},
		{"paid patient defense at 9", 9, mnkUse(character.MonkPatientDefense, true), nil, 0, 1},
		{"paid patient defense at 10", 10, mnkUse(character.MonkPatientDefense, true), []int{4, 7}, 11, 1},
		{"step of the wind", 5, mnkUse(character.MonkStepOfTheWind, true), nil, 0, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mnkWith(t, mnkMonk(t, c.level), mnkState(0, false, 0))
			roller := dicetest.NewFixedRoller(c.roll...)
			res, err := useFocus(t, roller).Run(context.Background(), character.UseFocusFeatureCommand{Character: in, Use: c.use})
			r, ok := res.(character.FocusFeatureApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if r.Resolution().TemporaryHitPoints() != c.thp || mnkStateOf(t, r.Character()).FocusExpended() != c.spent || mnkStateOf(t, in).FocusExpended() != 0 {
				t.Errorf("Temporary Hit Points %d, expended %d", r.Resolution().TemporaryHitPoints(), mnkStateOf(t, r.Character()).FocusExpended())
			}
			wantEvents := 0
			if c.spent > 0 {
				wantEvents = 1
				ev, isEv := r.Events()[0].(character.FocusPointsExpended)
				if !isEv || ev.Feature() != c.use.Feature || ev.Count() != 1 || ev.CharacterID() != in.ID() {
					t.Errorf("events = %#v", r.Events())
				}
			}
			if len(r.Events()) != wantEvents || (len(c.roll) > 0 && len(roller.Requests()) != 1) || (len(c.roll) == 0 && len(roller.Requests()) != 0) {
				t.Errorf("events %d, requests %v", len(r.Events()), roller.Requests())
			}
			if len(c.roll) > 0 {
				if want := dice.MustExpression(2, dice.D8(), 0); roller.Requests()[0] != want {
					t.Errorf("request = %v, want 2d8", roller.Requests()[0])
				}
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-023, FR-043, FR-135, FR-150 - a character with no Monk levels is notAMonk even with a stored MonkState; below level 2
// is levelTooLow; with no point available notEnoughFocusPoints; a refusal changes nothing, has no event and does not call the roller; a Monk with no
// stored state is read as the initial state and the new state is written; a foreign state under the key is an error.
// Path: per case (unhappy: state error).
func TestUseFocusFeatureNotAvailable(t *testing.T) {
	t.Parallel()
	flurry := mnkUse(character.MonkFlurryOfBlows, true)
	for _, c := range []struct {
		name string
		char character.Character
		use  character.MonkFocusUse
		want character.MonkReason
	}{
		{"no Monk levels", mnkChar(t, cl("wizard", 5)), flurry, character.MonkNotAMonk},
		{"no Monk levels with a stored state", mnkWith(t, mnkChar(t, cl("wizard", 5)), mnkState(0, false, 0)), flurry, character.MonkNotAMonk},
		{"level 1", mnkMonk(t, 1), flurry, character.MonkLevelTooLow},
		{"no points", mnkWith(t, mnkMonk(t, 3), mnkState(3, false, 0)), flurry, character.MonkNotEnoughFocusPoints},
		{"flurry without the point", mnkMonk(t, 3), mnkUse(character.MonkFlurryOfBlows, false), character.MonkNotEligible},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := useFocus(t, roller).Run(context.Background(), character.UseFocusFeatureCommand{Character: c.char, Use: c.use})
			if got, ok := res.(character.FocusFeatureNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Fatalf("= %#v, %v; want not available %v", res, err, c.want)
			}
		})
	}
	res, err := useFocus(t, dicetest.NewFixedRoller()).Run(context.Background(), character.UseFocusFeatureCommand{Character: mnkMonk(t, 2), Use: flurry})
	if r, ok := res.(character.FocusFeatureApplied); err != nil || !ok || mnkStateOf(t, r.Character()).FocusExpended() != 1 {
		t.Errorf("a Monk with no stored state = %#v, %v", res, err)
	}
	foreign := must(character.WithFeatureState(mnkMonk(t, 2), character.MustFeatureStateKey[character.ChosenSubclass]("monk"), must(character.NewChosenSubclass("warrior_of_the_open_hand"))))
	res, err = useFocus(t, dicetest.NewFixedRoller()).Run(context.Background(), character.UseFocusFeatureCommand{Character: foreign, Use: flurry})
	wantInvalid(t, err, "foreign state")
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
}

// Requirement: doc/specs/class-monk.md FR-126 - a roller that returns its error, panics or is exhausted is a *srderr.PortError for dice.Port with the cause
// reachable, the result is nil and the Character unchanged.
// Path: per case (unhappy: collaborator failure).
func TestUseFocusFeatureRollerFailures(t *testing.T) {
	t.Parallel()
	for _, c := range mnkFailingRollers() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := useFocus(t, c.roller).Run(context.Background(), character.UseFocusFeatureCommand{Character: mnkMonk(t, 10), Use: mnkUse(character.MonkPatientDefense, true)})
			mnkWantPortFailure(t, res, err, c.cause)
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-129, FR-147 - the result is identical with the no-op and with a recording telemetry; the constructor refuses the zero
// progression and a nil roller, and Run refuses the zero use case, a zero Character and a cancelled context before any signal.
// Path: per case (unhappy: invalid input, cancellation).
func TestUseFocusFeatureInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	good := character.UseFocusFeatureCommand{Character: mnkMonk(t, 3), Use: mnkUse(character.MonkFlurryOfBlows, true)}
	rec, opts := mnkTelemetry()
	withRec, err := useFocus(t, dicetest.NewFixedRoller(), opts...).Run(context.Background(), good)
	plain, err2 := useFocus(t, dicetest.NewFixedRoller()).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) || len(rec.Tracer.Spans()) != 1 {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	_, err = character.NewUseFocusFeature(character.MonkProgression{}, dicetest.NewFixedRoller())
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseFocusFeature(mnkProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseFocusFeature{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	_, err = useFocus(t, dicetest.NewFixedRoller()).Run(context.Background(), character.UseFocusFeatureCommand{})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec2, opts2 := mnkTelemetry()
	res, err := useFocus(t, dicetest.NewFixedRoller(), opts2...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}

// Requirement: doc/specs/class-monk.md FR-130, FR-147 and section 7 - the signals of a call: a success, a rejection and a port failure each record one span, one
// feature_used increment, one duration with srd5e.class monk and srd5e.feature the Focus feature used; a rejection sets no error status; a port failure sets
// error status, counts once on the errors counter with error.type and records no rejection.
// Path: per case (happy / unhappy: state error, collaborator failure).
func TestUseFocusFeatureSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("applied", func(t *testing.T) {
		t.Parallel()
		rec, opts := mnkTelemetry()
		if _, err := useFocus(t, dicetest.NewFixedRoller(), opts...).Run(ctx, character.UseFocusFeatureCommand{Character: mnkMonk(t, 3), Use: mnkUse(character.MonkStepOfTheWind, true)}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanUseFocusFeature, feature: "step_of_the_wind", status: telemetry.StatusUnset, outcome: "success", logged: true})
	})
	t.Run("rejected", func(t *testing.T) {
		t.Parallel()
		rec, opts := mnkTelemetry()
		if _, err := useFocus(t, dicetest.NewFixedRoller(), opts...).Run(ctx, character.UseFocusFeatureCommand{Character: mnkMonk(t, 1), Use: mnkUse(character.MonkFlurryOfBlows, true)}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanUseFocusFeature, feature: "flurry_of_blows", status: telemetry.StatusUnset, outcome: "failure", rejection: "level_too_low", logged: true})
	})
	t.Run("port failure", func(t *testing.T) {
		t.Parallel()
		rec, opts := mnkTelemetry()
		if _, err := useFocus(t, dicetest.NewFixedRoller(), opts...).Run(ctx, character.UseFocusFeatureCommand{Character: mnkMonk(t, 10), Use: mnkUse(character.MonkPatientDefense, true)}); err == nil {
			t.Fatal("want a port failure")
		}
		checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanUseFocusFeature, feature: "patient_defense", status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
	})
}
