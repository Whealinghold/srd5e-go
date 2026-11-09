package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RerollTargetReason says why NewRerollTarget refused its arguments. The zero
// value is not a reason.
type RerollTargetReason int

const (
	// RerollTargetDiceRolledOutOfRange: diceRolled is below 1 or above dice.MaxCount.
	RerollTargetDiceRolledOutOfRange RerollTargetReason = iota + 1
	// RerollTargetPositionOutOfRange: position is below 0 or not below diceRolled.
	RerollTargetPositionOutOfRange
)

// String returns "dice_rolled_out_of_range", "position_out_of_range" or "unknown".
func (r RerollTargetReason) String() string {
	switch r {
	case RerollTargetDiceRolledOutOfRange:
		return "dice_rolled_out_of_range"
	case RerollTargetPositionOutOfRange:
		return "position_out_of_range"
	}
	return "unknown"
}

// RerollTargetError reports the argument NewRerollTarget refused. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text. Build it
// with a keyed literal.
type RerollTargetError struct {
	Reason   RerollTargetReason
	Argument string // "diceRolled" or "position"
	Value    int
}

// Error renders the error. The text is not a contract.
func (e *RerollTargetError) Error() string {
	return "character: invalid reroll target " + e.Argument + " " + strconv.Itoa(e.Value) + ": " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *RerollTargetError) Unwrap() error { return srderr.ErrInvalidInput }

// RerollTarget is the one die a Heroic Inspiration reroll replaces: its size,
// how many dice were rolled together (2 for Advantage or Disadvantage) and which
// of them is chosen (SRD 5.2.1, "Heroic Inspiration", p. 8; "Advantage/
// Disadvantage", "Interactions with Rerolls", p. 8). It does not carry the
// original value. It is immutable and comparable; the zero RerollTarget is
// invalid and ExpendHeroicInspiration.Expend refuses it. Safe for concurrent use.
type RerollTarget struct {
	die                  dice.Die
	diceRolled, position int
}

// NewRerollTarget returns the target. It returns a *RerollTargetError for a
// diceRolled outside 1 to dice.MaxCount, and then for a position outside
// 0 to diceRolled-1. It never panics, whatever the arguments. The die is not
// checked here: Expend refuses the zero Die.
func NewRerollTarget(die dice.Die, diceRolled, position int) (RerollTarget, error) {
	if diceRolled < 1 || diceRolled > dice.MaxCount {
		return RerollTarget{}, &RerollTargetError{Reason: RerollTargetDiceRolledOutOfRange, Argument: "diceRolled", Value: diceRolled}
	}
	if position < 0 || position >= diceRolled {
		return RerollTarget{}, &RerollTargetError{Reason: RerollTargetPositionOutOfRange, Argument: "position", Value: position}
	}
	return RerollTarget{die: die, diceRolled: diceRolled, position: position}, nil
}

// MustRerollTarget is NewRerollTarget for arguments known to be valid. It
// panics with the *RerollTargetError otherwise.
func MustRerollTarget(die dice.Die, diceRolled, position int) RerollTarget {
	t, err := NewRerollTarget(die, diceRolled, position)
	if err != nil {
		panic(err)
	}
	return t
}

// Die returns the die to reroll.
func (t RerollTarget) Die() dice.Die { return t.die }

// DiceRolled returns how many dice were rolled together.
func (t RerollTarget) DiceRolled() int { return t.diceRolled }

// Position returns the index of the chosen die among those rolled.
func (t RerollTarget) Position() int { return t.position }

// IsZero reports whether t is the zero RerollTarget, or has the zero Die.
func (t RerollTarget) IsZero() bool { return t.diceRolled == 0 || t.die.IsZero() }
