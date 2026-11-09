package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// HigherLevelEquipment is the closed outcome of looking up the Starting
// Equipment at Higher Levels table (SRD 5.2.1, "Starting at Higher Levels",
// p. 24), a guide for the GM that comes on top of the normal starting equipment.
// Variants: NoHigherLevelEquipment, HigherLevelEquipmentAllowance.
type HigherLevelEquipment interface{ isHigherLevelEquipment() }

// NoHigherLevelEquipment is the table's answer for starting level 1: nothing
// beyond the normal starting equipment.
type NoHigherLevelEquipment struct{}

// RolledGold is the rolled part of an allowance: a dice expression and the
// multiplier, in GP, applied to its total (1d10 x 25 GP). It is an immutable
// value, comparable with ==. The zero RolledGold is invalid and means no rolled
// gold; see IsZero.
type RolledGold struct {
	expression dice.Expression
	multiplier int
}

// Expression returns the dice expression rolled.
func (g RolledGold) Expression() dice.Expression { return g.expression }

// Multiplier returns the GP each point of the roll is worth, as a Cost in GP.
func (g RolledGold) Multiplier() content.Cost {
	return content.MustCost(g.multiplier, content.CoinGP, 0)
}

// IsZero reports whether g is the invalid zero RolledGold.
func (g RolledGold) IsZero() bool { return g.expression.IsZero() }

// HigherLevelEquipmentAllowance is the extra gold and the magic items the table
// gives a starting level of 2 or more. It is an immutable value, comparable with
// ==. The zero value is invalid; get one from HigherLevelEquipmentRules.
type HigherLevelEquipmentAllowance struct {
	fixed  int
	rolled RolledGold
	items  [4]int // Common, Uncommon, Rare, Very Rare
}

// FixedGold returns the fixed gold, as a Cost in GP.
func (a HigherLevelEquipmentAllowance) FixedGold() content.Cost {
	return content.MustCost(a.fixed, content.CoinGP, 0)
}

// RolledGold returns a new slice holding the one RolledGold of the allowance, or
// nothing for starting levels 2 to 4.
func (a HigherLevelEquipmentAllowance) RolledGold() []RolledGold {
	if a.rolled.IsZero() {
		return nil
	}
	return []RolledGold{a.rolled}
}

// MagicItems returns a new map of the number of magic items by rarity, with the
// non-zero counts only. Only Common, Uncommon, Rare and Very Rare appear.
func (a HigherLevelEquipmentAllowance) MagicItems() map[content.ItemRarity]int {
	out := map[content.ItemRarity]int{}
	for i, n := range a.items {
		if n > 0 {
			out[content.RarityCommon+content.ItemRarity(i)] = n
		}
	}
	return out
}

// TotalGold returns the total gold in GP: the fixed gold plus, for an allowance
// with rolled gold, *roll times the multiplier. It is pure.
//
// It returns an error matching srderr.ErrInvalidInput (a
// *HigherLevelEquipmentError) when the allowance has rolled gold and roll is nil
// or outside 1 to the faces of the die, and when it has none and a roll is given.
// It never panics.
func (a HigherLevelEquipmentAllowance) TotalGold(roll *int) (int, error) {
	if a.rolled.IsZero() {
		if roll != nil {
			return 0, &HigherLevelEquipmentError{Argument: "roll", InvalidValue: *roll}
		}
		return a.fixed, nil
	}
	if roll == nil {
		return 0, &HigherLevelEquipmentError{Argument: "roll"}
	}
	if *roll < 1 || *roll > a.rolled.expression.Die().Sides() {
		return 0, &HigherLevelEquipmentError{Argument: "roll", InvalidValue: *roll}
	}
	return a.fixed + *roll*a.rolled.multiplier, nil
}

func (NoHigherLevelEquipment) isHigherLevelEquipment()        {}
func (HigherLevelEquipmentAllowance) isHigherLevelEquipment() {}

// HigherLevelEquipmentError reports an invalid argument of the starting money
// rules: the argument name and the offending value (0 for a missing roll). It
// matches srderr.ErrInvalidInput with errors.Is.
type HigherLevelEquipmentError struct {
	Argument     string
	InvalidValue int
}

// Error renders the error. The text is not a contract.
func (e *HigherLevelEquipmentError) Error() string {
	return "character: invalid starting money argument: " + e.Argument
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *HigherLevelEquipmentError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *HigherLevelEquipmentError with the same
// Argument and InvalidValue.
func (e *HigherLevelEquipmentError) Is(target error) bool {
	t, ok := target.(*HigherLevelEquipmentError)
	return ok && t != nil && e != nil && *t == *e
}

// HigherLevelEquipmentRules is the pure rules service for the Starting
// Equipment at Higher Levels table. The zero value is ready to use. It is
// stateless and safe for concurrent use.
type HigherLevelEquipmentRules struct{}

// ForStartingLevel returns NoHigherLevelEquipment for level 1 (and for the zero
// Level, which is not a level) and the allowance of the level's tier otherwise
// (SRD 5.2.1, p. 24):
//
//	2 to 4:   0 GP; Common 1
//	5 to 10:  500 GP plus 1d10 x 25 GP; Common 1, Uncommon 1
//	11 to 16: 5,000 GP plus 1d10 x 250 GP; Common 2, Uncommon 3, Rare 1
//	17 to 20: 20,000 GP plus 1d10 x 250 GP; Common 2, Uncommon 4, Rare 3, Very Rare 1
func (HigherLevelEquipmentRules) ForStartingLevel(level Level) HigherLevelEquipment {
	d10 := dice.MustExpression(1, dice.D10(), 0)
	switch n := level.Value(); {
	case n >= 17:
		return HigherLevelEquipmentAllowance{fixed: 20000, rolled: RolledGold{d10, 250}, items: [4]int{2, 4, 3, 1}}
	case n >= 11:
		return HigherLevelEquipmentAllowance{fixed: 5000, rolled: RolledGold{d10, 250}, items: [4]int{2, 3, 1, 0}}
	case n >= 5:
		return HigherLevelEquipmentAllowance{fixed: 500, rolled: RolledGold{d10, 25}, items: [4]int{1, 1, 0, 0}}
	case n >= 2:
		return HigherLevelEquipmentAllowance{items: [4]int{1, 0, 0, 0}}
	}
	return NoHigherLevelEquipment{}
}
