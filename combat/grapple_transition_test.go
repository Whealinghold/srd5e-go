package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: per case
// Path: per case.
func TestGrapplesBegin(t *testing.T) {
	t.Parallel()
	tentacle, maw := combat.GraspingPart(combat.MustGraspingBodyPart("tentacle")), combat.GraspingPart(combat.MustGraspingBodyPart("maw"))
	cases := []struct {
		name, requirement, path string
		state                   combat.Grapples
		next                    combat.Grapple
		rejection               combat.GrappleRejection
		problem                 combat.GrappleProblem
	}{
		{"first grapple", `FR-022, table 4.10.4; "Grappling", p. 182`, "happy", grapplesOf(t), held("a", "b", hand(0)), 0, 0},
		{"second part, second target", "FR-023", "unhappy (rule interaction)", grapplesOf(t, held("a", "b", hand(0))), held("a", "c", hand(1)), 0, 0},
		{"second part, same target", `FR-025; "Condition", p. 179`, "unhappy (state error)", grapplesOf(t, held("a", "b", hand(0))), held("a", "b", hand(1)), combat.GrappleRejectionTargetAlreadyGrappled, 0},
		{"same part, other target", "FR-024", "unhappy (state error)", grapplesOf(t, held("a", "b", hand(0))), held("a", "c", hand(0)), combat.GrappleRejectionPartInUse, 0},
		{"same part, same target", "FR-024, FR-026", "unhappy (rule interaction)", grapplesOf(t, held("a", "b", hand(0))), held("a", "b", hand(0)), combat.GrappleRejectionPartInUse, 0},
		{"tentacle then maw", "FR-023", "unhappy (rule interaction)", grapplesOf(t, held("a", "b", tentacle)), held("a", "c", maw), 0, 0},
		{"two grapplers, one target", "FR-025", "unhappy (state error)", grapplesOf(t, held("a", "b", hand(0))), held("d", "b", hand(0)), combat.GrappleRejectionTargetAlreadyGrappled, 0},
		{"same part, two creatures", "FR-015", "unhappy (boundary)", grapplesOf(t, held("a", "b", hand(0))), held("d", "c", hand(0)), 0, 0},
		{"grappler itself held", "FR-027", "unhappy (rule interaction)", grapplesOf(t, held("d", "a", hand(0))), held("a", "b", hand(0)), 0, 0},
		{"full set, part in use", "plan P10; FR-024 is checked before the cap", "unhappy (boundary)", fullGrapples(t, 256), held("g0", "other", hand(0)), combat.GrappleRejectionPartInUse, 0},
		{"an unbuilt grapple", "package design", "unhappy (invalid input)", grapplesOf(t), combat.Grapple{}, 0, combat.GrappleProblemMissingValue},
		{"a negative range literal", "package design", "unhappy (invalid input)", grapplesOf(t), combat.Grapple{Grappler: gid("a"), Target: gid("b"), Part: hand(0), RangeFeet: -1}, 0, combat.GrappleProblemNegativeRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state.All()
			tr := c.state.Begin(c.next)
			if !slices.Equal(c.state.All(), before) {
				t.Errorf("the input set changed [FR-028]")
			}
			switch {
			case c.rejection != 0:
				if tr != (combat.GrappleTransitionRejected{Rejection: c.rejection}) {
					t.Errorf("= %#v, want rejected %v [%s]", tr, c.rejection, c.requirement)
				}
			case c.problem != 0:
				if tr != (combat.GrappleTransitionInvalid{Problem: c.problem}) {
					t.Errorf("= %#v, want invalid %v [%s]", tr, c.problem, c.requirement)
				}
			default:
				a := appliedGrapple(t, tr)
				if !a.Grapples().Equal(grapplesOf(t, append(c.state.All(), c.next)...)) || len(a.Endings()) != 0 {
					t.Errorf("= %v, want the set plus the grapple and no ending [%s]", a.Grapples().All(), c.requirement)
				}
				want, err := combat.NewGrappleStarted(c.next.Grappler, c.next.Target, c.next.Part)
				if err != nil || !slices.Equal(a.Events(), []combat.Event{want}) {
					t.Errorf("events = %v, want one GrappleStarted [FR-022]", a.Events())
				}
				combattest.CheckEventsCopy(t, a.Events)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-029, FR-030, FR-031, FR-032, FR-059 ("Ending a Grapple", p. 182) - Release removes the
// grapple the grappler holds, with no event and one ending (released); the freed part can begin another grapple; a
// grapple not held by that grappler is NotHeld; equal ids are invalid.
// Path: per case.
func TestGrapplesRelease(t *testing.T) {
	t.Parallel()
	ab, ac := held("a", "b", hand(0)), held("a", "c", hand(1))
	s := grapplesOf(t, ab, ac)
	a := appliedGrapple(t, s.Release(gid("a"), gid("b")))
	if !a.Grapples().Equal(grapplesOf(t, ac)) || len(a.Events()) != 0 {
		t.Errorf("Release = %v, %v, want c still held and no event [FR-029]", a.Grapples().All(), a.Events())
	}
	if e := a.Endings(); len(e) != 1 || e[0].Grapple() != ab || e[0].Cause() != combat.GrappleEndCauseReleased {
		t.Errorf("endings = %v, want the released ab [FR-029]", e)
	}
	if a.Grapples().IsHolding(gid("a"), hand(0)) != combat.PartStateNotHoldingGrapple {
		t.Error("the released part is still holding [FR-030]")
	}
	if _, ok := a.Grapples().GrappleOf(gid("b")).(combat.NotInGrapple); !ok {
		t.Error("the released target is still grappled [FR-030, FR-059]")
	}
	if _, ok := a.Grapples().Begin(held("a", "d", hand(0))).(combat.GrappleTransitionApplied); !ok {
		t.Error("the freed part cannot begin another grapple [FR-030]")
	}
	notHeld := combat.GrappleTransitionRejected{Rejection: combat.GrappleRejectionNotHeld}
	if tr := s.Release(gid("0"), gid("b")); tr != (combat.GrappleTransitionRejected{Rejection: combat.GrappleRejectionNotHeld}) {
		t.Errorf("Release by a grappler that sorts before the holder = %#v, want NotHeld [FR-031]", tr)
	}
	for name, tr := range map[string]combat.GrappleTransition{
		"target free":          s.Release(gid("a"), gid("zed")),
		"held by another":      s.Release(gid("d"), gid("b")),
		"empty set":            combat.Grapples{}.Release(gid("a"), gid("b")),
		"the grappler as held": s.Release(gid("b"), gid("a")),
	} {
		if tr != notHeld {
			t.Errorf("%s: Release = %#v, want NotHeld [FR-031]", name, tr)
		}
	}
	if tr := s.Release(gid("a"), gid("a")); tr != (combat.GrappleTransitionInvalid{Problem: combat.GrappleProblemSameGrapplerAndTarget}) {
		t.Errorf("Release of itself = %#v, want invalid [FR-067]", tr)
	}
	if !s.Equal(grapplesOf(t, ab, ac)) {
		t.Error("Release changed its receiver [FR-031]")
	}
}

// Requirement: doc/specs/grappling.md FR-033, FR-034, FR-059 - Escape removes the grapple holding the target with one ending
// (escaped) and no event; a target that is not held is NotGrappled and the set is unchanged.
// Path: per case.
func TestGrapplesEscape(t *testing.T) {
	t.Parallel()
	ab := held("a", "b", hand(0))
	s := grapplesOf(t, ab, held("a", "c", hand(1)))
	a := appliedGrapple(t, s.Escape(gid("b")))
	if e := a.Endings(); len(e) != 1 || e[0].Grapple() != ab || e[0].Cause() != combat.GrappleEndCauseEscaped || len(a.Events()) != 0 {
		t.Errorf("Escape endings = %v, events %v, want one escaped ending [FR-033]", e, a.Events())
	}
	if a.Grapples().Len() != 1 || a.Grapples().IsHolding(gid("a"), hand(0)) != combat.PartStateNotHoldingGrapple {
		t.Errorf("Escape left %v [FR-059]", a.Grapples().All())
	}
	if tr := s.Escape(gid("zed")); tr != (combat.GrappleTransitionRejected{Rejection: combat.GrappleRejectionNotGrappled}) {
		t.Errorf("Escape of a free creature = %#v, want NotGrappled [FR-034]", tr)
	}
	if s.Len() != 2 {
		t.Error("Escape changed its receiver")
	}
}

// Requirement: doc/specs/grappling.md FR-060, FR-061, FR-059 - ConditionEnded removes the grapple holding the target with
// one GrappleRecordDropped and no ending; a target that is not held is NotGrappled.
// Path: per case.
func TestGrapplesConditionEnded(t *testing.T) {
	t.Parallel()
	s := grapplesOf(t, held("a", "b", hand(0)))
	a := appliedGrapple(t, s.ConditionEnded(gid("b")))
	want, _ := combat.NewGrappleRecordDropped(gid("a"), gid("b"))
	if a.Grapples().Len() != 0 || len(a.Endings()) != 0 || !slices.Equal(a.Events(), []combat.Event{want}) {
		t.Errorf("ConditionEnded = %v, %v, %v, want one record dropped [FR-060]", a.Grapples().All(), a.Events(), a.Endings())
	}
	if tr := s.ConditionEnded(gid("a")); tr != (combat.GrappleTransitionRejected{Rejection: combat.GrappleRejectionNotGrappled}) {
		t.Errorf("ConditionEnded of a non-target = %#v [FR-061]", tr)
	}
	if s.Len() != 1 {
		t.Error("ConditionEnded changed its receiver [FR-061]")
	}
}

// Requirement: per case
// Path: per case.
func TestGrapplesReview(t *testing.T) {
	t.Parallel()
	conds := func(cs ...combat.Condition) combat.ConditionSet { return condSet(t, cs) }
	actor, incap := conds(), conds(combat.Incapacitated{})
	cases := []struct {
		name, requirement, path string
		rangeFeet, distance     int
		grappler                combat.ConditionSet
		dead                    bool
		ending                  combat.GrappleEndCause
	}{
		{"adjacent, can act", "FR-051, FR-058, table 4.10.2", "happy", 5, 0, actor, false, 0},
		{"at range", `FR-051; "Ending a Grapple", p. 182 ("exceeds")`, "unhappy (boundary)", 5, 5, actor, false, 0},
		{"one foot past range", "FR-050", "unhappy (boundary)", 5, 6, actor, false, combat.GrappleEndCauseOutOfRange},
		{"far beyond range", "FR-050", "unhappy (boundary)", 5, 60, actor, false, combat.GrappleEndCauseOutOfRange},
		{"range 0, distance 0", "FR-051", "unhappy (boundary)", 0, 0, actor, false, 0},
		{"range 0, distance 1", "FR-050", "unhappy (boundary)", 0, 1, actor, false, combat.GrappleEndCauseOutOfRange},
		{"long reach at limit", "FR-051", "unhappy (boundary)", 15, 15, actor, false, 0},
		{"long reach one past", "FR-050", "unhappy (boundary)", 15, 16, actor, false, combat.GrappleEndCauseOutOfRange},
		{"Incapacitated, adjacent", `FR-049; "Incapacitated", p. 184`, "happy", 5, 0, incap, false, combat.GrappleEndCauseGrapplerIncapacitated},
		{"Unconscious implies Incapacitated", `FR-049, FR-087; "Unconscious", p. 191`, "unhappy (rule interaction)", 5, 0, conds(combat.Unconscious{}), false, combat.GrappleEndCauseGrapplerIncapacitated},
		{"Incapacitated and out of range", "FR-052, FR-088", "unhappy (rule interaction)", 5, 6, incap, false, combat.GrappleEndCauseGrapplerIncapacitated},
		{"dead, not Incapacitated", `FR-104; "Dead", p. 180`, "unhappy (rule interaction)", 5, 0, actor, true, combat.GrappleEndCauseGrapplerIncapacitated},
		{"dead and out of range", "FR-104, FR-088", "unhappy (rule interaction)", 5, 60, actor, true, combat.GrappleEndCauseGrapplerIncapacitated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			g := combat.MustGrapple(gid("a"), gid("b"), hand(0), 13, c.rangeFeet)
			s := grapplesOf(t, g)
			obs := combat.MustGrappleObservation(gid("a"), gid("b"), c.grappler, c.distance, combat.WithGrapplerDead(c.dead))
			a := appliedGrapple(t, s.Review([]combat.GrappleObservation{obs}))
			if c.ending == 0 {
				if !a.Grapples().Equal(s) || len(a.Endings()) != 0 || len(a.Events()) != 0 {
					t.Errorf("Review ended a held grapple: %v [%s]", a.Endings(), c.requirement)
				}
				return
			}
			if e := a.Endings(); len(e) != 1 || e[0].Grapple() != g || e[0].Cause() != c.ending || a.Grapples().Len() != 0 {
				t.Errorf("Review endings = %v, want one %v [%s]", e, c.ending, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-053, FR-056, FR-057, FR-059 - a grapple with no observation is left alone, no
// observations or an empty set change nothing, and several endings follow the order the grapples began, not the order of the
// observations.
// Path: unhappy (empty and minimal).
func TestGrapplesReviewSelection(t *testing.T) {
	t.Parallel()
	ab, cd, ef := held("a", "b", hand(0)), held("c", "d", hand(0)), held("e", "f", hand(0))
	s := grapplesOf(t, ab, cd, ef)
	far := func(g, tg string) combat.GrappleObservation {
		return combat.MustGrappleObservation(gid(g), gid(tg), combat.ConditionSet{}, 60)
	}
	a := appliedGrapple(t, s.Review([]combat.GrappleObservation{far("e", "f"), far("a", "b")}))
	if e := a.Endings(); len(e) != 2 || e[0].Grapple() != ab || e[1].Grapple() != ef {
		t.Errorf("endings = %v, want ab then ef, the order begun [FR-057]", e)
	}
	if !a.Grapples().Equal(grapplesOf(t, cd)) {
		t.Errorf("remaining = %v, want only cd, which had no observation [FR-053, FR-059]", a.Grapples().All())
	}
	for name, c := range map[string]struct {
		tr   combat.GrappleTransition
		want int
	}{
		"nil": {s.Review(nil), 3}, "empty": {s.Review([]combat.GrappleObservation{}), 3}, "empty set": {combat.Grapples{}.Review(nil), 0},
	} {
		a := appliedGrapple(t, c.tr)
		if a.Grapples().Len() != c.want || len(a.Endings()) != 0 || len(a.Events()) != 0 {
			t.Errorf("%s: Review changed something [FR-056]", name)
		}
	}
}

// Requirement: doc/specs/grappling.md FR-054, FR-055, FR-103, table 4.10.7 - Review returns Invalid for an observation of a
// grapple the set does not hold and for two observations of one grapple; an unknown grapple is reported before any
// duplicate, and the first unknown wins.
// Path: per case.
func TestGrapplesReviewInvalid(t *testing.T) {
	t.Parallel()
	s := grapplesOf(t, held("a", "b", hand(0)))
	ob := func(g, tg string) combat.GrappleObservation {
		return combat.MustGrappleObservation(gid(g), gid(tg), combat.ConditionSet{}, 0)
	}
	cases := []struct {
		name, requirement, path string
		obs                     []combat.GrappleObservation
		problem                 combat.GrappleProblem
	}{
		{"unknown grapple", "FR-054", "unhappy (invalid input)", []combat.GrappleObservation{ob("a", "z")}, combat.GrappleProblemUnknownGrapple},
		{"right ids, wrong way round", "FR-054", "unhappy (invalid input)", []combat.GrappleObservation{ob("b", "a")}, combat.GrappleProblemUnknownGrapple},
		{"an unbuilt observation", "FR-054", "unhappy (invalid input)", []combat.GrappleObservation{{}}, combat.GrappleProblemUnknownGrapple},
		{"duplicate only", "FR-055, table 4.10.7", "unhappy (invalid input)", []combat.GrappleObservation{ob("a", "b"), ob("a", "b")}, combat.GrappleProblemDuplicateObservation},
		{"unknown and duplicate", "FR-103", "unhappy (rule interaction)", []combat.GrappleObservation{ob("a", "z"), ob("a", "z")}, combat.GrappleProblemUnknownGrapple},
		{"duplicate then unknown", "FR-103", "unhappy (rule interaction)", []combat.GrappleObservation{ob("a", "b"), ob("a", "b"), ob("a", "z")}, combat.GrappleProblemUnknownGrapple},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if tr := s.Review(c.obs); tr != (combat.GrappleTransitionInvalid{Problem: c.problem}) {
				t.Errorf("Review = %#v, want invalid %v [%s]", tr, c.problem, c.requirement)
			}
		})
	}
}
