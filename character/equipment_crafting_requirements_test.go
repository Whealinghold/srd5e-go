package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func plainGear(id string, cost content.Cost) content.PlainGear {
	return must(content.NewPlainGear(content.ItemID(id), id, noSource, price(cost), lbs(1), "", "Text."))
}

// Requirement: per case equipSpec FR-040, FR-041, FR-042, FR-043, FR-044,
// FR-045, FR-046, FR-047, FR-048, FR-066 (SRD 5.2.1, "Crafting Nonmagical
// Items", p. 103) - the "Crafting requirements" rows.
// Path: per case (happy; unhappy rule interaction, missing content, empty and.
// minimal, boundary).
func TestRequirementsFor(t *testing.T) {
	t.Parallel()
	override := must(content.NewCraftingOverride(gp(25), 1, idsOf("herbalism_kit")))
	synth := must(content.NewCraftingOverride(gp(3), 2, nil))
	scribing := character.MustScrollScribing(scrollRows())
	scroll := func(level, cost int) content.EquipmentEntry {
		return must(content.NewScrollVariant("scroll_x", "Scroll", noSource, "scroll_family", price(gp(cost)), "", "", level))
	}
	vehicle := must(content.NewAirborneOrWaterborneVehicle("galley_x", "Galley", noSource, 4, 1, 0, 0, 15, 500, 0, content.NoWeight{}, price(gp(30000))))
	toolFamily := must(content.NewToolFamily("gaming_set", "Gaming Set", noSource, content.ToolGroupOther, 3, []content.ToolUtilizeAction{content.MustToolUtilizeAction("Play", 10)}, nil, idsOf("dragonchess_x"), content.CostVaries{}, content.WeightVaries{}))
	type req struct {
		raw   content.Cost
		days  int
		tools []string
		basis character.CraftingBasis
	}
	cases := []struct {
		name   string
		item   content.EquipmentEntry
		scribe character.ScrollScribing
		req    *req
		reason character.NotCraftableReason
	}{
		{"Plate Armor", armorOf(t, "plate_x", catH, noneRule(t, 18), 15, true, 65, gp(1500)), character.ScrollScribing{}, &req{cp(75000), 150, []string{"smiths_tools"}, character.CraftingGenericRule}, 0},
		{"Shield, no tool", shieldOf(t), character.ScrollScribing{}, &req{cp(500), 1, nil, character.CraftingGenericRule}, 0},
		{"Cart", must(content.NewTackHarnessOrDrawnVehicle("cart_x", "Cart", noSource, lbs(200), price(gp(15)))), character.ScrollScribing{}, &req{cp(750), 2, nil, character.CraftingGenericRule}, 0},
		{"Gaming Set, dragonchess", must(content.NewToolVariant("dragonchess_x", "Dragonchess", noSource, "gaming_set", price(gp(1)), lbs(1))), character.ScrollScribing{}, &req{cp(50), 1, nil, character.CraftingGenericRule}, 0},
		{"Arcane Focus, Crystal", must(content.NewFocusVariant("crystal_x", "Crystal", noSource, "arcane_focus", price(gp(10)), lbs(1), "", "", "")), character.ScrollScribing{}, &req{cp(500), 1, []string{"jewelers_tools", "woodcarvers_tools"}, character.CraftingGenericRule}, 0},
		{"Potion of Healing", must(content.NewMagicGear("potion_x", "Potion", noSource, price(gp(50)), content.NoWeight{}, "", "Text.", override)), character.ScrollScribing{}, &req{gp(25), 1, []string{"herbalism_kit"}, character.CraftingItemOverride}, 0},
		{"synthetic gear override", must(content.NewMagicGear("synth_x", "Synth", noSource, price(gp(50)), content.NoWeight{}, "", "Text.", synth)), character.ScrollScribing{}, &req{gp(3), 2, nil, character.CraftingItemOverride}, 0},
		{"gear with a rate and an override", must(content.NewMagicGear("rated_x", "Rated", noSource, price(content.MustCost(5, content.CoinSP, content.CostPerDay)), content.NoWeight{}, "", "Text.", synth)), character.ScrollScribing{}, &req{gp(3), 2, nil, character.CraftingItemOverride}, 0},
		{"magic gear without override", must(content.NewMagicGear("bare_x", "Bare", noSource, price(gp(50)), content.NoWeight{}, "", "Text.", content.CraftingOverride{})), character.ScrollScribing{}, &req{cp(2500), 5, nil, character.CraftingGenericRule}, 0},
		{"Spell Scroll (Cantrip)", scroll(0, 30), scribing, &req{gp(15), 1, nil, character.CraftingScrollRow}, 0},
		{"Spell Scroll (Level 1)", scroll(1, 50), scribing, &req{gp(25), 1, nil, character.CraftingScrollRow}, 0},
		{"Spell Scroll without rows", scroll(0, 30), character.ScrollScribing{}, nil, character.NotCraftableScribingTableMissing},
		{"Gaming Set family", toolFamily, character.ScrollScribing{}, nil, character.NotCraftablePriceVaries},
		{"Spell Scroll family", must(content.NewScrollFamily("scroll_family", "Spell Scroll", noSource, "", "Text.", idsOf("scroll_x"))), scribing, nil, character.NotCraftablePriceVaries},
		{"gear family", must(content.NewGearFamily("gear_family", "Gear", noSource, content.CostVaries{}, content.WeightVaries{}, "", "Text.", idsOf("v_x"))), character.ScrollScribing{}, nil, character.NotCraftablePriceVaries},
		{"Horse, Draft", mountOf(t, "horse_x", 540), character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Spellcasting Services 3", serviceRows()[3], character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Spell Scroll Cost row", scrollRows()[0], character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Food, Drink, Lodging", must(content.NewFoodDrinkOrLodging("meal_x", "Meal", noSource, price(gp(1)))), character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Hireling", must(content.NewHireling("hireling_x", "Hireling", noSource, price(gp(1)))), character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Wretched, cost 0", must(content.NewLifestyleExpense("wretched_x", "Wretched", noSource, price(gp(0)), "Text.")), character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Feed per day", must(content.NewTackHarnessOrDrawnVehicle("feed_x", "Feed", noSource, lbs(10), price(content.MustCost(5, content.CoinCP, content.CostPerDay)))), character.ScrollScribing{}, nil, character.NotCraftableRatedPrice},
		{"gear at 0 GP", plainGear("free_x", gp(0)), character.ScrollScribing{}, nil, character.NotCraftableFreeOfCharge},
		{"Galley", vehicle, character.ScrollScribing{}, nil, character.NotCraftableNoCraftingRule},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.Crafting{}.RequirementsFor(c.item, craftTools(), c.scribe)
			if err != nil {
				t.Fatal(err)
			}
			switch v := got.(type) {
			case character.CraftingRequirements:
				if c.req == nil || v.RawMaterials() != c.req.raw || v.Days() != c.req.days || !slices.Equal(v.Tools(), idsOf(c.req.tools...)) || v.Basis() != c.req.basis {
					t.Errorf("requirements = %v, %d days, %v, %v; want %+v", v.RawMaterials(), v.Days(), v.Tools(), v.Basis(), c.req)
				}
			case character.NotCraftable:
				if c.req != nil || v.Reason != c.reason {
					t.Errorf("NotCraftable(%v), want %+v / %v", v.Reason, c.req, c.reason)
				}
			}
		})
	}
}

