package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const gold = 100

// Requirement: per case (doc/specs/magic-item-rules.md FR-074 to FR-076, FR-081, FR-083, FR-084 and table "Crafting").
// Path: per case.
func TestMagicItemCraftingRequirementsFor(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Magic Item Crafting Time and Cost", p. 207`
	plateNested := character.NotCraftable{Reason: character.NotCraftableNotAnItem}
	cases := []struct {
		name, requirement, path string
		in                      character.MagicItemCraftingInput
		hours, workers          int
		cost                    int64
		perWorker               [2]int
	}{
		{"Rare, alone", p, "happy", character.MagicItemCraftingInput{Rarity: content.RarityRare, Workers: 1}, 400, 1, 2000 * gold, [2]int{400, 1}},
		{"Common consumable", p, "unhappy (rule interaction)", character.MagicItemCraftingInput{Rarity: content.RarityCommon, Consumable: true, Workers: 1}, 20, 1, 25 * gold, [2]int{20, 1}},
		{"Very Rare consumable", p, "unhappy (rule interaction)", character.MagicItemCraftingInput{Rarity: content.RarityVeryRare, Consumable: true, Workers: 1}, 500, 1, 10000 * gold, [2]int{500, 1}},
		{"Legendary", p, "unhappy (boundary)", character.MagicItemCraftingInput{Rarity: content.RarityLegendary, Workers: 1}, 2000, 1, 100000 * gold, [2]int{2000, 1}},
		{"one assistant", p, "happy", character.MagicItemCraftingInput{Rarity: content.RarityRare, Workers: 2}, 400, 2, 2000 * gold, [2]int{400, 2}},
		{"three workers, not rounded", p, "unhappy (rule interaction)", character.MagicItemCraftingInput{Rarity: content.RarityCommon, Workers: 3}, 40, 3, 50 * gold, [2]int{40, 3}},
		{"consumable incorporating an item, pay", p, "unhappy (rule interaction)", character.MagicItemCraftingInput{Rarity: content.RarityRare, Consumable: true, Choice: character.PayPurchaseCost, IncorporatedCostCopper: 1500 * gold, Workers: 1}, 200, 1, 2500 * gold, [2]int{200, 1}},
		{"+1 Armor (Plate Armor), pay", p, "happy", character.MagicItemCraftingInput{Rarity: content.RarityRare, Choice: character.PayPurchaseCost, IncorporatedCostCopper: 1500 * gold, Workers: 1}, 400, 1, 3500 * gold, [2]int{400, 1}},
		{"+1 Armor (Plate Armor), craft", p, "happy", character.MagicItemCraftingInput{Rarity: content.RarityRare, Choice: character.CraftIncorporatedItem, IncorporatedCostCopper: 1500 * gold, IncorporatedOutcome: plateNested, Workers: 1}, 400, 1, 2000 * gold, [2]int{400, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.MagicItemCrafting{}.RequirementsFor(c.in)
			r, ok := got.(character.MagicItemCraftingRequirements)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v [%s]", got, err, c.requirement)
			}
			if r.TotalWorkHours() != c.hours || r.CostCopperPieces() != c.cost || r.WorkHoursPerWorker().Numerator() != c.perWorker[0] || r.WorkHoursPerWorker().Denominator() != c.perWorker[1] ||
				r.WorkHoursPerDay() != 8 || !r.DaysNeedNotBeConsecutive() || r.IncorporatedChoice() != c.in.Choice {
				t.Errorf("= %+v [%s]", r, c.requirement)
			}
			if (c.in.Choice == character.CraftIncorporatedItem) != (r.IncorporatedOutcome() != nil) {
				t.Errorf("nested outcome = %v", r.IncorporatedOutcome())
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-075, FR-076 - an Artifact is not craftable, even when consumable or crafted by the
// equipment rules (FR-076 over FR-075); a Spell Scroll or Potion of Healing hands over to the equipment rules.
// Path: unhappy (rule interaction).
func TestMagicItemCraftingArtifactAndEquipmentRules(t *testing.T) {
	t.Parallel()
	artifact := character.MagicItemNotCraftable{Reason: character.MagicItemNotCraftableArtifact}
	equipRules := content.MagicItemCraftingBasis(must(content.NewEquipmentRulesCrafting("spell_scroll_x")))
	nested := character.CraftingOutcome(character.NotCraftable{Reason: character.NotCraftableScribingTableMissing})
	for name, in := range map[string]character.MagicItemCraftingInput{
		"artifact":                      {Rarity: content.RarityArtifact, Workers: 1},
		"consumable artifact":           {Rarity: content.RarityArtifact, Consumable: true, Workers: 1},
		"artifact with equipment rules": {Rarity: content.RarityArtifact, Basis: equipRules, GearOutcome: nested, Workers: 1},
	} {
		if got, err := (character.MagicItemCrafting{}).RequirementsFor(in); err != nil || got != artifact {
			t.Errorf("%s: %#v, %v", name, got, err)
		}
	}
	got, err := character.MagicItemCrafting{}.RequirementsFor(character.MagicItemCraftingInput{Rarity: content.RarityUncommon, Consumable: true, Basis: equipRules, GearOutcome: nested, Workers: 1})
	if err != nil || got != (character.MagicItemUsesEquipmentRules{Item: "spell_scroll_x", Outcome: nested}) {
		t.Errorf("equipment rules: %#v, %v", got, err)
	}
	for name, in := range map[string]character.MagicItemCraftingInput{
		"no workers":            {Rarity: content.RarityRare},
		"negative cost":         {Rarity: content.RarityRare, Workers: 1, IncorporatedCostCopper: -1},
		"no gear outcome":       {Rarity: content.RarityRare, Basis: equipRules, Workers: 1},
		"craft without outcome": {Rarity: content.RarityRare, Choice: character.CraftIncorporatedItem, Workers: 1},
		"unknown rarity":        {Rarity: 0, Workers: 1},
	} {
		if got, err := (character.MagicItemCrafting{}).RequirementsFor(in); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %#v, %v", name, got, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-077 to FR-080, FR-082, FR-087 - shortfalls in the order of members then spells, assistants last).
// Path: per case.
func TestMagicItemCraftingShortfalls(t *testing.T) {
	t.Parallel()
	team := must(character.NewCraftingTeam("crafter", []character.CraftingMemberID{"helper"}))
	q := func(m string, arcana bool, tools []string, spells ...content.SpellID) character.CrafterQualification {
		return must(character.NewCrafterQualification(character.CraftingMemberID(m), arcana, idsOf(tools...), spells))
	}
	options := idsOf("alchemists_supplies", "herbalism_kit")
	spells := []content.SpellID{"fireball", "shield"}
	full := []character.CrafterQualification{q("crafter", true, []string{"herbalism_kit"}, "fireball", "shield"), q("helper", true, []string{"alchemists_supplies"})}
	kinds := func(s []character.MagicItemCraftingShortfall) []character.ShortfallKind {
		var out []character.ShortfallKind
		for _, x := range s {
			out = append(out, x.Kind)
		}
		return out
	}
	cases := []struct {
		name      string
		team      character.CraftingTeam
		quals     []character.CrafterQualification
		spells    []content.SpellID
		allowance character.AssistantAllowance
		want      []character.ShortfallKind
	}{
		{"nothing lacking, any one tool of the set", team, full, spells, 0, nil},
		{"no Arcana for the assistant", team, []character.CrafterQualification{full[0], q("helper", false, []string{"alchemists_supplies"})}, spells, 0, []character.ShortfallKind{character.ShortfallArcanaProficiency}},
		{"no tool for the crafter", team, []character.CrafterQualification{q("crafter", true, nil, "fireball", "shield"), full[1]}, spells, 0, []character.ShortfallKind{character.ShortfallToolProficiency}},
		{"a tool outside the set", team, []character.CrafterQualification{q("crafter", true, []string{"smiths_tools"}, "fireball", "shield"), full[1]}, spells, 0, []character.ShortfallKind{character.ShortfallToolProficiency}},
		{"one spell missing", team, []character.CrafterQualification{q("crafter", true, []string{"herbalism_kit"}, "fireball"), full[1]}, spells, 0, []character.ShortfallKind{character.ShortfallSpellNotPrepared}},
		{"assistants are not asked for spells", team, full, spells, 0, nil},
		{"two assistants, no allowance", must(character.NewCraftingTeam("crafter", []character.CraftingMemberID{"helper", "other"})), append(slicesClone(full), q("other", true, []string{"herbalism_kit"})), spells, 0, []character.ShortfallKind{character.ShortfallTooManyAssistants}},
		{"two assistants, GM allows 2", must(character.NewCraftingTeam("crafter", []character.CraftingMemberID{"helper", "other"})), append(slicesClone(full), q("other", true, []string{"herbalism_kit"})), spells, must(character.NewAssistantAllowance(2)), nil},
		{"everything lacking, in order", must(character.NewCraftingTeam("crafter", []character.CraftingMemberID{"helper", "other"})), nil, spells,
			0, []character.ShortfallKind{character.ShortfallArcanaProficiency, character.ShortfallToolProficiency, character.ShortfallArcanaProficiency, character.ShortfallToolProficiency,
				character.ShortfallArcanaProficiency, character.ShortfallToolProficiency, character.ShortfallSpellNotPrepared, character.ShortfallSpellNotPrepared, character.ShortfallTooManyAssistants}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := kinds(character.MagicItemCrafting{}.Shortfalls(c.team, c.quals, options, c.spells, c.allowance))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("= %v, want %v (SRD 5.2.1, \"Crafting Magic Items\", pp. 206-207)", got, c.want)
			}
		})
	}
	short := character.MagicItemCrafting{}.Shortfalls(team, nil, options, spells, 0)
	if short[0].Member != "crafter" || short[1].Member != "crafter" || !reflect.DeepEqual(short[1].Tools, options) || short[4].Spell != "fireball" || short[5].Spell != "shield" {
		t.Errorf("fields: %+v", short)
	}
	for k := character.ShortfallArcanaProficiency; k <= character.ShortfallTooManyAssistants; k++ {
		if k.String() == "unknown" {
			t.Errorf("kind %d", k)
		}
	}
	if character.ShortfallKind(0).String() != "unknown" || character.ShortfallToolProficiency.String() != "tool" {
		t.Error("String")
	}
}

func slicesClone(q []character.CrafterQualification) []character.CrafterQualification {
	return append([]character.CrafterQualification(nil), q...)
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-079 and table "Crafting tools").
// Path: per case.
func TestMagicItemCraftingToolsFor(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Magic Item Tools", p. 207`
	tools := craftTools()
	plate := armorOf(t, "plate_x", catH, noneRule(t, 18), 15, true, 65, gp(1500))
	club := weaponOf("club_x", content.WeaponMelee, gp(1))
	cases := []struct {
		name, requirement, path string
		category                content.MagicItemCategory
		base                    content.EquipmentEntry
		want                    []content.ItemID
	}{
		{"ring", p, "happy", content.MagicItemRing, nil, idsOf("jewelers_tools")},
		{"rod", p, "happy", content.MagicItemRod, nil, idsOf("woodcarvers_tools")},
		{"staff", p, "happy", content.MagicItemStaff, nil, idsOf("woodcarvers_tools")},
		{"wand", p, "happy", content.MagicItemWand, nil, idsOf("woodcarvers_tools")},
		{"scroll", p, "happy", content.MagicItemScroll, nil, idsOf("calligraphers_supplies")},
		{"potion", p, "unhappy (rule interaction)", content.MagicItemPotion, nil, idsOf("alchemists_supplies", "herbalism_kit")},
		{"+1 Armor (Plate Armor)", p, "unhappy (rule interaction)", content.MagicItemArmor, plate, idsOf("smiths_tools")},
		{"+1 Club", p, "unhappy (rule interaction)", content.MagicItemWeapon, club, idsOf("carpenters_tools", "woodcarvers_tools")},
		{"wondrous, no base", p, "unhappy (empty case)", content.MagicItemWondrousItem, nil, idsOf("tinkers_tools")},
		{"wondrous, based on plate", p, "unhappy (empty case)", content.MagicItemWondrousItem, plate, idsOf("tinkers_tools", "smiths_tools")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.MagicItemCrafting{}.ToolsFor(c.category, c.base, tools)
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Errorf("= %v, %v, want %v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	for name, run := range map[string]func() error{
		"armor without a base": func() error {
			_, e := character.MagicItemCrafting{}.ToolsFor(content.MagicItemArmor, nil, tools)
			return e
		},
		"unknown category": func() error { _, e := character.MagicItemCrafting{}.ToolsFor(0, nil, tools); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-085, FR-086 and table "Raw materials").
// Path: per case.
func TestCraftingMaterialsIsAvailable(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Time and Cost" (Raw Materials), p. 207`
	city, town, village := content.SettlementCity, content.SettlementTown, content.SettlementVillage
	for _, c := range []struct {
		s    content.SettlementSize
		roll int
		want bool
	}{
		{city, 75, true}, {city, 76, false}, {town, 25, true}, {town, 26, false}, {village, 25, true}, {village, 26, false},
		{village, 1, true}, {city, 100, false}, {city, 1, true},
	} {
		if got, err := (character.CraftingMaterials{}).IsAvailable(c.s, c.roll); err != nil || got != c.want {
			t.Errorf("IsAvailable(%v, %d) = %v, %v [%s]", c.s, c.roll, got, err, p)
		}
	}
	for _, c := range []struct {
		s    content.SettlementSize
		roll int
	}{{city, 0}, {city, 101}, {0, 50}} {
		if _, err := (character.CraftingMaterials{}).IsAvailable(c.s, c.roll); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("IsAvailable(%v, %d): %v", c.s, c.roll, err)
		}
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-082, FR-083 - value objects of the crafting team and work hours validate their input.
// Path: unhappy (invalid input).
func TestCraftingValueObjects(t *testing.T) {
	t.Parallel()
	if h, err := character.NewWorkHours(40, 3); err != nil || h.Numerator() != 40 || h.Denominator() != 3 {
		t.Errorf("NewWorkHours = %+v, %v", h, err)
	}
	for _, in := range [][2]int{{-1, 1}, {1, 0}} {
		if _, err := character.NewWorkHours(in[0], in[1]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewWorkHours(%v): %v", in, err)
		}
	}
	for _, n := range []int{-1, 0, 1} {
		if _, err := character.NewAssistantAllowance(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewAssistantAllowance(%d): %v", n, err)
		}
	}
	if a, err := character.NewAssistantAllowance(2); err != nil || a.Max() != 2 || character.AssistantAllowance(0).Max() != 1 {
		t.Errorf("allowance %v %v", a, err)
	}
	team := must(character.NewCraftingTeam("c", []character.CraftingMemberID{"a"}))
	if team.Crafter() != "c" || team.Workers() != 2 || team.IsZero() || !(character.CraftingTeam{}).IsZero() {
		t.Errorf("team = %+v", team)
	}
	as := team.Assistants()
	as[0] = "x"
	if team.Assistants()[0] != "a" {
		t.Error("Assistants is not a copy")
	}
	for name, run := range map[string]func() error{
		"blank crafter":   func() error { _, e := character.NewCraftingTeam(" ", nil); return e },
		"blank assistant": func() error { _, e := character.NewCraftingTeam("c", []character.CraftingMemberID{""}); return e },
		"repeated":        func() error { _, e := character.NewCraftingTeam("c", []character.CraftingMemberID{"c"}); return e },
		"blank member":    func() error { _, e := character.NewCrafterQualification("", true, nil, nil); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if q := must(character.NewCrafterQualification("c", true, nil, nil)); q.Member() != "c" {
		t.Error("Member")
	}
}
