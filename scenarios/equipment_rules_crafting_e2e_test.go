package scenarios_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Services, repairs and crafting of equipment-rules.md, through the public
// packages only. The entries are synthetic (the bundled data has no equipment
// yet); the figures are the ones printed on SRD 5.2.1, pp. 100-103.

func grpD3IDs(ids ...string) []content.ItemID {
	out := make([]content.ItemID, 0, len(ids))
	for _, id := range ids {
		out = append(out, content.ItemID(id))
	}
	return out
}

func grpD3Level(n int) content.SpellLevel { return grpD3Must(content.NewSpellLevel(n)) }

func grpD3Service(lo, hi int, a content.Availability, cost content.Cost) content.SpellcastingService {
	return grpD3Must(content.NewSpellcastingService("service_x", "Service", content.SourceCitation{}, lo, hi, a, grpD3Price(cost)))
}

// grpD3ServiceRows is the printed Spellcasting Services table (p. 102).
func grpD3ServiceRows() []content.SpellcastingService {
	return []content.SpellcastingService{
		grpD3Service(0, 0, content.AvailabilityVillageTownOrCity, grpD3GP(30)),
		grpD3Service(1, 1, content.AvailabilityVillageTownOrCity, grpD3GP(50)),
		grpD3Service(2, 2, content.AvailabilityVillageTownOrCity, grpD3GP(200)),
		grpD3Service(3, 3, content.AvailabilityTownOrCityOnly, grpD3GP(300)),
		grpD3Service(4, 5, content.AvailabilityTownOrCityOnly, grpD3GP(2000)),
		grpD3Service(6, 8, content.AvailabilityCityOnly, grpD3GP(20000)),
		grpD3Service(9, 9, content.AvailabilityCityOnly, grpD3GP(100000)),
	}
}

func grpD3ScrollRow(level, days int, cost content.Cost) content.SpellScrollCost {
	return grpD3Must(content.NewSpellScrollCost("scroll_cost_x", "Scroll Cost", content.SourceCitation{}, level, days, grpD3Price(cost)))
}

// grpD3ScrollRows is the printed Spell Scroll Costs table (p. 103).
func grpD3ScrollRows() []content.SpellScrollCost {
	days := []int{1, 1, 3, 5, 10, 25, 40, 50, 60, 120}
	gold := []int{15, 25, 100, 150, 1000, 1500, 10000, 12500, 15000, 50000}
	rows := make([]content.SpellScrollCost, 0, len(days))
	for l := range days {
		rows = append(rows, grpD3ScrollRow(l, days[l], grpD3GP(gold[l])))
	}
	return rows
}

func grpD3CraftTool(id string, craft ...content.CraftReference) content.Tool {
	u := []content.ToolUtilizeAction{content.MustToolUtilizeAction("Do a thing", 10)}
	return grpD3Must(content.NewStandaloneTool(content.ItemID(id), id, content.SourceCitation{}, content.ToolGroupArtisans, ability.Dexterity, u, craft, grpD3Price(grpD3GP(5)), grpD3Lbs(1)))
}

func grpD3Item(id string) content.CraftReference {
	return grpD3Must(content.NewCraftItem(content.ItemID(id)))
}

func grpD3Set(name content.CraftSetName, except ...string) content.CraftReference {
	return grpD3Must(content.NewCraftSet(name, grpD3IDs(except...)))
}

// grpD3Toolbox mirrors the Craft lists of the SRD (pp. 93-94) on made-up ids.
func grpD3Toolbox() []content.Tool {
	return []content.Tool{
		grpD3CraftTool("carpenters_tools", grpD3Item("club_x")),
		grpD3CraftTool("woodcarvers_tools", grpD3Item("club_x"), grpD3Item("arcane_focus"), grpD3Set(content.CraftRangedWeapons, "musket_x", "sling_x")),
		grpD3CraftTool("smiths_tools", grpD3Set(content.CraftMeleeWeapons, "whip_x", "club_x"), grpD3Set(content.CraftMediumArmor, "hide_x"), grpD3Set(content.CraftHeavyArmor)),
		grpD3CraftTool("jewelers_tools", grpD3Item("arcane_focus")),
		grpD3CraftTool("leatherworkers_tools", grpD3Item("whip_x"), grpD3Item("sling_x"), grpD3Item("hide_x")),
		grpD3CraftTool("tinkers_tools", grpD3Item("musket_x")),
	}
}

