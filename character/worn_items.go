package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// WornSlot is a kind of worn item limited to one per creature. The zero value is
// WornSlotNone: an item with no slot (a ring, an amulet, a belt, a Shield) is
// not limited.
type WornSlot int

// The slots (SRD 5.2.1, "Multiple Items of the Same Kind", p. 103). Footwear,
// gloves and bracers hold one pair (two halves); the others hold one item.
const (
	WornSlotNone WornSlot = iota
	WornSlotFootwear
	WornSlotGloves
	WornSlotBracers
	WornSlotArmor
	WornSlotHeadwear
	WornSlotCloak
)

// Capacity returns how many items the slot holds, 2 for a pair slot, 1 for the
// others and 0 for WornSlotNone (unlimited).
func (s WornSlot) Capacity() int {
	switch {
	case s >= WornSlotFootwear && s <= WornSlotBracers:
		return 2
	case s >= WornSlotArmor && s <= WornSlotCloak:
		return 1
	}
	return 0
}

// String returns the snake_case name of the slot, or "none".
func (s WornSlot) String() string {
	names := []string{"none", "footwear", "gloves", "bracers", "armor", "headwear", "cloak"}
	if s < 0 || int(s) >= len(names) {
		return "unknown"
	}
	return names[s]
}

// SlotException is a Game Master's exception to the one-per-slot rule. The zero
// value is SlotExceptionNone.
type SlotException int

// The slot exceptions.
const (
	SlotExceptionNone SlotException = iota
	SlotExceptionGMAllows
)

// PairException is a Game Master's exception to the rule that paired items must
// both be worn. The zero value is PairExceptionNone.
type PairException int

// The pair exceptions.
const (
	PairExceptionNone PairException = iota
	PairExceptionGMAllows
)

// FitJudgement is the host's judgement of whether a fixed-size item fits the
// wearer. The zero value is FitNotSupplied.
type FitJudgement int

// The fit judgements.
const (
	FitNotSupplied FitJudgement = iota
	FitFits
	FitDoesNotFit
)

// WornItem is one worn item copy: its slot (WornSlotNone for none) and its pair
// (the zero PairID for none). Immutable and comparable.
type WornItem struct {
	instance MagicItemInstanceID
	slot     WornSlot
	pair     PairID
}

// NewWornItem returns the item. It returns an error matching
// srderr.ErrInvalidInput for a blank instance or an unknown slot.
func NewWornItem(instance MagicItemInstanceID, slot WornSlot, pair PairID) (WornItem, error) {
	if !instance.Valid() || slot < WornSlotNone || slot > WornSlotCloak {
		return WornItem{}, invalidInput("worn item: blank instance or unknown slot")
	}
	return WornItem{instance: instance, slot: slot, pair: pair}, nil
}

// InstanceID returns the copy.
func (w WornItem) InstanceID() MagicItemInstanceID { return w.instance }

// Slot returns the slot.
func (w WornItem) Slot() WornSlot { return w.slot }

// PairID returns the pair, or the zero PairID.
func (w WornItem) PairID() PairID { return w.pair }

// weight is how much of its slot the item fills: a pair-slot item with no pair fills the whole slot.
func (w WornItem) weight() int {
	if w.pair == "" && w.slot.Capacity() == 2 {
		return 2
	}
	return 1
}

// WornItems are the items a creature wears. Immutable: Wear returns a new value.
// The zero value is no items. Compare with Equal.
type WornItems struct{ items []WornItem }

// Items returns a new slice of the worn items in the order worn.
func (w WornItems) Items() []WornItem { return slices.Clone(w.items) }

// Equal reports whether both hold the same items in the same order.
func (w WornItems) Equal(o WornItems) bool { return slices.Equal(w.items, o.items) }

func (w WornItems) used(slot WornSlot) int {
	n := 0
	for _, it := range w.items {
		if it.slot == slot {
			n += it.weight()
		}
	}
	return n
}

// WearResult is the closed set of outcomes of WornItems.Wear. Variants:
// WornItemsWorn, WornItemsSlotOccupied, WornItemsDoesNotFit.
type WearResult interface{ isWearResult() }

// WornItemsWorn: the item is worn; Items is the new value.
type WornItemsWorn struct{ Items WornItems }

// WornItemsSlotOccupied: the slot is full.
type WornItemsSlotOccupied struct{ Slot WornSlot }

// WornItemsDoesNotFit: a fixed-size item does not fit, or no judgement was supplied.
type WornItemsDoesNotFit struct{}

func (WornItemsWorn) isWearResult()         {}
func (WornItemsSlotOccupied) isWearResult() {}
func (WornItemsDoesNotFit) isWearResult()   {}

// Wear adds item. An item that Adjusts fits any wearer; a Fixed one needs
// FitFits, else WornItemsDoesNotFit (the model never judges anatomy). An item
// with a slot is refused with WornItemsSlotOccupied at capacity unless
// SlotExceptionGMAllows; an item with no slot is never limited (SRD 5.2.1,
// "Multiple Items of the Same Kind", "Wearing and Wielding Items", p. 103). It
// returns an error matching srderr.ErrInvalidInput for a zero item or an
// unknown fit or exception.
func (w WornItems) Wear(item WornItem, fit content.ItemFit, judgement FitJudgement, exception SlotException) (WearResult, error) {
	if !item.instance.Valid() || fit != content.ItemFitAdjusts && fit != content.ItemFitFixed ||
		exception != SlotExceptionNone && exception != SlotExceptionGMAllows {
		return nil, invalidInput("wear: zero item or unknown fit or exception")
	}
	if fit == content.ItemFitFixed && judgement != FitFits {
		return WornItemsDoesNotFit{}, nil
	}
	if c := item.slot.Capacity(); c > 0 && exception != SlotExceptionGMAllows && w.used(item.slot)+item.weight() > c {
		return WornItemsSlotOccupied{Slot: item.slot}, nil
	}
	return WornItemsWorn{Items: WornItems{items: append(slices.Clone(w.items), item)}}, nil
}

// PairedItemBenefit decides whether a paired item imparts its benefits. The zero
// value is ready to use.
type PairedItemBenefit struct{}

// Of reports whether the items of pair give their benefits: when both halves of
// the pair are worn, or one with PairExceptionGMAllows (SRD 5.2.1, "Paired
// Items", p. 103; "Magic Item Rules", p. 205). Halves of different pairs worn
// together give none.
func (PairedItemBenefit) Of(pair PairID, worn WornItems, exception PairException) bool {
	n := 0
	for _, it := range worn.items {
		if it.pair == pair && pair != "" {
			n++
		}
	}
	return n >= 2 || n == 1 && exception == PairExceptionGMAllows
}
