package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollRangeError reports a roll outside the faces of its die. It matches
// srderr.ErrInvalidInput with errors.Is. Construct it with a keyed literal.
type RollRangeError struct {
	// Name is the input that was out of range ("roll").
	Name string
	// Value is the rejected value.
	Value int
	// Min and Max are the inclusive bounds the value had to fall within.
	Min, Max int
}

// Error renders the error. The text is not a contract.
func (e *RollRangeError) Error() string {
	return "character: " + e.Name + " " + strconv.Itoa(e.Value) + " outside " + strconv.Itoa(e.Min) + " to " + strconv.Itoa(e.Max)
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *RollRangeError) Unwrap() error { return srderr.ErrInvalidInput }

// HitPointsRegained returns the Hit Points one spent Hit Point Die regains: the
// die roll plus the Constitution modifier, and at least 1 (SRD 5.2.1, "Short
// Rest", "Spend Hit Point Dice", p. 187). The result is not capped by the Hit
// Point maximum; the host that applies Hit Points does that.
//
// It returns a *RollRangeError (matching srderr.ErrInvalidInput) when roll is
// below 1 or above die.Faces() (any roll, for an invalid die). It never panics.
func HitPointsRegained(die content.HitPointDie, roll int, constitutionModifier ability.Modifier) (int, error) {
	if roll < 1 || roll > die.Faces() {
		return 0, &RollRangeError{Name: "roll", Value: roll, Min: 1, Max: die.Faces()}
	}
	return regainedHitPoints(roll, constitutionModifier), nil
}

// regainedHitPoints is the roll plus the Constitution modifier, at least 1; the
// roll is already known to be within its die.
func regainedHitPoints(roll int, constitutionModifier ability.Modifier) int {
	return max(1, roll+constitutionModifier.Value())
}
