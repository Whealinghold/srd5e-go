package content

import (
	"slices"
	"strconv"
)

// AmmunitionData is the amount and storage of an ammunition row (SRD 5.2.1,
// "Ammunition", p. 96). The zero value is invalid; build it with
// [NewAmmunitionData]. Comparable.
type AmmunitionData struct {
	amount  int
	storage string
}

// NewAmmunitionData returns data for amount pieces held in the printed
// storage ("Quiver").
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for an amount below 1 or above 2^31 - 1, then BlockBlankText or
// BlockInvalidCharacter for a storage that breaks the text rule (schema
// FR-028). It never panics.
func NewAmmunitionData(amount int, storage string) (AmmunitionData, error) {
	if amount < 1 || amount > equipmentMaxInt {
		return AmmunitionData{}, blockError("AmmunitionData", BlockOutOfRange)
	}
	if err := checkText("AmmunitionData", storage); err != nil {
		return AmmunitionData{}, err
	}
	return AmmunitionData{amount: amount, storage: storage}, nil
}

// Amount returns the number of pieces.
func (a AmmunitionData) Amount() int { return a.amount }

// Storage returns the printed storage.
func (a AmmunitionData) Storage() string { return a.storage }

// PackItem is one line of the contents of a [Pack]: an item, how many and the
// printed unit ("flasks"). The zero value is invalid; build it with
// [NewPackItem]. Comparable.
type PackItem struct {
	id       ItemID
	quantity int
	unit     string
}

// NewPackItem returns a line of quantity of the item id, with unit "" for
// none.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockOutOfRange for a
// quantity below 1 or above 2^31 - 1; BlockBlankText or BlockInvalidCharacter
// for a non-empty unit that breaks the text rule. It never panics.
func NewPackItem(id ItemID, quantity int, unit string) (PackItem, error) {
	if _, err := ParseID(string(id)); err != nil {
		return PackItem{}, err
	}
	if quantity < 1 || quantity > equipmentMaxInt {
		return PackItem{}, blockError("PackItem", BlockOutOfRange)
	}
	if unit != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkText("PackItem", unit); err != nil {
			return PackItem{}, err
		}
	}
	return PackItem{id: id, quantity: quantity, unit: unit}, nil
}

// ID returns the item.
func (p PackItem) ID() ItemID { return p.id }

// Quantity returns how many.
func (p PackItem) Quantity() int { return p.quantity }

// Unit returns the printed unit, or "" for none.
func (p PackItem) Unit() string { return p.unit }

// CraftingOverride is the crafting data of an item whose cost or time differs
// from the general rule: the raw materials, the days and the tools (SRD 5.2.1,
// "Brewing Potions of Healing", p. 103). It is data only. The zero value is
// invalid; build it with [NewCraftingOverride]. It holds a slice, so it is
// not comparable with ==; compare with [CraftingOverride.Equal].
type CraftingOverride struct {
	rawMaterials Cost
	days         int
	toolIDs      []ItemID
}

// NewCraftingOverride returns an override with the raw materials, at least 1
// day and the tools (nil or empty for none).
//
// It returns an error wrapping srderr.ErrInvalidInput: a *[BlockError]
// (BlockInvalidMember) for invalid raw materials, BlockOutOfRange for days
// below 1 or above 2^31 - 1, then the *[IDError] of [ParseID] for an invalid
// tool id. It never panics.
func NewCraftingOverride(rawMaterials Cost, days int, toolIDs []ItemID) (CraftingOverride, error) {
	switch {
	case !rawMaterials.Valid():
		return CraftingOverride{}, blockError("CraftingOverride", BlockInvalidMember)
	case days < 1 || days > equipmentMaxInt:
		return CraftingOverride{}, blockError("CraftingOverride", BlockOutOfRange)
	}
	for _, id := range toolIDs {
		if _, err := ParseID(string(id)); err != nil {
			return CraftingOverride{}, err
		}
	}
	return CraftingOverride{rawMaterials: rawMaterials, days: days, toolIDs: slices.Clone(toolIDs)}, nil
}

// RawMaterials returns the cost of the raw materials.
func (c CraftingOverride) RawMaterials() Cost { return c.rawMaterials }

// Days returns the number of days.
func (c CraftingOverride) Days() int { return c.days }

