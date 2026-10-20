package content_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

func refFields(refs []content.Reference) []string {
	var out []string
	for _, r := range refs {
		if r.Constraint() != nil || r.Target().Kind() != content.KindEquipment {
			out = append(out, "bad:"+r.Field())
			continue
		}
		out = append(out, r.Field()+"="+r.Target().ID().String())
	}
	return out
}

// Requirement: equipSpec FR-108 - ToolUtilizeAction is a value object with a
// non-blank description and a DC of at least 1; ToolGroup and CraftSetName are
// closed enums.
// Path: per case.
func TestToolParts(t *testing.T) {
	t.Parallel()
	if a, err := content.NewToolUtilizeAction("Open a lock", 1); err != nil || a.Description() != "Open a lock" || a.DC() != 1 {
		t.Errorf("NewToolUtilizeAction = %+v, %v [FR-108 boundary: DC 1]", a, err)
	}
	for _, c := range []struct {
		name, text string
		dc         int
		want       content.BlockProblem
	}{
		{"blank description", " ", 10, content.BlockBlankText}, {"control character", "a\x00", 10, content.BlockInvalidCharacter},
		{"DC 0", "x", 0, content.BlockOutOfRange}, {"DC above the bound", "x", beyond(), content.BlockOutOfRange},
		{"description before DC", "", 0, content.BlockBlankText},
	} {
		_, err := content.NewToolUtilizeAction(c.text, c.dc)
		wantBlockError(t, err, "ToolUtilizeAction", c.want)
		assertMustPanics(t, func() { content.MustToolUtilizeAction(c.text, c.dc) })
	}
	if content.ToolGroupArtisans.String() != "Artisan's Tools" || content.ToolGroupOther.String() != "Other Tools" || content.ToolGroup(0).Valid() || content.ToolGroup(3).String() != "unknown" {
		t.Error("ToolGroup is Artisan's Tools or Other Tools [FR-113]")
	}
	for i, n := range []string{"MeleeWeapons", "RangedWeapons", "MediumArmor", "HeavyArmor"} {
		if s := content.CraftSetName(i + 1); s.String() != n || !s.Valid() {
			t.Errorf("CraftSetName(%d) = %q, want %q [FR-031]", i+1, s, n)
		}
	}
	if content.CraftSetName(0).Valid() || content.CraftSetName(5).Valid() {
		t.Error("CraftSetName outside 1 to 4 is valid")
	}
	// Craft references.
	if c, err := content.NewCraftItem("club_x"); err != nil || c.ID() != "club_x" {
		t.Errorf("NewCraftItem = %+v, %v [FR-031 happy]", c, err)
	}
	if _, err := content.NewCraftItem("Bad"); err == nil {
		t.Error("NewCraftItem accepted an invalid id [FR-107]")
	}
	except := []content.ItemID{"a", "b"}
	set, err := content.NewCraftSet(content.CraftMeleeWeapons, except)
	if err != nil || set.Set() != content.CraftMeleeWeapons || !slices.Equal(set.Except(), except) {
		t.Errorf("NewCraftSet = %+v, %v [FR-031 happy]", set, err)
	}
	except[0] = "z"
	got := set.Except()
	got[1] = "z"
	if !slices.Equal(set.Except(), []content.ItemID{"a", "b"}) {
		t.Error("CraftSet shares its exception list [FR-007]")
	}
	if _, err := content.NewCraftSet(0, nil); err == nil {
		t.Error("NewCraftSet accepted an invalid set name")
	}
	if _, err := content.NewCraftSet(content.CraftHeavyArmor, []content.ItemID{"Bad"}); err == nil {
		t.Error("NewCraftSet accepted an invalid exception id")
	}
}

func testUtilize() []content.ToolUtilizeAction {
	return []content.ToolUtilizeAction{content.MustToolUtilizeAction("Do a thing", 10)}
}

