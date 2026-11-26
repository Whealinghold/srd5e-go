package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: doc/specs/returning-to-life.md FR-013, FR-016 - the RevivalEvent family has one member, CreatureReturnedToLife, with the constant kind creature_returned_to_life, the creature as its only combatant (a fresh copy) and the Hit Points it returned with.
// Path: unhappy (name collision).
func TestRevivalEventFamily(t *testing.T) {
	t.Parallel()
	var e combat.RevivalEvent = rtlApplied(t, rtlCommand(t, 6)).Events()[0].(combat.CreatureReturnedToLife)
	combattest.CheckEvent(t, e, "creature_returned_to_life", gid("hero"))
	combattest.CheckEventKinds(t, []combat.Event{e})
	combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{e} })
	if got := e.(combat.CreatureReturnedToLife).HitPoints(); got != 6 {
		t.Errorf("HitPoints = %d, want 6", got)
	}
	list := e.Combatants()
	list[0] = gid("zzz")
	if e.Combatants()[0] != gid("hero") {
		t.Error("Combatants returned a view of the event's own list")
	}
}