// ToolIDs returns the tools, as a new slice; nil for none.
func (c CraftingOverride) ToolIDs() []ItemID { return slices.Clone(c.toolIDs) }

// Equal reports whether c and o hold the same values.
func (c CraftingOverride) Equal(o CraftingOverride) bool {
	return c.rawMaterials == o.rawMaterials && c.days == o.days && slices.Equal(c.toolIDs, o.toolIDs)
}

// AdventuringGear is a row of the Adventuring Gear table (SRD 5.2.1, pp.
// 94-100): a closed set with the variants [PlainGear], [MagicGear],
// [GearFamily], [ScrollFamily], [AmmunitionVariant], [FocusVariant],
// [ScrollVariant] and [Pack], all built on the unexported gearCommon. It carries
// no magic-item flag; the four entries
// that are magic items by their own text are recognised by id.
type AdventuringGear interface {
	EquipmentEntry
	// Weight returns the printed weight.
	Weight() WeightSpec
	// Note returns the printed remark, or "" for none.
	Note() string
	// Text returns the verbatim description, or "" when the entry has none (a
	// variant may omit it).
	Text() string
	isAdventuringGear()
}

// GearVariant is a row of a sub-table of a family: a closed set with the
// variants [AmmunitionVariant], [FocusVariant] and [ScrollVariant], all built on
// the unexported gearVariantBase. A variant
// has no variants of its own.
type GearVariant interface {
	AdventuringGear
	// VariantOf returns the family the row belongs to.
	VariantOf() ItemID
	isGearVariant()
}

// gearCommon is what every gear entry has: the identity, the weight, the
// remark and the description.
type gearCommon struct {
	equipmentHeader
	weight WeightSpec
	note   string
	text   string
}

func newGearCommon(block string, id ItemID, name string, source SourceCitation, weight WeightSpec, note, text string, textRequired bool) (gearCommon, error) {
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return gearCommon{}, err
	}
	if !validWeightSpec(weight) {
		return gearCommon{}, blockError(block, BlockInvalidMember)
	}
	if note != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkText(block, note); err != nil {
			return gearCommon{}, err
		}
	}
	switch {
	case text == "" && textRequired:
		return gearCommon{}, blockError(block, BlockMissing)
	case text != "": // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkText(block, text); err != nil {
			return gearCommon{}, err
		}
	}
	return gearCommon{equipmentHeader: h, weight: weight, note: note, text: text}, nil
}

// Weight returns the printed weight.
func (g gearCommon) Weight() WeightSpec { return g.weight }

// Note returns the printed remark, or "" for none.
func (g gearCommon) Note() string { return g.note }

// Text returns the verbatim description, or "" for none.
func (g gearCommon) Text() string { return g.text }

// EquipmentCategory returns [EquipmentAdventuringGear].
func (gearCommon) EquipmentCategory() EquipmentCategory { return EquipmentAdventuringGear }

func (gearCommon) isAdventuringGear() {}

// PlainGear is a gear row with a fixed price and a description. The zero value
// is invalid; build it with [NewPlainGear]. Comparable.
type PlainGear struct {
	gearCommon
	price FixedCost
}

var _ AdventuringGear = PlainGear{}

// NewPlainGear returns the gear id called name, with the verbatim text
// (required) and note "" for none.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid weight or price; BlockBlankText or
// BlockInvalidCharacter for a note or text that breaks the text rule;
// BlockMissing for no text. It never panics.
func NewPlainGear(id ItemID, name string, source SourceCitation, price FixedCost, weight WeightSpec, note, text string) (PlainGear, error) {
	g, err := newGearCommon("PlainGear", id, name, source, weight, note, text, true)
	if err != nil {
		return PlainGear{}, err
	}
	if !price.cost.Valid() {
		return PlainGear{}, blockError("PlainGear", BlockInvalidMember)
	}
	return PlainGear{gearCommon: g, price: price}, nil
}

// Price returns the price.
func (g PlainGear) Price() FixedCost { return g.price }

// MagicGear is a gear row that is a magic item by its own text and may carry
// crafting data. The zero value is invalid; build it with [NewMagicGear]. It
// holds a slice, so it is not comparable with ==.
type MagicGear struct {
	gearCommon
	price    FixedCost
	override CraftingOverride
}

var _ AdventuringGear = MagicGear{}