// Requirement: equipSpec FR-007, FR-029, FR-030, FR-107, FR-119 - the three
// tool types hold what they were built with, copy their lists, and list their
// reference sites (existence only, plan item P4).
// Path: happy and unhappy (immutability).
func TestTools(t *testing.T) {
	t.Parallel()
	item, _ := content.NewCraftItem("club_x")
	set, _ := content.NewCraftSet(content.CraftRangedWeapons, []content.ItemID{"bow_x"})
	craft := []content.CraftReference{item, set}
	price := fixedCost(5, content.CoinGP)
	w := content.MustWeight(5, "")

	s, err := content.NewStandaloneTool("tool_x", "Tool X", noSource, content.ToolGroupArtisans, ability.Dexterity, testUtilize(), craft, price, w)
	if err != nil {
		t.Fatal(err)
	}
	if s.Group() != content.ToolGroupArtisans || s.Ability() != ability.Dexterity || s.Price() != price || s.Weight() != content.WeightSpec(w) ||
		s.EquipmentCategory() != content.EquipmentTool || s.Utilize()[0].DC() != 10 || len(s.Craft()) != 2 {
		t.Errorf("StandaloneTool accessors = %+v [FR-029]", s)
	}
	var _ content.Tool = s
	if got, want := refFields(s.References()), []string{"craft/0/id=club_x", "craft/1/except/0=bow_x"}; !slices.Equal(got, want) {
		t.Errorf("StandaloneTool references = %v, want %v [R2-equipment sites]", got, want)
	}
	s.Utilize()[0] = content.ToolUtilizeAction{}
	s.Craft()[0] = nil
	if s.Utilize()[0].DC() != 10 || s.Craft()[0] == nil {
		t.Error("mutating a returned list changed the tool [FR-007]")
	}
	if empty, err := content.NewStandaloneTool("tool_y", "Tool Y", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, price, content.NoWeight{}); err != nil || empty.Craft() != nil || len(empty.References()) != 0 {
		t.Errorf("a tool with an empty Craft list = %+v, %v [FR-033 empty case]", empty, err)
	}

	fam, err := content.NewToolFamily("family_x", "Family X", noSource, content.ToolGroupOther, ability.Charisma, testUtilize(), nil, []content.ItemID{"v_a", "v_b"}, content.CostVaries{}, content.WeightVaries{})
	if err != nil {
		t.Fatal(err)
	}
	if fam.Price() != content.Price(content.CostVaries{}) || fam.Weight() != content.WeightSpec(content.WeightVaries{}) || !slices.Equal(fam.Variants(), []content.ItemID{"v_a", "v_b"}) ||
		fam.Group() != content.ToolGroupOther || fam.Ability() != ability.Charisma || len(fam.Utilize()) != 1 || fam.Craft() != nil || fam.EquipmentCategory() != content.EquipmentTool {
		t.Errorf("ToolFamily accessors = %+v [FR-030]", fam)
	}
	var _ content.Tool = fam
	fam.Variants()[0] = "z"
	if fam.Variants()[0] != "v_a" {
		t.Error("mutating Variants changed the family [FR-007]")
	}
	if got, want := refFields(fam.References()), []string{"variants/0=v_a", "variants/1=v_b"}; !slices.Equal(got, want) {
		t.Errorf("ToolFamily references = %v, want %v [R2-equipment sites]", got, want)
	}

	v, err := content.NewToolVariant("v_a", "Variant A", noSource, "family_x", price, w)
	if err != nil {
		t.Fatal(err)
	}
	if v.VariantOf() != "family_x" || v.Price() != price || v.Weight() != content.WeightSpec(w) || v.EquipmentCategory() != content.EquipmentTool {
		t.Errorf("ToolVariant accessors = %+v [FR-029]", v)
	}
	var _ content.Tool = v
	if got, want := refFields(v.References()), []string{"variantOf=family_x"}; !slices.Equal(got, want) {
		t.Errorf("ToolVariant references = %v, want %v [R2-equipment sites]", got, want)
	}
}

