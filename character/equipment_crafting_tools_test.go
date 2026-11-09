package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case equipSpec FR-037, FR-038, FR-039 (SRD 5.2.1, Craft
// entries, pp. 93-94; p. 103) - which tools craft an item: by id, by a set that
// includes it unless excepted, through a focus variant's family; ordered by id.
// Path: per case (happy; unhappy rule interaction, empty and minimal).
func TestToolsFor(t *testing.T) {
	t.Parallel()
	focusVariant := must(content.NewFocusVariant("crystal_x", "Crystal", noSource, "arcane_focus", price(gp(10)), lbs(1), "", "", ""))
	ammo := must(content.NewAmmunitionVariant("bolts_x", "Bolts", noSource, "ammo_family", price(gp(1)), lbs(1), "", "", must(content.NewAmmunitionData(20, "Case"))))
	toolVariant := must(content.NewToolVariant("dragonchess_x", "Dragonchess", noSource, "gaming_set", price(gp(1)), lbs(1)))
	scroll := must(content.NewScrollVariant("scroll_x", "Scroll", noSource, "scroll_family", price(gp(30)), "", "", 0))
	entries := []struct {
		name string
		item content.EquipmentEntry
		want []string
	}{
		{"Club", weaponOf("club_x", content.WeaponMelee, gp(1)), []string{"carpenters_tools", "woodcarvers_tools"}},
		{"Longsword (melee set)", weaponOf("longsword_x", content.WeaponMelee, gp(15)), []string{"smiths_tools"}},
		{"Chain Shirt (medium set)", armorOf(t, "chain_shirt_x", catM, cappedRule(t, 13, 2), 0, false, 20, gp(50)), []string{"smiths_tools"}},
		{"Plate Armor (heavy set)", armorOf(t, "plate_x", catH, noneRule(t, 18), 15, true, 65, gp(1500)), []string{"smiths_tools"}},
		{"Longbow (ranged set)", weaponOf("longbow_x", content.WeaponRanged, gp(50)), []string{"woodcarvers_tools"}},
		{"Arcane Focus", must(content.NewGearFamily("arcane_focus", "Arcane Focus", noSource, content.CostVaries{}, content.WeightVaries{}, "", "Text.", []content.ItemID{"crystal_x"})), []string{"jewelers_tools", "woodcarvers_tools"}},
		{"Whip (excepted from melee)", weaponOf("whip_x", content.WeaponMelee, gp(2)), []string{"leatherworkers_tools"}},
		{"Hide Armor (excepted from medium)", armorOf(t, "hide_x", catM, cappedRule(t, 12, 2), 0, false, 12, gp(10)), []string{"leatherworkers_tools"}},
		{"Musket (named, excepted from ranged)", weaponOf("musket_x", content.WeaponRanged, gp(500)), []string{"tinkers_tools"}},
		{"Sling (named, excepted from ranged)", weaponOf("sling_x", content.WeaponRanged, gp(1)), []string{"leatherworkers_tools"}},
		{"Crystal (through variantOf)", focusVariant, []string{"jewelers_tools", "woodcarvers_tools"}},
		{"ammunition variant of an unnamed family", ammo, nil},
		{"tool variant", toolVariant, nil},
		{"scroll variant", scroll, nil},
		{"Shield (the GM assigns)", shieldOf(t), nil},
		{"a Tool against the melee set", toolOf("other_tools"), nil},
	}
	for _, c := range entries {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.Crafting{}.ToolsFor(c.item, craftTools())
			if !slices.Equal(got, idsOf(c.want...)) {
				t.Errorf("tools = %v, want %v", got, c.want)
			}
		})
	}
}

// Requirement: equipSpec FR-037 - a tool named by a family is matched through
// the family of an ammunition variant, and a scroll's family is not.
// Path: unhappy (rule interaction).
func TestToolsForVariantFamilies(t *testing.T) {
	t.Parallel()
	tools := []content.Tool{toolOf("fletchers_tools", itemRef("ammo_family")), toolOf("scribes_tools", itemRef("scroll_family"))}
	ammo := must(content.NewAmmunitionVariant("bolts_x", "Bolts", noSource, "ammo_family", price(gp(1)), lbs(1), "", "", must(content.NewAmmunitionData(20, "Case"))))
	scroll := must(content.NewScrollVariant("scroll_x", "Scroll", noSource, "scroll_family", price(gp(30)), "", "", 0))
	if got := (character.Crafting{}).ToolsFor(ammo, tools); !slices.Equal(got, idsOf("fletchers_tools")) {
		t.Errorf("ammunition = %v", got)
	}
	if got := (character.Crafting{}).ToolsFor(scroll, tools); len(got) != 0 {
		t.Errorf("scroll variant matched through its family: %v", got)
	}
	direct := []content.Tool{toolOf("scribes_tools", itemRef("scroll_x"))}
	if got := (character.Crafting{}).ToolsFor(scroll, direct); !slices.Equal(got, idsOf("scribes_tools")) {
		t.Errorf("scroll named directly = %v", got)
	}
}

// Requirement: equipSpec FR-039 - the result is ordered by id whatever the
// input order, holds each id once, is empty with no tools or a nil item, and is
// a fresh slice; the input is not altered.
// Path: unhappy (immutability; empty and minimal).
func TestToolsForOrdering(t *testing.T) {
	t.Parallel()
	club := weaponOf("club_x", content.WeaponMelee, gp(1))
	a, b := toolOf("woodcarvers_tools", itemRef("club_x")), toolOf("carpenters_tools", itemRef("club_x"))
	tools := []content.Tool{a, b, a}
	got := character.Crafting{}.ToolsFor(club, tools)
	if !slices.Equal(got, idsOf("carpenters_tools", "woodcarvers_tools")) {
		t.Errorf("tools = %v", got)
	}
	if tools[0].ID() != "woodcarvers_tools" || tools[1].ID() != "carpenters_tools" {
		t.Error("input altered")
	}
	if got := (character.Crafting{}).ToolsFor(club, nil); len(got) != 0 {
		t.Errorf("no tools = %v", got)
	}
	if got := (character.Crafting{}).ToolsFor(nil, tools); len(got) != 0 {
		t.Errorf("nil item = %v", got)
	}
	_ = ability.Strength // the tool fixtures use abilities; keep the import honest
}