// NewMagicGear returns the gear id called name, with the verbatim text
// (required). The zero CraftingOverride means none.
//
// It returns the errors of [NewPlainGear]. It never panics.
func NewMagicGear(id ItemID, name string, source SourceCitation, price FixedCost, weight WeightSpec, note, text string, override CraftingOverride) (MagicGear, error) {
	g, err := newGearCommon("MagicGear", id, name, source, weight, note, text, true)
	if err != nil {
		return MagicGear{}, err
	}
	if !price.cost.Valid() {
		return MagicGear{}, blockError("MagicGear", BlockInvalidMember)
	}
	return MagicGear{gearCommon: g, price: price, override: CraftingOverride{
		rawMaterials: override.rawMaterials, days: override.days, toolIDs: slices.Clone(override.toolIDs),
	}}, nil
}

// Price returns the price.
func (g MagicGear) Price() FixedCost { return g.price }

// CraftingOverride returns the crafting data and true, or the zero value and
// false when the entry has none.
func (g MagicGear) CraftingOverride() (CraftingOverride, bool) {
	if g.override.days == 0 {
		return CraftingOverride{}, false
	}
	return CraftingOverride{rawMaterials: g.override.rawMaterials, days: g.override.days, toolIDs: slices.Clone(g.override.toolIDs)}, true
}

// References returns the sites of the crafting tools
// (craftingOverride/toolIds/<k>), existence only.
func (g MagicGear) References() []Reference {
	return itemReferences("craftingOverride/toolIds", g.override.toolIDs)
}

// GearFamily is a gear row printed once with a sub-table of variants, whose
// price or weight may be "Varies". The zero value is invalid; build it with
// [NewGearFamily]. It holds a slice, so it is not comparable with ==.
type GearFamily struct {
	gearCommon
	price    Price
	variants []ItemID
}

var _ AdventuringGear = GearFamily{}

// NewGearFamily returns the family id called name with at least one variant
// and the verbatim text (required).
//
// It returns the errors of [NewPlainGear], and a *[BlockError]:
// BlockInvalidMember for an invalid price or variant id, BlockEmpty for no
// variant, BlockDuplicate for a repeated variant. It never panics.
func NewGearFamily(id ItemID, name string, source SourceCitation, price Price, weight WeightSpec, note, text string, variants []ItemID) (GearFamily, error) {
	g, err := newGearCommon("GearFamily", id, name, source, weight, note, text, true)
	if err != nil {
		return GearFamily{}, err
	}
	if !validPrice(price) {
		return GearFamily{}, blockError("GearFamily", BlockInvalidMember)
	}
	if err := checkVariantList("GearFamily", variants); err != nil {
		return GearFamily{}, err
	}
	return GearFamily{gearCommon: g, price: price, variants: slices.Clone(variants)}, nil
}

func checkVariantList(block string, variants []ItemID) error {
	if len(variants) == 0 {
		return blockError(block, BlockEmpty)
	}
	return checkSet(block, variants, 1, equipmentMaxInt, ItemID.Valid)
}

// Price returns the price: a [FixedCost] or [CostVaries].
func (g GearFamily) Price() Price { return g.price }

// Variants returns the variants, as a new slice.
func (g GearFamily) Variants() []ItemID { return slices.Clone(g.variants) }

// References returns the sites of the variants (variants/<j>), existence only.
func (g GearFamily) References() []Reference { return itemReferences("variants", g.variants) }

// ScrollFamily is the Spell Scroll family, which prints no single price and no
// weight: its price is [CostVaries] and its weight [NoWeight]. The zero value
// is invalid; build it with [NewScrollFamily]. It holds a slice, so it is not
// comparable with ==.
type ScrollFamily struct {
	gearCommon
	variants []ItemID
}

var _ AdventuringGear = ScrollFamily{}

// NewScrollFamily returns the family id called name with at least one variant
// and the verbatim text (required).
//
// It returns the errors of [NewPlainGear], and a *[BlockError]:
// BlockInvalidMember for an invalid variant id, BlockEmpty for no variant,
// BlockDuplicate for a repeated variant. It never panics.
func NewScrollFamily(id ItemID, name string, source SourceCitation, note, text string, variants []ItemID) (ScrollFamily, error) {
	g, err := newGearCommon("ScrollFamily", id, name, source, NoWeight{}, note, text, true)
	if err != nil {
		return ScrollFamily{}, err
	}
	if err := checkVariantList("ScrollFamily", variants); err != nil {
		return ScrollFamily{}, err
	}
	return ScrollFamily{gearCommon: g, variants: slices.Clone(variants)}, nil
}

