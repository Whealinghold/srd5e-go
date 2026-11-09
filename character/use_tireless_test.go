package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func useTireless(t *testing.T, roller character.Roller, opts ...character.Option) character.UseTireless {
	t.Helper()
	return must(character.NewUseTireless(rgrProgression(t), roller, opts...))
}

// Requirement: doc/specs/class-ranger.md FR-072, FR-073, FR-100, T4 (SRD 5.2.1, "Level 10: Tireless", p. 59) - UseTireless rolls 1d8 through the roller and grants the roll plus the greater of 1 and the Wisdom modifier as Temporary Hit Points at the cost of a Magic action, spends one
// use and emits TirelessUsed with the Character id; rows TL-01 to TL-08 of T4, with the same result on every run for a fixed roller (including a natural 1 and a natural 8).
// Path: per case (T4): happy / unhappy (boundary, determinism).
func TestUseTirelessAccepted(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label                  string
		roll, wisdom, hp, uses int
	}{
		{"TL-01", 5, 16, 8, 3}, {"TL-02", 8, 20, 13, 5}, {"TL-03", 1, 10, 2, 1}, {"TL-04", 1, 8, 2, 1}, {"TL-05", 8, 1, 9, 1}, {"TL-06", 4, 12, 5, 1}, {"TL-07", 4, 14, 6, 2}, {"TL-08", 3, 30, 13, 10},
	} {
		wis := must(ability.NewScore(c.wisdom)).Modifier()
		char := rgrRanger(t, 10)
		before := char
		for run := 0; run < 2; run++ {
			roller := dicetest.NewFixedRoller(c.roll)
			res, err := useTireless(t, roller).Run(context.Background(), character.UseTirelessCommand{Character: char, Wisdom: wis})
			a, ok := res.(character.UseTirelessAccepted)
			if err != nil || !ok {
				t.Fatalf("%s: %#v, %v", c.label, res, err)
			}
			e := a.Events()[0].(character.TirelessUsed)
			if a.Rolled() != c.roll || a.TemporaryHitPoints() != c.hp || a.Cost() != character.RangerMagicAction || a.State().TirelessSpent() != 1 || !a.State().Equal(rgrStateOf(t, a.Character())) ||
				e.Rolled() != c.roll || e.TemporaryHitPoints() != c.hp || e.CharacterID() != char.ID() || !char.Equal(before) || len(roller.Requests()) != 1 || roller.Requests()[0].Die() != dice.D8() {
				t.Errorf("%s: %+v, event %+v, requests %v", c.label, a, e, roller.Requests())
			}
			a.Events()[0] = nil
			if a.Events()[0] == nil {
				t.Errorf("%s: Events returned shared storage", c.label)
			}
			if uses := character.TirelessUses(wis); uses != c.uses {
				t.Errorf("%s: uses = %d, want %d", c.label, uses, c.uses)
			}
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-071, FR-073, FR-074 (SRD 5.2.1, "Level 10: Tireless", p. 59) - Tireless below level 10 is rejected with feature_not_available, with no use left with no_uses_remaining (a Wisdom decrease below the expended count leaves none) and rolls no die in either case; the last use is
// accepted; a character with no Ranger levels is NotARanger and rolls nothing.
// Path: per case (unhappy: boundary, state error).
func TestUseTirelessRejected(t *testing.T) {
	t.Parallel()
	spent := func(n int) character.Character {
		return rgrWith(t, rgrRanger(t, 10), rgrState(t, character.RangerStateParts{TirelessSpent: n}))
	}
	for _, c := range []struct {
		name   string
		char   character.Character
		wisdom int
		want   character.RangerReason
	}{
		{"level 9", rgrRanger(t, 9), 3, character.RangerFeatureNotAvailable},
		{"all uses expended", spent(3), 3, character.RangerNoUsesRemaining},
		{"more expended than the maximum", spent(5), 3, character.RangerNoUsesRemaining},
		{"Wisdom dropped below the expended count", spent(2), 2, character.RangerNoUsesRemaining},
		{"the minimum of one use", spent(1), -1, character.RangerNoUsesRemaining},
	} {
		roller := dicetest.NewFixedRoller(5)
		res, err := useTireless(t, roller).Run(context.Background(), character.UseTirelessCommand{Character: c.char, Wisdom: modifierOf(t, c.wisdom)})
		if got, ok := res.(character.UseTirelessRejected); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
			t.Errorf("%s = %#v, %v; want rejected %v with no roll (requests %v)", c.name, res, err, c.want, roller.Requests())
		}
	}
	for _, n := range []int{0, 2} {
		res, err := useTireless(t, dicetest.NewFixedRoller(5)).Run(context.Background(), character.UseTirelessCommand{Character: spent(n), Wisdom: modifierOf(t, 3)})
		if _, ok := res.(character.UseTirelessAccepted); err != nil || !ok {
			t.Errorf("with %d spent of 3: %#v, %v; want accepted (the last use is accepted)", n, res, err)
		}
	}
	roller := dicetest.NewFixedRoller(5)
	res, err := useTireless(t, roller).Run(context.Background(), character.UseTirelessCommand{Character: fgtFighter(t, 10), Wisdom: modifierOf(t, 3)})
	if _, ok := res.(character.UseTirelessNotARanger); err != nil || !ok || len(roller.Requests()) != 0 {
		t.Errorf("a Fighter = %#v, %v", res, err)
	}
}

// Requirement: doc/specs/class-ranger.md FR-077, FR-107 - a roller that returns its error, panics, is exhausted or answers outside 1 to 8 (0 or 9) is a *srderr.PortError for dice.Port with the cause reachable, the result is nil, no use is spent and the Character is unchanged;
// the failure is recorded once as an error span, one feature_used increment (outcome failure, no rejection) and one errors increment with error.type, and is never logged.
// Path: per case (unhappy: collaborator failure, invalid input, telemetry on failure).
func TestUseTirelessRollerFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("roller down")
	for _, c := range []struct {
		name   string
		roller character.Roller
		cause  error
	}{
		{"an error", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), boom},
		{"exhausted", dicetest.NewFixedRoller(), dicetest.ErrExhausted},
		{"a panic", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(boom) }), boom},
		{"a roll of 9", dicetest.NewFixedRoller(9), dicetest.ErrValueOutOfRange},
		{"a roll of 0", dicetest.NewFixedRoller(0), dicetest.ErrValueOutOfRange},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			char := rgrRanger(t, 10)
			res, err := useTireless(t, c.roller, opts...).Run(context.Background(), character.UseTirelessCommand{Character: char, Wisdom: modifierOf(t, 3)})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
				t.Fatalf("= %v, %v; want a PortError for dice with the cause", res, err)
			}
			if _, stored, _ := character.ReadFeatureState(char.FeatureStates(), character.RangerStateKey()); stored {
				t.Error("a state was written")
			}
			w := rgrWant(character.SpanUseTireless, "tireless", "")
			w.status, w.outcome, w.logged, w.errorType = telemetry.StatusError, "failure", false, srderr.ErrorType(err)
			checkFeatureCall(t, rec, w)
		})
	}
}