// Requirement: equipSpec FR-029, FR-030, FR-107 - the tool constructors reject
// an invalid part, an empty Utilize list and an empty or repeated variant list.
// Path: per case.
func TestNewToolRejects(t *testing.T) {
	t.Parallel()
	price, w := fixedCost(5, content.CoinGP), content.WeightSpec(content.NoWeight{})
	item, _ := content.NewCraftItem("club_x")
	bad := []content.ToolUtilizeAction{{}}
	cases := []struct {
		name  string
		block string
		make  func() error
		want  content.BlockProblem
	}{
		{"no utilize", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, nil, nil, price, w)
			return err
		}, content.BlockEmpty},
		{"zero group", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, 0, ability.Wisdom, testUtilize(), nil, price, w)
			return err
		}, content.BlockInvalidMember},
		{"zero ability", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, 0, testUtilize(), nil, price, w)
			return err
		}, content.BlockInvalidMember},
		{"zero utilize action", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, bad, nil, price, w)
			return err
		}, content.BlockInvalidMember},
		{"nil craft reference", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), []content.CraftReference{nil}, price, w)
			return err
		}, content.BlockInvalidMember},
		{"zero craft item", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), []content.CraftReference{content.CraftItem{}}, price, w)
			return err
		}, content.BlockInvalidMember},
		{"zero craft set", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), []content.CraftReference{content.CraftSet{}}, price, w)
			return err
		}, content.BlockInvalidMember},
		{"zero price", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), []content.CraftReference{item}, content.FixedCost{}, w)
			return err
		}, content.BlockInvalidMember},
		{"nil weight", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, price, nil)
			return err
		}, content.BlockInvalidMember},
		{"bad name", "StandaloneTool", func() error {
			_, err := content.NewStandaloneTool("a", "", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, price, w)
			return err
		}, content.BlockInvalidName},
		{"family without variants", "ToolFamily", func() error {
			_, err := content.NewToolFamily("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, nil, content.CostVaries{}, w)
			return err
		}, content.BlockOutOfRange},
		{"family with a repeated variant", "ToolFamily", func() error {
			_, err := content.NewToolFamily("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, []content.ItemID{"b", "b"}, content.CostVaries{}, w)
			return err
		}, content.BlockDuplicate},
		{"family with a bad price", "ToolFamily", func() error {
			_, err := content.NewToolFamily("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, []content.ItemID{"b"}, nil, w)
			return err
		}, content.BlockInvalidMember},
		{"family with a bad name", "ToolFamily", func() error {
			_, err := content.NewToolFamily("a", "", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, []content.ItemID{"b"}, content.CostVaries{}, w)
			return err
		}, content.BlockInvalidName},
		{"family with a bad body", "ToolFamily", func() error {
			_, err := content.NewToolFamily("a", "A", noSource, 0, ability.Wisdom, testUtilize(), nil, []content.ItemID{"b"}, content.CostVaries{}, w)
			return err
		}, content.BlockInvalidMember},
		{"variant with a zero price", "ToolVariant", func() error {
			_, err := content.NewToolVariant("a", "A", noSource, "f", content.FixedCost{}, w)
			return err
		}, content.BlockInvalidMember},
		{"variant with a nil weight", "ToolVariant", func() error {
			_, err := content.NewToolVariant("a", "A", noSource, "f", price, nil)
			return err
		}, content.BlockInvalidMember},
		{"variant with a bad name", "ToolVariant", func() error {
			_, err := content.NewToolVariant("a", " ", noSource, "f", price, w)
			return err
		}, content.BlockInvalidName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wantBlockError(t, c.make(), c.block, c.want)
		})
	}
	if _, err := content.NewToolVariant("a", "A", noSource, "Bad Id", price, w); err == nil {
		t.Error("a variant of an invalid family id was accepted [FR-107]")
	}
	if _, err := content.NewToolFamily("a", "A", noSource, content.ToolGroupOther, ability.Wisdom, testUtilize(), nil, []content.ItemID{"Bad Id"}, content.CostVaries{}, w); err == nil {
		t.Error("a family with an invalid variant id was accepted [FR-107]")
	}
}