// Price returns [CostVaries].
func (ScrollFamily) Price() Price { return CostVaries{} }

// Variants returns the variants, as a new slice.
func (g ScrollFamily) Variants() []ItemID { return slices.Clone(g.variants) }

// References returns the sites of the variants (variants/<j>), existence only.
func (g ScrollFamily) References() []Reference { return itemReferences("variants", g.variants) }

// gearVariantBase is what the three gear variants share.
type gearVariantBase struct {
	gearCommon
	variantOf ItemID
	price     FixedCost
}

func newGearVariantBase(block string, id ItemID, name string, source SourceCitation, variantOf ItemID, price FixedCost, weight WeightSpec, note, text string) (gearVariantBase, error) {
	g, err := newGearCommon(block, id, name, source, weight, note, text, false)
	if err != nil {
		return gearVariantBase{}, err
	}
	if _, err := ParseID(string(variantOf)); err != nil {
		return gearVariantBase{}, err
	}
	if !price.cost.Valid() {
		return gearVariantBase{}, blockError(block, BlockInvalidMember)
	}
	return gearVariantBase{gearCommon: g, variantOf: variantOf, price: price}, nil
}

// VariantOf returns the family the row belongs to.
func (g gearVariantBase) VariantOf() ItemID { return g.variantOf }

// Price returns the price.
func (g gearVariantBase) Price() FixedCost { return g.price }

func (gearVariantBase) isGearVariant() {}

// References returns the site of the family (variantOf), existence only.
func (g gearVariantBase) References() []Reference {
	return []Reference{itemReference("variantOf", g.variantOf)}
}

// AmmunitionVariant is one row of the Ammunition sub-table. The zero value is
// invalid; build it with [NewAmmunitionVariant]. Comparable.
type AmmunitionVariant struct {
	gearVariantBase
	ammunition AmmunitionData
}

var _ GearVariant = AmmunitionVariant{}

// NewAmmunitionVariant returns the variant id called name of the family
// variantOf, with the ammunition data and an optional text and note.
//
// It returns the errors of [NewPlainGear] except that the text is optional,
// the *[IDError] of [ParseID] for an invalid family id, and a *[BlockError]
// (BlockInvalidMember) for invalid ammunition data. It never panics.
func NewAmmunitionVariant(id ItemID, name string, source SourceCitation, variantOf ItemID, price FixedCost, weight WeightSpec, note, text string, ammunition AmmunitionData) (AmmunitionVariant, error) {
	b, err := newGearVariantBase("AmmunitionVariant", id, name, source, variantOf, price, weight, note, text)
	if err != nil {
		return AmmunitionVariant{}, err
	}
	if _, err := NewAmmunitionData(ammunition.amount, ammunition.storage); err != nil {
		return AmmunitionVariant{}, blockError("AmmunitionVariant", BlockInvalidMember)
	}
	return AmmunitionVariant{gearVariantBase: b, ammunition: ammunition}, nil
}

// Ammunition returns the amount and storage.
func (v AmmunitionVariant) Ammunition() AmmunitionData { return v.ammunition }

// FocusVariant is one row of the Arcane Focus, Druidic Focus or Holy Symbol
// sub-table, which may also count as a weapon. The zero value is invalid; build
// it with [NewFocusVariant]. Comparable.
type FocusVariant struct {
	gearVariantBase
	alsoCountsAs ItemID
}

var _ GearVariant = FocusVariant{}

// NewFocusVariant returns the variant id called name of the family variantOf.
// alsoCountsAs is the weapon the focus counts as, "" for none.
//
// It returns the errors of [NewAmmunitionVariant] with the *[IDError] of
// [ParseID] for a non-empty invalid alsoCountsAs. It never panics.
func NewFocusVariant(id ItemID, name string, source SourceCitation, variantOf ItemID, price FixedCost, weight WeightSpec, note, text string, alsoCountsAs ItemID) (FocusVariant, error) {
	b, err := newGearVariantBase("FocusVariant", id, name, source, variantOf, price, weight, note, text)
	if err != nil {
		return FocusVariant{}, err
	}
	if alsoCountsAs != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		if _, err := ParseID(string(alsoCountsAs)); err != nil {
			return FocusVariant{}, err
		}
	}
	return FocusVariant{gearVariantBase: b, alsoCountsAs: alsoCountsAs}, nil
}

