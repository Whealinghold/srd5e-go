package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// grappleEnc has a (grappler, side A) holding b (grappled, side B).
func grappleEnc(tb testing.TB, grappler []combat.Condition) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encC(tb, "a", combat.SideA, pcStatus(tb, 20, 20, 0), grappler...), encC(tb, "b", combat.SideB, pcStatus(tb, 20, 20, 0), combat.Grappled{Grappler: cid("a")}))
	return e
}

// Requirement: per case
// Path: per case.
func TestGrappleCases(t *testing.T) {
	t.Parallel()
	type row struct {
		name, requirement, path string
		grappler                []combat.Condition
		run                     func(e combat.Encounter) (encTr, error)
		ended                   bool
		cause                   string
	}
	check := func(d, r int) func(combat.Encounter) (encTr, error) {
		return func(e combat.Encounter) (encTr, error) { return e.CheckGrapple(cid("b"), d, r) }
	}
	rows := []row{
		{"grappler Incapacitated", "FR-086; p. 182", "happy", condConds(combat.Incapacitated{}), check(5, 5), true, "grappler_incapacitated"},
		{"grappler Stunned (implied)", "FR-086; p. 182", "unhappy (rule interaction)", condConds(combat.Stunned{}), check(5, 5), true, "grappler_incapacitated"},
		{"distance exceeds range", "FR-086; p. 182", "happy", nil, check(6, 5), true, "out_of_range"},
		{"distance equals range", "FR-086; p. 182", "unhappy (boundary)", nil, check(5, 5), false, ""},
		{"both causes", "FR-086; p. 182", "unhappy (rule interaction)", condConds(combat.Incapacitated{}), check(10, 5), true, "grappler_incapacitated"},
		{"released", "FR-093; p. 182", "happy", nil, func(e combat.Encounter) (encTr, error) { return e.ReleaseGrapple(cid("b")) }, true, "released"},
		{"escape succeeds", "FR-094; p. 182", "happy", nil, func(e combat.Encounter) (encTr, error) { return e.EscapeGrapple(cid("b"), d20.Success) }, true, "escaped"},
		{"escape fails", "FR-094; p. 182", "unhappy (rule interaction)", nil, func(e combat.Encounter) (encTr, error) { return e.EscapeGrapple(cid("b"), d20.Failure) }, false, ""},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			e := grappleEnc(t, r.grappler)
			tr, err := r.run(e)
			after, events := encOK(t, tr, err)
			if !r.ended {
				if len(events) != 0 || !after.Equal(e) || !encExplicit(encMember(t, after, "b"), content.ConditionGrappled) {
					t.Errorf("a grapple that holds must change nothing and emit nothing [%s]", r.requirement)
				}
				return
			}
			if !slices.Equal(encKinds(events), []string{"condition_removed", "grapple_ended"}) {
				t.Fatalf("events = %v, want ConditionRemoved then GrappleEnded [%s]", encKinds(events), r.requirement)
			}
			if _, ok := events[0].(combat.ConditionEventOccurred); !ok {
				t.Errorf("first event is %T, want a ConditionEventOccurred", events[0])
			}
			env, ok := events[1].(combat.GrappleEventOccurred)
			if !ok || env.Event().(combat.GrappleEnded).Cause().String() != r.cause || !slices.Equal(env.Combatants(), []combat.CombatantID{"b", "a"}) {
				t.Errorf("GrappleEnded = %#v, want cause %s about [b a] [%s]", events[1], r.cause, r.requirement)
			}
			if encExplicit(encMember(t, after, "b"), content.ConditionGrappled) {
				t.Errorf("b is still Grappled")
			}
			encOthersEqual(t, e, after, "b")
			checkEnvelopes(t, events)
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-086, FR-087, FR-093, FR-094, table 4.11.1 - a combatant not holding Grappled is refused DoesNotHaveCondition by all three commands; a negative distance or range and an outcome that is not success or failure are errors.
// Path: unhappy (state, invalid input).
func TestGrappleCommandsOnACombatantThatIsNotGrappled(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	want := combat.DoesNotHaveConditionRejection{ID: cid("b"), Kind: content.ConditionGrappled}
	for name, f := range map[string]func() (encTr, error){
		"check":   func() (encTr, error) { return e.CheckGrapple(cid("b"), 5, 5) },
		"release": func() (encTr, error) { return e.ReleaseGrapple(cid("b")) },
		"escape":  func() (encTr, error) { return e.EscapeGrapple(cid("b"), d20.Success) },
	} {
		tr, err := f()
		if r := encNo(t, e, tr, err); r != want {
			t.Errorf("%s: rejection = %#v, want DoesNotHaveCondition(Grappled)", name, r)
		}
	}
	g := grappleEnc(t, nil)
	for name, f := range map[string]func() (encTr, error){
		"negative distance": func() (encTr, error) { return g.CheckGrapple(cid("b"), -1, 5) },
		"negative range":    func() (encTr, error) { return g.CheckGrapple(cid("b"), 5, -1) },
		"outcome":           func() (encTr, error) { return g.EscapeGrapple(cid("b"), 0) },
		"zero id":           func() (encTr, error) { return g.ReleaseGrapple("") },
		"zero id check":     func() (encTr, error) { return g.CheckGrapple("", 1, 1) },
		"zero encounter":    func() (encTr, error) { return (combat.Encounter{}).CheckGrapple(cid("b"), 1, 1) },
	} {
		if res, err := f(); err == nil || res != nil {
			t.Errorf("%s: want an ErrInvalidInput error, got %v, %v", name, res, err)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-087, FR-078, table 4.9.1 - a grappler the encounter does not hold is UnknownCombatant carrying the grappler id (class invalid), looked up before the dead rejection; a dead grappled combatant with a known grappler is CombatantIsDead and the grapple is not ended; a dead grappler's hold is not ended by the check.
// Path: unhappy (rule interaction).
func TestGrapplerLookupAndDeadCombatants(t *testing.T) {
	t.Parallel()
	ghost := func(hp combat.HitPointStatus) combat.Encounter {
		e, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "b", combat.SideB, hp, combat.Grappled{Grappler: cid("ghost")}))
		return e
	}
	for _, hp := range []combat.HitPointStatus{pcStatus(t, 20, 20, 0), deadStatus(t, combat.MonsterDeath)} {
		e := ghost(hp)
		tr, err := e.CheckGrapple(cid("b"), 5, 5)
		r := encNo(t, e, tr, err)
		if r != (combat.UnknownCombatantRejection{ID: cid("ghost")}) || r.String() != "unknown_combatant" {
			t.Errorf("rejection = %#v, want UnknownCombatant(ghost) [FR-087]", r)
		}
	}
	deadB, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "b", combat.SideB, deadStatus(t, combat.MonsterDeath), combat.Grappled{Grappler: cid("a")}))
	for name, f := range map[string]func() (encTr, error){
		"check":   func() (encTr, error) { return deadB.CheckGrapple(cid("b"), 5, 5) },
		"release": func() (encTr, error) { return deadB.ReleaseGrapple(cid("b")) },
		"escape":  func() (encTr, error) { return deadB.EscapeGrapple(cid("b"), d20.Success) },
	} {
		tr, err := f()
		if r := encNo(t, deadB, tr, err); r != (combat.CombatantIsDeadRejection{ID: cid("b")}) {
			t.Errorf("%s: rejection = %#v, want CombatantIsDead [FR-079]", name, r)
		}
	}
	notGrappled, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "b", combat.SideB, deadStatus(t, combat.MonsterDeath)))
	tr, err := notGrappled.ReleaseGrapple(cid("b"))
	if r := encNo(t, notGrappled, tr, err); r != (combat.CombatantIsDeadRejection{ID: cid("b")}) {
		t.Errorf("release of a dead, not Grappled combatant = %#v, want CombatantIsDead [table 4.9.1]", r)
	}
	deadA, _ := encStart(t, encC(t, "a", combat.SideA, deadStatus(t, combat.MonsterDeath)), encC(t, "b", combat.SideB, pcStatus(t, 20, 20, 0), combat.Grappled{Grappler: cid("a")}))
	tr, err = deadA.CheckGrapple(cid("b"), 5, 5)
	if _, events := encOK(t, tr, err); len(events) != 0 {
		t.Errorf("a dead grappler's hold must not be ended by the check [FR-086]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-088 - no other command checks or removes Grappled: Incapacitating the grappler, damage, healing, conditions and advancing the turn leave b Grappled.
// Path: unhappy (rule interaction).
func TestNoAutomaticEndOfAGrapple(t *testing.T) {
	t.Parallel()
	e := grappleEnc(t, nil)
	tr, err := e.ApplyCondition(cid("a"), combat.Incapacitated{})
	e, _ = encOK(t, tr, err)
	tr, err = e.ApplyDamage(cid("a"), encSlash(30), nil, combat.OtherSource, combat.NormalHit)
	e, _ = encOK(t, tr, err)
	e, _ = advance(t, e)
	e, _ = advance(t, e)
	if !encExplicit(encMember(t, e, "b"), content.ConditionGrappled) {
		t.Errorf("b is no longer Grappled though no grapple command ran [FR-088]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-086, FR-087 - the grappled combatant may be the first of the order and the grappler any known one, not only the first: a grapple of a (first) held by c (third) ends when c is Incapacitated, and a grapple at distance 0 and range 0 holds.
// Path: unhappy (boundary).
func TestGrappleCheckWithTheGrappledFirstAndTheGrapplerLater(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, encC(t, "a", combat.SideA, pcStatus(t, 20, 20, 0), combat.Grappled{Grappler: cid("c")}), encPC(t, "b", combat.SideB, 20, 20),
		encC(t, "c", combat.SideB, pcStatus(t, 20, 20, 0), combat.Incapacitated{}))
	tr, err := e.CheckGrapple(cid("a"), 0, 0)
	_, events := encOK(t, tr, err)
	if !slices.Equal(encKinds(events), []string{"condition_removed", "grapple_ended"}) {
		t.Errorf("events = %v, want the grapple ended by the Incapacitated grappler c", encKinds(events))
	}
	holds, _ := encStart(t, encC(t, "a", combat.SideA, pcStatus(t, 20, 20, 0), combat.Grappled{Grappler: cid("c")}), encPC(t, "b", combat.SideB, 20, 20), encPC(t, "c", combat.SideB, 20, 20))
	tr, err = holds.CheckGrapple(cid("a"), 0, 0)
	if _, events := encOK(t, tr, err); len(events) != 0 {
		t.Errorf("a grapple at distance 0 and range 0 holds: events %v", encKinds(events))
	}
}
