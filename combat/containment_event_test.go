package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// containmentEvents returns one event of each of the three kinds, produced by the real rules.
func containmentEvents(t *testing.T) []combat.ContainmentEvent {
	t.Helper()
	contained := ctContained(t, mustEnter(t, ctEnterInput(t, "behir"), 1))
	var out []combat.ContainmentEvent
	for _, e := range contained.Events() {
		if c, ok := e.(combat.OccupantContained); ok {
			out = append(out, c)
		}
	}
	died := ctReleased(t, ctRelease(t, ctDied(t, "behir")))
	for _, e := range died.Events() {
		if c, ok := e.(combat.ContainerTrapped); ok {
			out = append(out, c)
		}
	}
	gone := ctReleased(t, ctRelease(t, ctDied(t, "cube")))
	for _, e := range gone.Events() {
		if c, ok := e.(combat.OccupantReleased); ok {
			out = append(out, c)
		}
	}
	return out
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-064 - the three containment events are ContainmentEvent and Event members with the kinds occupant_contained, container_trapped and occupant_released, unique, past tense, with Combatants a fresh copy [occupant, container], and equal by value.
// Path: unhappy (immutability, duplicate).
func TestContainmentEventFamily(t *testing.T) {
	t.Parallel()
	events := containmentEvents(t)
	kinds := []string{"occupant_contained", "container_trapped", "occupant_released"}
	if len(events) != len(kinds) {
		t.Fatalf("got %d events, want %d", len(events), len(kinds))
	}
	wide := make([]combat.Event, len(events))
	for i, e := range events {
		var _ combat.Event = e
		combattest.CheckEvent(t, e, kinds[i], gid("p"), gid("m"))
		wide[i] = e
	}
	combattest.CheckEventKinds(t, wide)
	again := containmentEvents(t)
	for i := range events {
		if events[i] != again[i] {
			t.Errorf("%s: events built from equal inputs must be equal", kinds[i])
		}
	}
	if events[0].(combat.OccupantContained).Mode() != combat.ContainmentModeSwallowed || events[1].(combat.ContainerTrapped).Feet() != 15 ||
		events[2].(combat.OccupantReleased).Cause() != combat.ContainmentLeaveCauseContainerDied {
		t.Error("the event accessors disagree with the rules that produced them")
	}
}
