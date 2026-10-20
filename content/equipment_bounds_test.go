package content_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: equipSpec FR-080, FR-091, FR-098, FR-102, FR-103, FR-108, FR-112 - every
// integer of the equipment types is accepted exactly at its lower and upper bound
// (2^31 - 1 above), and the enum constants have the ordinals their names list.
// Path: unhappy (boundary).
func TestEquipmentBoundsAreInclusive(t *testing.T) {
	t.Parallel()
	const top = math.MaxInt32
	price := fixedCost(1, content.CoinGP)
	w := content.WeightSpec(content.NoWeight{})
	ok := func(name string, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("%s at its bound: %v", name, err)
		}
	}
	_, err := content.NewCost(top, content.CoinPP, content.CostPerMile)
	ok("NewCost amount", err)
	_, err = content.NewWeaponFixedDamage(2, content.DamageFire)
	ok("fixed damage 2", err)
	_, err = content.NewWeaponFixedDamage(top, content.DamageFire)
	ok("fixed damage max", err)
	_, err = content.NewRangePair(1, 2)
	ok("range 1/2", err)
	_, err = content.NewRangePair(1, top)
	ok("range 1/max", err)
	_, err = content.NewDonDoffMinutes(top)
	ok("minutes max", err)
	_, err = content.NewCappedDexterity(top)
	ok("cap max", err)
	_, err = content.NewBaseRule(1, content.NoDexterity{})
	ok("base 1", err)
	_, err = content.NewBaseRule(30, content.NoDexterity{})
	ok("base 30", err)
	_, err = content.NewBonusRule(1)
	ok("bonus 1", err)
	_, err = content.NewBonusRule(10)
	ok("bonus 10", err)
	rule, _ := content.NewBaseRule(11, content.FullDexterity{})
	for _, s := range []int{1, 30} {
		_, err = content.NewArmor("a", "A", noSource, content.ArmorLight, rule, s, false, content.Weight{}, price)
		ok("strength", err)
	}
	_, err = content.NewToolUtilizeAction("x", top)
	ok("dc max", err)
	_, err = content.NewAmmunitionData(top, "Q")
	ok("ammunition max", err)
	_, err = content.NewPackItem("a", top, "")
	ok("quantity max", err)
	_, err = content.NewCraftingOverride(content.MustCost(1, content.CoinGP, 0), top, nil)
	ok("days max", err)
	_, err = content.NewMount("a", "A", noSource, top, price)
	ok("capacity max", err)
	_, err = content.NewAirborneOrWaterborneVehicle("a", "A", noSource, top, top, top, top, top, top, top, w, price)
	ok("vehicle max", err)
	_, err = content.NewAirborneOrWaterborneVehicle("a", "A", noSource, 0.5, 1, 1, 0.5, 1, 1, 1, w, price)
	ok("vehicle min", err)
	_, err = content.NewSpellScrollCost("a", "A", noSource, 0, top, price)
	ok("scroll cost days max", err)
	if _, err := content.NewCost(0, content.CoinCP, content.CostRate(-1)); err == nil {
		t.Error("a negative CostRate was accepted")
	}
	if content.AmmunitionArrow.String() != "Arrow" || content.AmmunitionBolt.String() != "Bolt" || content.AmmunitionBullet.String() != "Bullet" || content.AmmunitionNeedle.String() != "Needle" {
		t.Error("AmmunitionType constants are out of order")
	}
	if content.ToolGroupArtisans.String() != "Artisan's Tools" || content.CraftMeleeWeapons.String() != "MeleeWeapons" || content.CraftHeavyArmor.String() != "HeavyArmor" ||
		content.ArmorShield.String() != "Shield" || content.WeaponMartial.String() != "Martial" || content.WeaponRanged.String() != "Ranged" ||
		content.EquipmentSpellScrollCost.String() != "spellScrollCost" || content.SettlementCity.String() != "city" || content.AvailabilityCityOnly.String() != "City only" {
		t.Error("an equipment enum constant has the wrong ordinal")
	}
}
