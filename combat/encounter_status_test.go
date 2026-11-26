package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/encounter-aggregate.md FR-011, FR-042, FR-090 - there are two sides, named side_a and side_b; the zero Side and a third value are invalid; the end reasons and statuses are prefixed variants of their sets.
// Path: unhappy (invalid input).
func TestSidesEndReasonsAndStatuses(t *testing.T) {
	t.Parallel()
	for side, want := range map[combat.Side]struct {
		name  string
		valid bool
	}{combat.SideA: {"side_a", true}, combat.SideB: {"side_b", true}, 0: {"unknown", false}, 3: {"unknown", false}, -1: {"unknown", false}} {
		if side.String() != want.name || side.Valid() != want.valid {
			t.Errorf("Side(%d) = %q valid %v, want %q %v", int(side), side.String(), side.Valid(), want.name, want.valid)
		}
	}
	e := encABC(t)
	defeated := runEnd(t, e, combat.EndReasonSideDefeated{Side: combat.SideB}).(combat.EndEncounterApplied).Encounter()
	agreed := runEnd(t, e, combat.EndReasonMutualAgreement{}).(combat.EndEncounterApplied).Encounter()
	if defeated.Status() == agreed.Status() || e.Status() == defeated.Status() || defeated.Equal(agreed) {
		t.Errorf("the statuses of different end reasons compare equal [FR-053]")
	}
	if defeated.Status() != (combat.EncounterEndedStatus{Reason: combat.EndReasonSideDefeated{Side: combat.SideB}}) {
		t.Errorf("status = %#v", defeated.Status())
	}
}
