package content

import "slices"

// BoolFactOverride is a per-item departure from a boolean category default
// (consumable, Arcane Focus, nonmagical Quarterstaff, must be worn), or the
// standard fact: use the category default. No nilable field stands for a state.
// The zero value is [StandardBoolFact]. Comparable and safe for concurrent use.
type BoolFactOverride struct{ set, value bool }

// StandardBoolFact returns the standard fact: the category default applies.
func StandardBoolFact() BoolFactOverride { return BoolFactOverride{} }

// OverriddenBoolFact returns a departure from the default, to v.
func OverriddenBoolFact(v bool) BoolFactOverride { return BoolFactOverride{set: true, value: v} }

// Value returns the overriding value and true, or false and false for the
// standard fact.
func (o BoolFactOverride) Value() (bool, bool) { return o.value, o.set }

// ActivationTimeOverride is a per-item departure from the activation time of
// its category, or the standard fact. The zero value is
// [StandardActivationTime]. Comparable and safe for concurrent use.
type ActivationTimeOverride struct {
	set   bool
	value ActivationTime
}

// StandardActivationTime returns the standard fact: the category default
// applies.
func StandardActivationTime() ActivationTimeOverride { return ActivationTimeOverride{} }

// OverriddenActivationTime returns a departure from the default, to v.
// [NewMagicItemFacts] rejects a v that is not a member.
func OverriddenActivationTime(v ActivationTime) ActivationTimeOverride {
	return ActivationTimeOverride{set: true, value: v}
}

// Value returns the overriding time and true, or 0 and false for the standard
// fact.
func (o ActivationTimeOverride) Value() (ActivationTime, bool) { return o.value, o.set }

// ResilienceOverride is a per-item departure from the default resilience, or
// the standard fact. The zero value is [StandardResilience]. Comparable and safe
// for concurrent use.
type ResilienceOverride struct {
	set   bool
	value MagicItemResilience
}

// StandardResilience returns the standard fact: the default resilience applies.
func StandardResilience() ResilienceOverride { return ResilienceOverride{} }

// OverriddenResilience returns a departure from the default, to v.
// [NewMagicItemFacts] rejects a v that is not a member.
func OverriddenResilience(v MagicItemResilience) ResilienceOverride {
	return ResilienceOverride{set: true, value: v}
}

// Value returns the overriding resilience and true, or 0 and false for the
// standard fact.
func (o ResilienceOverride) Value() (MagicItemResilience, bool) { return o.value, o.set }

// CurseStatus says whether an item is cursed (SRD 5.2.1, "Cursed Items", p.
// 206), a closed set of two members. The zero CurseStatus is invalid.
// Comparable and safe for concurrent use.
type CurseStatus int

// The curse statuses.
const (
	NotCursed CurseStatus = iota + 1
	Cursed
)

// Valid reports whether s is [NotCursed] or [Cursed].
func (s CurseStatus) Valid() bool { return s == NotCursed || s == Cursed }

// String returns "notCursed" or "cursed", or "unknown" for an undefined value.
func (s CurseStatus) String() string { return enumString([]string{"notCursed", "cursed"}, int(s)) }

// ItemFit says whether an item made for a specific creature adjusts to its
// wearer (SRD 5.2.1, "Magic Item Rules", Items Made for Specific Creatures, p.
// 205; "Armor", p. 204), a closed set of two members. The zero ItemFit is
// invalid. Comparable and safe for concurrent use.
type ItemFit int

// The item fits.
const (
	ItemFitAdjusts ItemFit = iota + 1
	ItemFitFixed
)

// Valid reports whether f is [ItemFitAdjusts] or [ItemFitFixed].
func (f ItemFit) Valid() bool { return f == ItemFitAdjusts || f == ItemFitFixed }

// String returns "adjusts" or "fixed", or "unknown" for an undefined value.
func (f ItemFit) String() string { return enumString([]string{"adjusts", "fixed"}, int(f)) }

// BaseItemSpec says which nonmagical items a magic item is based on, a closed
// family with three variants: [SpecificBaseItems], [UnspecifiedBaseItem] and
// [NoBaseItem]. Consumers switch over it exhaustively. Every variant is
// immutable and safe for concurrent use.
type BaseItemSpec interface{ isBaseItemSpec() }

// SpecificBaseItems is a magic item based on one of the named equipment items.
// The zero value is invalid; build it with [NewSpecificBaseItems]. It holds a
// slice, so compare it with [SpecificBaseItems.Equal].
type SpecificBaseItems struct{ items []ItemID }

