package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The weapon fixtures are synthetic: ids, categories, dice, properties and
// masteries are the ones the case tables of weapon-properties-and-mastery.md
// print; weight and price are placeholders.

func plain(p content.WeaponProperty) content.WeaponPropertyUse { return must(content.NewPlainUse(p)) }

func rangePair(n, l int) content.RangePair { return must(content.NewRangePair(n, l)) }

func ammoUse(n, l int, a content.AmmunitionType) content.WeaponPropertyUse {
	return must(content.NewAmmunitionUse(rangePair(n, l), a))
}

func thrownUse(n, l int) content.WeaponPropertyUse {
	return must(content.NewThrownUse(rangePair(n, l)))
}

func versatileUse(count int, d dice.Die) content.WeaponPropertyUse {
	return must(content.NewVersatileUse(dice.MustExpression(count, d, 0)))
}

func twoHandedUse(c content.TwoHandedCondition) content.WeaponPropertyUse {
	return must(content.NewTwoHandedUse(c))
}

func diceDamage(count int, d dice.Die, t content.DamageType) content.WeaponDamage {
	return must(content.NewWeaponDiceDamage(dice.MustExpression(count, d, 0), t))
}

func wpn(id string, cat content.WeaponCategory, typ content.WeaponType, dmg content.WeaponDamage, m content.MasteryProperty, props ...content.WeaponPropertyUse) content.Weapon {
	return must(content.NewWeapon(content.ItemID(id), id, content.SourceCitation{}, cat, typ, dmg, props, m, lbs(1), price(gp(1))))
}

var (
	finesse = plain(content.WeaponPropertyFinesse)
	heavy   = plain(content.WeaponPropertyHeavy)
	light   = plain(content.WeaponPropertyLight)
	loading = plain(content.WeaponPropertyLoading)
	reach   = plain(content.WeaponPropertyReach)
)

const (
	simple  = content.WeaponSimple
	martial = content.WeaponMartial
	melee   = content.WeaponMelee
	ranged  = content.WeaponRanged
)

func weaponFixtures(tb testing.TB) map[string]content.Weapon {
	tb.Helper()
	d := diceDamage
	slashing, piercing, bludgeoning := content.DamageSlashing, content.DamagePiercing, content.DamageBludgeoning
	list := []content.Weapon{
		wpn("club", simple, melee, d(1, dice.D4(), bludgeoning), content.MasterySlow, light),
		wpn("dagger", simple, melee, d(1, dice.D4(), piercing), content.MasteryNick, finesse, light, thrownUse(20, 60)),
		wpn("dart", simple, ranged, d(1, dice.D4(), piercing), content.MasteryVex, finesse, thrownUse(20, 60)),
		wpn("handaxe", simple, melee, d(1, dice.D6(), slashing), content.MasteryVex, light, thrownUse(20, 60)),
		wpn("javelin", simple, melee, d(1, dice.D6(), piercing), content.MasterySlow, thrownUse(30, 120)),
		wpn("mace", simple, melee, d(1, dice.D6(), bludgeoning), content.MasterySap),
		wpn("spear", simple, melee, d(1, dice.D6(), piercing), content.MasterySap, thrownUse(20, 60), versatileUse(1, dice.D8())),
		wpn("light_crossbow", simple, ranged, d(1, dice.D8(), piercing), content.MasterySlow, ammoUse(80, 320, content.AmmunitionBolt), loading, twoHandedUse(content.TwoHandedAlways)),
		wpn("shortbow", simple, ranged, d(1, dice.D6(), piercing), content.MasteryVex, ammoUse(80, 320, content.AmmunitionArrow), twoHandedUse(content.TwoHandedAlways)),
		wpn("sling", simple, ranged, d(1, dice.D4(), bludgeoning), content.MasterySlow, ammoUse(30, 120, content.AmmunitionBullet)),
		wpn("longsword", martial, melee, d(1, dice.D8(), slashing), content.MasterySap, versatileUse(1, dice.D10())),
		wpn("rapier", martial, melee, d(1, dice.D8(), piercing), content.MasteryVex, finesse),
		wpn("scimitar", martial, melee, d(1, dice.D6(), slashing), content.MasteryNick, finesse, light),
		wpn("shortsword", martial, melee, d(1, dice.D6(), piercing), content.MasteryVex, finesse, light),
		wpn("whip", martial, melee, d(1, dice.D4(), slashing), content.MasterySlow, finesse, reach),
		wpn("greataxe", martial, melee, d(1, dice.D12(), slashing), content.MasteryCleave, heavy, twoHandedUse(content.TwoHandedAlways)),
		wpn("greatsword", martial, melee, d(2, dice.D6(), slashing), content.MasteryGraze, heavy, twoHandedUse(content.TwoHandedAlways)),
		wpn("glaive", martial, melee, d(1, dice.D10(), slashing), content.MasteryGraze, heavy, reach, twoHandedUse(content.TwoHandedAlways)),
		wpn("lance", martial, melee, d(1, dice.D10(), piercing), content.MasteryTopple, heavy, reach, twoHandedUse(content.TwoHandedUnlessMounted)),
		wpn("maul", martial, melee, d(2, dice.D6(), bludgeoning), content.MasteryTopple, heavy, twoHandedUse(content.TwoHandedAlways)),
		wpn("pike", martial, melee, d(1, dice.D10(), piercing), content.MasteryPush, heavy, reach, twoHandedUse(content.TwoHandedAlways)),
		wpn("longbow", martial, ranged, d(1, dice.D8(), piercing), content.MasterySlow, ammoUse(150, 600, content.AmmunitionArrow), heavy, twoHandedUse(content.TwoHandedAlways)),
		wpn("hand_crossbow", martial, ranged, d(1, dice.D6(), piercing), content.MasteryVex, ammoUse(30, 120, content.AmmunitionBolt), light, loading),
		wpn("heavy_crossbow", martial, ranged, d(1, dice.D10(), piercing), content.MasteryPush, ammoUse(100, 400, content.AmmunitionBolt), heavy, loading, twoHandedUse(content.TwoHandedAlways)),
		wpn("pistol", martial, ranged, d(1, dice.D10(), piercing), content.MasteryVex, ammoUse(30, 90, content.AmmunitionBullet), loading),
		wpn("blowgun", martial, ranged, must(content.NewWeaponFixedDamage(1, piercing)), content.MasteryVex, ammoUse(25, 100, content.AmmunitionNeedle), loading),
	}
	out := map[string]content.Weapon{}
	for _, w := range list {
		out[string(w.ID())] = w
	}
	return out
}

// resolvedOf returns the resolved-weapons map of the named fixtures.
func resolvedOf(ws map[string]content.Weapon, names ...string) map[content.ItemID]content.Weapon {
	out := map[content.ItemID]content.Weapon{}
	for _, n := range names {
		out[content.ItemID(n)] = ws[n]
	}
	return out
}

func ids(names ...string) []content.ItemID {
	out := make([]content.ItemID, len(names))
	for i, n := range names {
		out[i] = content.ItemID(n)
	}
	return out
}
