package content

import "math"

// CustomBackgroundCopperPieces is the value of the equipment of a custom
// background: 50 Gold Pieces, including unspent gold, at 100 Copper Pieces to
// the Gold Piece (SRD 5.2.1, "Creating a Background", step 5, p. 193;
// "Coins", p. 89).
const CustomBackgroundCopperPieces = 50 * copperPerGold

// copperPerGold is the number of Copper Pieces in one Gold Piece (SRD 5.2.1,
// "Coins", p. 89).
const copperPerGold = 100

// maxTotalCopper caps the running total of a valuation, so the sum never
// overflows an int64 however many items there are; any total above
// [CustomBackgroundCopperPieces] is wrong, and the cap is far above it.
const maxTotalCopper = 1 << 61

// EquipmentItemValue is what a caller's valuation says of one equipment item:
// its price in Copper Pieces and whether it is a Martial weapon or armor. The
// zero value is a free item that is not excluded; build others with
// [NewEquipmentItemValue]. Comparable.
type EquipmentItemValue struct {
	copperPieces           int
	isMartialWeaponOrArmor bool
}

// NewEquipmentItemValue returns the value of an item priced at copperPieces,
// from 0 to math.MaxInt32, that is a Martial weapon or armor when martial is
// true.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a price outside the range. It never panics.
func NewEquipmentItemValue(copperPieces int, martial bool) (EquipmentItemValue, error) {
	if copperPieces < 0 || copperPieces > math.MaxInt32 {
		return EquipmentItemValue{}, blockError("EquipmentItemValue", BlockOutOfRange)
	}
	return EquipmentItemValue{copperPieces: copperPieces, isMartialWeaponOrArmor: martial}, nil
}

// CopperPieces returns the price of one item in Copper Pieces.
func (v EquipmentItemValue) CopperPieces() int { return v.copperPieces }

// IsMartialWeaponOrArmor reports whether the item is a Martial weapon or
// armor, which a custom background may not take.
func (v EquipmentItemValue) IsMartialWeaponOrArmor() bool { return v.isMartialWeaponOrArmor }

// EquipmentValuation prices an item for [CustomBackgroundEquipmentProblems].
// It returns false when it has no price for the item. The caller supplies it
// (typically from equipment content), so this package imports no equipment.
type EquipmentValuation func(EquipmentItem) (EquipmentItemValue, bool)

// CustomBackgroundEquipmentProblem is the closed set of problems with the
// equipment option of a custom background. Variants:
// EquipmentValueNotFiftyGold, EquipmentItemUnpriced, EquipmentItemExcluded.
// Consumers switch over it exhaustively; the variants are immutable and
// comparable.
type CustomBackgroundEquipmentProblem interface{ isCustomBackgroundEquipmentProblem() }

// EquipmentValueNotFiftyGold says the option is worth CopperPieces Copper
// Pieces, gold included, not [CustomBackgroundCopperPieces]. A total beyond
// 2^61 is reported as 2^61. Construct it with a keyed literal.
type EquipmentValueNotFiftyGold struct{ CopperPieces int64 }

func (EquipmentValueNotFiftyGold) isCustomBackgroundEquipmentProblem() {}

// EquipmentItemUnpriced says the valuation had no price for the item called
// ItemName. Construct it with a keyed literal.
type EquipmentItemUnpriced struct{ ItemName string }

func (EquipmentItemUnpriced) isCustomBackgroundEquipmentProblem() {}

// EquipmentItemExcluded says the item called ItemName is a Martial weapon or
// armor. Construct it with a keyed literal.
type EquipmentItemExcluded struct{ ItemName string }

func (EquipmentItemExcluded) isCustomBackgroundEquipmentProblem() {}

// CustomBackgroundEquipmentProblems checks step 5 of "Creating a Background"
// (SRD 5.2.1, p. 193) for one equipment option: no Martial weapon or armor, and
// a value of exactly 50 Gold Pieces, unspent gold included (100 Copper Pieces
// to the Gold Piece). It is a pure function; run it on each of the options A
// and B of a custom background, never on an SRD background.
//
// It returns the problems, none for a valid option, in item order and then the
// total: an [EquipmentItemUnpriced] for each item valuation has no price for,
// an [EquipmentItemExcluded] for each Martial item (which still counts toward
// the total), and an [EquipmentValueNotFiftyGold] unless some item was unpriced
// (an unpriced item is not counted as free, so the total is not checked). A nil
// valuation prices nothing. It never panics.
func CustomBackgroundEquipmentProblems(option EquipmentOption, valuation EquipmentValuation) []CustomBackgroundEquipmentProblem {
	var problems []CustomBackgroundEquipmentProblem
	total, unpriced := int64(option.gold)*copperPerGold, false
	for _, item := range option.items {
		var v EquipmentItemValue
		priced := false
		if valuation != nil {
			v, priced = valuation(item)
		}
		if !priced {
			unpriced = true
			problems = append(problems, EquipmentItemUnpriced{ItemName: item.name})
			continue
		}
		if v.isMartialWeaponOrArmor {
			problems = append(problems, EquipmentItemExcluded{ItemName: item.name})
		}
		total = min(total+min(int64(item.quantity)*int64(v.copperPieces), maxTotalCopper), maxTotalCopper)
	}
	if !unpriced && total != CustomBackgroundCopperPieces {
		problems = append(problems, EquipmentValueNotFiftyGold{CopperPieces: total})
	}
	return problems
}

// MustEquipmentItemValue is like [NewEquipmentItemValue] but panics with its
// error when the arguments are invalid.
func MustEquipmentItemValue(copperPieces int, martial bool) EquipmentItemValue {
	v, err := NewEquipmentItemValue(copperPieces, martial)
	if err != nil {
		panic(err)
	}
	return v
}
