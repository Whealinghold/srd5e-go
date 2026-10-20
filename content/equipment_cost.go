package content

import "math"

// equipmentMaxInt bounds every integer an equipment type holds (a package
// design bound: the SRD prints no limit, content-equipment FR-091), so no
// value or sum depends on the width of the platform's int.
const equipmentMaxInt = 1<<31 - 1

// MaxCostAmount is the largest amount a [Cost] holds, 2^31 - 1. The SRD
// prints no limit; the bound keeps [Cost.ValueInCopper] exact on every
// platform.
const MaxCostAmount = equipmentMaxInt

// Coin is one of the five coin denominations (SRD 5.2.1, "Coins", p. 89), in
// ascending value. The zero Coin is invalid. Comparable and safe for
// concurrent use.
type Coin int

// The coins: Copper, Silver, Electrum, Gold and Platinum Pieces.
const (
	CoinCP Coin = iota + 1
	CoinSP
	CoinEP
	CoinGP
	CoinPP
)

func coinNames() []string { return []string{"CP", "SP", "EP", "GP", "PP"} }

// Valid reports whether c is one of the five coins.
func (c Coin) Valid() bool { return enumValid(coinNames(), int(c)) }

// String returns the printed abbreviation of c ("GP"), or "unknown".
func (c Coin) String() string { return enumString(coinNames(), int(c)) }

// ValueInCopper returns what one coin of c is worth in Copper Pieces (1, 10,
// 50, 100 and 1000), or 0 for an invalid coin.
func (c Coin) ValueInCopper() int {
	switch c {
	case CoinCP:
		return 1
	case CoinSP:
		return 10
	case CoinEP:
		return 50
	case CoinGP:
		return 100
	case CoinPP:
		return 1000
	}
	return 0
}

func parseCoin(s string) (Coin, error) {
	v, err := parseEnum("Coin", coinNames(), s)
	return Coin(v), err
}

// CostRate says what a [Cost] is charged per when it is not per item. The
// zero CostRate means per item and is not a member.
type CostRate int

// The cost rates.
const (
	CostPerDay CostRate = iota + 1
	CostPerMile
)

func costRateNames() []string { return []string{"Day", "Mile"} }

// Valid reports whether r is a member (the zero rate, per item, is not).
func (r CostRate) Valid() bool { return enumValid(costRateNames(), int(r)) }

// String returns the printed name of r ("Day"), or "unknown".
func (r CostRate) String() string { return enumString(costRateNames(), int(r)) }

func parseCostRate(s string) (CostRate, error) {
	v, err := parseEnum("CostRate", costRateNames(), s)
	return CostRate(v), err
}

// Cost is a printed price: a whole amount of one [Coin], optionally per Day or
// per Mile (SRD 5.2.1, "Coins", p. 89). It is an immutable value compared by
// value, so 5 SP and 50 CP are not equal although they are worth the same;
// order two costs with [CompareCost]. The zero Cost is invalid. Safe for
// concurrent use.
type Cost struct {
	amount int
	coin   Coin
	per    CostRate
}

// NewCost returns the cost of amount coins, per rate (the zero CostRate for
// per item).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockOutOfRange for an amount below 0 or above [MaxCostAmount], then
// BlockInvalidMember for an invalid coin or a rate that is neither zero nor a
// member. It never panics.
func NewCost(amount int, coin Coin, per CostRate) (Cost, error) {
	const block = "Cost"
	switch {
	case amount < 0 || amount > MaxCostAmount:
		return Cost{}, blockError(block, BlockOutOfRange)
	case !coin.Valid(), per != 0 && !per.Valid():
		return Cost{}, blockError(block, BlockInvalidMember)
	}
	return Cost{amount: amount, coin: coin, per: per}, nil
}

// MustCost is like [NewCost] but panics with its error when the arguments are
// invalid.
func MustCost(amount int, coin Coin, per CostRate) Cost {
	c, err := NewCost(amount, coin, per)
	if err != nil {
		panic(err)
	}
	return c
}

// Valid reports whether c is a cost [NewCost] would accept.
func (c Cost) Valid() bool {
	_, err := NewCost(c.amount, c.coin, c.per)
	return err == nil
}

// Amount returns the number of coins.
func (c Cost) Amount() int { return c.amount }

// Coin returns the denomination.
func (c Cost) Coin() Coin { return c.coin }

// Per returns the rate and true when the cost is per Day or per Mile, and the
// zero rate and false when it is per item.
func (c Cost) Per() (CostRate, bool) { return c.per, c.per != 0 }

// ValueInCopper returns the amount times the coin's worth in Copper Pieces.
func (c Cost) ValueInCopper() int64 { return int64(c.amount) * int64(c.coin.ValueInCopper()) }

