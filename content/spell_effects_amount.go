package content

import (
	"math"

	"github.com/Whealinghold/srd5e-go/dice"
)

// SpellAmount is the closed set of amounts a spell outcome can have:
// [RolledSpellAmount] and [FixedSpellAmount]. Every variant is an immutable,
// comparable value and safe for concurrent use. Consumers switch over it
// exhaustively.
type SpellAmount interface{ isSpellAmount() }

// RolledSpellAmount is an amount rolled with dice (SRD 5.2.1, "Damage Rolls",
// p. 16). The zero value is invalid; build it with [NewRolledSpellAmount].
type RolledSpellAmount struct{ roll dice.Expression }

// NewRolledSpellAmount returns the amount rolled by roll.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for the zero expression (its count is below 1). It never
// panics.
func NewRolledSpellAmount(roll dice.Expression) (RolledSpellAmount, error) {
	if roll.IsZero() {
		return RolledSpellAmount{}, blockError("RolledSpellAmount", BlockOutOfRange)
	}
	return RolledSpellAmount{roll: roll}, nil
}

// Roll returns the dice that give the amount.
func (a RolledSpellAmount) Roll() dice.Expression { return a.roll }

func (RolledSpellAmount) isSpellAmount() {}

// FixedSpellAmount is an amount printed as a number (SRD 5.2.1, "Heal",
// p. 139). The zero value is invalid; build it with [NewFixedSpellAmount].
type FixedSpellAmount struct{ value int }

// NewFixedSpellAmount returns the fixed amount v, v being at least 1.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for v below 1. It never panics.
func NewFixedSpellAmount(v int) (FixedSpellAmount, error) {
	if v < 1 {
		return FixedSpellAmount{}, blockError("FixedSpellAmount", BlockOutOfRange)
	}
	return FixedSpellAmount{value: v}, nil
}

// Value returns the amount.
func (a FixedSpellAmount) Value() int { return a.value }

func (FixedSpellAmount) isSpellAmount() {}

// checkSpellAmount reports why a is not an amount its constructor would
// accept: BlockMissing for nil.
func checkSpellAmount(a SpellAmount) error {
	switch v := a.(type) {
	case RolledSpellAmount:
		_, err := NewRolledSpellAmount(v.roll)
		return err
	case FixedSpellAmount:
		_, err := NewFixedSpellAmount(v.value)
		return err
	}
	return blockError("SpellAmount", BlockMissing)
}

// AmountStep is the growth of an amount for each slot level above a level
// (SRD 5.2.1, "Using a Higher-Level Spell Slot", pp. 104-105): ExtraDice more
// of the base die and ExtraFlat more to the flat modifier. The zero AmountStep
// is invalid; build it with [NewAmountStep]. Comparable and safe for
// concurrent use.
type AmountStep struct{ extraDice, extraFlat int }

// NewAmountStep returns the step of extraDice dice and extraFlat flat points.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a negative part or for both parts 0. It never panics.
func NewAmountStep(extraDice, extraFlat int) (AmountStep, error) {
	if extraDice < 0 || extraFlat < 0 || extraDice+extraFlat == 0 {
		return AmountStep{}, blockError("AmountStep", BlockOutOfRange)
	}
	return AmountStep{extraDice: extraDice, extraFlat: extraFlat}, nil
}

// ExtraDice returns the number of dice added for each slot level.
func (s AmountStep) ExtraDice() int { return s.extraDice }

// ExtraFlat returns the flat amount added for each slot level.
func (s AmountStep) ExtraFlat() int { return s.extraFlat }

// AmountScaling is the closed set of ways an amount grows: [NoAmountScaling],
// [PerSlotLevelAmount] and [ByCharacterLevelAmount]. Every variant is an
// immutable, comparable value and safe for concurrent use. Consumers switch
// over it exhaustively.
type AmountScaling interface{ isAmountScaling() }

// NoAmountScaling is an amount that does not grow. The zero value is valid.
type NoAmountScaling struct{}

func (NoAmountScaling) isAmountScaling() {}

// PerSlotLevelAmount is an amount that grows by Step for each slot level above
// AboveLevel (SRD 5.2.1, "Fireball", p. 131). The zero value is invalid; build
// it with [NewPerSlotLevelAmount].
type PerSlotLevelAmount struct {
	aboveLevel int
	step       AmountStep
}

// NewPerSlotLevelAmount returns the scaling of step per level above
// aboveLevel. Whether aboveLevel suits a spell is the spell's check
// ([SpellEffectModel.CheckSpellLevel]).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an invalid step. It never panics.
func NewPerSlotLevelAmount(aboveLevel int, step AmountStep) (PerSlotLevelAmount, error) {
	if _, err := NewAmountStep(step.extraDice, step.extraFlat); err != nil {
		return PerSlotLevelAmount{}, err
	}
	return PerSlotLevelAmount{aboveLevel: aboveLevel, step: step}, nil
}

// AboveLevel returns the slot level above which the amount grows.
func (s PerSlotLevelAmount) AboveLevel() int { return s.aboveLevel }

// Step returns the growth for each slot level.
func (s PerSlotLevelAmount) Step() AmountStep { return s.step }

func (PerSlotLevelAmount) isAmountScaling() {}

