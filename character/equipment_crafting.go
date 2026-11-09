package character

import "github.com/Whealinghold/srd5e-go/content"

// Crafting is the stateless service for Crafting Nonmagical Items (SRD 5.2.1,
// p. 103). The zero value is ready to use.
type Crafting struct{}

// copperPerDay is the Copper Pieces in 10 GP, the cost that takes one day
// to craft: 10 GP is 1,000 CP.
const copperPerDay = 1000

// RawMaterials returns half of cost in Copper Pieces, rounded down, as a Copper
// Piece cost: 1,500 GP gives 750 GP (75,000 CP), 1 CP gives 0 CP, 5 CP gives 2
// CP (SRD 5.2.1, "Raw Materials", p. 103). The SRD names no denomination;
// rounding in Copper Pieces, the smallest coin, is package design. A cost of 0
// gives 0 CP.
//
// It returns an error matching srderr.ErrInvalidInput for a cost with a rate
// (per Day or Mile) or an invalid cost.
func (Crafting) RawMaterials(cost content.Cost) (content.Cost, error) {
	if _, rated := cost.Per(); rated || !cost.Valid() {
		return content.Cost{}, invalidInput("crafting cost")
	}
	half := cost.ValueInCopper() / 2
	if half > content.MaxCostAmount {
		return content.Cost{}, invalidInput("raw materials out of range")
	}
	return content.NewCost(int(half), content.CoinCP, 0)
}

// Days returns the days to craft an item of cost: the cost in Gold Pieces
// divided by 10, a fraction rounded up to a whole day (SRD 5.2.1, "Time", p.
// 103). Silver and Copper Pieces count as exact fractions of a Gold Piece, so
// 10 GP 1 CP takes 2 days; that reading is package design.
//
// It returns an error matching srderr.ErrInvalidInput for a cost of 0, a cost
// with a rate or an invalid cost.
func (Crafting) Days(cost content.Cost) (int, error) {
	if _, rated := cost.Per(); rated || !cost.Valid() || cost.ValueInCopper() == 0 {
		return 0, invalidInput("crafting cost")
	}
	return int((cost.ValueInCopper() + copperPerDay - 1) / copperPerDay), nil
}