// Requirement: equipSpec FR-063, FR-067 - the tools of the requirements are a
// copy: mutating a read changes nothing, nor does mutating the override's list.
// Path: unhappy (immutability).
func TestRequirementsToolsAreCopies(t *testing.T) {
	t.Parallel()
	got, err := character.Crafting{}.RequirementsFor(armorOf(t, "plate_x", catH, noneRule(t, 18), 15, true, 65, gp(1500)), craftTools(), character.ScrollScribing{})
	if err != nil {
		t.Fatal(err)
	}
	r := got.(character.CraftingRequirements)
	r.Tools()[0] = "changed"
	if !slices.Equal(r.Tools(), idsOf("smiths_tools")) {
		t.Errorf("tools = %v after mutating a read", r.Tools())
	}
}

// Requirement: equipSpec FR-040 - a nil item, an entry too large for its
// figures, and the String of the enums.
// Path: unhappy (invalid input).
func TestRequirementsForInvalid(t *testing.T) {
	t.Parallel()
	_, err := character.Crafting{}.RequirementsFor(nil, nil, character.ScrollScribing{})
	wantInvalid(t, err, "nil item")
	huge := plainGear("huge_x", content.MustCost(content.MaxCostAmount, content.CoinPP, 0))
	_, err = character.Crafting{}.RequirementsFor(huge, nil, character.ScrollScribing{})
	wantInvalid(t, err, "huge")
	for b, want := range map[character.CraftingBasis]string{
		character.CraftingGenericRule: "genericRule", character.CraftingItemOverride: "itemOverride",
		character.CraftingScrollRow: "scrollRow", 0: "unknown",
	} {
		if b.String() != want {
			t.Errorf("basis %d = %q, want %q", int(b), b.String(), want)
		}
	}
	for r, want := range map[character.NotCraftableReason]string{
		character.NotCraftablePriceVaries: "priceVaries", character.NotCraftableRatedPrice: "ratedPrice",
		character.NotCraftableFreeOfCharge: "freeOfCharge", character.NotCraftableNotAnItem: "notAnItem",
		character.NotCraftableNoCraftingRule: "noCraftingRule", character.NotCraftableScribingTableMissing: "scribingTableMissing", 0: "unknown",
	} {
		if r.String() != want {
			t.Errorf("reason %d = %q, want %q", int(r), r.String(), want)
		}
	}
}