// ByCharacterLevelAmount is an amount chosen by total character level (SRD
// 5.2.1, "Acid Splash", p. 107). The zero value is invalid; build it with
// [NewByCharacterLevelAmount].
type ByCharacterLevelAmount struct{ table AmountLevelTable }

// NewByCharacterLevelAmount returns the scaling by the table.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// for the zero table. It never panics.
func NewByCharacterLevelAmount(table AmountLevelTable) (ByCharacterLevelAmount, error) {
	if table.t.n == 0 {
		return ByCharacterLevelAmount{}, blockError("ByCharacterLevelAmount", BlockEmpty)
	}
	return ByCharacterLevelAmount{table: table}, nil
}

// Table returns the table of amounts by level.
func (s ByCharacterLevelAmount) Table() AmountLevelTable { return s.table }

func (ByCharacterLevelAmount) isAmountScaling() {}

// checkAmountScaling reports why s is not a scaling its constructor would
// accept; a nil scaling is the valid [NoAmountScaling].
func checkAmountScaling(s AmountScaling) error {
	switch v := s.(type) {
	case nil, NoAmountScaling:
		return nil
	case PerSlotLevelAmount:
		_, err := NewPerSlotLevelAmount(v.aboveLevel, v.step)
		return err
	case ByCharacterLevelAmount:
		_, err := NewByCharacterLevelAmount(v.table)
		return err
	}
	return blockError("AmountScaling", BlockInvalidMember)
}

// ScalableAmount is an amount with its scaling: a base, and how it grows with
// the slot level or the character level (spell-effect-data.md FR-016 to
// FR-021). The zero ScalableAmount is invalid; build it with
// [NewScalableAmount]. Comparable and safe for concurrent use.
type ScalableAmount struct {
	base    SpellAmount
	scaling AmountScaling
}

// NewScalableAmount returns the amount base scaled by scaling; a nil scaling
// is [NoAmountScaling].
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockMissing
// for a nil base, the reason of the part's own check for an invalid base or
// scaling, and BlockConstraint for a step that adds dice to a [FixedSpellAmount]
// (FR-016). It never panics.
func NewScalableAmount(base SpellAmount, scaling AmountScaling) (ScalableAmount, error) {
	if err := checkSpellAmount(base); err != nil {
		return ScalableAmount{}, err
	}
	if err := checkAmountScaling(scaling); err != nil {
		return ScalableAmount{}, err
	}
	if scaling == nil {
		scaling = NoAmountScaling{}
	}
	if s, ok := scaling.(PerSlotLevelAmount); ok && s.step.extraDice > 0 {
		if _, fixed := base.(FixedSpellAmount); fixed {
			return ScalableAmount{}, blockError("ScalableAmount", BlockConstraint)
		}
	}
	return ScalableAmount{base: base, scaling: scaling}, nil
}

// Base returns the amount before scaling.
func (a ScalableAmount) Base() SpellAmount { return a.base }

// Scaling returns how the amount grows.
func (a ScalableAmount) Scaling() AmountScaling { return a.scaling }

// At returns the amount for a cast. slotLevelsAbove is the number of slot
// levels above the scaling's AboveLevel (at least 0) and characterLevel is the
// total character level (1 to 20). Each input is read only by the scaling that
// uses it: [PerSlotLevelAmount] reads slotLevelsAbove, [ByCharacterLevelAmount]
// reads characterLevel, [NoAmountScaling] reads neither (FR-041).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an input it reads that is out of range or for a result
// beyond the dice limits, and BlockMissing for the zero value. It never panics.
func (a ScalableAmount) At(slotLevelsAbove, characterLevel int) (SpellAmount, error) {
	switch s := a.scaling.(type) {
	case PerSlotLevelAmount:
		if slotLevelsAbove < 0 {
			return nil, blockError("ScalableAmount", BlockOutOfRange)
		}
		return a.grow(s.step, slotLevelsAbove)
	case ByCharacterLevelAmount:
		return s.table.ValueAt(characterLevel)
	}
	if a.base == nil {
		return nil, blockError("ScalableAmount", BlockMissing)
	}
	return a.base, nil
}

// grow adds n steps to the base. A result past the limits of dice.Expression
// or of int is out of range, never wrapped around.
func (a ScalableAmount) grow(step AmountStep, n int) (SpellAmount, error) {
	switch b := a.base.(type) {
	case RolledSpellAmount:
		dc, okDice := mulAdd(b.roll.Count(), n, step.extraDice)
		fl, okMod := mulAdd(b.roll.Modifier(), n, step.extraFlat)
		if okDice && okMod {
			if e, err := dice.NewExpression(dc, b.roll.Die(), fl); err == nil {
				return RolledSpellAmount{roll: e}, nil
			}
		}
	case FixedSpellAmount:
		if v, ok := mulAdd(b.value, n, step.extraFlat); ok {
			return FixedSpellAmount{value: v}, nil
		}
	default:
		return nil, blockError("ScalableAmount", BlockMissing)
	}
	return nil, blockError("ScalableAmount", BlockOutOfRange)
}

// mulAdd returns base + n*extra, or false when it does not fit in an int. n
// and extra are not negative.
func mulAdd(base, n, extra int) (int, bool) {
	if extra != 0 && n > (math.MaxInt-max(base, 0))/extra {
		return 0, false
	}
	return base + n*extra, true
}