func grpD3Weapon(id string, typ content.WeaponType, cost content.Cost) content.Weapon {
	d := grpD3Must(content.NewWeaponDiceDamage(grpD3Must(dice.ParseExpression("1d6")), content.DamageBludgeoning))
	return grpD3Must(content.NewWeapon(content.ItemID(id), id, content.SourceCitation{}, content.WeaponSimple, typ, d, nil, content.MasterySap, grpD3Lbs(1), grpD3Price(cost)))
}

func grpD3PlainGear(id string, cost content.Cost) content.PlainGear {
	return grpD3Must(content.NewPlainGear(content.ItemID(id), id, content.SourceCitation{}, grpD3Price(cost), grpD3Lbs(1), "", "Text."))
}

// A captain's galley takes damage and the crew prices the repair: 20 GP and a
// day per Hit Point, halved when supplies and skilled labor are abundant, and
// never more than the ship's own Hit Points (SRD 5.2.1, "Ship Repair", p. 101).
// Requirement: doc/specs/equipment-rules.md FR-029 to FR-032 - 20 GP and 1 day per Hit Point, abundant halves both exactly (half days), below 1 is invalid, QuoteShipRepairFor refuses more than the ship's Hit Points.
// Path: happy; unhappy (boundary, invalid input).
func TestGrpD3ShipRepairQuotes(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		hp       int
		supplies character.SuppliesAndLabor
		gold     int
		halfDays int
	}{
		{1, character.NotAbundant, 20, 2}, {10, character.NotAbundant, 200, 20},
		{1, character.Abundant, 10, 1}, {7, character.Abundant, 70, 7},
	} {
		got, err := character.QuoteShipRepair(c.hp, c.supplies)
		if err != nil || got.Cost() != grpD3GP(c.gold) || got.HalfDays() != c.halfDays {
			t.Errorf("repair %d HP, %v = %v, %d half days, %v; want %d GP, %d", c.hp, c.supplies, got.Cost(), got.HalfDays(), err, c.gold, c.halfDays)
		}
	}
	_, err := character.QuoteShipRepair(0, character.NotAbundant)
	grpD3WantInvalid(t, err, "0 Hit Points")
	_, err = character.QuoteShipRepair(-3, character.Abundant)
	grpD3WantInvalid(t, err, "-3 Hit Points")
	_, err = character.QuoteShipRepair(5, 0)
	grpD3WantInvalid(t, err, "undefined supplies")

	galley := grpD3Must(content.NewAirborneOrWaterborneVehicle("galley_x", "Galley", content.SourceCitation{}, 4, 80, 0, 150, 15, 500, 0, content.NoWeight{}, grpD3Price(grpD3GP(30000))))
	full, err := character.QuoteShipRepairFor(galley, 500, character.NotAbundant)
	if err != nil {
		t.Fatal(err)
	}
	if q, ok := full.(character.ShipRepairQuoted); !ok || q.Repair.Cost() != grpD3GP(10000) {
		t.Errorf("a full repair of the galley = %#v, want 10,000 GP", full)
	}
	over, err := character.QuoteShipRepairFor(galley, 501, character.NotAbundant)
	if err != nil || over != (character.ShipRepairExceedsShip{RequestedHitPoints: 501, ShipHitPoints: 500}) {
		t.Errorf("one point over the ship = %#v, %v", over, err)
	}
	_, err = character.QuoteShipRepairFor(galley, 0, character.NotAbundant)
	grpD3WantInvalid(t, err, "repair of 0 for a ship")
}