// Requirement: doc/specs/class-ranger.md FR-104, FR-105 - UseTireless records one span named srd5e.character.use_tireless with the class ranger and the feature tireless, one feature_used increment and one duration; a rejection counts the reason with no error status and not_a_ranger counts that;
// a foreign state is an error recorded on the span only; the result is identical with the no-op and with a recording telemetry; the constructor refuses the zero progression and a nil roller, and Run the zero use case and a zero Character, and a cancelled context, before any signal.
// Path: per case (happy / unhappy: invalid input, state error, determinism, cancellation).
func TestUseTirelessSignalsAndInvalidInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	good := character.UseTirelessCommand{Character: rgrRanger(t, 10), Wisdom: modifierOf(t, 3)}
	rec, opts := fgtTelemetry()
	withRec, err := useTireless(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, good)
	plain, err2 := useTireless(t, dicetest.NewFixedRoller(4)).Run(ctx, good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	checkFeatureCall(t, rec, rgrWant(character.SpanUseTireless, "tireless", ""))
	for _, c := range []struct {
		rejection string
		cmd       character.UseTirelessCommand
	}{
		{"feature_not_available", character.UseTirelessCommand{Character: rgrRanger(t, 9), Wisdom: modifierOf(t, 3)}},
		{"no_uses_remaining", character.UseTirelessCommand{Character: rgrWith(t, rgrRanger(t, 10), rgrState(t, character.RangerStateParts{TirelessSpent: 3})), Wisdom: modifierOf(t, 3)}},
		{"not_a_ranger", character.UseTirelessCommand{Character: fgtFighter(t, 10), Wisdom: modifierOf(t, 3)}},
	} {
		rec, opts := fgtTelemetry()
		if _, err := useTireless(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, c.cmd); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, rgrWant(character.SpanUseTireless, "tireless", c.rejection))
	}
	rec, opts = fgtTelemetry()
	res, err := useTireless(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, character.UseTirelessCommand{Character: rgrForeign(t, rgrRanger(t, 10))})
	wantInvalid(t, err, "foreign state")
	if spans := rec.Tracer.Spans(); res != nil || len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Errorf("foreign state recorded %+v", spans)
	}
	_, err = character.NewUseTireless(character.RangerProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseTireless(rgrProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseTireless{}.Run(ctx, good)
	wantInvalid(t, err, "zero use case")
	_, err = useTireless(t, dicetest.NewFixedRoller(1)).Run(ctx, character.UseTirelessCommand{})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec, opts = fgtTelemetry()
	roller := dicetest.NewFixedRoller(1)
	res, err = useTireless(t, roller, opts...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec.Tracer.Spans()))
	}
}
