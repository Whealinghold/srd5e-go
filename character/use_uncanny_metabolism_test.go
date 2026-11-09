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

func useUncanny(t *testing.T, roller character.Roller, opts ...character.Option) character.UseUncannyMetabolism {
	t.Helper()
	return must(character.NewUseUncannyMetabolism(mnkProgression(t), roller, opts...))
}

// Requirement: doc/specs/class-monk.md Table 4.5a, FR-046, FR-047 - UseUncannyMetabolism rolls one Martial Arts die of the level (one request), returns the Hit Points to
// regain (Monk level plus the roll, uncapped), the Focus Points regained and the new Character with all points regained and the feature used; the
// UncannyMetabolismUsed event carries them; the input is untouched.
// Path: per case (rows of Table 4.5a).
func TestUseUncannyMetabolismApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name              string
		level, roll, want int
		die               dice.Die
	}{{"d6 low", 2, 1, 3, dice.D6()}, {"d6 high", 2, 6, 8, dice.D6()}, {"d8", 5, 8, 13, dice.D8()}, {"d10", 11, 10, 21, dice.D10()}, {"d12", 20, 12, 32, dice.D12()}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mnkWith(t, mnkMonk(t, c.level), mnkState(2, false, 1))
			roller := dicetest.NewFixedRoller(c.roll)
			res, err := useUncanny(t, roller).Run(context.Background(), character.UseUncannyMetabolismCommand{Character: in, RolledInitiative: true})
			r, ok := res.(character.UncannyMetabolismApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if r.HitPointsToRegain() != c.want || r.Rolled() != c.roll || r.FocusPointsRegained() != 2 || mnkStateOf(t, r.Character()) != mnkState(0, true, 1) || mnkStateOf(t, in) != mnkState(2, false, 1) {
				t.Errorf("= %d Hit Points, %d regained, state %v", r.HitPointsToRegain(), r.FocusPointsRegained(), mnkStateOf(t, r.Character()))
			}
			ev, isEv := r.Events()[0].(character.UncannyMetabolismUsed)
			if len(r.Events()) != 1 || !isEv || ev.Rolled() != c.roll || ev.HitPointsToRegain() != c.want || ev.FocusPointsRegained() != 2 || ev.CharacterID() != in.ID() {
				t.Errorf("events = %#v", r.Events())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, c.die, 0) {
				t.Errorf("requests = %v, want one 1%v", reqs, c.die)
			}
			r.Events()[0] = nil
			if r.Events()[0] == nil {
				t.Error("Events returned a shared slice")
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-048, FR-049, FR-050, FR-023, FR-135 - a second use before a Long Rest is alreadyUsed, a trigger that is not Initiative is
// notEligible (also when used), a Long Rest makes it usable again and a Short Rest does not; no Monk levels is notAMonk; a refusal changes nothing and does not roll.
// Path: per case (unhappy: state error).
func TestUseUncannyMetabolismNotAvailable(t *testing.T) {
	t.Parallel()
	used := mnkWith(t, mnkMonk(t, 5), mnkState(1, true, 0))
	for _, c := range []struct {
		name string
		char character.Character
		init bool
		want character.MonkReason
	}{
		{"already used", used, true, character.MonkAlreadyUsed}, {"used and not Initiative", used, false, character.MonkNotEligible},
		{"not Initiative", mnkMonk(t, 5), false, character.MonkNotEligible}, {"level 1", mnkMonk(t, 1), true, character.MonkLevelTooLow},
		{"no Monk levels", mnkChar(t, cl("fighter", 5)), true, character.MonkNotAMonk},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := useUncanny(t, roller).Run(context.Background(), character.UseUncannyMetabolismCommand{Character: c.char, RolledInitiative: c.init})
			if got, ok := res.(character.UncannyMetabolismNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Fatalf("= %#v, %v; want not available %v", res, err, c.want)
			}
		})
	}
	state := mnkStateOf(t, used)
	for kind, wantUsed := range map[character.RestKind]bool{character.ShortRest: true, character.LongRest: false} {
		after := mnkWith(t, used, state.AfterRest(kind))
		res, err := useUncanny(t, dicetest.NewFixedRoller(5)).Run(context.Background(), character.UseUncannyMetabolismCommand{Character: after, RolledInitiative: true})
		if _, notAvail := res.(character.UncannyMetabolismNotAvailable); err != nil || notAvail != wantUsed {
			t.Errorf("after rest %v: %#v, %v; want not available = %v", kind, res, err, wantUsed)
		}
	}
	foreign := must(character.WithFeatureState(mnkMonk(t, 2), character.MustFeatureStateKey[character.ChosenSubclass]("monk"), must(character.NewChosenSubclass("warrior_of_the_open_hand"))))
	res, err := useUncanny(t, dicetest.NewFixedRoller(1)).Run(context.Background(), character.UseUncannyMetabolismCommand{Character: foreign, RolledInitiative: true})
	wantInvalid(t, err, "foreign state")
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
}

// Requirement: doc/specs/class-monk.md FR-126, FR-127 - a failing dice port is a *srderr.PortError with the cause and a nil result; all-1 and all-maximum rolls give the
// table minimum and maximum.
// Path: per case (unhappy: collaborator failure, determinism).
func TestUseUncannyMetabolismRollerFailures(t *testing.T) {
	t.Parallel()
	for _, c := range mnkFailingRollers() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := useUncanny(t, c.roller).Run(context.Background(), character.UseUncannyMetabolismCommand{Character: mnkMonk(t, 5), RolledInitiative: true})
			mnkWantPortFailure(t, res, err, c.cause)
		})
	}
	for roller, want := range map[character.Roller]int{dicetest.NewMinimumRoller(): 6, dicetest.NewMaximumRoller(): 13} {
		a, _ := useUncanny(t, roller).Run(context.Background(), character.UseUncannyMetabolismCommand{Character: mnkMonk(t, 5), RolledInitiative: true})
		if r, ok := a.(character.UncannyMetabolismApplied); !ok || r.HitPointsToRegain() != want {
			t.Errorf("= %#v, want %d Hit Points", a, want)
		}
	}
}