// NewSpecificBaseItems returns the base items, a copy of items: a non-empty set
// of valid ids.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an empty set, BlockInvalidMember for an invalid id and
// BlockDuplicate for a repeat. It never panics.
func NewSpecificBaseItems(items []ItemID) (SpecificBaseItems, error) {
	if err := checkSet("SpecificBaseItems", items, 1, noUpperBound, ItemID.Valid); err != nil {
		return SpecificBaseItems{}, err
	}
	return SpecificBaseItems{items: slices.Clone(items)}, nil
}

// Items returns the base item ids as a new slice.
func (s SpecificBaseItems) Items() []ItemID { return slices.Clone(s.items) }

// Equal reports whether s and t name the same items in the same order.
func (s SpecificBaseItems) Equal(t SpecificBaseItems) bool { return slices.Equal(s.items, t.items) }

func (SpecificBaseItems) isBaseItemSpec() {}

// UnspecifiedBaseItem is a magic item based on an item its description leaves
// open. The zero value is valid.
type UnspecifiedBaseItem struct{}

func (UnspecifiedBaseItem) isBaseItemSpec() {}

// NoBaseItem is a magic item based on no nonmagical item. The zero value is
// valid.
type NoBaseItem struct{}

func (NoBaseItem) isBaseItemSpec() {}

// IncorporatedItemSpec says whether a magic item incorporates another item, a
// closed family with two variants: [NoIncorporatedItem] and
// [IncorporatedItem]. Consumers switch over it exhaustively. Every variant is
// an immutable, comparable value and safe for concurrent use.
type IncorporatedItemSpec interface{ isIncorporatedItemSpec() }

// NoIncorporatedItem is a magic item that incorporates nothing. The zero value
// is valid.
type NoIncorporatedItem struct{}

func (NoIncorporatedItem) isIncorporatedItemSpec() {}

// IncorporatedItem is a magic item that incorporates an equipment item at a
// cost in Copper Pieces. The cost is an integer because the equipment Cost
// type belongs to the equipment data, which this package may not import. The
// zero value is invalid; build it with [NewIncorporatedItem].
type IncorporatedItem struct {
	item       ItemID
	costCopper int64
}

// NewIncorporatedItem returns an incorporated item with the cost in Copper
// Pieces.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid id and BlockOutOfRange for a negative
// cost. It never panics.
func NewIncorporatedItem(item ItemID, costCopperPieces int64) (IncorporatedItem, error) {
	if !item.Valid() {
		return IncorporatedItem{}, blockError("IncorporatedItem", BlockInvalidMember)
	}
	if costCopperPieces < 0 {
		return IncorporatedItem{}, blockError("IncorporatedItem", BlockOutOfRange)
	}
	return IncorporatedItem{item: item, costCopper: costCopperPieces}, nil
}

// Item returns the id of the incorporated equipment item.
func (i IncorporatedItem) Item() ItemID { return i.item }

// CostCopperPieces returns the cost in Copper Pieces.
func (i IncorporatedItem) CostCopperPieces() int64 { return i.costCopper }

func (IncorporatedItem) isIncorporatedItemSpec() {}

// MagicItemCraftingBasis says how a magic item is crafted, a closed family
// with two variants: [TableCrafting] (the Magic Item Crafting Time and Cost
// table) and [EquipmentRulesCrafting] (the equipment crafting rules of an
// item). Consumers switch over it exhaustively. Every variant is an immutable,
// comparable value and safe for concurrent use.
type MagicItemCraftingBasis interface{ isMagicItemCraftingBasis() }

// TableCrafting is crafting by the rarity table. The zero value is valid.
type TableCrafting struct{}

func (TableCrafting) isMagicItemCraftingBasis() {}

// EquipmentRulesCrafting is crafting by the equipment rules of an item. The
// zero value is invalid; build it with [NewEquipmentRulesCrafting].
type EquipmentRulesCrafting struct{ item ItemID }

// NewEquipmentRulesCrafting returns crafting by the rules of item.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid id. It never panics.
func NewEquipmentRulesCrafting(item ItemID) (EquipmentRulesCrafting, error) {
	if !item.Valid() {
		return EquipmentRulesCrafting{}, blockError("EquipmentRulesCrafting", BlockInvalidMember)
	}
	return EquipmentRulesCrafting{item: item}, nil
}

// Item returns the id of the equipment item whose rules apply.
func (c EquipmentRulesCrafting) Item() ItemID { return c.item }

func (EquipmentRulesCrafting) isMagicItemCraftingBasis() {}

