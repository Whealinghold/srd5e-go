package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func useRedirect(t *testing.T, roller character.Roller, opts ...character.Option) character.RedirectDeflectedAttack {
	t.Helper()
	return must(character.NewRedirectDeflectedAttack(mnkProgression(t), roller, opts...))
}

// Requirement: doc/specs/class-monk.md Table 4.6b, FR-056, FR-060, FR-061, FR-137 - RedirectDeflectedAttack rolls two Martial Arts dice (one request), deals their sum plus
// the Dexterity modifier (+3) on a failed save and nothing on a successful one, in the attack's damage type, against the Monk's Focus save DC (8 + 3 + the Proficiency Bonus),
// expends one Focus Point and emits DeflectedForceRedirected.
// Path: per case (rows of Table 4.6b).
func TestRedirectDeflectedAttackApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name            string
		level, roll     int
		outcome         d20.Outcome
		onFail, applied int
		dc              d20.DifficultyClass
		die             dice.Die
	}{
		{"failed save", 3, 5, d20.Failure, 8, 8, 13, dice.D6()}, {"successful save", 3, 5, d20.Success, 8, 0, 13, dice.D6()},
		{"d8 tier", 5, 16, d20.Failure, 19, 19, 14, dice.D8()},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mnkMonk(t, c.level)
			roller := dicetest.NewFixedRoller(c.roll/2, c.roll-c.roll/2)
			res, err := useRedirect(t, roller).Run(context.Background(), character.RedirectDeflectedAttackCommand{Character: in, Request: melee60(0), Outcome: c.outcome})
			r, ok := res.(character.RedirectApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			rr := r.Resolution()
			if rr.DamageOnFailedSave() != c.onFail || rr.DamageApplied() != c.applied || rr.DifficultyClass() != c.dc || rr.DamageType() != content.DamageSlashing ||
				mnkStateOf(t, r.Character()).FocusExpended() != 1 {
				t.Errorf("= %+v", rr)
			}
			ev, isEv := r.Events()[0].(character.DeflectedForceRedirected)
			if len(r.Events()) != 1 || !isEv || ev.DamageApplied() != c.applied || ev.DifficultyClass() != c.dc || ev.DamageType() != content.DamageSlashing || ev.CharacterID() != in.ID() {
				t.Errorf("events = %#v", r.Events())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(2, c.die, 0) {
				t.Errorf("requests = %v, want one 2%v", reqs, c.die)
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-057, FR-058, FR-059, FR-135, Table 4.11a - a redirect with damage left, a target out of range or unseen is notEligible (before the Focus
// Point check); with no Focus Point it is notEnoughFocusPoints; level 2 is levelTooLow; no Monk levels is notAMonk; none rolls or spends; an outcome that is neither success nor
// failure is an invalid command.
// Path: per case (unhappy: state error, boundary).
func TestRedirectDeflectedAttackNotAvailable(t *testing.T) {
	t.Parallel()
	far := melee60(0)
	far.Target.DistanceFeet = 6
	for _, c := range []struct {
		name string
		char character.Character
		req  character.MonkRedirectRequest
		want character.MonkReason
	}{
		{"damage left and no points", mnkWith(t, mnkMonk(t, 3), mnkState(3, false, 0)), melee60(2), character.MonkNotEligible},
		{"a melee target at 6 feet", mnkMonk(t, 3), far, character.MonkNotEligible},
		{"no points", mnkWith(t, mnkMonk(t, 3), mnkState(3, false, 0)), melee60(0), character.MonkNotEnoughFocusPoints},
		{"level 2", mnkMonk(t, 2), melee60(0), character.MonkLevelTooLow}, {"no Monk levels", mnkChar(t, cl("fighter", 5)), melee60(0), character.MonkNotAMonk},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(3, 3)
			res, err := useRedirect(t, roller).Run(context.Background(), character.RedirectDeflectedAttackCommand{Character: c.char, Request: c.req, Outcome: d20.Failure})
			if got, ok := res.(character.RedirectNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Fatalf("= %#v, %v; want not available %v", res, err, c.want)
			}
		})
	}
	res, err := useRedirect(t, dicetest.NewFixedRoller(3, 3)).Run(context.Background(), character.RedirectDeflectedAttackCommand{Character: mnkMonk(t, 3), Request: melee60(0)})
	wantInvalid(t, err, "outcome")
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
	foreign := must(character.WithFeatureState(mnkMonk(t, 3), character.MustFeatureStateKey[character.ChosenSubclass]("monk"), must(character.NewChosenSubclass("warrior_of_the_open_hand"))))
	_, err = useRedirect(t, dicetest.NewFixedRoller(3, 3)).Run(context.Background(), character.RedirectDeflectedAttackCommand{Character: foreign, Request: melee60(0), Outcome: d20.Failure})
	wantInvalid(t, err, "foreign state")
}

// Requirement: doc/specs/class-monk.md FR-126, FR-129, FR-147 - a failing dice port is a *srderr.PortError with the cause and no Focus Point is spent; equal results with the
// no-op and a recording telemetry; invalid construction, the zero use case, a zero Character and a cancelled context are refused before any signal; the signals of success,
// rejection and port failure.
// Path: per case (unhappy: collaborator failure, invalid input, cancellation).
func TestRedirectDeflectedAttackFailuresAndSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cmd := character.RedirectDeflectedAttackCommand{Character: mnkMonk(t, 3), Request: melee60(0), Outcome: d20.Failure}
	for _, c := range mnkFailingRollers() {
		res, err := useRedirect(t, c.roller).Run(ctx, cmd)
		mnkWantPortFailure(t, res, err, c.cause)
	}
	rec, opts := mnkTelemetry()
	withRec, err := useRedirect(t, dicetest.NewFixedRoller(2, 3), opts...).Run(ctx, cmd)
	plain, err2 := useRedirect(t, dicetest.NewFixedRoller(2, 3)).Run(ctx, cmd)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanRedirectDeflectedAttack, feature: "deflect_redirect", status: telemetry.StatusUnset, outcome: "success", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useRedirect(t, dicetest.NewFixedRoller(2, 3), opts...).Run(ctx, character.RedirectDeflectedAttackCommand{Character: mnkMonk(t, 3), Request: melee60(4), Outcome: d20.Failure})
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanRedirectDeflectedAttack, feature: "deflect_redirect", status: telemetry.StatusUnset, outcome: "failure", rejection: "not_eligible", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useRedirect(t, dicetest.NewFixedRoller(), opts...).Run(ctx, cmd)
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanRedirectDeflectedAttack, feature: "deflect_redirect", status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
	_, err = character.NewRedirectDeflectedAttack(character.MonkProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewRedirectDeflectedAttack(mnkProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.RedirectDeflectedAttack{}.Run(ctx, cmd)
	wantInvalid(t, err, "zero use case")
	_, err = useRedirect(t, dicetest.NewFixedRoller(1)).Run(ctx, character.RedirectDeflectedAttackCommand{Outcome: d20.Failure})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec2, opts2 := mnkTelemetry()
	res, err := useRedirect(t, dicetest.NewFixedRoller(1, 1), opts2...).Run(cancelled, cmd)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}

// Requirement: doc/specs/class-monk.md FR-141 - an Outcome that is neither success nor failure (zero, or above them) is refused before any signal and any roll.
// Path: per case (unhappy: invalid input, boundary).
func TestRedirectDeflectedAttackRefusesBadOutcomeBeforeAnySignal(t *testing.T) {
	t.Parallel()
	for _, bad := range []d20.Outcome{0, 3, -1} {
		rec, opts := mnkTelemetry()
		roller := dicetest.NewFixedRoller(3, 3)
		res, err := useRedirect(t, roller, opts...).Run(context.Background(), character.RedirectDeflectedAttackCommand{Character: mnkMonk(t, 3), Request: melee60(0), Outcome: bad})
		wantInvalid(t, err, "outcome")
		if res != nil || len(rec.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
			t.Errorf("outcome %d: result %v, spans %d, requests %v; want nothing", int(bad), res, len(rec.Tracer.Spans()), roller.Requests())
		}
	}
}