// Requirement: doc/specs/class-monk.md FR-129, FR-147 - equal results with the no-op and a recording telemetry; the constructor, the zero use case, a zero Character
// and a cancelled context are refused before any signal.
// Path: per case (unhappy: invalid input, cancellation).
func TestUseUncannyMetabolismInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	good := character.UseUncannyMetabolismCommand{Character: mnkMonk(t, 5), RolledInitiative: true}
	rec, opts := mnkTelemetry()
	withRec, err := useUncanny(t, dicetest.NewFixedRoller(4), opts...).Run(context.Background(), good)
	plain, err2 := useUncanny(t, dicetest.NewFixedRoller(4)).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) || len(rec.Tracer.Spans()) != 1 {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	_, err = character.NewUseUncannyMetabolism(character.MonkProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseUncannyMetabolism(mnkProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseUncannyMetabolism{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	_, err = useUncanny(t, dicetest.NewFixedRoller(1)).Run(context.Background(), character.UseUncannyMetabolismCommand{})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec2, opts2 := mnkTelemetry()
	res, err := useUncanny(t, dicetest.NewFixedRoller(1), opts2...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}

// Requirement: doc/specs/class-monk.md FR-130, FR-147 - success, rejection and port failure signals of UseUncannyMetabolism.
// Path: per case (happy / unhappy: state error, collaborator failure).
func TestUseUncannyMetabolismSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	run := func(t *testing.T, c character.Character, roller character.Roller, w wantCall) {
		t.Helper()
		rec, opts := mnkTelemetry()
		_, _ = useUncanny(t, roller, opts...).Run(ctx, character.UseUncannyMetabolismCommand{Character: c, RolledInitiative: true})
		w.class, w.span, w.feature = "monk", character.SpanUseUncannyMetabolism, "uncanny_metabolism"
		checkFeatureCall(t, rec, w)
	}
	t.Run("applied", func(t *testing.T) {
		t.Parallel()
		run(t, mnkMonk(t, 5), dicetest.NewFixedRoller(3), wantCall{status: telemetry.StatusUnset, outcome: "success", logged: true})
	})
	t.Run("already used", func(t *testing.T) {
		t.Parallel()
		run(t, mnkWith(t, mnkMonk(t, 5), mnkState(0, true, 0)), dicetest.NewFixedRoller(3), wantCall{status: telemetry.StatusUnset, outcome: "failure", rejection: "already_used", logged: true})
	})
	t.Run("port failure", func(t *testing.T) {
		t.Parallel()
		run(t, mnkMonk(t, 5), dicetest.NewFixedRoller(), wantCall{status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
	})
}
