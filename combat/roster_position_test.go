package combat_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rosInsertAll inserts the ids in turn into the tracker, each at its position, and returns the tracker and the last standing.
func rosInsertAll(tb testing.TB, t combat.TurnTracker, ids []string, positions []combat.RosterPosition) (combat.TurnTracker, combat.InitiativeStanding) {
	tb.Helper()
	var last combat.InitiativeStanding
	for i, id := range ids {
		step, err := t.InsertCombatant(rosProfile(tb, id), positions[i])
		if err != nil {
			tb.Fatalf("InsertCombatant(%s): %v", id, err)
		}
		a, ok := step.(combat.RosterStepApplied)
		if !ok {
			tb.Fatalf("InsertCombatant(%s) = %#v, want RosterStepApplied", id, step)
		}
		t, last = a.Tracker(), a.Standing()
	}
	return t, last
}

func rosTrackerOrder(t combat.TurnTracker) string {
	var ids []string
	for _, s := range t.Order().Standings() {
		ids = append(ids, string(s.Combatant))
	}
	return strings.Join(ids, ",")
}

// Requirement: per case (table 4.1a: FR-001, FR-002, FR-003, FR-006)
// Path: per case.
func TestPlacementCases(t *testing.T) {
	t.Parallel()
	type row struct {
		name, requirement, path string
		ids                     []string
		positions               []combat.RosterPosition
		order                   string
		count                   combat.InitiativeCount
	}
	rows := []row{
		{"after a combatant already past", "FR-002", "happy", []string{"x"}, []combat.RosterPosition{rosAfter("a")}, "a,x,b,c,d", 15},
		{"after the active combatant", "FR-002, FR-006", "happy", []string{"x"}, []combat.RosterPosition{rosAfter("b")}, "a,b,x,c,d", 15},
		{"after the last", "FR-002", "unhappy (boundary)", []string{"x"}, []combat.RosterPosition{rosAfter("d")}, "a,b,c,d,x", 5},
		{"two after one anchor", "FR-002", "unhappy (rule interaction)", []string{"x", "y"}, []combat.RosterPosition{rosAfter("a"), rosAfter("a")}, "a,y,x,b,c,d", 15},
		{"highest count", "FR-003", "unhappy (boundary)", []string{"x"}, []combat.RosterPosition{rosAt(20, rosBehind)}, "x,a,b,c,d", 20},
		{"tied with two, before", "FR-003", "unhappy (rule interaction)", []string{"x"}, []combat.RosterPosition{rosAt(15, rosBefore)}, "x,a,b,c,d", 15},
		{"tied with two, after", "FR-003", "unhappy (rule interaction)", []string{"x"}, []combat.RosterPosition{rosAt(15, rosBehind)}, "a,b,x,c,d", 15},
		{"tied with one, before", "FR-003", "unhappy (rule interaction)", []string{"x"}, []combat.RosterPosition{rosAt(10, rosBefore)}, "a,b,x,c,d", 10},
		{"tied with one, after", "FR-003", "unhappy (rule interaction)", []string{"x"}, []combat.RosterPosition{rosAt(10, rosBehind)}, "a,b,c,x,d", 10},
		{"between counts, before tied", "FR-003", "happy", []string{"x"}, []combat.RosterPosition{rosAt(7, rosBefore)}, "a,b,c,x,d", 7},
		{"between counts, after tied", "FR-003", "happy", []string{"x"}, []combat.RosterPosition{rosAt(7, rosBehind)}, "a,b,c,x,d", 7},
		{"lowest tie, after", "FR-003", "unhappy (boundary)", []string{"x"}, []combat.RosterPosition{rosAt(5, rosBehind)}, "a,b,c,d,x", 5},
		{"lowest tie, before", "FR-003", "unhappy (boundary)", []string{"x"}, []combat.RosterPosition{rosAt(5, rosBefore)}, "a,b,c,x,d", 5},
		{"below all", "FR-003, FR-001", "unhappy (boundary)", []string{"x"}, []combat.RosterPosition{rosAt(-3, rosBefore)}, "a,b,c,d,x", -3},
		{"zero count", "FR-001", "unhappy (boundary)", []string{"x"}, []combat.RosterPosition{rosAt(0, rosBehind)}, "a,b,c,d,x", 0},
		{"two at one count, before keeps the latest first", "FR-003", "unhappy (rule interaction)", []string{"x", "y"}, []combat.RosterPosition{rosAt(10, rosBefore), rosAt(10, rosBefore)}, "a,b,y,x,c,d", 10},
		{"two at one count, after keeps the latest last", "FR-003", "unhappy (rule interaction)", []string{"x", "y"}, []combat.RosterPosition{rosAt(10, rosBehind), rosAt(10, rosBehind)}, "a,b,c,x,y,d", 10},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			base := rosEnc(t).Tracker()
			got, standing := rosInsertAll(t, base, r.ids, r.positions)
			if o := rosTrackerOrder(got); o != r.order {
				t.Errorf("order = %s, want %s [%s]", o, r.order, r.requirement)
			}
			if standing.Combatant != cid(r.ids[len(r.ids)-1]) || standing.Count != r.count {
				t.Errorf("standing = %+v, want the last inserted at count %d [%s]", standing, r.count, r.requirement)
			}
			if s, err := got.Order().StandingOf(standing.Combatant); err != nil || s.Count != r.count {
				t.Errorf("the order holds %+v (%v), want count %d", s, err, r.count)
			}
			if got.Active().Combatant != cid("b") || got.Round() != 2 {
				t.Errorf("active %v in round %d, want b in round 2: a roster operation must not move the turn [FR-005]", got.Active().Combatant, got.Round())
			}
			prev := combat.InitiativeCount(1 << 30)
			for _, s := range got.Order().Standings() {
				if s.Count > prev {
					t.Errorf("counts increase along %s [FR-004]", rosTrackerOrder(got))
				}
				prev = s.Count
			}
		})
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-001 - both variants of RosterPosition compare by value (usable as a map key), and a count of any integer, negative and zero included, is accepted.
// Path: unhappy (boundary).
func TestRosterPositionsCompareByValue(t *testing.T) {
	t.Parallel()
	if rosAfter("a") != rosAfter("a") || rosAfter("a") == rosAfter("b") {
		t.Errorf("RosterAfterCombatant does not compare by its anchor")
	}
	if rosAt(3, rosBefore) != rosAt(3, rosBefore) || rosAt(3, rosBefore) == rosAt(3, rosBehind) || rosAt(3, rosBefore) == rosAt(4, rosBefore) || rosAfter("a") == rosAt(3, rosBefore) {
		t.Errorf("RosterAtInitiative does not compare by count and tie placement")
	}
	seen := map[combat.RosterPosition]int{rosAfter("a"): 1, rosAt(-1, rosBehind): 2}
	if seen[rosAfter("a")] != 1 || seen[rosAt(-1, rosBehind)] != 2 {
		t.Errorf("positions are not usable as map keys: %v", seen)
	}
	for _, count := range []combat.InitiativeCount{-1 << 30, -1, 0, 1 << 30} {
		if _, err := rosEnc(t).Tracker().InsertCombatant(rosProfile(t, "x"), rosAt(count, rosBehind)); err != nil {
			t.Errorf("count %d: %v, want any integer accepted", count, err)
		}
	}
	if combat.TiePlacementBeforeTied.String() != "before_tied" || combat.TiePlacementAfterTied.String() != "after_tied" || combat.TiePlacement(0).String() != "unknown" {
		t.Errorf("TiePlacement names are not before_tied, after_tied, unknown")
	}
	if combat.TiePlacement(0).Valid() || combat.TiePlacement(3).Valid() || !rosBefore.Valid() || !rosBehind.Valid() {
		t.Errorf("TiePlacement.Valid disagrees with the two defined values")
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-001, FR-010, FR-039 - a position that is nil, has a blank anchor or an undefined tie placement is invalid input to a tracker operation, never a panic.
// Path: unhappy (invalid input).
func TestInvalidPositionsAreInputErrors(t *testing.T) {
	t.Parallel()
	base := rosEnc(t).Tracker()
	for name, pos := range map[string]combat.RosterPosition{
		"nil": nil, "blank anchor": combat.RosterAfterCombatant{}, "zero tie placement": combat.RosterAtInitiative{Count: 3},
		"tie placement out of range": combat.RosterAtInitiative{Count: 3, Tie: 9},
	} {
		if _, err := base.InsertCombatant(rosProfile(t, "x"), pos); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("InsertCombatant with %s: err = %v, want ErrInvalidInput", name, err)
		}
		if _, err := base.MoveCombatant(cid("c"), pos); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("MoveCombatant with %s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-004 - after every roster operation the counts do not increase along the order, each id occurs once, and the order depends only on the previous order and the operation: the same sequence of operations gives an equal tracker twice.
// Path: unhappy (determinism).
func TestRosterSequencesKeepTheOrderValidAndAreDeterministic(t *testing.T) {
	t.Parallel()
	positions := []combat.RosterPosition{rosAfter("a"), rosAfter("d"), rosAt(20, rosBefore), rosAt(15, rosBehind), rosAt(10, rosBefore), rosAt(0, rosBehind), rosAt(-9, rosBefore), rosAfter("x1")}
	run := func() combat.TurnTracker {
		tr := rosEnc(t).Tracker()
		ids := []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7"}
		for i, id := range ids {
			var step combat.RosterStep
			var err error
			if step, err = tr.InsertCombatant(rosProfile(t, id), positions[i]); err != nil {
				t.Fatal(err)
			}
			tr = step.(combat.RosterStepApplied).Tracker()
			if i%3 == 2 { // move the previous joiner and drop an old one
				step, _ = tr.MoveCombatant(cid(ids[i-1]), rosAt(combat.InitiativeCount(i), rosBehind))
				tr = step.(combat.RosterStepApplied).Tracker()
				step, _ = tr.RemoveCombatant(cid("c"))
				if a, ok := step.(combat.RosterStepApplied); ok {
					tr = a.Tracker()
				}
			}
			seen := map[combat.CombatantID]bool{}
			prev := combat.InitiativeCount(1 << 30)
			for _, s := range tr.Order().Standings() {
				if s.Count > prev || seen[s.Combatant] {
					t.Fatalf("after step %d the order %s is not ranked and unique", i, rosTrackerOrder(tr))
				}
				prev, seen[s.Combatant] = s.Count, true
			}
		}
		return tr
	}
	if a, b := run(), run(); !a.Equal(b) || !slices.Equal(a.Order().Standings(), b.Order().Standings()) {
		t.Errorf("the same operations gave two different trackers: %s and %s", rosTrackerOrder(a), rosTrackerOrder(b))
	}
}
