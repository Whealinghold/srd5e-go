package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func useWholeness(t *testing.T, roller character.Roller, opts ...character.Option) character.UseWholenessOfBody {
	t.Helper()
	return must(character.NewUseWholenessOfBody(mnkProgression(t), roller, opts...))
}

// Requirement: doc/specs/class-monk.md Table 4.10a, FR-110, FR-111 - UseWholenessOfBody rolls one Martial Arts die (d8 at level 6), returns the roll plus the Character's Wisdom
// modifier (+3) as Hit Points to regain, uncapped, at least 1, spends one of the 3 uses and emits WholenessOfBodyUsed; the minimum of 1 is reached with a Wisdom of 8.
// Path: per case (rows of Table 4.10a).
func TestUseWholenessOfBodyApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name         string
		roll, wisdom int
		want         int
	}{{"typical", 3, 16, 6}, {"high", 6, 20, 11}, {"sum below the minimum", 1, 8, 1}, {"zero modifier", 1, 10, 1}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := restored(t, 355000, 200, scoresOf(t, 10, 16, 10, 10, c.wisdom, 10), cl("monk", 6))
			roller := dicetest.NewFixedRoller(c.roll)
			res, err := useWholeness(t, roller).Run(context.Background(), character.UseWholenessOfBodyCommand{Character: in, Subclass: openHand})
			r, ok := res.(character.WholenessOfBodyApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if r.HitPointsToRegain() != c.want || r.Rolled() != c.roll || mnkStateOf(t, r.Character()).WholenessOfBodySpent() != 1 {
				t.Errorf("= %d Hit Points, state %v", r.HitPointsToRegain(), mnkStateOf(t, r.Character()))
			}
			ev, isEv := r.Events()[0].(character.WholenessOfBodyUsed)
			if len(r.Events()) != 1 || !isEv || ev.Rolled() != c.roll || ev.HitPointsToRegain() != c.want || ev.CharacterID() != in.ID() {
				t.Errorf("events = %#v", r.Events())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D8(), 0) {
				t.Errorf("requests = %v, want one 1d8", reqs)
			}
			if in.AbilityScores().Modifier(ability.Wisdom).Value() != (c.wisdom-10)>>1 {
				t.Error("fixture wisdom changed")
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-112, FR-134, FR-135, FR-148, Table 4.11a - with every use spent it is noUsesRemaining; without the Warrior of the Open Hand it is
// notEligible (before the uses, and a level 5 Monk gets levelTooLow first); no Monk levels is notAMonk; a refusal changes nothing and does not roll; a Long Rest restores
// the uses and a Short Rest does not.
// Path: per case (unhappy: state error, rule interaction).
func TestUseWholenessOfBodyNotAvailable(t *testing.T) {
	t.Parallel()
	spent := mnkWith(t, mnkMonk(t, 6), mnkState(0, false, 3))
	for _, c := range []struct {
		name     string
		char     character.Character
		subclass content.SubclassID
		want     character.MonkReason
	}{
		{"all uses spent", spent, openHand, character.MonkNoUsesRemaining}, {"no subclass and no uses", spent, "", character.MonkNotEligible},
		{"level 5 with no subclass", mnkMonk(t, 5), "", character.MonkLevelTooLow}, {"no Monk levels", mnkChar(t, cl("fighter", 6)), openHand, character.MonkNotAMonk},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(4)
			res, err := useWholeness(t, roller).Run(context.Background(), character.UseWholenessOfBodyCommand{Character: c.char, Subclass: c.subclass})
			if got, ok := res.(character.WholenessOfBodyNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Fatalf("= %#v, %v; want not available %v", res, err, c.want)
			}
		})
	}
	state := mnkStateOf(t, spent)
	for kind, wantRefused := range map[character.RestKind]bool{character.ShortRest: true, character.LongRest: false} {
		after := mnkWith(t, spent, state.AfterRest(kind))
		res, err := useWholeness(t, dicetest.NewFixedRoller(4)).Run(context.Background(), character.UseWholenessOfBodyCommand{Character: after, Subclass: openHand})
		if _, refused := res.(character.WholenessOfBodyNotAvailable); err != nil || refused != wantRefused {
			t.Errorf("after rest %v: %#v, %v; want refused = %v", kind, res, err, wantRefused)
		}
	}
	foreign := must(character.WithFeatureState(mnkMonk(t, 6), character.MustFeatureStateKey[character.ChosenSubclass]("monk"), must(character.NewChosenSubclass("warrior_of_the_open_hand"))))
	res, err := useWholeness(t, dicetest.NewFixedRoller(4)).Run(context.Background(), character.UseWholenessOfBodyCommand{Character: foreign, Subclass: openHand})
	wantInvalid(t, err, "foreign state")
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
}

// Requirement: doc/specs/class-monk.md FR-126, FR-129, FR-147 - a failing dice port is a *srderr.PortError with the cause; equal results with the no-op and a recording
// telemetry; invalid construction, the zero use case, a zero Character and a cancelled context are refused before any signal; the signals of success, rejection and port failure.
// Path: per case (unhappy: collaborator failure, invalid input, cancellation).
func TestUseWholenessOfBodyFailuresAndSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cmd := character.UseWholenessOfBodyCommand{Character: mnkMonk(t, 6), Subclass: openHand}
	for _, c := range mnkFailingRollers() {
		res, err := useWholeness(t, c.roller).Run(ctx, cmd)
		mnkWantPortFailure(t, res, err, c.cause)
	}
	rec, opts := mnkTelemetry()
	withRec, err := useWholeness(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, cmd)
	plain, err2 := useWholeness(t, dicetest.NewFixedRoller(4)).Run(ctx, cmd)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanUseWholenessOfBody, feature: "wholeness_of_body", status: telemetry.StatusUnset, outcome: "success", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useWholeness(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, character.UseWholenessOfBodyCommand{Character: mnkMonk(t, 6)})
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanUseWholenessOfBody, feature: "wholeness_of_body", status: telemetry.StatusUnset, outcome: "failure", rejection: "not_eligible", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useWholeness(t, dicetest.NewFixedRoller(), opts...).Run(ctx, cmd)
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanUseWholenessOfBody, feature: "wholeness_of_body", status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
	_, err = character.NewUseWholenessOfBody(character.MonkProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseWholenessOfBody(mnkProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseWholenessOfBody{}.Run(ctx, cmd)
	wantInvalid(t, err, "zero use case")
	_, err = useWholeness(t, dicetest.NewFixedRoller(1)).Run(ctx, character.UseWholenessOfBodyCommand{})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec2, opts2 := mnkTelemetry()
	res, err := useWholeness(t, dicetest.NewFixedRoller(1), opts2...).Run(cancelled, cmd)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}
