package combat_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// fullGrapples builds a set of n grapples, grappler g<i> holding target t<i>.
func fullGrapples(tb testing.TB, n int) combat.Grapples {
	tb.Helper()
	gs := make([]combat.Grapple, n)
	for i := range gs {
		gs[i] = held(fmt.Sprintf("g%d", i), fmt.Sprintf("t%d", i), hand(0))
	}
	return grapplesOf(tb, gs...)
}

// Requirement: doc/specs/grappling.md Limits (plan P10) - a grapple set holds at most 256 grapples.
// Path: unhappy (boundary).
func TestNewGrapplesCap(t *testing.T) {
	t.Parallel()
	runCap(t, "grapples", 256, func(n int) error {
		gs := make([]combat.Grapple, n)
		for i := range gs {
			gs[i] = held(fmt.Sprintf("g%d", i), fmt.Sprintf("t%d", i), hand(0))
		}
		_, err := combat.NewGrapples(gs)
		return err
	})
}

// Requirement: doc/specs/grappling.md Limits (plan P10) - Begin on a set holding 255 grapples is applied; on a set holding 256 it is invalid with the limit_exceeded problem and the set is unchanged.
// Path: per case.
func TestBeginGrappleCap(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement string
		held              int
		applied           bool
	}{{"255 held, the 256th begins", "P10 at the cap", 255, true}, {"256 held, a 257th", "P10 cap plus one", 256, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := fullGrapples(t, c.held)
			tr := s.Begin(held("new", "newt", hand(0)))
			if c.applied {
				if got := appliedGrapple(t, tr).Grapples().Len(); got != 256 {
					t.Errorf("Len = %d, want 256 [%s]", got, c.requirement)
				}
			} else if inv, ok := tr.(combat.GrappleTransitionInvalid); !ok || inv.Problem != combat.GrappleProblemLimitExceeded {
				t.Errorf("transition = %#v, want invalid limit_exceeded [%s]", tr, c.requirement)
			}
			if s.Len() != c.held {
				t.Errorf("receiver changed: Len = %d", s.Len())
			}
		})
	}
}

// Requirement: doc/specs/grappling.md Limits (plan P10) - Review reads at most 256 observations; 257 is invalid with the limit_exceeded problem, checked before the unknown and duplicate problems.
// Path: per case.
func TestReviewGrapplesCap(t *testing.T) {
	t.Parallel()
	one := grapplesOf(t, held("a", "b", hand(0)))
	ob := combat.MustGrappleObservation(gid("a"), gid("b"), condSet(t, nil), 0)
	for _, c := range []struct {
		name, requirement string
		n                 int
		want              combat.GrappleProblem
	}{{"256 observations of one grapple are duplicates", "P10 at the cap", 256, combat.GrappleProblemDuplicateObservation},
		{"257 observations", "P10 cap plus one", 257, combat.GrappleProblemLimitExceeded}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			obs := make([]combat.GrappleObservation, c.n)
			for i := range obs {
				obs[i] = ob
			}
			inv, ok := one.Review(obs).(combat.GrappleTransitionInvalid)
			if !ok || inv.Problem != c.want {
				t.Errorf("Review(%d) = %#v, want invalid %v [%s]", c.n, inv, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md Limits (plan P10) - ExtraMovementCostPerFoot reads at most 256 carried creatures; 257 is a LimitExceededError and 0.
// Path: unhappy (boundary).
func TestExtraMovementCostCarriedCap(t *testing.T) {
	t.Parallel()
	gs := make([]combat.Grapple, 256)
	sizes := map[combat.CombatantID]content.CreatureSize{}
	carried := make([]combat.CombatantID, 256)
	for i := range gs {
		id := fmt.Sprintf("t%d", i)
		gs[i] = held("g", id, hand(i))
		sizes[gid(id)] = content.SizeMedium
		carried[i] = gid(id)
	}
	s := grapplesOf(t, gs...)
	runCap(t, "carried creatures", 256, func(n int) error {
		list := append(slices.Clone(carried), carried[:n-256]...)
		got, err := s.ExtraMovementCostPerFoot(gid("g"), content.SizeMedium, list, sizes)
		if err != nil && got != 0 {
			return fmt.Errorf("cost %d with error %w", got, err)
		}
		return err
	})
}
