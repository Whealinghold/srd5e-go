package content

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
)

// ToolGroup is Artisan's Tools or Other Tools (SRD 5.2.1, "Tools", pp.
// 93-94). The zero value is invalid. Comparable and safe for concurrent use.
type ToolGroup int

// The tool groups.
const (
	ToolGroupArtisans ToolGroup = iota + 1
	ToolGroupOther
)

func toolGroupNames() []string { return []string{"Artisan's Tools", "Other Tools"} }

// Valid reports whether g is a tool group.
func (g ToolGroup) Valid() bool { return enumValid(toolGroupNames(), int(g)) }

// String returns the printed name of g, or "unknown".
func (g ToolGroup) String() string { return enumString(toolGroupNames(), int(g)) }

func parseToolGroup(s string) (ToolGroup, error) {
	v, err := parseEnum("ToolGroup", toolGroupNames(), s)
	return ToolGroup(v), err
}

// ToolUtilizeAction is one Utilize action of a tool and the DC the entry
// prints for it (SRD 5.2.1, "Tools", pp. 93-94). The zero value is invalid;
// build it with [NewToolUtilizeAction]. Comparable.
type ToolUtilizeAction struct {
	description string
	dc          int
}

// NewToolUtilizeAction returns an action with the printed description and a
// DC of at least 1.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockBlankText
// or BlockInvalidCharacter for a description that breaks the text rule
// (schema FR-028), then BlockOutOfRange for a DC below 1 or above 2^31 - 1. It
// never panics.
func NewToolUtilizeAction(description string, dc int) (ToolUtilizeAction, error) {
	if err := checkText("ToolUtilizeAction", description); err != nil {
		return ToolUtilizeAction{}, err
	}
	if dc < 1 || dc > equipmentMaxInt {
		return ToolUtilizeAction{}, blockError("ToolUtilizeAction", BlockOutOfRange)
	}
	return ToolUtilizeAction{description: description, dc: dc}, nil
}

// MustToolUtilizeAction is like [NewToolUtilizeAction] but panics with its
// error when the arguments are invalid.
func MustToolUtilizeAction(description string, dc int) ToolUtilizeAction {
	a, err := NewToolUtilizeAction(description, dc)
	if err != nil {
		panic(err)
	}
	return a
}

// Description returns the printed description.
func (a ToolUtilizeAction) Description() string { return a.description }

// DC returns the Difficulty Class.
func (a ToolUtilizeAction) DC() int { return a.dc }

// CraftSetName names a set of equipment a tool can craft: all melee weapons,
// all ranged weapons, all medium armor or all heavy armor. A set expands over
// the weapons by [WeaponType] and the armor by [ArmorCategory]. The zero value
// is invalid. Comparable and safe for concurrent use.
type CraftSetName int

// The craft set names.
const (
	CraftMeleeWeapons CraftSetName = iota + 1
	CraftRangedWeapons
	CraftMediumArmor
	CraftHeavyArmor
)

func craftSetNames() []string {
	return []string{"MeleeWeapons", "RangedWeapons", "MediumArmor", "HeavyArmor"}
}

// Valid reports whether n is a craft set name.
func (n CraftSetName) Valid() bool { return enumValid(craftSetNames(), int(n)) }

// String returns the wire name of n, or "unknown".
func (n CraftSetName) String() string { return enumString(craftSetNames(), int(n)) }

func parseCraftSetName(s string) (CraftSetName, error) {
	v, err := parseEnum("CraftSetName", craftSetNames(), s)
	return CraftSetName(v), err
}

// CraftReference is one entry of a tool's Craft list: a closed set with the
// variants [CraftItem] and [CraftSet].
type CraftReference interface{ isCraftReference() }

// CraftItem is one piece of equipment a tool can craft. The zero value is
// invalid; build it with [NewCraftItem]. Comparable.
type CraftItem struct{ id ItemID }

func (CraftItem) isCraftReference() {}

// NewCraftItem returns a reference to the item id. It returns the *[IDError]
// of [ParseID], wrapping srderr.ErrInvalidInput, for an invalid id.
func NewCraftItem(id ItemID) (CraftItem, error) {
	if _, err := ParseID(string(id)); err != nil {
		return CraftItem{}, err
	}
	return CraftItem{id: id}, nil
}

// ID returns the item.
func (c CraftItem) ID() ItemID { return c.id }

// CraftSet is a set of equipment a tool can craft, less the listed
// exceptions. The zero value is invalid; build it with [NewCraftSet]. It
// holds a slice, so it is not comparable with ==.
type CraftSet struct {
	set    CraftSetName
	except []ItemID
}

func (CraftSet) isCraftReference() {}

