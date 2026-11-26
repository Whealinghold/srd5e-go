package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/encounter-aggregate.md - Encounter.ApplyHealing and Encounter.ReduceHitPointMaximum return the transition of a living combatant, with the new Hit Points and the events, and no error.
// Path: happy.
func TestEncounterHealingAndMaximumReductionSucceed(t *testing.T) {
	t.Parallel()
	e := encWith(t, encC(t, "a", combat.SideA, pcStatus(t, 5, 20, 0)))
	tr, err := e.ApplyHealing(cid("a"), 7)
	healed, events := encOK(t, tr, err)
	if got := encMember(t, healed, "a").HitPoints().HitPoints().Current(); got != 12 || len(events) == 0 {
		t.Errorf("healed to %d with %d events, want 12 and at least one event", got, len(events))
	}
	tr, err = e.ReduceHitPointMaximum(cid("a"), 5)
	reduced, events := encOK(t, tr, err)
	if got := encMember(t, reduced, "a").HitPoints().HitPoints().Maximum(); got != 15 || len(events) == 0 {
		t.Errorf("maximum reduced to %d with %d events, want 15 and at least one event", got, len(events))
	}
}
