package content

import "math"

// MaxMagicItemCopper is the largest amount a [PricedMagicItemValue] holds, so
// no amount differs between platform integer widths.
const MaxMagicItemCopper = math.MaxInt64

// MagicItemValue is the value of a magic item of a rarity, a closed family
// with two variants: [PricedMagicItemValue] and [PricelessMagicItemValue].
// Amounts are whole Copper Pieces, the unit of the equipment cost, because
// content may not import the equipment types. Every variant is an immutable,
// comparable value and safe for concurrent use.
type MagicItemValue interface{ isMagicItemValue() }

// PricedMagicItemValue is a value in Copper Pieces. The zero value is a valid
// value of 0; build others with [NewPricedMagicItemValue] or
// [PricedMagicItemValueFromGold].
type PricedMagicItemValue struct{ copperPieces int64 }

// NewPricedMagicItemValue returns a value of copperPieces Copper Pieces.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a negative amount. It never panics.
func NewPricedMagicItemValue(copperPieces int64) (PricedMagicItemValue, error) {
	if copperPieces < 0 {
		return PricedMagicItemValue{}, blockError("PricedMagicItemValue", BlockOutOfRange)
	}
	return PricedMagicItemValue{copperPieces: copperPieces}, nil
}

// PricedMagicItemValueFromGold returns a value of gold Gold Pieces, that is
// gold times 100 Copper Pieces.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a negative amount or one whose Copper Pieces overflow.
// It never panics.
func PricedMagicItemValueFromGold(gold int64) (PricedMagicItemValue, error) {
	if gold < 0 || gold > MaxMagicItemCopper/copperPerGold {
		return PricedMagicItemValue{}, blockError("PricedMagicItemValue", BlockOutOfRange)
	}
	return NewPricedMagicItemValue(gold * copperPerGold)
}

// CopperPieces returns the value in Copper Pieces.
func (v PricedMagicItemValue) CopperPieces() int64 { return v.copperPieces }

func (PricedMagicItemValue) isMagicItemValue() {}

// PricelessMagicItemValue is the value of an Artifact: no price is invented.
// The zero value is valid.
type PricelessMagicItemValue struct{}

func (PricelessMagicItemValue) isMagicItemValue() {}

// The days of crafting that are 8 hours of work each (SRD 5.2.1, "Magic Item
// Crafting Time and Cost", p. 207).
const craftingHoursPerDay = 8

// MagicItemCraftingRow is one row of the Magic Item Crafting Time and Cost
// table: a rarity, the days and hours of work and the cost. The zero value is
// invalid; build it with [NewMagicItemCraftingRow]. Comparable.
type MagicItemCraftingRow struct {
	rarity    ItemRarity
	days      int
	workHours int
	cost      PricedMagicItemValue
}

// NewMagicItemCraftingRow returns the row of rarity r. A row exists for every
// rarity but the Artifact, workHours is always 8 times days, and days is at
// least 1.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for a rarity that is not a member or is the Artifact,
// BlockOutOfRange for days below 1, BlockConstraint when workHours is not 8
// times days. It never panics.
func NewMagicItemCraftingRow(r ItemRarity, days, workHours int, cost PricedMagicItemValue) (MagicItemCraftingRow, error) {
	const block = "MagicItemCraftingRow"
	switch {
	case !r.Valid() || r == RarityArtifact: // equivalent mutant: the Artifact is the highest member, so == and >= agree once r is valid
		return MagicItemCraftingRow{}, blockError(block, BlockInvalidMember)
	case days < 1:
		return MagicItemCraftingRow{}, blockError(block, BlockOutOfRange)
	case days > math.MaxInt32/craftingHoursPerDay || workHours != craftingHoursPerDay*days:
		return MagicItemCraftingRow{}, blockError(block, BlockConstraint)
	}
	return MagicItemCraftingRow{rarity: r, days: days, workHours: workHours, cost: cost}, nil
}

// Rarity returns the rarity of the row.
func (r MagicItemCraftingRow) Rarity() ItemRarity { return r.rarity }

// Days returns the days of crafting.
func (r MagicItemCraftingRow) Days() int { return r.days }

// WorkHours returns the hours of work, 8 times the days.
func (r MagicItemCraftingRow) WorkHours() int { return r.workHours }

// Cost returns the cost of crafting.
func (r MagicItemCraftingRow) Cost() PricedMagicItemValue { return r.cost }

// MagicItemCraftingRowLookup is the result of looking up the crafting row of a
// rarity, a closed family with two variants: [MagicItemCraftingRowFound] and
// [MagicItemCraftingRowAbsent]. Every variant is an immutable, comparable value
// and safe for concurrent use.
type MagicItemCraftingRowLookup interface{ isMagicItemCraftingRowLookup() }

// MagicItemCraftingRowFound is a rarity that has a crafting row.
type MagicItemCraftingRowFound struct{ row MagicItemCraftingRow }

// Row returns the crafting row.
func (f MagicItemCraftingRowFound) Row() MagicItemCraftingRow { return f.row }

func (MagicItemCraftingRowFound) isMagicItemCraftingRowLookup() {}

// MagicItemCraftingRowAbsent is a rarity with no crafting row: the Artifact.
type MagicItemCraftingRowAbsent struct{}

func (MagicItemCraftingRowAbsent) isMagicItemCraftingRowLookup() {}

// ItemRarityFacts holds the value and the crafting row of a rarity (SRD 5.2.1,
// "Magic Item Rarities and Values", p. 206; "Magic Item Crafting Time and
// Cost", p. 207). Get one from [ItemRarityFacts.Of]; the zero value is what
// Of returns for a rarity that is not a member. Comparable and safe for
// concurrent use.
type ItemRarityFacts struct {
	value    MagicItemValue
	crafting MagicItemCraftingRowLookup
}

// Of returns the facts of rarity r: Common 100 GP, Uncommon 400, Rare 4,000,
// Very Rare 40,000, Legendary 200,000 and Artifact priceless; and the crafting
// row of the five rarities that have one (Common 5 days, 50 GP; Uncommon 10,
// 200; Rare 50, 2,000; Very Rare 125, 20,000; Legendary 250, 100,000), the
// Artifact having none. It returns the zero facts for a value that is not a
// rarity. It never panics.
func (ItemRarityFacts) Of(r ItemRarity) ItemRarityFacts {
	if !r.Valid() {
		return ItemRarityFacts{}
	}
	gold := func(n int) PricedMagicItemValue { return PricedMagicItemValue{copperPieces: int64(n) * copperPerGold} }
	var value, days, cost int
	switch r {
	case RarityCommon:
		value, days, cost = 100, 5, 50
	case RarityUncommon:
		value, days, cost = 400, 10, 200
	case RarityRare:
		value, days, cost = 4000, 50, 2000
	case RarityVeryRare:
		value, days, cost = 40000, 125, 20000
	case RarityLegendary:
		value, days, cost = 200000, 250, 100000
	case RarityArtifact:
		return ItemRarityFacts{value: PricelessMagicItemValue{}, crafting: MagicItemCraftingRowAbsent{}}
	}
	row := MagicItemCraftingRow{rarity: r, days: days, workHours: craftingHoursPerDay * days, cost: gold(cost)}
	return ItemRarityFacts{value: gold(value), crafting: MagicItemCraftingRowFound{row: row}}
}

// Value returns the value of the rarity.
func (f ItemRarityFacts) Value() MagicItemValue { return f.value }

// Crafting returns the crafting-row lookup of the rarity.
func (f ItemRarityFacts) Crafting() MagicItemCraftingRowLookup { return f.crafting }