// Requirement: equipSpec FR-066 - every goods entry type with a rate-free price
// above 0 follows the generic rule from its own price.
// Path: unhappy (rule interaction).
func TestRequirementsForGoodsTypes(t *testing.T) {
	t.Parallel()
	items := []content.EquipmentEntry{
		weaponOf("sword_x", content.WeaponMelee, gp(10)),
		plainGear("rope_x", gp(10)),
		must(content.NewPack("pack_x", "Pack", noSource, price(gp(10)), lbs(5), "", "Text.", []content.PackItem{must(content.NewPackItem("rope_x", 1, ""))})),
		must(content.NewAmmunitionVariant("bolts_x", "Bolts", noSource, "ammo_family", price(gp(10)), lbs(1), "", "", must(content.NewAmmunitionData(20, "Case")))),
	}
	for _, item := range items {
		got, err := character.Crafting{}.RequirementsFor(item, nil, character.ScrollScribing{})
		r, ok := got.(character.CraftingRequirements)
		if err != nil || !ok || r.RawMaterials() != gp(5) && r.RawMaterials() != cp(500) || r.Days() != 1 || r.Basis() != character.CraftingGenericRule {
			t.Errorf("%s = %v, %v", item.ID(), got, err)
		}
	}
	// A ToolFamily with a fixed price follows the generic rule too.
	fam := must(content.NewToolFamily("fam_x", "Fam", noSource, content.ToolGroupOther, 3, []content.ToolUtilizeAction{content.MustToolUtilizeAction("Play", 10)}, nil, idsOf("v_x"), price(gp(10)), lbs(1)))
	if got, err := (character.Crafting{}).RequirementsFor(fam, nil, character.ScrollScribing{}); err != nil || got.(character.CraftingRequirements).Days() != 1 {
		t.Errorf("tool family = %v, %v", got, err)
	}
	if got := (character.Crafting{}).ToolsFor(fam, []content.Tool{toolOf("a_tools", itemRef("fam_x")), fam}); !slices.Equal(got, idsOf("a_tools")) {
		t.Errorf("tools for a family item = %v", got)
	}
}

// Requirement: equipSpec FR-037 - a Tool in the tool list may be a family with its
// own Craft list and a variant, which has none.
// Path: unhappy (rule interaction).
func TestToolsForFamilyAndVariantTools(t *testing.T) {
	t.Parallel()
	famTool := must(content.NewToolFamily("fam_tools", "Fam", noSource, content.ToolGroupOther, 3, []content.ToolUtilizeAction{content.MustToolUtilizeAction("Play", 10)}, []content.CraftReference{itemRef("club_x")}, idsOf("v_x"), content.CostVaries{}, content.WeightVaries{}))
	variantTool := must(content.NewToolVariant("v_x", "V", noSource, "fam_tools", price(gp(1)), lbs(1)))
	got := character.Crafting{}.ToolsFor(weaponOf("club_x", content.WeaponMelee, gp(1)), []content.Tool{famTool, variantTool})
	if !slices.Equal(got, idsOf("fam_tools")) {
		t.Errorf("tools = %v", got)
	}
}

// Requirement: equipSpec FR-066 - a standalone tool is goods priced by itself.
// Path: unhappy (rule interaction).
func TestRequirementsForStandaloneTool(t *testing.T) {
	t.Parallel()
	got, err := character.Crafting{}.RequirementsFor(toolOf("any_tools"), nil, character.ScrollScribing{})
	if r, ok := got.(character.CraftingRequirements); err != nil || !ok || r.RawMaterials() != cp(250) || r.Days() != 1 {
		t.Errorf("standalone tool = %v, %v", got, err)
	}
}