// Requirement: equipSpec FR-041, FR-042, FR-108 - AmmunitionData, PackItem and
// CraftingOverride are value objects with validating constructors.
// Path: per case.
func TestGearParts(t *testing.T) {
	t.Parallel()
	if a, err := content.NewAmmunitionData(20, "Quiver"); err != nil || a.Amount() != 20 || a.Storage() != "Quiver" {
		t.Errorf("NewAmmunitionData = %+v, %v [FR-041 happy]", a, err)
	}
	for _, c := range []struct {
		amount  int
		storage string
		want    content.BlockProblem
	}{{0, "Q", content.BlockOutOfRange}, {beyond(), "Q", content.BlockOutOfRange}, {1, "", content.BlockBlankText}, {1, "a\x00", content.BlockInvalidCharacter}} {
		_, err := content.NewAmmunitionData(c.amount, c.storage)
		wantBlockError(t, err, "AmmunitionData", c.want)
	}
	if p, err := content.NewPackItem("rope_x", 2, "days"); err != nil || p.ID() != "rope_x" || p.Quantity() != 2 || p.Unit() != "days" {
		t.Errorf("NewPackItem = %+v, %v [FR-042 happy]", p, err)
	}
	if p, err := content.NewPackItem("rope_x", 1, ""); err != nil || p.Unit() != "" {
		t.Errorf("NewPackItem without a unit = %+v, %v [FR-042 happy]", p, err)
	}
	for _, c := range []struct {
		quantity int
		unit     string
		want     content.BlockProblem
	}{{0, "", content.BlockOutOfRange}, {beyond(), "", content.BlockOutOfRange}, {1, " ", content.BlockBlankText}} {
		_, err := content.NewPackItem("rope_x", c.quantity, c.unit)
		wantBlockError(t, err, "PackItem", c.want)
	}
	if _, err := content.NewPackItem("Bad Id", 1, ""); err == nil {
		t.Error("NewPackItem accepted an invalid id [FR-107]")
	}
	raw := content.MustCost(25, content.CoinGP, 0)
	tools := []content.ItemID{"kit_x"}
	o, err := content.NewCraftingOverride(raw, 1, tools)
	if err != nil || o.RawMaterials() != raw || o.Days() != 1 || !slices.Equal(o.ToolIDs(), tools) {
		t.Fatalf("NewCraftingOverride = %+v, %v [FR-108 happy]", o, err)
	}
	tools[0] = "z"
	o.ToolIDs()[0] = "z"
	if !slices.Equal(o.ToolIDs(), []content.ItemID{"kit_x"}) {
		t.Error("CraftingOverride shares its tool list [FR-007]")
	}
	if none, err := content.NewCraftingOverride(raw, 3, nil); err != nil || none.ToolIDs() != nil {
		t.Errorf("an override with no tools = %+v, %v [FR-108 empty list accepted]", none, err)
	}
	same, _ := content.NewCraftingOverride(raw, 1, []content.ItemID{"kit_x"})
	other, _ := content.NewCraftingOverride(raw, 2, []content.ItemID{"kit_x"})
	otherTools, _ := content.NewCraftingOverride(raw, 1, []content.ItemID{"kit_y"})
	otherCost, _ := content.NewCraftingOverride(content.MustCost(26, content.CoinGP, 0), 1, []content.ItemID{"kit_x"})
	if !o.Equal(same) || o.Equal(other) || o.Equal(otherTools) || o.Equal(otherCost) {
		t.Error("CraftingOverride.Equal does not compare every part [FR-108]")
	}
	for _, c := range []struct {
		raw  content.Cost
		days int
		want content.BlockProblem
	}{{content.Cost{}, 1, content.BlockInvalidMember}, {raw, 0, content.BlockOutOfRange}, {raw, beyond(), content.BlockOutOfRange}} {
		_, err := content.NewCraftingOverride(c.raw, c.days, nil)
		wantBlockError(t, err, "CraftingOverride", c.want)
	}
	if _, err := content.NewCraftingOverride(raw, 1, []content.ItemID{"Bad Id"}); err == nil {
		t.Error("NewCraftingOverride accepted an invalid tool id [FR-107]")
	}
}

