package combat_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/encounter-aggregate.md FR-013 - ProblemsOf finds the problems of an error that joins them, wrapped or not, in order, and none in other errors.
// Path: unhappy (invalid input).
func TestProblemsOfWalksJoinedAndWrappedErrors(t *testing.T) {
	t.Parallel()
	_, _, err := (combat.Encounter{}).Start(eid(), encEstablished(t, "a", "b"), []combat.Combatant{encPC(t, "a", combat.SideA, 5, 5), encPC(t, "a", combat.SideA, 5, 5)})
	want := []combat.EncounterProblem{combat.MissingSideProblem{Side: combat.SideB}, combat.DuplicateCombatantIDProblem{ID: cid("a")}, combat.TurnOrderEntryWithoutCombatantProblem{ID: cid("b")}}
	if got := combat.ProblemsOf(err); !deepEqual(got, want) {
		t.Fatalf("ProblemsOf = %#v, want %#v", got, want)
	}
	wrapped := fmt.Errorf("start: %w", err)
	if got := combat.ProblemsOf(wrapped); !deepEqual(got, want) {
		t.Errorf("ProblemsOf of a wrapped error = %#v, want the same problems", got)
	}
	if got := combat.ProblemsOf(errors.Join(errors.New("x"), err)); !deepEqual(got, want) {
		t.Errorf("ProblemsOf of a join with another error = %#v", got)
	}
}