// A party travels from a village to a city looking for a spellcaster: which
// spell levels each settlement offers, what the fee is with expensive
// components, and what a bad table looks like (SRD 5.2.1, "Spellcasting
// Services", p. 102).
// Requirement: doc/specs/equipment-rules.md FR-052 to FR-057, FR-067 - one row per level range, availability by settlement size, components added to the fee, a rated component cost is invalid, gaps and overlaps are a TableCoverageError, the table copies its rows.
// Path: happy; unhappy (rule interaction, invalid input, missing content, immutability).
func TestGrpD3SpellcastingServicesAcrossSettlements(t *testing.T) {
	t.Parallel()
	table := character.MustSpellcastingServices(grpD3ServiceRows())

	for _, c := range []struct {
		level   int
		fee     int
		village bool
		town    bool
		city    bool
	}{
		{0, 30, true, true, true}, {2, 200, true, true, true},
		{3, 300, false, true, true}, {5, 2000, false, true, true},
		{6, 20000, false, false, true}, {9, 100000, false, false, true},
	} {
		row := table.ForSpellLevel(grpD3Level(c.level))
		if row.Price().Cost() != grpD3GP(c.fee) {
			t.Errorf("level %d fee = %v, want %d GP", c.level, row.Price().Cost(), c.fee)
		}
		for size, want := range map[content.SettlementSize]bool{content.SettlementVillage: c.village, content.SettlementTown: c.town, content.SettlementCity: c.city} {
			if got := table.IsAvailable(row, size); got != want {
				t.Errorf("level %d available in settlement %v = %v, want %v", c.level, size, got, want)
			}
		}
	}

	row3 := table.ForSpellLevel(grpD3Level(3))
	total, err := table.TotalCost(row3, grpD3GP(100))
	if err != nil || total != grpD3CP(40000) {
		t.Errorf("level 3 with 100 GP of components = %v, %v; want 400 GP (40000 CP)", total, err)
	}
	if bare, err := table.TotalCost(row3, grpD3GP(0)); err != nil || bare != grpD3CP(30000) {
		t.Errorf("level 3 with no components = %v, %v; want 300 GP", bare, err)
	}
	_, err = table.TotalCost(row3, content.MustCost(5, content.CoinGP, content.CostPerDay))
	grpD3WantInvalid(t, err, "a component cost with a rate")

	// A level above the table has no row; the zero table has none either.
	if got := table.ForSpellLevel(content.SpellLevel(10)); got.Price() != (content.FixedCost{}) {
		t.Errorf("level 10 row = %v, want the zero row", got)
	}

	// A table with a hole at levels 4 and 5, and one that overlaps at level 8, are named precisely.
	rows := grpD3ServiceRows()
	_, err = character.NewSpellcastingServices(slices.Delete(slices.Clone(rows), 4, 5))
	var gap *character.TableCoverageError
	if !errors.As(err, &gap) || !slices.Equal(gap.Uncovered, []content.SpellLevel{grpD3Level(4), grpD3Level(5)}) || len(gap.Overlapping) != 0 {
		t.Errorf("gap error = %v, %+v", err, gap)
	}
	grpD3WantInvalid(t, err, "a table with a gap")
	overlapping := append(slices.Clone(rows), grpD3Service(8, 8, content.AvailabilityCityOnly, grpD3GP(1)))
	_, err = character.NewSpellcastingServices(overlapping)
	var over *character.TableCoverageError
	if !errors.As(err, &over) || !slices.Equal(over.Overlapping, []content.SpellLevel{grpD3Level(8)}) {
		t.Errorf("overlap error = %v, %+v", err, over)
	}
	if _, err = character.NewSpellcastingServices(nil); !errors.As(err, &gap) || len(gap.Uncovered) != 10 {
		t.Errorf("no rows = %v, want ten uncovered levels", err)
	}
	// The MustX form panics with the same error.
	func() {
		defer func() {
			if r, _ := recover().(error); !errors.As(r, &gap) {
				t.Errorf("MustSpellcastingServices panic = %v, want a *TableCoverageError", r)
			}
		}()
		character.MustSpellcastingServices(nil)
	}()

	// The table copied its rows: changing the source afterwards changes nothing.
	source := grpD3ServiceRows()
	copied := character.MustSpellcastingServices(source)
	source[1] = grpD3Service(1, 1, content.AvailabilityCityOnly, grpD3GP(999))
	if got := copied.ForSpellLevel(grpD3Level(1)).Price().Cost(); got != grpD3GP(50) {
		t.Errorf("level 1 fee after mutating the source = %v, want 50 GP", got)
	}
}