// NewCraftSet returns the set s less the except items (nil for none).
//
// It returns an error wrapping srderr.ErrInvalidInput: a *[BlockError]
// (BlockInvalidMember) for an invalid set name, then the *[IDError] of
// [ParseID] for an invalid id in except. It never panics.
func NewCraftSet(s CraftSetName, except []ItemID) (CraftSet, error) {
	if !s.Valid() {
		return CraftSet{}, blockError("CraftSet", BlockInvalidMember)
	}
	for _, id := range except {
		if _, err := ParseID(string(id)); err != nil {
			return CraftSet{}, err
		}
	}
	return CraftSet{set: s, except: slices.Clone(except)}, nil
}

// Set returns the name of the set.
func (c CraftSet) Set() CraftSetName { return c.set }

// Except returns the items left out of the set, as a new slice.
func (c CraftSet) Except() []ItemID { return slices.Clone(c.except) }

func validCraftReference(r CraftReference) bool {
	var err error
	switch v := r.(type) {
	case CraftItem:
		_, err = NewCraftItem(v.id)
	case CraftSet:
		_, err = NewCraftSet(v.set, v.except)
	default:
		return false
	}
	return err == nil
}

func craftReferences(craft []CraftReference) []Reference {
	var out []Reference
	for j, r := range craft {
		switch v := r.(type) {
		case CraftItem:
			out = append(out, itemReference("craft/"+strconv.Itoa(j)+"/id", v.id))
		case CraftSet:
			out = append(out, itemReferences("craft/"+strconv.Itoa(j)+"/except", v.except)...)
		}
	}
	return out
}

// Tool is a row of the Tools tables (SRD 5.2.1, "Tools", pp. 93-94): a closed
// set with the variants [StandaloneTool], [ToolFamily] and [ToolVariant]. A
// variant takes its ability and Utilize actions from its family.
type Tool interface {
	EquipmentEntry
	isTool()
}

// toolBody is what a standalone tool and a tool family share.
type toolBody struct {
	group   ToolGroup
	ability ability.Ability
	utilize []ToolUtilizeAction
	craft   []CraftReference
	weight  WeightSpec
}

func newToolBody(block string, group ToolGroup, ab ability.Ability, utilize []ToolUtilizeAction, craft []CraftReference, weight WeightSpec) (toolBody, error) {
	switch {
	case !group.Valid(), !ab.Valid(), !validWeightSpec(weight),
		slices.ContainsFunc(utilize, func(a ToolUtilizeAction) bool { _, err := NewToolUtilizeAction(a.description, a.dc); return err != nil }),
		slices.ContainsFunc(craft, func(r CraftReference) bool { return !validCraftReference(r) }):
		return toolBody{}, blockError(block, BlockInvalidMember)
	case len(utilize) == 0:
		return toolBody{}, blockError(block, BlockEmpty)
	}
	return toolBody{group: group, ability: ab, utilize: slices.Clone(utilize), craft: slices.Clone(craft), weight: weight}, nil
}

// StandaloneTool is a tool that is neither a family nor a variant. It is a
// [Tool] and an [EquipmentEntry]. Immutable, with its lists copied; the zero
// value is invalid. It holds slices, so it is not comparable with ==.
type StandaloneTool struct {
	equipmentHeader
	toolBody
	price FixedCost
}

var _ Tool = StandaloneTool{}

// NewStandaloneTool returns the tool id called name, with at least one Utilize
// action and a Craft list that may be empty.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid group, ability, action, craft reference,
// weight or price; BlockEmpty for no Utilize action. It never panics.
func NewStandaloneTool(id ItemID, name string, source SourceCitation, group ToolGroup, ab ability.Ability, utilize []ToolUtilizeAction, craft []CraftReference, price FixedCost, weight WeightSpec) (StandaloneTool, error) {
	const block = "StandaloneTool"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return StandaloneTool{}, err
	}
	b, err := newToolBody(block, group, ab, utilize, craft, weight)
	if err != nil {
		return StandaloneTool{}, err
	}
	if !price.cost.Valid() {
		return StandaloneTool{}, blockError(block, BlockInvalidMember)
	}
	return StandaloneTool{equipmentHeader: h, toolBody: b, price: price}, nil
}

func (StandaloneTool) isTool() {}

// EquipmentCategory returns [EquipmentTool].
func (StandaloneTool) EquipmentCategory() EquipmentCategory { return EquipmentTool }

// Group returns Artisan's Tools or Other Tools.
func (t StandaloneTool) Group() ToolGroup { return t.group }

// Ability returns the ability the tool's checks use.
func (t StandaloneTool) Ability() ability.Ability { return t.ability }

// Utilize returns the Utilize actions, as a new slice.
func (t StandaloneTool) Utilize() []ToolUtilizeAction { return slices.Clone(t.utilize) }

// Craft returns the Craft list, as a new slice; nil when it is empty.
func (t StandaloneTool) Craft() []CraftReference { return slices.Clone(t.craft) }

// Price returns the price.
func (t StandaloneTool) Price() FixedCost { return t.price }

// Weight returns the weight.
func (t StandaloneTool) Weight() WeightSpec { return t.weight }

