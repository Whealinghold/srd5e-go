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

func useIndomitable(t *testing.T, roller character.Roller, opts ...character.Option) character.UseIndomitable {
	t.Helper()
	return must(character.NewUseIndomitable(fgtProgression(t), roller, opts...))
}

// Requirement: doc/specs/class-fighter.md Table F, FR-080, FR-081, FR-082, FR-107, FR-126 - UseIndomitable rolls one 1d20, replaces only the die the caller named and recomputes the
// total with the modifiers and the Fighter level (9, not the total level): IN2 equals the DC and succeeds, IN1 is still failed with the use expended
// all the same, IN4 replaces the used die under Advantage and IN6 the discarded die under Disadvantage; the new Roll is carried and the original unaltered.
// Path: per case (rows IN1 to IN7).
func TestUseIndomitableRolled(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		mode     modifier.RollMode
		original []int
		replaced int
		rolled   int
		total    int
		succeeds bool
	}{
		{"IN1", modifier.Normal, []int{8}, 0, 1, 14, false}, {"IN2", modifier.Normal, []int{8}, 0, 2, 15, true}, {"IN3", modifier.Normal, []int{8}, 0, 20, 33, true},
		{"IN4", modifier.WithAdvantage, []int{3, 8}, 1, 2, 16, true}, {"IN5", modifier.WithAdvantage, []int{3, 8}, 0, 20, 33, true},
		{"IN6", modifier.WithDisadvantage, []int{1, 12}, 1, 15, 14, false}, {"IN7", modifier.WithDisadvantage, []int{1, 12}, 0, 10, 23, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := fgtWith(t, fgtChar(t, cl("fighter", 9), cl("wizard", 3)), fgtState(1, 0, 0))
			save := fgtCheck(t, d20.SavingThrow, c.mode, c.original, 4, 15)
			roller := dicetest.NewFixedRoller(c.rolled)
			res, err := useIndomitable(t, roller).Run(context.Background(), character.UseIndomitableCommand{Character: in, FailedSave: save, ReplacedDie: c.replaced})
			if err != nil {
				t.Fatal(err)
			}
			var char character.Character
			var rolled, total int
			var events []character.Event
			var check d20.CheckResult
			switch r := res.(type) {
			case character.IndomitableSucceeded:
				char, rolled, total, events, check = r.Character(), r.Rolled(), r.NewTotal(), r.Events(), r.Check()
				if !c.succeeds {
					t.Error("succeeded, want still failed")
				}
			case character.IndomitableStillFailed:
				char, rolled, total, events, check = r.Character(), r.Rolled(), r.NewTotal(), r.Events(), r.Check()
				if c.succeeds {
					t.Error("still failed, want succeeded")
				}
			default:
				t.Fatalf("result = %#v", res)
			}
			ev, ok := events[0].(character.IndomitableUsed)
			if rolled != c.rolled || total != c.total || fgtStateOf(t, char) != fgtState(1, 0, 1) || len(events) != 1 || !ok || ev.Rolled() != c.rolled || check.Roll().Dice()[c.replaced] != c.rolled {
				t.Errorf("rolled %d total %d state %v events %v", rolled, total, fgtStateOf(t, char), events)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D20(), 0) {
				t.Errorf("requests = %v, want exactly one 1d20", reqs)
			}
			if !equalInts(save.Roll().Dice(), c.original) {
				t.Error("the original roll was altered")
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-077, FR-078, FR-079, FR-127, FR-129, IN8 - UseIndomitable is IndomitableNotAvailable with, in order, notAFighter,
// featureNotGained (level 8), notASavingThrow, dieNotRolled (IN8), saveNotFailed and noUsesRemaining; the roller is not called, not even for dieNotRolled,
// and the Character is unchanged.
// Path: per case (unhappy: boundary, invalid input, state error).
func TestUseIndomitableNotAvailable(t *testing.T) {
	t.Parallel()
	failed := fgtCheck(t, d20.SavingThrow, modifier.WithAdvantage, []int{3, 8}, 4, 15)
	passed := fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{18}, 4, 15)
	check := fgtCheck(t, d20.AbilityCheck, modifier.Normal, []int{3}, 4, 15)
	for _, c := range []struct {
		name     string
		char     character.Character
		save     d20.CheckResult
		replaced int
		want     character.FighterReason
	}{
		{"no Fighter levels", fgtChar(t, cl("wizard", 9)), failed, 0, character.FighterNotAFighter},
		{"level 8", fgtFighter(t, 8), failed, 0, character.FighterFeatureNotGained},
		{"an ability check", fgtFighter(t, 9), check, 0, character.FighterNotASavingThrow},
		{"IN8 a die not rolled", fgtFighter(t, 9), failed, 2, character.FighterDieNotRolled},
		{"a passed save", fgtFighter(t, 9), passed, 0, character.FighterSaveNotFailed},
		{"no uses remaining", fgtWith(t, fgtFighter(t, 9), fgtState(0, 0, 1)), failed, 1, character.FighterNoUsesRemaining},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := useIndomitable(t, roller).Run(context.Background(), character.UseIndomitableCommand{Character: c.char, FailedSave: c.save, ReplacedDie: c.replaced})
			if got, ok := res.(character.IndomitableNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Errorf("= %#v, %v, requests %v; want not available %v and no roll", res, err, roller.Requests(), c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-083, FR-110, FR-112, FR-149 - Indomitable uses return only on a Long Rest (the state after a Short Rest keeps them expended),
// a failing roller is a *srderr.PortError for dice.Port with the result nil, the result is the same with the no-op and a recording telemetry, and the invalid-input and
// cancellation refusals are made before any signal.
// Path: per case (unhappy: state error, collaborator failure, invalid input, cancellation).
func TestUseIndomitableFailuresAndInvalidInput(t *testing.T) {
	t.Parallel()
	save := fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{5}, 4, 15)
	good := character.UseIndomitableCommand{Character: fgtFighter(t, 9), FailedSave: save}
	res, err := useIndomitable(t, dicetest.NewFixedRoller()).Run(context.Background(), good)
	var pe *srderr.PortError
	if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, dicetest.ErrExhausted) {
		t.Errorf("failing roller: %v, %v", res, err)
	}
	_, opts := fgtTelemetry()
	withRec, err := useIndomitable(t, dicetest.NewFixedRoller(4), opts...).Run(context.Background(), good)
	plain, err2 := useIndomitable(t, dicetest.NewFixedRoller(4)).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v vs %#v", withRec, plain)
	}
	used := fgtStateOf(t, withRec.(character.IndomitableSucceeded).Character())
	if used.AfterRest(character.ShortRest).IndomitableExpended() != 1 || used.AfterRest(character.LongRest).IndomitableExpended() != 0 {
		t.Error("Indomitable must return on a Long Rest only")
	}
	_, err = character.NewUseIndomitable(character.FighterProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseIndomitable(fgtProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	uc := useIndomitable(t, dicetest.NewFixedRoller(1))
	for name, run := range map[string]func() (character.UseIndomitableResult, error){
		"zero use case": func() (character.UseIndomitableResult, error) {
			return character.UseIndomitable{}.Run(context.Background(), good)
		},
		"zero character": func() (character.UseIndomitableResult, error) {
			return uc.Run(context.Background(), character.UseIndomitableCommand{FailedSave: save})
		},
		"zero save": func() (character.UseIndomitableResult, error) {
			return uc.Run(context.Background(), character.UseIndomitableCommand{Character: good.Character})
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
