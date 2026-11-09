package character_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func usePeerless(t *testing.T, roller character.Roller, opts ...character.Option) character.UsePeerlessSkill {
	t.Helper()
	return must(character.NewUsePeerlessSkill(roller, opts...))
}

func peerlessCmd(c character.Character, failed character.AdjustableD20Result) character.UsePeerlessSkillCommand {
	return character.UsePeerlessSkillCommand{Character: c, Subclass: lore, Failed: failed}
}

// Requirement: doc/specs/class-bard.md FR-081, FR-082, FR-104 - UsePeerlessSkill rolls one d10 (level 14) through the dice port and adds it to the failed test; the Character has one use fewer only when the test became a success, and is returned unchanged
// when it still fails; a natural 1 attack stays a miss.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestUsePeerlessSkillApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		failed character.AdjustableD20Result
		roll   int
		want   d20.Outcome
		uses   int
	}{
		{"reaches the DC", brdAbility(t, 10, 0, 15), 5, d20.Success, 2},
		{"one short", brdAbility(t, 10, 0, 15), 4, d20.Failure, 3},
		{"natural 1 attack", brdAttack(t, 1, 0, 5), 9, d20.Failure, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			bard := brdBard(t, 14, 16)
			roller := dicetest.NewFixedRoller(c.roll)
			res, err := usePeerless(t, roller).Run(context.Background(), peerlessCmd(bard, c.failed))
			applied, ok := res.(character.UsePeerlessSkillApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if applied.Applied().Result().Outcome() != c.want || brdPoolOf(t, applied.Character()).Remaining() != c.uses || brdPoolOf(t, bard).Remaining() != 3 {
				t.Errorf("outcome %v, pool %d, input pool %d; want %v, %d, 3", applied.Applied().Result().Outcome(), brdPoolOf(t, applied.Character()).Remaining(), brdPoolOf(t, bard).Remaining(), c.want, c.uses)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D10(), 0) {
				t.Errorf("requests = %v, want one d10", reqs)
			}
			if wantEvents := 3 - c.uses; len(applied.Events()) != wantEvents {
				t.Errorf("%d events, want %d", len(applied.Events()), wantEvents)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-083 - a refused Peerless Skill rolls nothing: not a Bard, level 13, no College of Lore, no uses, a saving throw, a result that was not a failure.
// Path: per case (unhappy: boundary, state error).
func TestUsePeerlessSkillRejected(t *testing.T) {
	t.Parallel()
	failed := brdAbility(t, 5, 0, 15)
	for _, c := range []struct {
		name   string
		cmd    character.UsePeerlessSkillCommand
		reason character.BardReason
	}{
		{"not a Bard", peerlessCmd(fgtFighter(t, 14), failed), character.BardNotABard},
		{"level 13", peerlessCmd(brdBard(t, 13, 16), failed), character.BardFeatureUnavailable},
		{"no subclass", character.UsePeerlessSkillCommand{Character: brdBard(t, 14, 16), Failed: failed}, character.BardFeatureUnavailable},
		{"no uses", peerlessCmd(brdWithPool(t, brdBard(t, 14, 16), 3, 0), failed), character.BardNoUsesRemaining},
		{"a saving throw", peerlessCmd(brdBard(t, 14, 16), brdSave(t, 5, 0, 15)), character.BardIneligibleTrigger},
		{"a success", peerlessCmd(brdBard(t, 14, 16), brdAbility(t, 18, 0, 15)), character.BardIneligibleTrigger},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(4)
			res, err := usePeerless(t, roller).Run(context.Background(), c.cmd)
			if err != nil || res != (character.UsePeerlessSkillRejected{Reason: c.reason}) || len(roller.Requests()) != 0 {
				t.Errorf("= %#v, %v, %d rolls; want %v and no roll", res, err, len(roller.Requests()), c.reason)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-036, FR-101 - a *srderr.PortError from the dice roller is returned unchanged with a nil result; a zero use case, a zero Character, a nil or zero result, a cancelled context and a nil roller are errors.
// Path: per case (unhappy: collaborator failure, invalid input).
func TestUsePeerlessSkillFailures(t *testing.T) {
	t.Parallel()
	failed := brdAbility(t, 5, 0, 15)
	port := &srderr.PortError{Port: "dice", Op: "roll", Err: errBoom}
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, port })
	if res, err := usePeerless(t, failing).Run(context.Background(), peerlessCmd(brdBard(t, 14, 16), failed)); res != nil || !errors.Is(err, errBoom) {
		t.Errorf("port failure = %v, %v", res, err)
	}
	uc := usePeerless(t, dicetest.NewFixedRoller(3))
	for name, run := range map[string]func() error{
		"zero use case": func() error {
			_, e := character.UsePeerlessSkill{}.Run(context.Background(), peerlessCmd(brdBard(t, 14, 16), failed))
			return e
		},
		"zero Character": func() error {
			_, e := uc.Run(context.Background(), peerlessCmd(character.Character{}, failed))
			return e
		},
		"nil result": func() error { _, e := uc.Run(context.Background(), peerlessCmd(brdBard(t, 14, 16), nil)); return e },
		"zero result": func() error {
			_, e := uc.Run(context.Background(), peerlessCmd(brdBard(t, 14, 16), character.AdjustableAttack{}))
			return e
		},
		"no pool": func() error {
			_, e := uc.Run(context.Background(), peerlessCmd(restored(t, 355000, 200, flatScores(t, 10), cl("bard", 14)), failed))
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, peerlessCmd(brdBard(t, 14, 16), failed)); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: error = %v", err)
	}
	if _, err := character.NewUsePeerlessSkill(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil roller: error = %v", err)
	}
}
