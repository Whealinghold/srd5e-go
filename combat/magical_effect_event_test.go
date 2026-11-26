package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: doc/specs/magical-effects.md FR-147 and table 4.15a - the four events of the MagicalEffectEvent family have the constant kinds curse_removed, shape_shifted, reverted_to_true_form and creature_teleported, unique together, and report their combatants in the table's order as a fresh copy.
// Path: unhappy (name collision).
func TestMagicalEffectEventFamily(t *testing.T) {
	t.Parallel()
	removed := (combat.CurseRemoval{}).Remove(gid("a"), meEffects(meCurse("c1")), meEID("c1"), combat.RemoveCurseSpell).(combat.RemoveCurseApplied).Events()[0]
	shifted := meShift(t, combat.ConditionSet{}, meEffects(), nil).(combat.ShiftShapeApplied).Events()[0]
	reverted, err := (combat.ShapeShifting{}).Revert(combat.RevertToTrueFormCommand{Creature: gid("a"), State: meShifted(t)})
	if err != nil {
		t.Fatal(err)
	}
	revertedEvent := reverted.(combat.RevertToTrueFormApplied).Events()[0]
	teleported := meTeleport(t, combat.TeleportFields{Teleporter: gid("a"), Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired,
		Policy: combat.Accompany, Touched: []combat.CombatantID{gid("b"), gid("c")}}).(combat.TeleportArrivedAtDestination).Events()[0]

	combattest.CheckEvent(t, removed, "curse_removed", gid("a"))
	combattest.CheckEvent(t, shifted, "shape_shifted", gid("hero"))
	combattest.CheckEvent(t, revertedEvent, "reverted_to_true_form", gid("a"))
	combattest.CheckEvent(t, teleported, "creature_teleported", gid("a"), gid("b"), gid("c"))
	combattest.CheckEventKinds(t, []combat.Event{removed, shifted, revertedEvent, teleported})
	combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{teleported} })

	// The teleporter's own accessors and the copy of the combatant list.
	tp := teleported.(combat.CreatureTeleported)
	list := tp.Combatants()
	list[1] = gid("zzz")
	if tp.Combatants()[1] != gid("b") {
		t.Error("Combatants returned a view of the event's own list")
	}
}