// Requirement: doc/specs/equipment-rules.md FR-049 to FR-051, FR-067 - the scroll table has one row per level 0 to 9; a gap or repeat is a TableCoverageError; the table copies its rows.
// Path: happy; unhappy (invalid input, missing content, immutability).
func TestGrpD3ScrollScribingTableCoverage(t *testing.T) {
	t.Parallel()
	table := character.MustScrollScribing(grpD3ScrollRows())
	if table.IsZero() {
		t.Fatal("a complete table is not the zero table")
	}
	for l, want := range []struct{ days, gold int }{{1, 15}, {1, 25}, {3, 100}, {5, 150}, {10, 1000}, {25, 1500}, {40, 10000}, {50, 12500}, {60, 15000}, {120, 50000}} {
		row := table.ForSpellLevel(grpD3Level(l))
		if row.Days() != want.days || row.Price().Cost() != grpD3GP(want.gold) {
			t.Errorf("level %d = %d days, %v; want %d, %d GP", l, row.Days(), row.Price().Cost(), want.days, want.gold)
		}
	}

	rows := grpD3ScrollRows()
	_, err := character.NewScrollScribing(slices.Delete(slices.Clone(rows), 4, 5))
	var gap *character.TableCoverageError
	if !errors.As(err, &gap) || !slices.Equal(gap.Missing, []content.SpellLevel{grpD3Level(4)}) {
		t.Errorf("missing level 4 = %v, %+v", err, gap)
	}
	grpD3WantInvalid(t, err, "a scroll table with a gap")
	_, err = character.NewScrollScribing(append(slices.Clone(rows), grpD3ScrollRow(2, 9, grpD3GP(1))))
	var rep *character.TableCoverageError
	if !errors.As(err, &rep) || !slices.Equal(rep.Repeated, []content.SpellLevel{grpD3Level(2)}) {
		t.Errorf("repeated level 2 = %v, %+v", err, rep)
	}
	if _, err = character.NewScrollScribing(nil); !errors.As(err, &gap) || len(gap.Missing) != 10 {
		t.Errorf("no rows = %v, want ten missing levels", err)
	}
	if _, err = content.NewSpellLevel(-1); err == nil {
		t.Error("spell level -1 must be rejected upstream")
	}
	if _, err = content.NewSpellLevel(10); err == nil {
		t.Error("spell level 10 must be rejected upstream")
	}

	source := grpD3ScrollRows()
	copied := character.MustScrollScribing(source)
	source[0] = grpD3ScrollRow(0, 99, grpD3GP(1))
	if got := copied.ForSpellLevel(grpD3Level(0)).Days(); got != 1 {
		t.Errorf("cantrip days after mutating the source = %d, want 1", got)
	}
	if got := (character.ScrollScribing{}).ForSpellLevel(grpD3Level(0)); got.Days() != 0 {
		t.Errorf("the zero table gave a row: %v", got)
	}
}

