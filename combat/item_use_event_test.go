package combat_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/item-use-effects.md FR-072 - the seven item-use events are one closed family of immutable values, each produced by its operation: unique constant snake_case kinds, the user first, a fresh duplicate-free combatant list (a creature using an item on itself is named once), comparable by value.
// Path: happy; unhappy (immutability, duplicate).
func TestItemUseEventsAreAClosedFamily(t *testing.T) {
	t.Parallel()
	thrown, err := thrownRun(t, thrownIn(), dicetest.NewFixedRoller(1, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	applied, _ := combat.BasicPoisonRules{}.ApplyToWeapon(gid("archer"))
	delivered, _ := applied.Coatings()[0].AfterWeaponDamage(content.DamagePiercing, 1)
	drink, _ := combat.AntitoxinRules{}.Drink(gid("drinker"))
	wornOff, _ := drink.Effect().Advance(time.Hour)
	potion, err := potionRun(t, "medic", "hero", pcStatus(t, 5, 20, 0), combat.AdministerPotion{}, dicetest.NewFixedRoller(1, 1))
	if err != nil {
		t.Fatal(err)
	}
	kit, err := combat.FreshHealersKit().UseToStabilize(gid("medic"), gid("hero"), dyingStatus(t, 20, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	sorted, err := combat.FreshHealersKit().UseToStabilize(gid("adam"), gid("zed"), dyingStatus(t, 20, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	combattest.CheckEvent(t, sorted.(combat.HealersKitUsed).Events()[1], "healers_kit_used", gid("adam"), gid("zed"))
	family := []combat.ItemUseEvent{
		thrown.(combat.ThrownItemResolved).Events()[0], applied.Events()[0], delivered.Events()[0], drink.Events()[0], wornOff.Events()[0],
		potion.(combat.PotionOfHealingUsed).Events()[1].(combat.ItemUseEvent), kit.(combat.HealersKitUsed).Events()[1].(combat.ItemUseEvent),
	}
	want := []struct {
		kind string
		who  []combat.CombatantID
	}{
		{"thrown_item_used", []combat.CombatantID{gid("thrower"), gid("target")}}, {"basic_poison_applied", []combat.CombatantID{gid("archer")}},
		{"basic_poison_spent", []combat.CombatantID{gid("archer")}}, {"antitoxin_drunk", []combat.CombatantID{gid("drinker")}},
		{"antitoxin_wore_off", []combat.CombatantID{gid("drinker")}}, {"potion_of_healing_used", []combat.CombatantID{gid("medic"), gid("hero")}},
		{"healers_kit_used", []combat.CombatantID{gid("medic"), gid("hero")}},
	}
	var events []combat.Event
	for i, e := range family {
		combattest.CheckEvent(t, e, want[i].kind, want[i].who...)
		events = append(events, e)
	}
	combattest.CheckEventKinds(t, events)
	if a, b := delivered.Events()[0], delivered.Events()[0]; a != b {
		t.Error("equal events are not equal with ==")
	}
	expired, _ := applied.Coatings()[0].Advance(time.Minute)
	if expired.Events()[0] == delivered.Events()[0] {
		t.Error("a delivered and an expired BasicPoisonSpent compare equal")
	}
}