// MagicItemFactsFields holds the members of a [MagicItemFacts] for
// [NewMagicItemFacts]. Construct it with a keyed literal. Every member's zero
// value (nil for the interfaces) means its default: the standard fact,
// [NotCursed], [NoBaseItem], [NoIncorporatedItem], [TableCrafting],
// [DoesNotRecharge], [NoChargeRecovery] and [ItemFitAdjusts].
type MagicItemFactsFields struct {
	Consumable             BoolFactOverride
	ArcaneFocus            BoolFactOverride
	NonmagicalQuarterstaff BoolFactOverride
	MustBeWorn             BoolFactOverride
	ActivationTime         ActivationTimeOverride
	Curse                  CurseStatus
	Resilience             ResilienceOverride
	// AllowedSpells is a set of spell ids; nil for none.
	AllowedSpells    []SpellID
	BaseItem         BaseItemSpec
	IncorporatedItem IncorporatedItemSpec
	CraftingBasis    MagicItemCraftingBasis
	RechargeTime     ItemRechargeTime
	ChargeRecovery   ChargeRecovery
	Fit              ItemFit
}

// MagicItemFacts are the structured facts of one magic item that the SRD
// prints as prose: the departures from its category defaults, curse, allowed
// spells, base item, incorporated item, crafting basis, recharge and fit. The
// host supplies them as a value; the package never infers a fact from the
// description, and the entry keeps charges and effects as verbatim text.
// Immutable, with its collections copied on construction and on read; the zero
// MagicItemFacts is invalid. It holds slices, so it is not comparable with ==;
// compare with [MagicItemFacts.Equal]. Safe for concurrent use.
type MagicItemFacts struct {
	id MagicItemID
	f  MagicItemFactsFields
}

// NewMagicItemFacts returns the facts of the magic item id with the members of
// f, a zero member taking its default.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] with BlockInvalidMember for
// the first member that is not a valid value of its type, in the order of the
// fields, and for the allowed spells BlockInvalidMember or BlockDuplicate. It
// never panics.
func NewMagicItemFacts(id MagicItemID, f MagicItemFactsFields) (MagicItemFacts, error) {
	const block = "MagicItemFacts"
	if _, err := ParseID(string(id)); err != nil {
		return MagicItemFacts{}, err
	}
	if f.Curse == 0 {
		f.Curse = NotCursed
	}
	if f.BaseItem == nil {
		f.BaseItem = NoBaseItem{}
	}
	if f.IncorporatedItem == nil {
		f.IncorporatedItem = NoIncorporatedItem{}
	}
	if f.CraftingBasis == nil {
		f.CraftingBasis = TableCrafting{}
	}
	if f.RechargeTime == 0 {
		f.RechargeTime = DoesNotRecharge
	}
	if f.ChargeRecovery == nil {
		f.ChargeRecovery = NoChargeRecovery{}
	}
	if f.Fit == 0 {
		f.Fit = ItemFitAdjusts
	}
	if v, set := f.ActivationTime.Value(); set && !v.Valid() || !f.Curse.Valid() {
		return MagicItemFacts{}, blockError(block, BlockInvalidMember)
	}
	if v, set := f.Resilience.Value(); set && !v.Valid() || !f.RechargeTime.Valid() || !f.Fit.Valid() {
		return MagicItemFacts{}, blockError(block, BlockInvalidMember)
	}
	if err := checkSet(block, f.AllowedSpells, 0, noUpperBound, SpellID.Valid); err != nil {
		return MagicItemFacts{}, err
	}
	if !validBaseItem(f.BaseItem) || !validIncorporated(f.IncorporatedItem) ||
		!validCraftingBasis(f.CraftingBasis) || !validChargeRecovery(f.ChargeRecovery) {
		return MagicItemFacts{}, blockError(block, BlockInvalidMember)
	}
	f.AllowedSpells = cloneOrNil(f.AllowedSpells)
	return MagicItemFacts{id: id, f: f}, nil
}

// MustMagicItemFacts is like [NewMagicItemFacts] but panics with its error when
// the arguments are invalid.
func MustMagicItemFacts(id MagicItemID, f MagicItemFactsFields) MagicItemFacts {
	v, err := NewMagicItemFacts(id, f)
	if err != nil {
		panic(err)
	}
	return v
}

// MagicItemFactsForEntry returns the default facts of entry: every override the
// standard fact, [NotCursed], no allowed spells, [NoBaseItem],
// [NoIncorporatedItem], [TableCrafting], [DoesNotRecharge], [NoChargeRecovery]
// and [ItemFitAdjusts]. The category defaults themselves are
// [MagicItemCategoryFacts]. It never panics.
func MagicItemFactsForEntry(entry MagicItemEntry) MagicItemFacts {
	f, _ := NewMagicItemFacts(entry.ID(), MagicItemFactsFields{}) // an entry's id is valid; the zero entry gives the zero facts
	return f
}

