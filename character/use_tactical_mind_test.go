package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func useTacticalMind(t *testing.T, roller character.Roller, opts ...character.Option) character.UseTacticalMind {
	t.Helper()
	return must(character.NewUseTacticalMind(fgtProgression(t), roller, opts...))
}

func failedCheck(t *testing.T, total int) d20.CheckResult {
	t.Helper()
	return fgtCheck(t, d20.AbilityCheck, modifier.Normal, []int{total}, 0, 15)
}

// Requirement: doc/specs/class-fighter.md Table D, FR-067, FR-068, FR-069, FR-070, FR-107 - UseTacticalMind rolls one 1d10 and adds it to the failed ability check: a new
// total that meets the DC is TacticalMindSucceeded with one Second Wind use expended and a TacticalMindUsed event, a total still below it is
// TacticalMindStillFailed with the Character unchanged and no event; both carry the roll and the new total, and the roll of the check is untouched.
// Path: per case (rows TM1 to TM5, roll extremes).
func TestUseTacticalMindRolled(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name        string
		total, roll int
		newTotal    int
		succeeds    bool
	}{{"TM1", 12, 3, 15, true}, {"TM2", 12, 2, 14, false}, {"TM3", 14, 1, 15, true}, {"TM4", 5, 10, 15, true}, {"TM5", 5, 9, 14, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := fgtWith(t, fgtFighter(t, 2), fgtState(1, 1, 0))
			roller := dicetest.NewFixedRoller(c.roll)
			check := failedCheck(t, c.total)
			res, err := useTacticalMind(t, roller).Run(context.Background(), character.UseTacticalMindCommand{Character: in, FailedCheck: check})
			if err != nil {
				t.Fatal(err)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D10(), 0) {
				t.Errorf("requests = %v, want exactly one 1d10", reqs)
			}
			switch r := res.(type) {
			case character.TacticalMindSucceeded:
				ev, ok := r.Events()[0].(character.TacticalMindUsed)
				if !c.succeeds || r.NewTotal() != c.newTotal || r.Rolled() != c.roll || fgtStateOf(t, r.Character()) != fgtState(2, 1, 0) || !ok || ev.Rolled() != c.roll || len(r.Events()) != 1 || r.Check().Roll() != check.Roll() {
					t.Errorf("succeeded = %d, %d, %v, %v", r.NewTotal(), r.Rolled(), fgtStateOf(t, r.Character()), r.Events())
				}
			case character.TacticalMindStillFailed:
				if c.succeeds || r.NewTotal() != c.newTotal || r.Rolled() != c.roll || !r.Character().Equal(in) || len(r.Events()) != 0 || r.Check().Roll() != check.Roll() {
					t.Errorf("still failed = %d, %d, %v", r.NewTotal(), r.Rolled(), r.Events())
				}
			default:
				t.Fatalf("result = %#v", res)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-064, FR-065, FR-066, FR-129, TM6 - UseTacticalMind is TacticalMindNotAvailable with, in order, notAFighter,
// featureNotGained (level 1), notAnAbilityCheck (a saving throw), checkNotFailed (TM6) and noUsesRemaining; the roller is not called and the Character is
// unchanged.
// Path: per case (unhappy: boundary, state error).
func TestUseTacticalMindNotAvailable(t *testing.T) {
	t.Parallel()
	save := fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{5}, 0, 15)
	for _, c := range []struct {
		name  string
		char  character.Character
		check d20.CheckResult
		want  character.FighterReason
	}{
		{"no Fighter levels", fgtChar(t, cl("wizard", 5)), failedCheck(t, 5), character.FighterNotAFighter},
		{"level 1", fgtFighter(t, 1), failedCheck(t, 5), character.FighterFeatureNotGained},
		{"a saving throw", fgtFighter(t, 2), save, character.FighterNotAnAbilityCheck},
		{"TM6 a check that succeeded", fgtFighter(t, 2), failedCheck(t, 15), character.FighterCheckNotFailed},
		{"no uses remaining", fgtWith(t, fgtFighter(t, 2), fgtState(2, 0, 0)), failedCheck(t, 5), character.FighterNoUsesRemaining},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := useTacticalMind(t, roller).Run(context.Background(), character.UseTacticalMindCommand{Character: c.char, FailedCheck: c.check})
			if got, ok := res.(character.TacticalMindNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Errorf("= %#v, %v, requests %v; want not available %v and no roll", res, err, roller.Requests(), c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-110, FR-112, FR-149 - a failing roller is a *srderr.PortError for dice.Port, the result nil; the result is the same with
// the no-op and a recording telemetry; the constructor refuses the zero progression and a nil roller, and Run refuses the zero use case, a zero
// Character, a zero check result and a cancelled context.
// Path: per case (unhappy: collaborator failure, invalid input, cancellation).
func TestUseTacticalMindFailuresAndInvalidInput(t *testing.T) {
	t.Parallel()
	good := character.UseTacticalMindCommand{Character: fgtFighter(t, 2), FailedCheck: failedCheck(t, 5)}
	res, err := useTacticalMind(t, dicetest.NewFixedRoller()).Run(context.Background(), good)
	var pe *srderr.PortError
	if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, dicetest.ErrExhausted) {
		t.Errorf("failing roller: %v, %v", res, err)
	}
	_, opts := fgtTelemetry()
	withRec, err := useTacticalMind(t, dicetest.NewFixedRoller(4), opts...).Run(context.Background(), good)
	plain, err2 := useTacticalMind(t, dicetest.NewFixedRoller(4)).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v vs %#v", withRec, plain)
	}
	_, err = character.NewUseTacticalMind(character.FighterProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseTacticalMind(fgtProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	uc := useTacticalMind(t, dicetest.NewFixedRoller(1))
	for name, run := range map[string]func() (character.UseTacticalMindResult, error){
		"zero use case": func() (character.UseTacticalMindResult, error) {
			return character.UseTacticalMind{}.Run(context.Background(), good)
		},
		"zero character": func() (character.UseTacticalMindResult, error) {
			return uc.Run(context.Background(), character.UseTacticalMindCommand{FailedCheck: good.FailedCheck})
		},
		"zero check": func() (character.UseTacticalMindResult, error) {
			return uc.Run(context.Background(), character.UseTacticalMindCommand{Character: good.Character})
		},
	} {
		if r, err := run(); r != nil {
			t.Errorf("%s: result %v", name, r)
		} else {
			wantInvalid(t, err, name)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := uc.Run(cancelled, good); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", r, err)
	}
}
