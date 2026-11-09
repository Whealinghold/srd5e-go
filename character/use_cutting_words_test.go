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

func useCutting(t *testing.T, roller character.Roller, opts ...character.Option) character.UseCuttingWords {
	t.Helper()
	return must(character.NewUseCuttingWords(roller, opts...))
}

func cuttingCmd(c character.Character, trigger character.CuttingWordsTrigger) character.UseCuttingWordsCommand {
	return character.UseCuttingWordsCommand{Character: c, Subclass: lore, Trigger: trigger}
}

// Requirement: doc/specs/class-bard.md FR-072, FR-073, FR-075, FR-107 - UseCuttingWords rolls one Bardic Die of the Bard's size through the dice port, subtracts it from the check or the damage, returns a Character with one use fewer whether or not the success became a
// failure, and leaves the input Character alone.
// Path: per case (happy / unhappy: boundary).
func TestUseCuttingWordsApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		trigger character.CuttingWordsTrigger
		roll    int
		check   d20.Outcome
		damage  int
	}{
		{"success becomes failure", character.CuttingWordsRoll{Result: brdAbility(t, 15, 0, 15)}, 1, d20.Failure, 0},
		{"still succeeds", character.CuttingWordsRoll{Result: brdAbility(t, 20, 0, 15)}, 4, d20.Success, 0},
		{"damage reduced", character.CuttingWordsDamage{Total: 9}, 4, 0, 5},
		{"damage clamped", character.CuttingWordsDamage{Total: 3}, 8, 0, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			bard := brdBard(t, 5, 16)
			roller := dicetest.NewFixedRoller(c.roll)
			res, err := useCutting(t, roller).Run(context.Background(), cuttingCmd(bard, c.trigger))
			applied, ok := res.(character.UseCuttingWordsApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if got := brdPoolOf(t, applied.Character()); got.Remaining() != 2 || brdPoolOf(t, bard).Remaining() != 3 {
				t.Errorf("pool = %d, input pool = %d; want 2 and 3", got.Remaining(), brdPoolOf(t, bard).Remaining())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D8(), 0) {
				t.Errorf("requests = %v, want one d8", reqs)
			}
			if r := applied.Applied().Result(); r != nil && r.Outcome() != c.check || applied.Applied().Damage() != c.damage || len(applied.Events()) != 1 {
				t.Errorf("result %v, damage %d, %d events", r, applied.Applied().Damage(), len(applied.Events()))
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-074 - a refused Cutting Words rolls nothing and spends nothing: not a Bard, below level 3, no College of Lore, no uses left, a saving throw; the Character is not returned changed.
// Path: per case (unhappy: state error).
func TestUseCuttingWordsRejected(t *testing.T) {
	t.Parallel()
	ok := character.CuttingWordsRoll{Result: brdAbility(t, 16, 0, 15)}
	for _, c := range []struct {
		name   string
		cmd    character.UseCuttingWordsCommand
		reason character.BardReason
	}{
		{"not a Bard", cuttingCmd(fgtFighter(t, 5), ok), character.BardNotABard},
		{"below level 3", cuttingCmd(brdBard(t, 2, 16), ok), character.BardFeatureUnavailable},
		{"no subclass", character.UseCuttingWordsCommand{Character: brdBard(t, 5, 16), Trigger: ok}, character.BardFeatureUnavailable},
		{"no uses", cuttingCmd(brdWithPool(t, brdBard(t, 5, 16), 3, 0), ok), character.BardNoUsesRemaining},
		{"a saving throw", cuttingCmd(brdBard(t, 5, 16), character.CuttingWordsRoll{Result: brdSave(t, 16, 0, 15)}), character.BardIneligibleTrigger},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(4)
			res, err := useCutting(t, roller).Run(context.Background(), c.cmd)
			if err != nil || res != (character.UseCuttingWordsRejected{Reason: c.reason}) || len(roller.Requests()) != 0 {
				t.Errorf("= %#v, %v, %d rolls; want %v and no roll", res, err, len(roller.Requests()), c.reason)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-036, FR-101, FR-118 - a *srderr.PortError from the dice roller is returned unchanged with a nil result; a zero use case, a zero Character, a nil trigger, a cancelled context, a trigger the rules cannot take, a Bard with no
// pool and another state type under the pool key are errors.
// Path: per case (unhappy: collaborator failure, invalid input, state error).
func TestUseCuttingWordsFailures(t *testing.T) {
	t.Parallel()
	ok := character.CuttingWordsRoll{Result: brdAbility(t, 16, 0, 15)}
	port := &srderr.PortError{Port: "dice", Op: "roll", Err: errBoom}
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, port })
	if res, err := useCutting(t, failing).Run(context.Background(), cuttingCmd(brdBard(t, 5, 16), ok)); res != nil || !errors.Is(err, errBoom) {
		t.Errorf("port failure = %v, %v", res, err)
	}
	uc := useCutting(t, dicetest.NewFixedRoller(3))
	foreign := must(character.WithFeatureState(restored(t, 355000, 200, flatScores(t, 10), cl("bard", 5)), character.MustFeatureStateKey[character.ChosenSubclass]("bardic_inspiration_pool"), must(character.NewChosenSubclass("x"))))
	for name, run := range map[string]func() error{
		"zero use case": func() error {
			_, e := character.UseCuttingWords{}.Run(context.Background(), cuttingCmd(brdBard(t, 5, 16), ok))
			return e
		},
		"zero Character": func() error { _, e := uc.Run(context.Background(), cuttingCmd(character.Character{}, ok)); return e },
		"nil trigger":    func() error { _, e := uc.Run(context.Background(), cuttingCmd(brdBard(t, 5, 16), nil)); return e },
		"negative damage": func() error {
			_, e := uc.Run(context.Background(), cuttingCmd(brdBard(t, 5, 16), character.CuttingWordsDamage{Total: -1}))
			return e
		},
		"no pool": func() error {
			_, e := uc.Run(context.Background(), cuttingCmd(restored(t, 355000, 200, flatScores(t, 10), cl("bard", 5)), ok))
			return e
		},
		"another state": func() error { _, e := uc.Run(context.Background(), cuttingCmd(foreign, ok)); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, cuttingCmd(brdBard(t, 5, 16), ok)); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: error = %v", err)
	}
	if _, err := character.NewUseCuttingWords(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil roller: error = %v", err)
	}
}
