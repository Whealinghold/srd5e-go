package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-028, FR-054, FR-057 - releaseHeld ends the grapple of an Engulfed-grappled creature for the causes escaped and pulled out (ended as escaped) and container died (ended as released) and for no other cause; a swallowed creature's grapple is never touched.
// Path: per case.
func TestReleaseHeldEndsTheGrappleForThreeCauses(t *testing.T) {
	t.Parallel()
	whelm := internalTerms(t, func(in *ContainmentTermsInput) {
		in.Mode, in.Entry = ContainmentModeEngulfedGrappled, EntryAutomatic{}
		in.Grapple = MustGrappleHoldTerms(14, "Whelm")
	})
	swallow := internalTerms(t, func(in *ContainmentTermsInput) { in.Mode, in.Entry = ContainmentModeSwallowed, EntryAutomatic{} })
	grapple := MustGrapple("m", "p", MustGraspingBodyPart("Whelm 1"), d20.DifficultyClass(14), 0)
	conds := MustConditionSet([]Condition{grapple.Condition()}, nil)
	rec := func(terms ContainmentTerms) Containment {
		return Containment{"m", "p", content.SizeMedium, terms, ContainmentInside(), 0, 1, 0, 0, false}
	}
	ends := map[ContainmentLeaveCause]GrappleEndCause{
		ContainmentLeaveCauseEscaped: GrappleEndCauseEscaped, ContainmentLeaveCausePulledOut: GrappleEndCauseEscaped, ContainmentLeaveCauseContainerDied: GrappleEndCauseReleased}
	for cause := ContainmentLeaveCauseEscaped; cause <= ContainmentLeaveCauseOccupantRemoved; cause++ {
		g, rel := releaseHeld(MustGrapples([]Grapple{grapple}), conds, rec(whelm), cause)
		want, ends := ends[cause]
		if ends != (g.Len() == 0) {
			t.Errorf("%v: grapples left = %d, want the grapple ended only for escaped, pulled out and container died", cause, g.Len())
		}
		if ends {
			if end, ok := rel.events[1].(GrappleEnded); !ok || end.Cause() != want {
				t.Errorf("%v: events = %v, want GrappleEnded as %v second", cause, rel.events, want)
			}
		} else if len(rel.events) > 0 {
			if _, isEnd := rel.events[0].(ConditionRemoved); isEnd && cause != ContainmentLeaveCauseOccupantRemoved && len(rel.Conditions().Effective()) == 0 {
				t.Errorf("%v: the Grappled condition was removed", cause)
			}
		}
		// A swallowed creature's record never touches the grapple.
		if kept, _ := releaseHeld(MustGrapples([]Grapple{grapple}), conds, rec(swallow), cause); kept.Len() != 1 {
			t.Errorf("%v: a swallowed creature's release ended a grapple", cause)
		}
	}
}