// CompareCost orders two costs by their value in Copper Pieces, like
// cmp.Compare: 5 SP and 50 CP give 0.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput
// (BlockConstraint) when the rates differ, such as 5 SP per Day against 5 SP,
// because the two are not comparable. It never panics.
func CompareCost(a, b Cost) (int, error) {
	if a.per != b.per {
		return 0, blockError("Cost", BlockConstraint)
	}
	switch av, bv := a.ValueInCopper(), b.ValueInCopper(); {
	case av < bv:
		return -1, nil
	case av > bv:
		return 1, nil
	}
	return 0, nil
}

// Price is a printed price: a closed set with the variants [FixedCost] and
// [CostVaries] (a table cell printed "Varies"). Only [ToolFamily],
// [GearFamily] and [ScrollFamily] hold a Price; every other entry holds a
// [FixedCost].
type Price interface{ isPrice() }

// FixedCost is a price that is a [Cost]. The zero FixedCost is invalid; build
// it with [NewFixedCost]. Comparable.
type FixedCost struct{ cost Cost }

func (FixedCost) isPrice() {}

// NewFixedCost returns the price c. It returns a *[BlockError]
// (BlockInvalidMember) wrapping srderr.ErrInvalidInput for an invalid cost.
func NewFixedCost(c Cost) (FixedCost, error) {
	if !c.Valid() {
		return FixedCost{}, blockError("FixedCost", BlockInvalidMember)
	}
	return FixedCost{cost: c}, nil
}

// MustFixedCost is like [NewFixedCost] but panics with its error when c is
// invalid.
func MustFixedCost(c Cost) FixedCost {
	f, err := NewFixedCost(c)
	if err != nil {
		panic(err)
	}
	return f
}

// Cost returns the cost.
func (f FixedCost) Cost() Cost { return f.cost }

// CostVaries is the price of an entry whose table cell is printed "Varies".
// Its zero value is the one valid value.
type CostVaries struct{}

func (CostVaries) isPrice() {}

func validPrice(p Price) bool {
	switch v := p.(type) {
	case FixedCost:
		return v.cost.Valid()
	case CostVaries:
		return true
	}
	return false
}

// WeightSpec is a printed weight: a closed set with the variants [Weight], a
// number of pounds, [WeightVaries] for a cell printed "Varies" and [NoWeight]
// for one printed "—".
type WeightSpec interface{ isWeightSpec() }

// Weight is a weight in pounds, with the printed parenthetical ("full") when
// there is one. The zero Weight is a weight of 0 pounds with no note.
// Comparable.
type Weight struct {
	pounds float64
	note   string
}

func (Weight) isWeightSpec() {}

// NewWeight returns a weight of pounds with note, "" for none.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for pounds that are negative, infinite or not a number, then BlockBlankText
// or BlockInvalidCharacter for a non-empty note that breaks the text rule
// (schema FR-028). It never panics.
func NewWeight(pounds float64, note string) (Weight, error) {
	if !(pounds >= 0) || math.IsInf(pounds, 0) { // NaN fails the first comparison
		return Weight{}, blockError("Weight", BlockOutOfRange)
	}
	if note != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkText("Weight", note); err != nil {
			return Weight{}, err
		}
	}
	return Weight{pounds: pounds, note: note}, nil
}

// MustWeight is like [NewWeight] but panics with its error when the arguments
// are invalid.
func MustWeight(pounds float64, note string) Weight {
	w, err := NewWeight(pounds, note)
	if err != nil {
		panic(err)
	}
	return w
}

// Pounds returns the weight in pounds.
func (w Weight) Pounds() float64 { return w.pounds }

// Note returns the printed parenthetical, or "" when there is none.
func (w Weight) Note() string { return w.note }

// WeightVaries is the weight of an entry whose table cell is printed
// "Varies". Its zero value is the one valid value.
type WeightVaries struct{}

func (WeightVaries) isWeightSpec() {}

// NoWeight is the weight of an entry whose table cell is printed "—". Its
// zero value is the one valid value.
type NoWeight struct{}

func (NoWeight) isWeightSpec() {}

func validWeightSpec(w WeightSpec) bool {
	switch v := w.(type) {
	case Weight:
		_, err := NewWeight(v.pounds, v.note)
		return err == nil
	case WeightVaries, NoWeight:
		return true
	}
	return false
}

// validFixedWeight reports whether w is a valid weight that is not
// WeightVaries: a Weight or NoWeight.
func validFixedWeight(w WeightSpec) bool {
	_, varies := w.(WeightVaries)
	return !varies && validWeightSpec(w)
}