// References returns the sites of the Craft list: craft/<j>/id for an item and
// craft/<j>/except/<k> for each exception, existence only.
func (t StandaloneTool) References() []Reference { return craftReferences(t.craft) }

// ToolFamily is a tool printed once with a list of variants and no single
// price (Gaming Set, Musical Instrument). It is a [Tool] and an
// [EquipmentEntry]. Immutable, with its lists copied; the zero value is
// invalid. It holds slices, so it is not comparable with ==.
type ToolFamily struct {
	equipmentHeader
	toolBody
	variants []ItemID
	price    Price
}

var _ Tool = ToolFamily{}

// NewToolFamily returns the family id called name, with at least one Utilize
// action and at least one variant.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid group, ability, action, craft reference,
// weight, price or variant id; BlockEmpty for no Utilize action or no
// variant; BlockDuplicate for a repeated variant. It never panics.
func NewToolFamily(id ItemID, name string, source SourceCitation, group ToolGroup, ab ability.Ability, utilize []ToolUtilizeAction, craft []CraftReference, variants []ItemID, price Price, weight WeightSpec) (ToolFamily, error) {
	const block = "ToolFamily"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return ToolFamily{}, err
	}
	b, err := newToolBody(block, group, ab, utilize, craft, weight)
	if err != nil {
		return ToolFamily{}, err
	}
	if !validPrice(price) {
		return ToolFamily{}, blockError(block, BlockInvalidMember)
	}
	if err := checkSet(block, variants, 1, equipmentMaxInt, ItemID.Valid); err != nil {
		return ToolFamily{}, err
	}
	return ToolFamily{equipmentHeader: h, toolBody: b, variants: slices.Clone(variants), price: price}, nil
}

func (ToolFamily) isTool() {}

// EquipmentCategory returns [EquipmentTool].
func (ToolFamily) EquipmentCategory() EquipmentCategory { return EquipmentTool }

// Group returns Artisan's Tools or Other Tools.
func (t ToolFamily) Group() ToolGroup { return t.group }

// Ability returns the ability the tool's checks use.
func (t ToolFamily) Ability() ability.Ability { return t.ability }

// Utilize returns the Utilize actions, as a new slice.
func (t ToolFamily) Utilize() []ToolUtilizeAction { return slices.Clone(t.utilize) }

// Craft returns the Craft list, as a new slice; nil when it is empty.
func (t ToolFamily) Craft() []CraftReference { return slices.Clone(t.craft) }

// Variants returns the variants, as a new slice.
func (t ToolFamily) Variants() []ItemID { return slices.Clone(t.variants) }

// Price returns the price: a [FixedCost] or [CostVaries].
func (t ToolFamily) Price() Price { return t.price }

// Weight returns the weight.
func (t ToolFamily) Weight() WeightSpec { return t.weight }

// References returns the sites of the variants (variants/<j>) and of the Craft
// list, existence only.
func (t ToolFamily) References() []Reference {
	return append(itemReferences("variants", t.variants), craftReferences(t.craft)...)
}

// ToolVariant is one variant of a [ToolFamily]. It has a price and a weight
// and nothing else: its ability and Utilize actions are its family's. It is a
// [Tool] and an [EquipmentEntry]. Immutable; the zero value is invalid.
// Comparable.
type ToolVariant struct {
	equipmentHeader
	variantOf ItemID
	price     FixedCost
	weight    WeightSpec
}

var _ Tool = ToolVariant{}

// NewToolVariant returns the variant id called name of the family variantOf.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id or family id, then a *[BlockError]:
// BlockInvalidName; BlockInvalidMember for an invalid price or weight. It
// never panics.
func NewToolVariant(id ItemID, name string, source SourceCitation, variantOf ItemID, price FixedCost, weight WeightSpec) (ToolVariant, error) {
	const block = "ToolVariant"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return ToolVariant{}, err
	}
	if _, err := ParseID(string(variantOf)); err != nil {
		return ToolVariant{}, err
	}
	if !price.cost.Valid() || !validWeightSpec(weight) {
		return ToolVariant{}, blockError(block, BlockInvalidMember)
	}
	return ToolVariant{equipmentHeader: h, variantOf: variantOf, price: price, weight: weight}, nil
}

func (ToolVariant) isTool() {}

// EquipmentCategory returns [EquipmentTool].
func (ToolVariant) EquipmentCategory() EquipmentCategory { return EquipmentTool }

// VariantOf returns the family this tool is a variant of.
func (t ToolVariant) VariantOf() ItemID { return t.variantOf }

// Price returns the price.
func (t ToolVariant) Price() FixedCost { return t.price }

// Weight returns the weight.
func (t ToolVariant) Weight() WeightSpec { return t.weight }

// References returns the site of the family (variantOf), existence only.
func (t ToolVariant) References() []Reference {
	return []Reference{itemReference("variantOf", t.variantOf)}
}