// A blacksmith prices the raw materials and the time to forge: half the
// purchase cost (rounded down in Copper Pieces) and a day per 10 GP, rounded
// up (SRD 5.2.1, "Crafting Nonmagical Items", p. 103).
// Requirement: doc/specs/equipment-rules.md FR-033 to FR-036 - raw materials are half the cost in CP rounded down; days are the GP value over 10 rounded up; a rate is invalid; zero cost is 0 CP raw materials but invalid for days.
// Path: happy; unhappy (boundary, invalid input).
func TestGrpD3CraftingRawMaterialsAndDays(t *testing.T) {
	t.Parallel()
	c := character.Crafting{}
	for _, row := range []struct {
		cost content.Cost
		raw  int // copper pieces
		days int
	}{
		{grpD3GP(1500), 75000, 150}, {grpD3GP(15), 750, 2}, {grpD3GP(10), 500, 1},
		{grpD3CP(1001), 500, 2}, {grpD3GP(45), 2250, 5}, {grpD3GP(20), 1000, 2},
		{grpD3CP(1), 0, 1}, {grpD3CP(5), 2, 1},
	} {
		raw, err := c.RawMaterials(row.cost)
		if err != nil || raw != grpD3CP(row.raw) {
			t.Errorf("RawMaterials(%v) = %v, %v; want %d CP", row.cost, raw, err, row.raw)
		}
		days, err := c.Days(row.cost)
		if err != nil || days != row.days {
			t.Errorf("Days(%v) = %d, %v; want %d", row.cost, days, err, row.days)
		}
	}
	rated := content.MustCost(5, content.CoinGP, content.CostPerDay)
	_, err := c.RawMaterials(rated)
	grpD3WantInvalid(t, err, "RawMaterials with a rate")
	_, err = c.Days(rated)
	grpD3WantInvalid(t, err, "Days with a rate")
	if raw, err := c.RawMaterials(grpD3GP(0)); err != nil || raw != grpD3CP(0) {
		t.Errorf("RawMaterials(0) = %v, %v; want 0 CP", raw, err)
	}
	_, err = c.Days(grpD3GP(0))
	grpD3WantInvalid(t, err, "Days of a free item")
}

// Which tools craft an item, the order the answer comes in, and who gets
// assigned by the Game Master (SRD 5.2.1, "Crafting Nonmagical Items", Tools,
// p. 103; Craft entries, pp. 93-94).
// Requirement: doc/specs/equipment-rules.md FR-037 to FR-039 - tools naming the item, its family or a set that includes it without an exception; ordered by id, once each; empty when none names it.
// Path: happy; unhappy (rule interaction, empty and minimal).
func TestGrpD3WhichToolsCraftAnItem(t *testing.T) {
	t.Parallel()
	toolbox := grpD3Toolbox()
	crystal := grpD3Must(content.NewFocusVariant("crystal_x", "Crystal", content.SourceCitation{}, "arcane_focus", grpD3Price(grpD3GP(10)), grpD3Lbs(1), "", "", ""))
	for _, c := range []struct {
		name string
		item content.EquipmentEntry
		want []string
	}{
		{"Club, named by two tools", grpD3Weapon("club_x", content.WeaponMelee, grpD3GP(1)), []string{"carpenters_tools", "woodcarvers_tools"}},
		{"Longsword, in the melee set", grpD3Weapon("longsword_x", content.WeaponMelee, grpD3GP(15)), []string{"smiths_tools"}},
		{"Whip, excepted from the melee set", grpD3Weapon("whip_x", content.WeaponMelee, grpD3GP(2)), []string{"leatherworkers_tools"}},
		{"Longbow, in the ranged set", grpD3Weapon("longbow_x", content.WeaponRanged, grpD3GP(50)), []string{"woodcarvers_tools"}},
		{"Musket, excepted and named", grpD3Weapon("musket_x", content.WeaponRanged, grpD3GP(500)), []string{"tinkers_tools"}},
		{"Plate, in the heavy armor set", grpD3Armor("plate_x", content.ArmorHeavy, grpD3NoDex(18), 15, true, 65, grpD3GP(1500)), []string{"smiths_tools"}},
		{"Chain Shirt, in the medium armor set", grpD3Armor("chain_shirt_x", content.ArmorMedium, grpD3Capped(13, 2), 0, false, 20, grpD3GP(50)), []string{"smiths_tools"}},
		{"Hide, excepted from the medium set", grpD3Armor("hide_x", content.ArmorMedium, grpD3Capped(12, 2), 0, false, 12, grpD3GP(10)), []string{"leatherworkers_tools"}},
		{"Crystal, through its family", crystal, []string{"jewelers_tools", "woodcarvers_tools"}},
		{"Shield, the Game Master assigns", grpD3Armor("shield_x", content.ArmorShield, grpD3Must(content.NewBonusRule(2)), 0, false, 6, grpD3GP(10)), nil},
	} {
		got := character.Crafting{}.ToolsFor(c.item, toolbox)
		if !slices.Equal(got, grpD3IDs(c.want...)) {
			t.Errorf("%s: tools = %v, want %v", c.name, got, c.want)
		}
	}
	// Input order and repeats change nothing.
	club := grpD3Weapon("club_x", content.WeaponMelee, grpD3GP(1))
	reversed := slices.Clone(toolbox)
	slices.Reverse(reversed)
	doubled := append(slices.Clone(toolbox), toolbox...)
	want := grpD3IDs("carpenters_tools", "woodcarvers_tools")
	for _, in := range [][]content.Tool{reversed, doubled} {
		if got := (character.Crafting{}).ToolsFor(club, in); !slices.Equal(got, want) {
			t.Errorf("tools from a reordered toolbox = %v, want %v", got, want)
		}
	}
	if got := (character.Crafting{}).ToolsFor(club, nil); len(got) != 0 {
		t.Errorf("no tools in the catalogue gave %v", got)
	}
}

