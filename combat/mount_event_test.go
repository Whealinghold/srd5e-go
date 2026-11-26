package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

func mntEvents(t *testing.T, control combat.MountControl) []combat.MountEvent {
	t.Helper()
	ctx := context.Background()
	in := mntInput(t)
	in.Control = control
	mounted, _ := combat.NewMountCreature().Run(ctx, mntCommand(t, in))
	dismounted, _ := combat.NewDismountCreature().Run(ctx, mntDismountCommand(t, mntMounted(t, control), mntTurn(t, combat.RiderTurnActive, 30, 30)))
	fell, _ := mntFallWith(t, dicetest.NewFixedRoller(), mntMounted(t, control), combat.FallingOffTriggerMountMovedAgainstItsWill, d20.AutomaticFailure{}, combat.LandingSpaceUnoccupiedWithin5Feet)
	return []combat.MountEvent{mounted.(combat.MountCreatureApplied).Events()[0], dismounted.(combat.DismountCreatureApplied).Events()[0], fell.(combat.ResolveFallingOffFellOff).Events()[0]}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-055, FR-068, FR-069 - the three events are MountEvent and Event members with the kinds creature_mounted, creature_dismounted and fell_off_mount, unique, about [rider, mount] as a fresh copy, and equal by value (a different field differs).
// Path: unhappy (immutability).
func TestMountEvents(t *testing.T) {
	t.Parallel()
	events := mntEvents(t, combat.MountControlControlled)
	kinds := []string{"creature_mounted", "creature_dismounted", "fell_off_mount"}
	for i, e := range events {
		var _ combat.Event = e
		combattest.CheckEvent(t, e, kinds[i], gid("rider"), gid("horse"))
	}
	combattest.CheckEventKinds(t, []combat.Event{events[0], events[1], events[2]})
	again, other := mntEvents(t, combat.MountControlControlled), mntEvents(t, combat.MountControlIndependent)
	for i := range events {
		if events[i] != again[i] || events[i] == other[i] {
			t.Errorf("%s: events must compare by every field", kinds[i])
		}
	}
}
