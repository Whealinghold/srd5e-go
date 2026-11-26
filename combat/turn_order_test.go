package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/initiative-and-actions.md FR-153 - a TieBreak with no id or a blank id is an invalid-input error; one id is accepted; the ids are copied.
// Path: unhappy (empty case).
func TestNewTieBreak(t *testing.T) {
	t.Parallel()
	for name, ids := range map[string][]combat.CombatantID{"nil": nil, "empty": {}, "blank": {trkA, ""}} {
		if tb, err := combat.NewTieBreak(ids); !errors.Is(err, srderr.ErrInvalidInput) || len(tb.IDs()) != 0 {
			t.Errorf("%s: NewTieBreak = %+v, %v; want a zero TieBreak and invalid input", name, tb, err)
		}
	}
	ids := []combat.CombatantID{trkA}
	tb, err := combat.NewTieBreak(ids)
	if err != nil {
		t.Fatal(err)
	}
	ids[0] = trkB
	got := tb.IDs()
	got[0] = trkC
	if tb.IDs()[0] != trkA {
		t.Error("TieBreak shares its list [FR-096]")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-153 - NewTieBreak never panics for any ids.
// Path: unhappy (invalid input).
func FuzzNewTieBreak(f *testing.F) {
	f.Add("a", "b")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, a, b string) {
		tb, err := combat.NewTieBreak([]combat.CombatantID{combat.CombatantID(a), combat.CombatantID(b)})
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("error %v does not match ErrInvalidInput", err)
		}
		if err == nil && len(tb.IDs()) != 2 {
			t.Fatalf("tie-break %+v lost its ids", tb)
		}
	})
}

// Requirement: doc/specs/initiative-and-actions.md FR-026, FR-154 - the order is a defensively copied view; StandingOf returns the standing of a combatant and an invalid-input error for an id not in the order; Equal compares ids and counts.
// Path: unhappy (immutability).
func TestTurnOrderViewAndQueries(t *testing.T) {
	t.Parallel()
	order := trkABC(t).Order()
	view := order.Standings()
	view[0] = combat.InitiativeStanding{Combatant: trkC, Count: 1}
	if got := order.Standings(); got[0].Combatant != trkA || order.Len() != 3 {
		t.Errorf("order = %v after changing a view, want it unchanged [FR-026]", got)
	}
	if s, err := order.StandingOf(trkB); err != nil || s != (combat.InitiativeStanding{Combatant: trkB, Count: 90}) {
		t.Errorf("StandingOf(b) = %+v, %v", s, err)
	}
	if _, err := order.StandingOf("x"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("StandingOf(x) = %v, want invalid input [FR-154]", err)
	}
	if !order.Equal(trkABC(t).Order()) || order.Equal(combat.TurnOrder{}) {
		t.Error("Equal is wrong")
	}
	var zero combat.TurnOrder
	if zero.Len() != 0 || len(zero.Standings()) != 0 {
		t.Error("the zero TurnOrder is not empty")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-120 - the String of each reason is its snake_case name and an undefined reason is unknown.
// Path: unhappy (boundary).
func TestTieBreakRejectionReasonString(t *testing.T) {
	t.Parallel()
	want := []string{"unknown_id", "repeated_id", "not_tied", "spans_groups", "partial", "duplicate_group"}
	for i, w := range want {
		if got := combat.TieBreakRejectionReason(i + 1).String(); got != w {
			t.Errorf("reason %d = %q, want %q", i+1, got, w)
		}
	}
	for _, r := range []combat.TieBreakRejectionReason{0, 7} {
		if r.String() != "unknown" {
			t.Errorf("reason %d = %q, want unknown", r, r.String())
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-154 - StandingOf finds the first and the last standing of the order and an id that sorts before every id is not in it.
// Path: unhappy (boundary).
func TestTurnOrderStandingOfEnds(t *testing.T) {
	t.Parallel()
	order := trkABC(t).Order()
	for _, want := range order.Standings() {
		if got, err := order.StandingOf(want.Combatant); err != nil || got != want {
			t.Errorf("StandingOf(%s) = %+v, %v; want %+v", want.Combatant, got, err, want)
		}
	}
	if _, err := order.StandingOf("A"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("StandingOf(A) = %v, want invalid input", err)
	}
}
