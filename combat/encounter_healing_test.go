package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestApplyHealingCases(t *testing.T) {
	t.Parallel()
	u := condConds(combat.Unconscious{})
	type row struct {
		name, requirement, path string
		before                  combat.Combatant
		amount                  int
		kinds                   []string
		cur                     int
		life                    string
		unconscious, prone      bool
		knocked                 bool
	}
	a := func(hp combat.HitPointStatus, conds ...combat.Condition) combat.Combatant {
		return encC(t, "a", combat.SideA, hp, conds...)
	}
	pc := func(cur, max int) combat.HitPointStatus { return pcStatus(t, cur, max, 0) }
	rows := []row{
		{"partial healing", "FR-030; p. 17", "happy", a(pc(14, 20)), 5, []string{"hit_points_restored"}, 19, "alive", false, false, false},
		{"healing capped", "FR-030; p. 17", "unhappy (boundary)", a(pc(14, 20)), 8, []string{"hit_points_restored"}, 20, "alive", false, false, false},
		{"exactly to maximum", "FR-030; p. 17", "unhappy (boundary)", a(pc(14, 20)), 6, []string{"hit_points_restored"}, 20, "alive", false, false, false},
		{"already at maximum", "FR-032; p. 17", "unhappy (boundary)", a(pc(20, 20)), 4, []string{}, 20, "alive", false, false, false},
		{"zero healing", "FR-032; p. 17", "unhappy (boundary)", a(pc(14, 20)), 0, []string{}, 14, "alive", false, false, false},
		{"zero healing at 0, Unconscious", "FR-032; p. 17", "unhappy (boundary)", a(dyingStatus(t, 20, 0, 0), u...), 0, []string{}, 0, "dying", true, false, false},
		{"wakes at 1 Hit Point", "FR-031; p. 17", "happy", a(dyingStatus(t, 20, 0, 0), u...), 1, []string{"hit_points_restored", "recovered", "condition_removed"}, 1, "alive", false, true, false},
		{"Stable but Unconscious, healed", "FR-031; p. 18", "unhappy (rule interaction)", a(status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), u...), 3,
			[]string{"hit_points_restored", "recovered", "condition_removed"}, 3, "alive", false, true, false},
		{"knocked out, healed above 0", "FR-092; pp. 17, 184", "unhappy (rule interaction)", a(pc(1, 12), u...).WithKnockedOut(true), 3,
			[]string{"hit_points_restored", "condition_removed"}, 4, "alive", false, true, false},
		{"spell's Unconscious, healed above 0", "FR-092; p. 179", "unhappy (rule interaction)", a(pc(5, 12), u...), 3,
			[]string{"hit_points_restored"}, 8, "alive", true, false, false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			e := encWith(t, r.before)
			tr, err := e.ApplyHealing(cid("a"), r.amount)
			after, events := encOK(t, tr, err)
			if !slices.Equal(encKinds(events), r.kinds) {
				t.Errorf("events = %v, want %v [%s]", encKinds(events), r.kinds, r.requirement)
			}
			c := encMember(t, after, "a")
			if got := c.HitPoints().HitPoints().Current(); got != r.cur || encLife(c) != r.life {
				t.Errorf("after: %d %s, want %d %s [%s]", got, encLife(c), r.cur, r.life, r.requirement)
			}
			if encExplicit(c, content.ConditionUnconscious) != r.unconscious || encExplicit(c, content.ConditionProne) != r.prone || c.KnockedOut() {
				t.Errorf("Unconscious %v Prone %v knocked out %v, want %v %v false [%s]", encExplicit(c, content.ConditionUnconscious), encExplicit(c, content.ConditionProne), c.KnockedOut(), r.unconscious, r.prone, r.requirement)
			}
			if len(r.kinds) == 0 && !after.Equal(e) {
				t.Errorf("healing that restores nothing changed the encounter [FR-032]")
			}
			encOthersEqual(t, e, after, "a")
			checkEnvelopes(t, events)
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-030, table 4.5.1 - the capped healing event states the Hit Points actually regained.
// Path: unhappy (boundary).
func TestHealingEventStatesWhatWasRegained(t *testing.T) {
	t.Parallel()
	e := encWith(t, encPC(t, "a", combat.SideA, 14, 20))
	tr, err := e.ApplyHealing(cid("a"), 8)
	_, events := encOK(t, tr, err)
	if got := events[0].(combat.HitPointEventOccurred).Event().(combat.HitPointsRestored).Regained(); got != 6 {
		t.Errorf("regained = %d, want 6 [table 4.5.1]", got)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-080 - a negative healing amount is an ErrInvalidInput error, before any rejection.
// Path: unhappy (invalid input).
func TestNegativeHealingIsAnError(t *testing.T) {
	t.Parallel()
	e := encWith(t, encPC(t, "a", combat.SideA, 14, 20))
	if tr, err := e.ApplyHealing(cid("a"), -1); err == nil || tr != nil {
		t.Errorf("ApplyHealing(-1) = %v, %v, want an error", tr, err)
	}
	ended := endedEncounter(t, e)
	if tr, err := ended.ApplyHealing(cid("a"), -1); err == nil || tr != nil {
		t.Errorf("an invalid amount on an ended encounter must be an error, not a rejection [FR-080, FR-078]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-098, FR-092 - a knock-out marks the combatant, healing above 0 ends it and clears the marker; first aid is the host's RemoveCondition(Unconscious), which also clears the marker; a held Unconscious is never marked.
// Path: unhappy (rule interaction).
func TestKnockedOutMarkerLifecycle(t *testing.T) {
	t.Parallel()
	e := encWith(t, encPC(t, "a", combat.SideA, 6, 12))
	tr, err := e.ApplyDamage(cid("a"), encSlash(6), nil, combat.MeleeAttackKnockOut, combat.NormalHit)
	ko, _ := encOK(t, tr, err)
	if !encMember(t, ko, "a").KnockedOut() {
		t.Fatal("a knock-out did not set the marker [FR-098]")
	}
	// First aid: the host removes Unconscious.
	rm, err := ko.RemoveCondition(cid("a"), content.ConditionUnconscious)
	woke, events := encOK(t, rm, err)
	c := encMember(t, woke, "a")
	if c.KnockedOut() || encHas(c, content.ConditionUnconscious) || !encExplicit(c, content.ConditionProne) || !slices.Equal(encKinds(events), []string{"condition_removed"}) {
		t.Errorf("after first aid: knocked out %v, Unconscious %v, Prone %v, events %v [FR-092, FR-066]", c.KnockedOut(), encHas(c, content.ConditionUnconscious), encExplicit(c, content.ConditionProne), encKinds(events))
	}
	// Dropping to 0 later clears the marker: that Unconscious belongs to the Dying rules.
	drop, err := ko.ApplyDamage(cid("a"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
	again, _ := encOK(t, drop, err)
	if got := encMember(t, again, "a"); got.KnockedOut() || encLife(got) != "dying" {
		t.Errorf("after dropping to 0: knocked out %v life %s, want false dying [FR-098]", got.KnockedOut(), encLife(got))
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-098 - removing a condition other than Unconscious keeps the knocked-out marker.
// Path: unhappy (rule interaction).
func TestRemovingAnotherConditionKeepsTheKnockedOutMarker(t *testing.T) {
	t.Parallel()
	e := encWith(t, encC(t, "a", combat.SideA, pcStatus(t, 1, 12, 0), combat.Unconscious{}, combat.Blinded{}).WithKnockedOut(true))
	tr, err := e.RemoveCondition(cid("a"), content.ConditionBlinded)
	after, _ := encOK(t, tr, err)
	if !encMember(t, after, "a").KnockedOut() {
		t.Errorf("removing Blinded cleared the knocked-out marker [FR-098]")
	}
}