// Requirement: equipSpec FR-039, FR-081, FR-095, FR-114 - the eight gear types
// hold what they were built with; a gear entry has a Weight, Note and Text; a
// variant names its family; the Spell Scroll family is CostVaries and NoWeight.
// Path: happy.
func TestGear(t *testing.T) {
	t.Parallel()
	price, w := fixedCost(2, content.CoinSP), content.WeightSpec(content.MustWeight(1, "full"))
	plain, err := content.NewPlainGear("gear_x", "Gear X", noSource, price, w, "a note", "Text.")
	if err != nil {
		t.Fatal(err)
	}
	if plain.Price() != price || plain.Weight() != w || plain.Note() != "a note" || plain.Text() != "Text." || plain.EquipmentCategory() != content.EquipmentAdventuringGear || plain.Key().Kind() != content.KindEquipment {
		t.Errorf("PlainGear accessors = %+v [FR-039]", plain)
	}
	var _ content.AdventuringGear = plain

	override, _ := content.NewCraftingOverride(content.MustCost(25, content.CoinGP, 0), 1, []content.ItemID{"kit_x"})
	magic, err := content.NewMagicGear("potion_x", "Potion X", noSource, price, content.NoWeight{}, "", "Text.", override)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := magic.CraftingOverride(); !ok || !got.Equal(override) || magic.Price() != price {
		t.Errorf("MagicGear override = %+v, %v [FR-094]", got, ok)
	}
	if got, _ := magic.CraftingOverride(); got.ToolIDs() == nil {
		t.Error("lost the override tools")
	}
	if got, want := refFields(magic.References()), []string{"craftingOverride/toolIds/0=kit_x"}; !slices.Equal(got, want) {
		t.Errorf("MagicGear references = %v, want %v [R2-equipment sites]", got, want)
	}
	var _ content.AdventuringGear = magic
	bare, err := content.NewMagicGear("potion_y", "Potion Y", noSource, price, content.NoWeight{}, "", "Text.", content.CraftingOverride{})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := bare.CraftingOverride(); ok || got.Days() != 0 || len(bare.References()) != 0 {
		t.Errorf("MagicGear without an override reports %+v, %v [FR-094]", got, ok)
	}

	family, err := content.NewGearFamily("family_x", "Family X", noSource, content.CostVaries{}, content.WeightVaries{}, "", "Text.", []content.ItemID{"v_a"})
	if err != nil {
		t.Fatal(err)
	}
	if family.Price() != content.Price(content.CostVaries{}) || family.Weight() != content.WeightSpec(content.WeightVaries{}) || !slices.Equal(family.Variants(), []content.ItemID{"v_a"}) {
		t.Errorf("GearFamily accessors = %+v [FR-039]", family)
	}
	family.Variants()[0] = "z"
	if family.Variants()[0] != "v_a" {
		t.Error("GearFamily shares its variant list [FR-007]")
	}
	if got, want := refFields(family.References()), []string{"variants/0=v_a"}; !slices.Equal(got, want) {
		t.Errorf("GearFamily references = %v, want %v", got, want)
	}
	var _ content.AdventuringGear = family

	scrolls, err := content.NewScrollFamily("scroll_x", "Scroll X", noSource, "", "Text.", []content.ItemID{"s_0", "s_1"})
	if err != nil {
		t.Fatal(err)
	}
	if scrolls.Price() != content.Price(content.CostVaries{}) || scrolls.Weight() != content.WeightSpec(content.NoWeight{}) || len(scrolls.Variants()) != 2 || len(scrolls.References()) != 2 {
		t.Errorf("ScrollFamily = %+v [FR-081]", scrolls)
	}
	scrolls.Variants()[0] = "z"
	if scrolls.Variants()[0] != "s_0" {
		t.Error("ScrollFamily shares its variant list [FR-007]")
	}
	var _ content.AdventuringGear = scrolls

	ammo, _ := content.NewAmmunitionData(20, "Quiver")
	av, err := content.NewAmmunitionVariant("arrows_x", "Arrows X", noSource, "family_x", price, w, "", "", ammo)
	if err != nil {
		t.Fatal(err)
	}
	if av.VariantOf() != "family_x" || av.Ammunition() != ammo || av.Price() != price || av.Text() != "" {
		t.Errorf("AmmunitionVariant = %+v [FR-041]", av)
	}
	var _ content.GearVariant = av
	fv, err := content.NewFocusVariant("staff_x", "Staff X", noSource, "family_x", price, w, "held", "", "quarterstaff_x")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := fv.AlsoCountsAs(); !ok || got != "quarterstaff_x" || fv.Note() != "held" {
		t.Errorf("FocusVariant AlsoCountsAs = %q, %v [FR-040]", got, ok)
	}
	if got, want := refFields(fv.References()), []string{"variantOf=family_x", "alsoCountsAs=quarterstaff_x"}; !slices.Equal(got, want) {
		t.Errorf("FocusVariant references = %v, want %v", got, want)
	}
	plainFocus, err := content.NewFocusVariant("orb_x", "Orb X", noSource, "family_x", price, w, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := plainFocus.AlsoCountsAs(); ok || got != "" || len(plainFocus.References()) != 1 {
		t.Errorf("a focus that counts as nothing = %q, %v, %v", got, ok, plainFocus.References())
	}
	var _ content.GearVariant = fv
	sv, err := content.NewScrollVariant("s_0", "Scroll 0", noSource, "scroll_x", price, "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if sv.SpellLevel() != 0 || sv.Weight() != content.WeightSpec(content.NoWeight{}) || sv.VariantOf() != "scroll_x" {
		t.Errorf("ScrollVariant = %+v [FR-081, FR-114]", sv)
	}
	if got, want := refFields(sv.References()), []string{"variantOf=scroll_x"}; !slices.Equal(got, want) {
		t.Errorf("ScrollVariant references = %v, want %v", got, want)
	}
	var _ content.GearVariant = sv

	contents := []content.PackItem{mustPackItem(t, "rope_x", 2, "days"), mustPackItem(t, "torch_x", 1, "")}
	pack, err := content.NewPack("pack_x", "Pack X", noSource, price, w, "", "Text.", contents)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(pack.Contents(), contents) || pack.Price() != price {
		t.Errorf("Pack = %+v [FR-042]", pack)
	}
	pack.Contents()[0] = content.PackItem{}
	contents[1] = content.PackItem{}
	if pack.Contents()[0] != mustPackItem(t, "rope_x", 2, "days") || pack.Contents()[1] != mustPackItem(t, "torch_x", 1, "") {
		t.Error("Pack shares its contents [FR-007]")
	}
	if got, want := refFields(pack.References()), []string{"contents/0/id=rope_x", "contents/1/id=torch_x"}; !slices.Equal(got, want) {
		t.Errorf("Pack references = %v, want %v", got, want)
	}
	var _ content.AdventuringGear = pack
}

func mustPackItem(t *testing.T, id content.ItemID, quantity int, unit string) content.PackItem {
	t.Helper()
	p, err := content.NewPackItem(id, quantity, unit)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Requirement: equipSpec FR-037, FR-039, FR-081, FR-114 - the gear constructors
// reject an invalid weight, price, note, text, variant list or spell level.
// Path: per case.
func TestNewGearRejects(t *testing.T) {
	t.Parallel()
	price, w := fixedCost(2, content.CoinSP), content.WeightSpec(content.NoWeight{})
	ammo, _ := content.NewAmmunitionData(1, "Q")
	cases := []struct {
		name  string
		block string
		make  func() error
		want  content.BlockProblem
	}{
		{"plain without text", "PlainGear", func() error { _, err := content.NewPlainGear("a", "A", noSource, price, w, "", ""); return err }, content.BlockMissing},
		{"plain with blank text", "PlainGear", func() error { _, err := content.NewPlainGear("a", "A", noSource, price, w, "", " "); return err }, content.BlockBlankText},
		{"plain with blank note", "PlainGear", func() error { _, err := content.NewPlainGear("a", "A", noSource, price, w, " ", "t"); return err }, content.BlockBlankText},
		{"plain with nil weight", "PlainGear", func() error { _, err := content.NewPlainGear("a", "A", noSource, price, nil, "", "t"); return err }, content.BlockInvalidMember},
		{"plain with zero price", "PlainGear", func() error {
			_, err := content.NewPlainGear("a", "A", noSource, content.FixedCost{}, w, "", "t")
			return err
		}, content.BlockInvalidMember},
		{"plain with bad name", "PlainGear", func() error { _, err := content.NewPlainGear("a", "", noSource, price, w, "", "t"); return err }, content.BlockInvalidName},
		{"magic with zero price", "MagicGear", func() error {
			_, err := content.NewMagicGear("a", "A", noSource, content.FixedCost{}, w, "", "t", content.CraftingOverride{})
			return err
		}, content.BlockInvalidMember},
		{"magic without text", "MagicGear", func() error {
			_, err := content.NewMagicGear("a", "A", noSource, price, w, "", "", content.CraftingOverride{})
			return err
		}, content.BlockMissing},
		{"family with a bad price", "GearFamily", func() error {
			_, err := content.NewGearFamily("a", "A", noSource, nil, w, "", "t", []content.ItemID{"b"})
			return err
		}, content.BlockInvalidMember},
		{"family without variants", "GearFamily", func() error {
			_, err := content.NewGearFamily("a", "A", noSource, content.CostVaries{}, w, "", "t", nil)
			return err
		}, content.BlockEmpty},
		{"family with a repeated variant", "GearFamily", func() error {
			_, err := content.NewGearFamily("a", "A", noSource, content.CostVaries{}, w, "", "t", []content.ItemID{"b", "b"})
			return err
		}, content.BlockDuplicate},
		{"family without text", "GearFamily", func() error {
			_, err := content.NewGearFamily("a", "A", noSource, content.CostVaries{}, w, "", "", []content.ItemID{"b"})
			return err
		}, content.BlockMissing},
		{"scroll family without variants", "ScrollFamily", func() error {
			_, err := content.NewScrollFamily("a", "A", noSource, "", "t", nil)
			return err
		}, content.BlockEmpty},
		{"scroll family without text", "ScrollFamily", func() error {
			_, err := content.NewScrollFamily("a", "A", noSource, "", "", []content.ItemID{"b"})
			return err
		}, content.BlockMissing},
		{"ammunition with zero data", "AmmunitionVariant", func() error {
			_, err := content.NewAmmunitionVariant("a", "A", noSource, "f", price, w, "", "", content.AmmunitionData{})
			return err
		}, content.BlockInvalidMember},
		{"ammunition with zero price", "AmmunitionVariant", func() error {
			_, err := content.NewAmmunitionVariant("a", "A", noSource, "f", content.FixedCost{}, w, "", "", ammo)
			return err
		}, content.BlockInvalidMember},
		{"variant with a blank text", "AmmunitionVariant", func() error {
			_, err := content.NewAmmunitionVariant("a", "A", noSource, "f", price, w, "", " ", ammo)
			return err
		}, content.BlockBlankText},
		{"scroll variant with a bad name", "ScrollVariant", func() error {
			_, err := content.NewScrollVariant("a", "", noSource, "f", price, "", "", 0)
			return err
		}, content.BlockInvalidName},
		{"focus variant with a bad name", "FocusVariant", func() error {
			_, err := content.NewFocusVariant("a", "", noSource, "f", price, w, "", "", "")
			return err
		}, content.BlockInvalidName},
		{"pack with a bad name", "Pack", func() error {
			_, err := content.NewPack("a", "", noSource, price, w, "", "t", []content.PackItem{mustPackItem(t, "x", 1, "")})
			return err
		}, content.BlockInvalidName},
		{"scroll variant level 10", "ScrollVariant", func() error {
			_, err := content.NewScrollVariant("a", "A", noSource, "f", price, "", "", 10)
			return err
		}, content.BlockOutOfRange},
		{"scroll variant level -1", "ScrollVariant", func() error {
			_, err := content.NewScrollVariant("a", "A", noSource, "f", price, "", "", -1)
			return err
		}, content.BlockOutOfRange},
		{"pack without contents", "Pack", func() error {
			_, err := content.NewPack("a", "A", noSource, price, w, "", "t", nil)
			return err
		}, content.BlockEmpty},
		{"pack with a zero line", "Pack", func() error {
			_, err := content.NewPack("a", "A", noSource, price, w, "", "t", []content.PackItem{{}})
			return err
		}, content.BlockInvalidMember},
		{"pack with zero price", "Pack", func() error {
			_, err := content.NewPack("a", "A", noSource, content.FixedCost{}, w, "", "t", []content.PackItem{mustPackItem(t, "x", 1, "")})
			return err
		}, content.BlockInvalidMember},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wantBlockError(t, c.make(), c.block, c.want)
		})
	}
	// A level from 0 to 9 is accepted on a scroll row; the bounds are 0 and 9.
	for _, level := range []int{0, 1, 9} {
		if _, err := content.NewScrollVariant("a", "A", noSource, "f", price, "", "", level); err != nil {
			t.Errorf("scroll variant level %d: %v [FR-114 boundary]", level, err)
		}
	}
	if _, err := content.NewFocusVariant("a", "A", noSource, "f", price, w, "", "", "Bad Id"); err == nil {
		t.Error("a focus variant that counts as an invalid id was accepted [FR-107]")
	}
	if _, err := content.NewAmmunitionVariant("a", "A", noSource, "Bad Id", price, w, "", "", ammo); err == nil {
		t.Error("a variant of an invalid family id was accepted [FR-107]")
	}
}

