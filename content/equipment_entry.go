package content

import "strconv"

// EquipmentCategory is one of the twelve tables of the Equipment chapter
// (SRD 5.2.1, "Equipment", pp. 89-103), in table order. The zero value is
// invalid. Comparable and safe for concurrent use.
type EquipmentCategory int

// The equipment categories.
const (
	EquipmentWeapon EquipmentCategory = iota + 1
	EquipmentArmor
	EquipmentTool
	EquipmentAdventuringGear
	EquipmentMount
	EquipmentTackHarnessOrDrawnVehicle
	EquipmentAirborneOrWaterborneVehicle
	EquipmentLifestyleExpense
	EquipmentFoodDrinkOrLodging
	EquipmentHireling
	EquipmentSpellcastingService
	EquipmentSpellScrollCost
)

func equipmentCategoryNames() []string {
	return []string{
		"weapon", "armor", "tool", "adventuringGear", "mount", "tackHarnessOrDrawnVehicle",
		"airborneOrWaterborneVehicle", "lifestyleExpense", "foodDrinkOrLodging", "hireling",
		"spellcastingService", "spellScrollCost",
	}
}

// Valid reports whether c is one of the twelve categories.
func (c EquipmentCategory) Valid() bool { return enumValid(equipmentCategoryNames(), int(c)) }

// String returns the lower camel case name of c, or "unknown".
func (c EquipmentCategory) String() string { return enumString(equipmentCategoryNames(), int(c)) }

// EquipmentEntry is an [Entry] of kind [KindEquipment]: one row of an
// Equipment table. It is a closed set whose variants are [Weapon], [Armor],
// [StandaloneTool], [ToolFamily], [ToolVariant], [PlainGear], [MagicGear],
// [GearFamily], [ScrollFamily], [AmmunitionVariant], [FocusVariant],
// [ScrollVariant], [Pack], [Mount], [TackHarnessOrDrawnVehicle],
// [AirborneOrWaterborneVehicle], [LifestyleExpense], [FoodDrinkOrLodging],
// [Hireling], [SpellcastingService] and [SpellScrollCost]; [Tool] and
// [AdventuringGear] are closed subsets of it. Every variant is immutable.
type EquipmentEntry interface {
	Entry
	// ID returns the entry id.
	ID() ItemID
	// Name returns the entry name as printed.
	Name() string
	// Source returns the citation, or the zero [SourceCitation] when the entry
	// cites none.
	Source() SourceCitation
	// EquipmentCategory returns the table the entry belongs to.
	EquipmentCategory() EquipmentCategory
	isEquipmentEntry()
}

// equipmentHeader is the identity every equipment entry starts with, embedded
// in each entry type.
type equipmentHeader struct {
	id     ID
	name   string
	source SourceCitation
}

// newEquipmentHeader validates the identity of an entry of the named block:
// the id (an *IDError), then the name (BlockInvalidName).
func newEquipmentHeader(block string, id ItemID, name string, source SourceCitation) (equipmentHeader, error) {
	pid, err := ParseID(string(id))
	if err != nil {
		return equipmentHeader{}, err
	}
	if err := checkName(block, name); err != nil {
		return equipmentHeader{}, err
	}
	return equipmentHeader{id: pid, name: name, source: source}, nil
}

// Key returns the key of the entry: kind [KindEquipment] and its id.
func (h equipmentHeader) Key() Key { return NewKey(KindEquipment, h.id) }

// ID returns the entry id.
func (h equipmentHeader) ID() ItemID { return ItemID(h.id.String()) }

// Name returns the entry name as printed.
func (h equipmentHeader) Name() string { return h.name }

// Source returns the citation of the entry, or the zero [SourceCitation].
func (h equipmentHeader) Source() SourceCitation { return h.source }

func (equipmentHeader) isEquipmentEntry() {}

// itemReference returns the existence-only reference at field to the entry id
// (content-equipment FR-107; plan item P4: the category of the target is not
// checked).
func itemReference(field string, id ItemID) Reference {
	return Reference{field: field, target: id.Key()}
}

// itemReferences lists the references at prefix/<j> to each of ids.
func itemReferences(prefix string, ids []ItemID) []Reference {
	out := make([]Reference, 0, len(ids))
	for j, id := range ids {
		out = append(out, itemReference(prefix+"/"+strconv.Itoa(j), id))
	}
	return out
}