// AlsoCountsAs returns the weapon the focus counts as and true, or "" and
// false for none.
func (v FocusVariant) AlsoCountsAs() (ItemID, bool) { return v.alsoCountsAs, v.alsoCountsAs != "" }

// References returns the sites of the family (variantOf) and of the weapon
// (alsoCountsAs), existence only.
func (v FocusVariant) References() []Reference {
	refs := v.gearVariantBase.References()
	if v.alsoCountsAs != "" {
		refs = append(refs, itemReference("alsoCountsAs", v.alsoCountsAs))
	}
	return refs
}

// Spell levels an equipment type may hold: a spell level is a plain int here,
// a deliberate exception recorded in content-equipment FR-118.
const (
	equipmentMinSpellLevel = 0
	equipmentMaxSpellLevel = 9
)

func validSpellLevel(level int) bool {
	return level >= equipmentMinSpellLevel && level <= equipmentMaxSpellLevel
}

// ScrollVariant is one row of the Spell Scroll family, with its spell level
// (0 for a cantrip). Its weight is [NoWeight]. The zero value is invalid;
// build it with [NewScrollVariant]. Comparable.
type ScrollVariant struct {
	gearVariantBase
	spellLevel int
}

var _ GearVariant = ScrollVariant{}

// NewScrollVariant returns the variant id called name of the family
// variantOf, for scrolls of spellLevel 0 to 9.
//
// It returns the errors of [NewAmmunitionVariant], and a *[BlockError]:
// BlockOutOfRange for a level outside 0 to 9, BlockConstraint for a weight
// that is not [NoWeight]. It never panics.
func NewScrollVariant(id ItemID, name string, source SourceCitation, variantOf ItemID, price FixedCost, note, text string, spellLevel int) (ScrollVariant, error) {
	b, err := newGearVariantBase("ScrollVariant", id, name, source, variantOf, price, NoWeight{}, note, text)
	if err != nil {
		return ScrollVariant{}, err
	}
	if !validSpellLevel(spellLevel) {
		return ScrollVariant{}, blockError("ScrollVariant", BlockOutOfRange)
	}
	return ScrollVariant{gearVariantBase: b, spellLevel: spellLevel}, nil
}

// SpellLevel returns the spell level of the scroll, 0 for a cantrip.
func (v ScrollVariant) SpellLevel() int { return v.spellLevel }

// Pack is a gear row with listed contents (SRD 5.2.1, "Burglar's Pack" to
// "Scholar's Pack", pp. 96-99). The zero value is invalid; build it with
// [NewPack]. It holds a slice, so it is not comparable with ==.
type Pack struct {
	gearCommon
	price    FixedCost
	contents []PackItem
}

var _ AdventuringGear = Pack{}

// NewPack returns the pack id called name with at least one content line and
// the verbatim text (required).
//
// It returns the errors of [NewPlainGear], and a *[BlockError]: BlockEmpty for
// no contents, BlockInvalidMember for an invalid line. It never panics.
func NewPack(id ItemID, name string, source SourceCitation, price FixedCost, weight WeightSpec, note, text string, contents []PackItem) (Pack, error) {
	g, err := newGearCommon("Pack", id, name, source, weight, note, text, true)
	if err != nil {
		return Pack{}, err
	}
	switch {
	case len(contents) == 0:
		return Pack{}, blockError("Pack", BlockEmpty)
	case !price.cost.Valid(), slices.ContainsFunc(contents, func(p PackItem) bool { _, err := NewPackItem(p.id, p.quantity, p.unit); return err != nil }):
		return Pack{}, blockError("Pack", BlockInvalidMember)
	}
	return Pack{gearCommon: g, price: price, contents: slices.Clone(contents)}, nil
}

// Price returns the price.
func (p Pack) Price() FixedCost { return p.price }

// Contents returns the content lines in printed order, as a new slice.
func (p Pack) Contents() []PackItem { return slices.Clone(p.contents) }

// References returns the sites of the contents (contents/<j>/id), existence
// only.
func (p Pack) References() []Reference {
	out := make([]Reference, 0, len(p.contents))
	for j, c := range p.contents {
		out = append(out, itemReference("contents/"+strconv.Itoa(j)+"/id", c.id))
	}
	return out
}
