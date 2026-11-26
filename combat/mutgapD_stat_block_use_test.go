package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/monster-stat-block-actions.md FR-001 to FR-009 - a use that happened costs one of an action, a Bonus Action
// or a Reaction; the zero transition costs none, so it reports no action choice, no grant and no reaction.
// Path: unhappy (boundary).
func TestZeroStatBlockUseTransitionHasNoEconomy(t *testing.T) {
	t.Parallel()
	var tr combat.StatBlockUseTransition
	if _, ok := tr.ActionChoice(); ok {
		t.Error("ActionChoice ok = true on the zero transition, want false")
	}
	if _, ok := tr.BonusActionGrant(); ok {
		t.Error("BonusActionGrant ok = true on the zero transition, want false")
	}
	if _, ok := tr.Reaction(); ok {
		t.Error("Reaction ok = true on the zero transition, want false")
	}
	if tr.Cost() != 0 {
		t.Errorf("Cost = %v, want the zero cost", tr.Cost())
	}
}