// A smith plans a suit of Plate, a brewer a Potion of Healing, a scribe a scroll,
// and the clerk turns away what cannot be crafted by the rule (SRD 5.2.1,
// "Crafting Nonmagical Items", p. 103; "Brewing Potions of Healing";
// "Scribing Spell Scrolls").
// Requirement: doc/specs/equipment-rules.md FR-040 to FR-048, FR-063, FR-066 - generic rule, item override, scroll row, and each NotCraftable reason in its order; invalid entries are errors; tools are copies.
// Path: happy; unhappy (rule interaction, missing content, empty and minimal, boundary, immutability, invalid input).
func TestGrpD3CraftingRequirementsForEachKindOfEntry(t *testing.T) {
	t.Parallel()
	src := content.SourceCitation{}
	plate := grpD3Armor("plate_x", content.ArmorHeavy, grpD3NoDex(18), 15, true, 65, grpD3GP(1500))
	override := grpD3Must(content.NewCraftingOverride(grpD3GP(25), 1, grpD3IDs("herbalism_kit")))
	potion := grpD3Must(content.NewMagicGear("potion_x", "Potion of Healing", src, grpD3Price(grpD3GP(50)), content.NoWeight{}, "", "Text.", override))
	scroll := func(level, cost int) content.EquipmentEntry {
		return grpD3Must(content.NewScrollVariant("scroll_x", "Scroll", src, "scroll_family", grpD3Price(grpD3GP(cost)), "", "", level))
	}
	scribing := character.MustScrollScribing(grpD3ScrollRows())
	rated := func(id string) content.EquipmentEntry {
		return grpD3Must(content.NewTackHarnessOrDrawnVehicle(content.ItemID(id), id, src, grpD3Lbs(10), grpD3Price(content.MustCost(5, content.CoinCP, content.CostPerDay))))
	}
	galley := grpD3Must(content.NewAirborneOrWaterborneVehicle("galley_x", "Galley", src, 4, 1, 0, 0, 15, 500, 0, content.NoWeight{}, grpD3Price(grpD3GP(30000))))
	gamingSet := grpD3Must(content.NewToolFamily("gaming_set", "Gaming Set", src, content.ToolGroupOther, ability.Wisdom,
		[]content.ToolUtilizeAction{content.MustToolUtilizeAction("Play", 10)}, nil, grpD3IDs("dragonchess_x"), content.CostVaries{}, content.WeightVaries{}))

	type want struct {
		raw   content.Cost
		days  int
		tools []string
		basis character.CraftingBasis
	}
	for _, c := range []struct {
		name   string
		item   content.EquipmentEntry
		scribe character.ScrollScribing
		req    *want
		reason character.NotCraftableReason
	}{
		{"Plate Armor: 750 GP, 150 days, smith's tools", plate, character.ScrollScribing{}, &want{grpD3CP(75000), 150, []string{"smiths_tools"}, character.CraftingGenericRule}, 0},
		{"Potion of Healing: its own 25 GP and 1 day", potion, character.ScrollScribing{}, &want{grpD3GP(25), 1, []string{"herbalism_kit"}, character.CraftingItemOverride}, 0},
		{"Spell Scroll (Cantrip): the table row, no tools", scroll(0, 30), scribing, &want{grpD3GP(15), 1, nil, character.CraftingScrollRow}, 0},
		{"Spell Scroll (Level 3): the table row", scroll(3, 300), scribing, &want{grpD3GP(150), 5, nil, character.CraftingScrollRow}, 0},
		{"Spell Scroll without the table", scroll(0, 30), character.ScrollScribing{}, nil, character.NotCraftableScribingTableMissing},
		{"Gaming Set family: price varies", gamingSet, character.ScrollScribing{}, nil, character.NotCraftablePriceVaries},
		{"Feed: rated price", rated("feed_x"), character.ScrollScribing{}, nil, character.NotCraftableRatedPrice},
		{"free gear", grpD3PlainGear("free_x", grpD3GP(0)), character.ScrollScribing{}, nil, character.NotCraftableFreeOfCharge},
		{"Horse: not an item", grpD3Must(content.NewMount("horse_x", "Horse", src, 540, grpD3Price(grpD3GP(50)))), character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Hireling: not an item", grpD3Must(content.NewHireling("hireling_x", "Hireling", src, grpD3Price(grpD3GP(1)))), character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Wretched lifestyle at 0 GP: not an item, not free", grpD3Must(content.NewLifestyleExpense("wretched_x", "Wretched", src, grpD3Price(grpD3GP(0)), "Text.")), character.ScrollScribing{}, nil, character.NotCraftableNotAnItem},
		{"Galley: no crafting rule", galley, character.ScrollScribing{}, nil, character.NotCraftableNoCraftingRule},
	} {
		got, err := character.Crafting{}.RequirementsFor(c.item, grpD3Toolbox(), c.scribe)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		switch v := got.(type) {
		case character.CraftingRequirements:
			if c.req == nil || v.RawMaterials() != c.req.raw || v.Days() != c.req.days ||
				!slices.Equal(v.Tools(), grpD3IDs(c.req.tools...)) || v.Basis() != c.req.basis {
				t.Errorf("%s: requirements = %v, %d days, %v, %v; want %+v", c.name, v.RawMaterials(), v.Days(), v.Tools(), v.Basis(), c.req)
			}
		case character.NotCraftable:
			if c.req != nil || v.Reason != c.reason {
				t.Errorf("%s: NotCraftable(%v), want %+v / %v", c.name, v.Reason, c.req, c.reason)
			}
		default:
			t.Errorf("%s: outcome = %#v", c.name, got)
		}
	}

	// The tools are the caller's copy.
	got, err := character.Crafting{}.RequirementsFor(plate, grpD3Toolbox(), character.ScrollScribing{})
	if err != nil {
		t.Fatal(err)
	}
	r := got.(character.CraftingRequirements)
	r.Tools()[0] = "changed"
	if !slices.Equal(r.Tools(), grpD3IDs("smiths_tools")) {
		t.Errorf("tools = %v after mutating a read", r.Tools())
	}

	// Invalid entries are errors, never a guess.
	_, err = character.Crafting{}.RequirementsFor(nil, nil, character.ScrollScribing{})
	grpD3WantInvalid(t, err, "a nil item")
	huge := grpD3PlainGear("huge_x", content.MustCost(content.MaxCostAmount, content.CoinPP, 0))
	_, err = character.Crafting{}.RequirementsFor(huge, nil, character.ScrollScribing{})
	grpD3WantInvalid(t, err, "an item priced beyond the range")
}
