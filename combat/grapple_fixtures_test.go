package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
)

// gid builds a CombatantID.
func gid(s string) combat.CombatantID { return combat.MustCombatantID(s) }

// hand builds hand i.
func hand(i int) combat.GraspingHand { return combat.MustGraspingHand(i) }

// held builds the grapple in which grappler holds target with part, escape DC 13, range 5.
func held(grappler, target string, part combat.GraspingPart) combat.Grapple {
	return combat.MustGrapple(gid(grappler), gid(target), part, d20.DifficultyClass(13), 5)
}

// grapplesOf builds a set from grapples, failing the test on an invalid one.
func grapplesOf(tb testing.TB, gs ...combat.Grapple) combat.Grapples {
	tb.Helper()
	s, err := combat.NewGrapples(gs)
	if err != nil {
		tb.Fatalf("NewGrapples: %v", err)
	}
	return s
}

// appliedGrapple returns the applied transition or fails the test.
func appliedGrapple(tb testing.TB, tr combat.GrappleTransition) combat.GrappleTransitionApplied {
	tb.Helper()
	a, ok := tr.(combat.GrappleTransitionApplied)
	if !ok {
		tb.Fatalf("transition = %#v, want applied", tr)
	}
	return a
}
