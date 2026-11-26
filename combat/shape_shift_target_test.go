package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/shape-changing-spells.md FR-051 - the zero AnimatedObjectTerms holds no terms and reports none of them.
// Path: unhappy (empty and minimal).
func TestZeroAnimatedObjectTermsHoldNothing(t *testing.T) {
	t.Parallel()
	var z combat.AnimatedObjectTerms
	if !z.IsZero() || z.IsFriendlyToCasterAndAllies() || z.TakesTurnsAfterCaster() || z.ObeysCaster() || z.Combatants() != nil {
		t.Errorf("zero terms = %+v", z)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-018, FR-019 - a TargetResponse is Willing or Unwilling with a saving throw outcome, and the two are distinct values.
// Path: happy.
func TestTargetResponsesAreValues(t *testing.T) {
	t.Parallel()
	var a, b combat.TargetResponse = combat.Unwilling{Save: d20.Success}, combat.Unwilling{Save: d20.Success}
	if a != b || a == combat.TargetResponse(combat.Willing{}) || a == combat.TargetResponse(combat.Unwilling{Save: d20.Failure}) {
		t.Error("responses do not compare by value")
	}
}
