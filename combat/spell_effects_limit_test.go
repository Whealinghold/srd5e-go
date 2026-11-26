package combat_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// seCapRun runs the Fireball command edited by edit and returns its error, with a roller that has no die so a call that
// reaches a roll would fail differently.
func seCapRun(tb testing.TB, edit func(*combat.ResolveSpellEffectsCommand)) error {
	tb.Helper()
	cmd := seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil))
	edit(&cmd)
	_, err := seRun(tb, cmd, dicetest.NewMinimumRoller())
	return err
}

// Requirement: doc/specs/spell-effects.md Limits (plan P10), FR-062 - a ResolveSpellEffects command holds at most 256 targets, 256 assignments, 64 part indexes and an instance count of 100; one more of any is an error matching ErrInvalidSpellEffectsCommand and LimitExceededError (srderr.ErrInvalidInput), and the command is unchanged.
// Path: unhappy (boundary).
func TestResolveSpellEffectsCaps(t *testing.T) {
	t.Parallel()
	t.Run("targets", func(t *testing.T) {
		t.Parallel()
		runSpellCap(t, "spell effect targets", 256, func(n int) error {
			return seCapRun(t, func(c *combat.ResolveSpellEffectsCommand) {
				c.Targets = nil
				for i := range n {
					c.Targets = append(c.Targets, seTarget(fmt.Sprintf("t%d", i), nil))
				}
			})
		})
	})
	t.Run("assignments", func(t *testing.T) {
		t.Parallel()
		runSpellCap(t, "spell effect assignments", 256, func(n int) error {
			return seCapRun(t, func(c *combat.ResolveSpellEffectsCommand) {
				c.Assignments = make([]combat.InstanceAssignment, n)
			})
		})
	})
	t.Run("part indexes", func(t *testing.T) {
		t.Parallel()
		runSpellCap(t, "spell effect part indexes", 64, func(n int) error {
			return seCapRun(t, func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes = make([]int, n) })
		})
	})
	t.Run("instance count", func(t *testing.T) {
		t.Parallel()
		runSpellCap(t, "spell effect instances", 100, func(n int) error {
			return seCapRun(t, func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes, c.InstanceCount = []int{0}, n })
		})
	})
}

// Requirement: doc/specs/spell-effects.md FR-062, plan P10 - a host-supplied InstanceCount far over the cap (math.MaxInt, 2^30, 2^31 - 1) with part indexes is rejected with ErrInvalidSpellEffectsCommand and never panics (it once made a slice of that length).
// Path: unhappy (invalid input).
func TestHugeInstanceCountIsRejectedNotAPanic(t *testing.T) {
	t.Parallel()
	for _, n := range []int{math.MaxInt, 1 << 30, math.MaxInt32, 101} {
		err := seCapRun(t, func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes, c.InstanceCount = []int{0}, n })
		if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
			t.Errorf("InstanceCount %d: err = %v, want ErrInvalidSpellEffectsCommand", n, err)
		}
	}
}

// runSpellCap is runCap for an error that matches ErrInvalidSpellEffectsCommand as well as the LimitExceededError. The
// call at the cap may fail for another reason (its targets are not a valid cast), so only the cap error is asserted.
func runSpellCap(t *testing.T, collection string, max int, build func(n int) error) {
	t.Helper()
	t.Run("at the cap", func(t *testing.T) {
		t.Parallel()
		var le combat.LimitExceededError
		if err := build(max); errors.As(err, &le) {
			t.Errorf("%d %s: err = %v, want no cap error", max, collection, err)
		}
	})
	t.Run("cap plus one", func(t *testing.T) {
		t.Parallel()
		err := build(max + 1)
		wantLimit(t, err, collection, max)
		if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
			t.Errorf("err = %v, want ErrInvalidSpellEffectsCommand", err)
		}
	})
}