// TestGearOwnsItsLists: MagicGear hands out a copy of its override tool list,
// and GearFamily copies the variant list it is built from, so neither can be
// changed through a shared slice.
// Requirement: FR-007, FR-094.
// Path: happy.
func TestGearOwnsItsLists(t *testing.T) {
	t.Parallel()
	override, err := content.NewCraftingOverride(content.MustCost(25, content.CoinGP, 0), 1, []content.ItemID{"kit_x"})
	if err != nil {
		t.Fatal(err)
	}
	magic, err := content.NewMagicGear("potion_x", "Potion X", noSource, fixedCost(2, content.CoinSP), content.NoWeight{}, "", "Text.", override)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := magic.CraftingOverride()
	got.ToolIDs()[0] = "z" // ToolIDs copies, so this must not reach the entry
	if again, _ := magic.CraftingOverride(); again.ToolIDs()[0] != "kit_x" {
		t.Error("MagicGear.CraftingOverride shares its tool list [FR-007]")
	}

	variants := []content.ItemID{"v_a"}
	family, err := content.NewGearFamily("family_x", "Family X", noSource, content.CostVaries{}, content.WeightVaries{}, "", "Text.", variants)
	if err != nil {
		t.Fatal(err)
	}
	variants[0] = "z"
	if family.Variants()[0] != "v_a" {
		t.Error("NewGearFamily keeps the caller's variant slice [FR-007]")
	}
}
