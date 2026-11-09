package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func useSecondWind(t *testing.T, roller character.Roller, opts ...character.Option) character.UseSecondWind {
	t.Helper()
	return must(character.NewUseSecondWind(fgtProgression(t), roller, opts...))
}

func runSecondWind(t *testing.T, c character.Character, roll int) (character.UseSecondWindResult, *dicetest.FixedRoller) {
	t.Helper()
	roller := dicetest.NewFixedRoller(roll)
	res, err := useSecondWind(t, roller).Run(context.Background(), character.UseSecondWindCommand{Character: c})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res, roller
}

// Requirement: doc/specs/class-fighter.md Table C, FR-051, FR-052, FR-053, FR-054, FR-055, FR-059, FR-107 - UseSecondWind rolls one 1d10 and returns the
// Hit Points to regain (roll plus the Fighter level, not the total level, uncapped), the new Character with exactly one Second Wind use more expended and
// the other counts untouched, the Bonus Action cost, the Tactical Shift offer from level 5 and the SecondWindUsed event; at the roll extremes too.
// Path: per case (rows SW1 to SW6).
func TestUseSecondWindApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		char      character.Character
		roll, hit int
		tactical  bool
	}{
		{"SW1 level 1, roll 1", fgtFighter(t, 1), 1, 2, false},
		{"SW2 level 1, roll 10", fgtFighter(t, 1), 10, 11, false},
		{"SW3 level 5", fgtFighter(t, 5), 6, 11, true},
		{"SW4 level 20, roll 1", fgtFighter(t, 20), 1, 21, true},
		{"SW5 level 20, roll 10", fgtFighter(t, 20), 10, 30, true},
		{"SW6 Fighter 3 with Wizard 2", fgtChar(t, cl("fighter", 3), cl("wizard", 2)), 4, 7, false},
		{"level 4 offers no shift", fgtFighter(t, 4), 5, 9, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := fgtWith(t, c.char, fgtState(0, 1, 1))
			res, roller := runSecondWind(t, in, c.roll)
			r, ok := res.(character.SecondWindApplied)
			if !ok {
				t.Fatalf("result = %#v", res)
			}
			if r.HitPointsToRegain() != c.hit || r.Rolled() != c.roll || r.Cost() != character.FighterBonusAction || fgtStateOf(t, r.Character()) != fgtState(1, 1, 1) {
				t.Errorf("= %d Hit Points, rolled %d, state %v", r.HitPointsToRegain(), r.Rolled(), fgtStateOf(t, r.Character()))
			}
			if _, shift := r.TacticalShift().(character.TacticalShiftOffered); shift != c.tactical {
				t.Errorf("Tactical Shift offered = %v, want %v", shift, c.tactical)
			}
			ev, ok := r.Events()[0].(character.SecondWindUsed)
			if len(r.Events()) != 1 || !ok || ev.Rolled() != c.roll || ev.HitPointsToRegain() != c.hit || ev.CharacterID() != in.ID() {
				t.Errorf("events = %#v", r.Events())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D10(), 0) {
				t.Errorf("requests = %v, want exactly one 1d10", reqs)
			}
			if got := fgtStateOf(t, in); got != fgtState(0, 1, 1) {
				t.Errorf("Run altered the input: %v", got)
			}
			r.Events()[0] = nil
			if r.Events()[0] == nil {
				t.Error("Events returned a shared slice")
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-056, FR-057, FR-113, FR-129, FR-108 - with no unexpended use the result is SecondWindNotAvailable(noUsesRemaining)
// and a character with no Fighter levels is notAFighter even when a FighterState is stored; the Character is unchanged, there are no events and
// the roller is not called.
// Path: per case (unhappy: state error).
func TestUseSecondWindNotAvailable(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		char character.Character
		want character.FighterReason
	}{
		{"all uses expended", fgtWith(t, fgtFighter(t, 3), fgtState(2, 0, 0)), character.FighterNoUsesRemaining},
		{"more expended than the maximum", fgtWith(t, fgtFighter(t, 3), fgtState(5, 0, 0)), character.FighterNoUsesRemaining},
		{"no Fighter levels", fgtChar(t, cl("wizard", 3)), character.FighterNotAFighter},
		{"no Fighter levels with a stored state", fgtWith(t, fgtChar(t, cl("wizard", 3)), fgtState(0, 0, 0)), character.FighterNotAFighter},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := useSecondWind(t, roller).Run(context.Background(), character.UseSecondWindCommand{Character: c.char})
			if got, ok := res.(character.SecondWindNotAvailable); err != nil || !ok || got.Reason != c.want {
				t.Fatalf("= %#v, %v; want not available %v", res, err, c.want)
			}
			if len(roller.Requests()) != 0 {
				t.Errorf("requests = %v, want none", roller.Requests())
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-152 - a Fighter with no stored state is treated as having the initial state and the new state is written under the
// key; a stored state of another type under the key is an error matching srderr.ErrInvalidInput.
// Path: per case (unhappy: state error).
func TestUseSecondWindAbsentAndForeignState(t *testing.T) {
	t.Parallel()
	res, _ := runSecondWind(t, fgtFighter(t, 2), 3)
	if got := fgtStateOf(t, res.(character.SecondWindApplied).Character()); got != fgtState(1, 0, 0) {
		t.Errorf("state after the first use = %v, want one expended", got)
	}
	foreign := must(character.WithFeatureState(fgtFighter(t, 2), character.MustFeatureStateKey[character.ChosenSubclass]("fighter"), must(character.NewChosenSubclass("champion"))))
	roller := dicetest.NewFixedRoller(3)
	got, err := useSecondWind(t, roller).Run(context.Background(), character.UseSecondWindCommand{Character: foreign})
	wantInvalid(t, err, "foreign state")
	if got != nil || len(roller.Requests()) != 0 {
		t.Errorf("result %v, requests %v; want nil and none", got, roller.Requests())
	}
}

// Requirement: doc/specs/class-fighter.md FR-110 - a roller that returns its error, panics or answers outside its contract is a *srderr.PortError for dice.Port
// with the cause reachable, the result is nil and the Character unchanged.
// Path: per case (unhappy: collaborator failure).
func TestUseSecondWindRollerFailures(t *testing.T) {
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
		{"a value out of range", dicetest.NewFixedRoller(11), dicetest.ErrValueOutOfRange},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := useSecondWind(t, c.roller).Run(context.Background(), character.UseSecondWindCommand{Character: fgtFighter(t, 5)})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
				t.Errorf("= %v, %v; want a PortError for dice with the cause", res, err)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-112, FR-149 - the result is identical with the no-op and with a recording telemetry; the constructor refuses the zero
// progression and a nil roller, and Run refuses the zero use case, a zero Character and a cancelled context before any signal.
// Path: per case (unhappy: invalid input, cancellation).
func TestUseSecondWindInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	good := character.UseSecondWindCommand{Character: fgtFighter(t, 5)}
	rec, opts := fgtTelemetry()
	withRec, err := useSecondWind(t, dicetest.NewFixedRoller(4), opts...).Run(context.Background(), good)
	plain, err2 := useSecondWind(t, dicetest.NewFixedRoller(4)).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	if len(rec.Tracer.Spans()) != 1 {
		t.Errorf("spans = %d, want 1", len(rec.Tracer.Spans()))
	}
	_, err = character.NewUseSecondWind(character.FighterProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseSecondWind(fgtProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseSecondWind{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	_, err = useSecondWind(t, dicetest.NewFixedRoller(1)).Run(context.Background(), character.UseSecondWindCommand{})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec2, opts2 := fgtTelemetry()
	roller := dicetest.NewFixedRoller(1)
	res, err := useSecondWind(t, roller, opts2...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}
