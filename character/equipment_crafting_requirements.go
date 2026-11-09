package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// CraftingBasis says which rule produced a [CraftingRequirements]. The zero
// value is invalid.
type CraftingBasis int

// The crafting bases.
const (
	// CraftingGenericRule: half the price and a day per 10 GP (SRD 5.2.1, p. 103).
	CraftingGenericRule CraftingBasis = iota + 1
	// CraftingItemOverride: the item states its own requirements.
	CraftingItemOverride
	// CraftingScrollRow: the Spell Scroll Costs table.
	CraftingScrollRow
)

// String returns "genericRule", "itemOverride", "scrollRow" or "unknown".
func (b CraftingBasis) String() string {
	switch b {
	case CraftingGenericRule:
		return "genericRule"
	case CraftingItemOverride:
		return "itemOverride"
	case CraftingScrollRow:
		return "scrollRow"
	}
	return "unknown"
}

// NotCraftableReason says why an entry has no crafting requirements. The zero
// value is invalid.
type NotCraftableReason int

// The reasons an entry cannot be crafted by the rule.
const (
	// NotCraftablePriceVaries: the price is printed "Varies".
	NotCraftablePriceVaries NotCraftableReason = iota + 1
	// NotCraftableRatedPrice: the price has a rate, such as per day.
	NotCraftableRatedPrice
	// NotCraftableFreeOfCharge: the price is 0.
	NotCraftableFreeOfCharge
	// NotCraftableNotAnItem: a service, expense or table row, not goods.
	NotCraftableNotAnItem
	// NotCraftableNoCraftingRule: the SRD has no rule for making it (a vehicle).
	NotCraftableNoCraftingRule
	// NotCraftableScribingTableMissing: a scroll, but no Spell Scroll Costs table.
	NotCraftableScribingTableMissing
)

// String returns the lower camel case name of r, or "unknown".
func (r NotCraftableReason) String() string {
	switch r {
	case NotCraftablePriceVaries:
		return "priceVaries"
	case NotCraftableRatedPrice:
		return "ratedPrice"
	case NotCraftableFreeOfCharge:
		return "freeOfCharge"
	case NotCraftableNotAnItem:
		return "notAnItem"
	case NotCraftableNoCraftingRule:
		return "noCraftingRule"
	case NotCraftableScribingTableMissing:
		return "scribingTableMissing"
	}
	return "unknown"
}

// CraftingOutcome is the closed set of results of [Crafting.RequirementsFor]:
// [CraftingRequirements] and [NotCraftable].
//
//sumtype:decl
type CraftingOutcome interface{ isCraftingOutcome() }

// CraftingRequirements are the Tools, Raw Materials and Time to craft an item.
// The tools are alternatives: any one suffices. It is an immutable value; it
// holds a slice, so it is not comparable with ==.
type CraftingRequirements struct {
	rawMaterials content.Cost
	days         int
	tools        []content.ItemID
	basis        CraftingBasis
}

// NotCraftable is the outcome for an entry with no crafting requirements.
type NotCraftable struct {
	Reason NotCraftableReason
}

func (CraftingRequirements) isCraftingOutcome() {}
func (NotCraftable) isCraftingOutcome()         {}

// RawMaterials returns the cost of the Raw Materials.
func (r CraftingRequirements) RawMaterials() content.Cost { return r.rawMaterials }

// Days returns the days of work, 8 hours a day, which need not be consecutive.
func (r CraftingRequirements) Days() int { return r.days }

// Tools returns a fresh slice of the alternative tool ids; empty when the GM
// assigns the tools.
func (r CraftingRequirements) Tools() []content.ItemID { return slices.Clone(r.tools) }

// Basis returns the rule that produced the requirements.
func (r CraftingRequirements) Basis() CraftingBasis { return r.basis }

// RequirementsFor returns the crafting requirements of item (SRD 5.2.1,
// "Crafting Nonmagical Items", p. 103), or a [NotCraftable] with the reason.
// The first case that applies wins: a service, expense or table row is
// not an item and a vehicle has no crafting rule; a magic gear's own override;
// a scroll's row in scribing (no table gives scribingTableMissing); a price
// that varies, has a rate or is 0; then the generic rule. A tool variant,
// focus and ammunition variant, and plain-priced Tack row follow the generic
// rule from their own price. The zero scribing means "no table".
//
// It returns an error matching srderr.ErrInvalidInput for a nil item or a
// figure above [content.MaxCostAmount].
func (c Crafting) RequirementsFor(item content.EquipmentEntry, tools []content.Tool, scribing ScrollScribing) (CraftingOutcome, error) {
	switch v := item.(type) {
	case nil:
		return nil, invalidInput("nil item")
	case content.Mount, content.LifestyleExpense, content.FoodDrinkOrLodging, content.Hireling,
		content.SpellcastingService, content.SpellScrollCost:
		return NotCraftable{Reason: NotCraftableNotAnItem}, nil
	case content.AirborneOrWaterborneVehicle:
		return NotCraftable{Reason: NotCraftableNoCraftingRule}, nil
	case content.MagicGear:
		if o, ok := v.CraftingOverride(); ok {
			return CraftingRequirements{rawMaterials: o.RawMaterials(), days: o.Days(), tools: o.ToolIDs(), basis: CraftingItemOverride}, nil
		}
	case content.ScrollVariant:
		if scribing.IsZero() {
			return NotCraftable{Reason: NotCraftableScribingTableMissing}, nil
		}
		row := scribing.ForSpellLevel(content.SpellLevel(v.SpellLevel()))
		return CraftingRequirements{rawMaterials: row.Price().Cost(), days: row.Days(), basis: CraftingScrollRow}, nil
	}
	price := priceOf(item)
	if price == nil {
		return nil, invalidInput("entry type")
	}
	fixed, isFixed := price.(content.FixedCost)
	if !isFixed {
		return NotCraftable{Reason: NotCraftablePriceVaries}, nil
	}
	cost := fixed.Cost()
	if _, rated := cost.Per(); rated {
		return NotCraftable{Reason: NotCraftableRatedPrice}, nil
	}
	if cost.Amount() == 0 {
		return NotCraftable{Reason: NotCraftableFreeOfCharge}, nil
	}
	raw, err := c.RawMaterials(cost)
	if err != nil {
		return nil, err
	}
	days, err := c.Days(cost)
	if err != nil {
		return nil, err
	}
	return CraftingRequirements{rawMaterials: raw, days: days, tools: c.ToolsFor(item, tools), basis: CraftingGenericRule}, nil
}

// priceOf returns the printed price of an entry that has one, or nil for an
// entry type with none.
func priceOf(item content.EquipmentEntry) content.Price {
	switch v := item.(type) {
	case content.Weapon:
		return v.Price()
	case content.Armor:
		return v.Price()
	case content.StandaloneTool:
		return v.Price()
	case content.ToolFamily:
		return v.Price()
	case content.ToolVariant:
		return v.Price()
	case content.PlainGear:
		return v.Price()
	case content.MagicGear:
		return v.Price()
	case content.GearFamily:
		return v.Price()
	case content.ScrollFamily:
		return v.Price()
	case content.AmmunitionVariant:
		return v.Price()
	case content.FocusVariant:
		return v.Price()
	case content.Pack:
		return v.Price()
	case content.TackHarnessOrDrawnVehicle:
		return v.Price()
	}
	return nil
}
