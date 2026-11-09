package character_test

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Synthetic equipment entries for the crafting tests; ids and prices are made
// up, the numbers follow the spec's case tables.

var noSource = content.SourceCitation{}

func weaponOf(id string, typ content.WeaponType, cost content.Cost) content.Weapon {
	d := must(content.NewWeaponDiceDamage(must(dice.ParseExpression("1d6")), content.DamageBludgeoning))
	return must(content.NewWeapon(content.ItemID(id), id, noSource, content.WeaponSimple, typ, d, nil, content.MasterySap, lbs(1), price(cost)))
}

func itemRef(id string) content.CraftReference {
	return must(content.NewCraftItem(content.ItemID(id)))
}

func setRef(name content.CraftSetName, except ...string) content.CraftReference {
	ids := make([]content.ItemID, 0, len(except))
	for _, e := range except {
		ids = append(ids, content.ItemID(e))
	}
	return must(content.NewCraftSet(name, ids))
}

func toolOf(id string, craft ...content.CraftReference) content.Tool {
	u := []content.ToolUtilizeAction{content.MustToolUtilizeAction("Do a thing", 10)}
	return must(content.NewStandaloneTool(content.ItemID(id), id, noSource, content.ToolGroupArtisans, ability.Dexterity, u, craft, price(gp(5)), lbs(1)))
}

// craftTools mirrors the Craft lists of the spec's "Which tools craft an item"
// table on made-up ids.
func craftTools() []content.Tool {
	return []content.Tool{
		toolOf("carpenters_tools", itemRef("club_x")),
		toolOf("woodcarvers_tools", itemRef("club_x"), itemRef("arcane_focus"), setRef(content.CraftRangedWeapons, "musket_x", "sling_x")),
		toolOf("smiths_tools", setRef(content.CraftMeleeWeapons, "whip_x", "club_x"), setRef(content.CraftMediumArmor, "hide_x"), setRef(content.CraftHeavyArmor)),
		toolOf("jewelers_tools", itemRef("arcane_focus")),
		toolOf("leatherworkers_tools", itemRef("whip_x"), itemRef("sling_x"), itemRef("hide_x")),
		toolOf("tinkers_tools", itemRef("musket_x")),
	}
}

func idsOf(ids ...string) []content.ItemID {
	out := make([]content.ItemID, 0, len(ids))
	for _, i := range ids {
		out = append(out, content.ItemID(i))
	}
	return out
}