func validBaseItem(b BaseItemSpec) bool {
	switch v := b.(type) {
	case UnspecifiedBaseItem, NoBaseItem:
		return true
	case SpecificBaseItems:
		_, err := NewSpecificBaseItems(v.items)
		return err == nil
	}
	return false
}

func validIncorporated(i IncorporatedItemSpec) bool {
	switch v := i.(type) {
	case NoIncorporatedItem:
		return true
	case IncorporatedItem:
		_, err := NewIncorporatedItem(v.item, v.costCopper)
		return err == nil
	}
	return false
}

func validCraftingBasis(c MagicItemCraftingBasis) bool {
	switch v := c.(type) {
	case TableCrafting:
		return true
	case EquipmentRulesCrafting:
		_, err := NewEquipmentRulesCrafting(v.item)
		return err == nil
	}
	return false
}

// ID returns the id of the magic item the facts describe.
func (m MagicItemFacts) ID() MagicItemID { return m.id }

// Consumable returns the override of the category's consumable default.
func (m MagicItemFacts) Consumable() BoolFactOverride { return m.f.Consumable }

// ArcaneFocus returns the override of the category's Arcane Focus default.
func (m MagicItemFacts) ArcaneFocus() BoolFactOverride { return m.f.ArcaneFocus }

// NonmagicalQuarterstaff returns the override of the category's nonmagical
// Quarterstaff default.
func (m MagicItemFacts) NonmagicalQuarterstaff() BoolFactOverride { return m.f.NonmagicalQuarterstaff }

// MustBeWorn returns the override of the category's must-be-worn default.
func (m MagicItemFacts) MustBeWorn() BoolFactOverride { return m.f.MustBeWorn }

// ActivationTime returns the override of the category's activation time.
func (m MagicItemFacts) ActivationTime() ActivationTimeOverride { return m.f.ActivationTime }

// Curse returns whether the item is cursed.
func (m MagicItemFacts) Curse() CurseStatus { return m.f.Curse }

// Resilience returns the override of the default resilience.
func (m MagicItemFacts) Resilience() ResilienceOverride { return m.f.Resilience }

// AllowedSpells returns the spell ids the item allows, as a new slice, nil for
// none.
func (m MagicItemFacts) AllowedSpells() []SpellID { return cloneOrNil(m.f.AllowedSpells) }

// BaseItem returns the base-item rule.
func (m MagicItemFacts) BaseItem() BaseItemSpec { return m.f.BaseItem }

// IncorporatedItem returns the incorporated-item rule.
func (m MagicItemFacts) IncorporatedItem() IncorporatedItemSpec { return m.f.IncorporatedItem }

// CraftingBasis returns how the item is crafted.
func (m MagicItemFacts) CraftingBasis() MagicItemCraftingBasis { return m.f.CraftingBasis }

// RechargeTime returns when the item recharges.
func (m MagicItemFacts) RechargeTime() ItemRechargeTime { return m.f.RechargeTime }

// ChargeRecovery returns how many charges the item regains.
func (m MagicItemFacts) ChargeRecovery() ChargeRecovery { return m.f.ChargeRecovery }

// Fit returns whether the item adjusts to its wearer.
func (m MagicItemFacts) Fit() ItemFit { return m.f.Fit }

// Equal reports whether m and o are the same facts: every member equal.
func (m MagicItemFacts) Equal(o MagicItemFacts) bool {
	a, b := m.f, o.f
	return m.id == o.id && a.Consumable == b.Consumable && a.ArcaneFocus == b.ArcaneFocus &&
		a.NonmagicalQuarterstaff == b.NonmagicalQuarterstaff && a.MustBeWorn == b.MustBeWorn &&
		a.ActivationTime == b.ActivationTime && a.Curse == b.Curse && a.Resilience == b.Resilience &&
		slices.Equal(a.AllowedSpells, b.AllowedSpells) && baseItemEqual(a.BaseItem, b.BaseItem) &&
		a.IncorporatedItem == b.IncorporatedItem && a.CraftingBasis == b.CraftingBasis &&
		a.RechargeTime == b.RechargeTime && a.ChargeRecovery == b.ChargeRecovery && a.Fit == b.Fit
}

func baseItemEqual(a, b BaseItemSpec) bool {
	if x, ok := a.(SpecificBaseItems); ok {
		y, ok := b.(SpecificBaseItems)
		return ok && x.Equal(y)
	}
	if _, ok := b.(SpecificBaseItems); ok {
		return false
	}
	return a == b
}
