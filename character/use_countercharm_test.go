package character_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func useCountercharm(t *testing.T, roller character.Roller, opts ...character.Option) character.UseCountercharm {
	t.Helper()
	return must(character.NewUseCountercharm(roller, opts...))
}

func countercharmCmd(c character.Character, circs ...modifier.RollCircumstance) character.UseCountercharmCommand {
	return character.UseCountercharmCommand{Character: c, Failed: countercharmFailed(), Circumstances: circs, Conditions: []content.ConditionKind{content.ConditionCharmed}}
}

// Requirement: doc/specs/class-bard.md FR-084, FR-085, FR-087, FR-105, FR-106 (SRD 5.2.1, "Level 7: Countercharm", p. 33) - UseCountercharm requests two d20 through the dice port (one when a Disadvantage cancels the Advantage), resolves the new save with the original
// modifiers and DC, reports both outcomes and returns no Character: the pool and the slots are untouched.
// Path: per case (happy / unhappy: rule interaction).
func TestUseCountercharmRerolled(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		circs []modifier.RollCircumstance
		faces []int
		count int
		want  d20.Outcome
	}{
		{"Advantage takes the higher d20", nil, []int{3, 17}, 2, d20.Success},
		{"a Disadvantage cancels to one d20", []modifier.RollCircumstance{modifier.Disadvantage{}}, []int{17}, 1, d20.Success},
		{"still failing", nil, []int{4, 6}, 2, d20.Failure},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			bard := brdBard(t, 7, 16)
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := useCountercharm(t, roller).Run(context.Background(), countercharmCmd(bard, c.circs...))
			got, ok := res.(character.UseCountercharmRerolled)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(c.count, dice.D20(), 0) {
				t.Errorf("requests = %v, want %dd20", reqs, c.count)
			}
			if got.Rerolled().Outcome() != c.want || got.Failed().Outcome() != d20.Failure || got.Rerolled().Target() != got.Failed().Target() || brdPoolOf(t, bard).Remaining() != 3 {
				t.Errorf("rerolled %v, failed %v, pool %d", got.Rerolled().Outcome(), got.Failed().Outcome(), brdPoolOf(t, bard).Remaining())
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-086 - a refused Countercharm rolls nothing: not a Bard, level 6, an effect with neither condition, a save that was not failed.
// Path: per case (unhappy: boundary).
func TestUseCountercharmRejected(t *testing.T) {
	t.Parallel()
	passed := countercharmCmd(brdBard(t, 7, 16))
	passed.Failed = brdCheck(t, d20.SavingThrow, 18, 0, 15)
	poisoned := countercharmCmd(brdBard(t, 7, 16))
	poisoned.Conditions = []content.ConditionKind{content.ConditionPoisoned}
	for _, c := range []struct {
		name   string
		cmd    character.UseCountercharmCommand
		reason character.BardReason
	}{
		{"not a Bard", countercharmCmd(fgtFighter(t, 7)), character.BardNotABard},
		{"level 6", countercharmCmd(brdBard(t, 6, 16)), character.BardBelowRequiredLevel},
		{"neither condition", poisoned, character.BardIneligibleEffect},
		{"the save passed", passed, character.BardSaveNotFailed},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(4, 5)
			res, err := useCountercharm(t, roller).Run(context.Background(), c.cmd)
			if err != nil || res != (character.UseCountercharmRejected{Reason: c.reason}) || len(roller.Requests()) != 0 {
				t.Errorf("= %#v, %v, %d rolls; want %v and no roll", res, err, len(roller.Requests()), c.reason)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-036, FR-101 - a *srderr.PortError from the dice roller is returned unchanged with a nil result; a zero use case, a zero Character, a zero saving throw, a cancelled context and a nil roller are errors.
// Path: per case (unhappy: collaborator failure, invalid input).
func TestUseCountercharmFailures(t *testing.T) {
	t.Parallel()
	port := &srderr.PortError{Port: "dice", Op: "roll", Err: errBoom}
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, port })
	if res, err := useCountercharm(t, failing).Run(context.Background(), countercharmCmd(brdBard(t, 7, 16))); res != nil || !errors.Is(err, errBoom) {
		t.Errorf("port failure = %v, %v", res, err)
	}
	uc := useCountercharm(t, dicetest.NewFixedRoller(3, 4))
	zero := countercharmCmd(brdBard(t, 7, 16))
	zero.Failed = d20.CheckResult{}
	for name, run := range map[string]func() error{
		"zero use case": func() error {
			_, e := character.UseCountercharm{}.Run(context.Background(), countercharmCmd(brdBard(t, 7, 16)))
			return e
		},
		"zero Character": func() error { _, e := uc.Run(context.Background(), countercharmCmd(character.Character{})); return e },
		"zero save":      func() error { _, e := uc.Run(context.Background(), zero); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, countercharmCmd(brdBard(t, 7, 16))); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: error = %v", err)
	}
	if _, err := character.NewUseCountercharm(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil roller: error = %v", err)
	}
}

// countercharmFailed is a failed saving throw: a natural 5 and +2 against DC 15.
func countercharmFailed() d20.CheckResult {
	return must(d20.ResolveCheck(d20.SavingThrow, d20.MustRoll(modifier.Normal, 5), d20.NewRollModifiers(modifier.New(2)), 15))
}
