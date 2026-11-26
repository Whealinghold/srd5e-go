package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// rosEvents returns one CombatantJoined, CombatantRemoved and CombatantRepositioned from the base encounter.
func rosEvents(t *testing.T) (combat.CombatantJoined, combat.CombatantRemoved, combat.CombatantRepositioned) {
	t.Helper()
	_, added := rosOK(t)(rosAdd(t, rosEnc(t), "x", combat.SideB, rosAt(20, rosBehind)))
	_, removed := rosOK(t)(rosEnc(t).RemoveCombatant(cid("c"), combat.RemovalCauseSurrendered))
	_, moved := rosOK(t)(rosEnc(t).RepositionCombatant(cid("d"), rosAfter("a")))
	return added[0].(combat.CombatantJoined), removed[0].(combat.CombatantRemoved), moved[0].(combat.CombatantRepositioned)
}

// Requirement: doc/specs/combatant-roster-changes.md FR-033, FR-012, FR-026, FR-032 - each roster event has its constant Kind and a Combatants list of exactly the subject, as a fresh copy; the three kinds are distinct and the events belong to the RosterEvent family, not to the encounter's.
// Path: happy.
func TestRosterEventsFollowTheEventContract(t *testing.T) {
	t.Parallel()
	joined, removed, moved := rosEvents(t)
	combattest.CheckEvent(t, joined, "combatant_joined", cid("x"))
	combattest.CheckEvent(t, removed, "combatant_removed", cid("c"))
	combattest.CheckEvent(t, moved, "combatant_repositioned", cid("d"))
	combattest.CheckEventKinds(t, []combat.Event{joined, removed, moved})
	for _, e := range []combat.RosterEvent{joined, removed, moved} {
		if _, ok := any(e).(combat.EncounterEvent); ok {
			t.Errorf("%T is an EncounterEvent: a roster event is its own family [FR-033]", e)
		}
		if _, ok := any(e).(interface{ Event() combat.TurnEvent }); ok {
			t.Errorf("%T wraps another event: none is an envelope [FR-033]", e)
		}
	}
	for _, e := range []interface {
		EncounterID() combat.EncounterID
		Round() int
	}{joined, removed, moved} {
		if e.EncounterID() != eid() || e.Round() != 2 {
			t.Errorf("%T carries encounter %v round %d, want the base encounter in round 2", e, e.EncounterID(), e.Round())
		}
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-033, FR-034 - roster events are immutable values equal by value (comparable with ==, Kind and Combatants included); precededBy is the entry directly before the subject after the operation, none when it is first; a reposition also carries the previous count and previous predecessor.
// Path: unhappy (boundary).
func TestRosterEventsCompareByValueAndCarryPrecededBy(t *testing.T) {
	t.Parallel()
	joined, removed, moved := rosEvents(t)
	again, removedAgain, movedAgain := rosEvents(t)
	if joined != again || removed != removedAgain || moved != movedAgain {
		t.Errorf("the same operation gave events that are not equal by value [FR-033]")
	}
	if _, other, _ := rosEvents(t); other != removed {
		t.Errorf("equal events compare unequal")
	}
	_, otherEvents := rosOK(t)(rosAdd(t, rosEnc(t), "x", combat.SideB, rosAt(20, rosBefore)))
	if otherEvents[0].(combat.CombatantJoined) != joined {
		t.Errorf("a different tie placement at the same position changed the event: the first entry has no predecessor either way")
	}
	_, middle := rosOK(t)(rosAdd(t, rosEnc(t), "x", combat.SideB, rosAt(7, rosBehind)))
	if m := middle[0].(combat.CombatantJoined); m == joined || !slices.Equal(m.PrecededBy(), []combat.CombatantID{"c"}) || m.Count() != 7 {
		t.Errorf("middle event %#v, want x at 7 preceded by c", m)
	}
	if len(joined.PrecededBy()) != 0 || joined.PrecededBy() == nil {
		t.Errorf("first entry: PrecededBy = %#v, want an empty list [FR-034]", joined.PrecededBy())
	}
	if moved.Count() != 15 || !slices.Equal(moved.PrecededBy(), []combat.CombatantID{"a"}) || moved.PreviousCount() != 5 || !slices.Equal(moved.PreviousPrecededBy(), []combat.CombatantID{"c"}) {
		t.Errorf("reposition event %#v, want d at 15 after a, previously 5 after c [FR-032]", moved)
	}
	if got := moved.PrecededBy(); len(got) == 1 {
		got[0] = "clobbered"
		if moved.PrecededBy()[0] != "a" {
			t.Errorf("PrecededBy returns the stored slice, not a copy [FR-035]")
		}
	}
	if joined.Reason() != combat.JoinReasonArrived || removed.Cause() != combat.RemovalCauseSurrendered || removed.Side() != combat.SideB || joined.Side() != combat.SideB {
		t.Errorf("reason, cause or side not carried: %#v %#v", joined, removed)
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-035 - the events of a transition are a copy: changing the returned slice leaves the stored list unchanged, and the order is the order they happened (grapple endings, then CombatantRemoved).
// Path: unhappy (immutability).
func TestRosterTransitionEventsAreACopyInOrder(t *testing.T) {
	t.Parallel()
	tr, err := rosGrapple(t, "c", "d").RemoveCombatant(cid("c"), combat.RemovalCauseFled)
	if err != nil {
		t.Fatal(err)
	}
	combattest.CheckEventsCopy(t, tr.Events)
	if got := encKinds(tr.Events()); !slices.Equal(got, []string{"condition_removed", "grapple_ended", "combatant_removed"}) {
		t.Errorf("events = %v, want the endings before combatant_removed", got)
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-012, FR-023 - JoinReason and RemovalCause are closed enums with snake_case names, an invalid zero value and a name for nothing outside the set.
// Path: unhappy (boundary).
func TestJoinReasonAndRemovalCauseAreClosedEnums(t *testing.T) {
	t.Parallel()
	for r, name := range map[combat.JoinReason]string{combat.JoinReasonSummoned: "summoned", combat.JoinReasonArrived: "arrived", 0: "unknown", 3: "unknown"} {
		if r.String() != name || r.Valid() != (name != "unknown") {
			t.Errorf("JoinReason(%d) = %q valid %v, want %q", int(r), r.String(), r.Valid(), name)
		}
	}
	for c, name := range map[combat.RemovalCause]string{combat.RemovalCauseFled: "fled", combat.RemovalCauseSurrendered: "surrendered",
		combat.RemovalCauseDisappeared: "disappeared", combat.RemovalCauseDeparted: "departed", 0: "unknown", 5: "unknown"} {
		if c.String() != name || c.Valid() != (name != "unknown") {
			t.Errorf("RemovalCause(%d) = %q valid %v, want %q", int(c), c.String(), c.Valid(), name)
		}
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-009, FR-019, FR-020, FR-042 - each rejection reports the snake_case result name its table names.
// Path: per case.
func TestRosterRejectionNames(t *testing.T) {
	t.Parallel()
	for r, want := range map[combat.RosterRejection]string{
		combat.RosterEncounterEndedRejection{}:                   "encounter_already_ended",
		combat.RosterUnknownCombatantRejection{ID: "z"}:          "unknown_combatant",
		combat.RosterDuplicateCombatantRejection{ID: "z"}:        "duplicate_combatant",
		combat.RosterCombatantIsActiveRejection{ID: "z"}:         "combatant_is_active",
		combat.RosterWouldEmptySideRejection{Side: combat.SideA}: "would_empty_side",
		combat.RosterLimitExceededRejection{}:                    "limit_exceeded",
	} {
		if r.String() != want {
			t.Errorf("%T.String() = %q, want %q", r, r.String(), want)
		}
	}
}
