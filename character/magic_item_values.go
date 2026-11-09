package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// IncorporatedItemCost builds the incorporated item of a magic item from a cost.
// The zero value is ready to use.
type IncorporatedItemCost struct{}

// FromCost returns the incorporated item whose cost is cost in Copper Pieces. It
// returns an error matching srderr.ErrInvalidInput for a rated cost (per Day or
// per Mile), an invalid cost or an invalid item id.
func (IncorporatedItemCost) FromCost(item content.ItemID, cost content.Cost) (content.IncorporatedItem, error) {
	if _, rated := cost.Per(); rated || !cost.Valid() {
		return content.IncorporatedItem{}, invalidInput("incorporated item: rated or invalid cost")
	}
	return content.NewIncorporatedItem(item, cost.ValueInCopper())
}

// MagicItemValues computes what a magic item is worth. The zero value is ready to use.
type MagicItemValues struct{}

// Of returns the value of an item of rarity: Artifact is always priceless; any
// other is the table value, halved for a consumable and then increased by the
// incorporated item's cost (halving covers the table value only, a design
// choice, SRD 5.2.1, "Magic Item Rarities and Values", p. 206). It returns an
// error matching srderr.ErrInvalidInput for an unknown rarity, a negative cost
// or a sum that overflows.
func (MagicItemValues) Of(rarity content.ItemRarity, consumable bool, incorporatedCostCopper int64) (content.MagicItemValue, error) {
	if incorporatedCostCopper < 0 {
		return nil, invalidInput("magic item value: negative incorporated cost")
	}
	if rarity == content.RarityArtifact {
		return content.PricelessMagicItemValue{}, nil
	}
	priced, ok := content.ItemRarityFacts{}.Of(rarity).Value().(content.PricedMagicItemValue)
	if !ok {
		return nil, invalidInput("magic item value: unknown rarity")
	}
	base := priced.CopperPieces()
	if consumable {
		base /= 2
	}
	if incorporatedCostCopper > content.MaxMagicItemCopper-base {
		return nil, invalidInput("magic item value: sum overflows")
	}
	return content.NewPricedMagicItemValue(base + incorporatedCostCopper)
}

// OfSpellScroll returns twice the cost of scribing the scroll, never halved
// (SRD 5.2.1, "Magic Item Rarities and Values", p. 206). It returns an error
// matching srderr.ErrInvalidInput for the zero row or a rated cost.
func (MagicItemValues) OfSpellScroll(scribing content.SpellScrollCost) (content.MagicItemValue, error) {
	cost := scribing.Price().Cost()
	if _, rated := cost.Per(); rated || !cost.Valid() {
		return nil, invalidInput("spell scroll value: zero row or rated cost")
	}
	return content.NewPricedMagicItemValue(2 * cost.ValueInCopper())
}
