package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// HitPointGain is the Hit Points a character gains with a level: a rolled Hit
// Point Die or its fixed value (SRD 5.2.1, "Gaining a Level", step 2, p. 23).
// It is a closed set. Variants: RolledHitPointGain, FixedHitPointGain.
type HitPointGain interface {
	// Amount returns the gain: the value plus the Constitution modifier, and at
	// least 1.
	Amount(constitution ability.Modifier) int
	isHitPointGain()
}

// RolledHitPointGain is a gain from a rolled Hit Point Die. It is an immutable
// value, comparable with ==. The zero value is invalid; build it with
// NewRolledHitPointGain.
type RolledHitPointGain struct {
	value int
	die   content.HitPointDie
}

// FixedHitPointGain is a gain of a class's fixed value instead of a roll. It is
// an immutable value, comparable with ==. The zero value is invalid; build it
// with NewFixedHitPointGain.
type FixedHitPointGain struct{ value int }

func (RolledHitPointGain) isHitPointGain() {}
func (FixedHitPointGain) isHitPointGain()  {}

// RolledHitPointGainRejection says why NewRolledHitPointGain rejected a value.
// The zero value is not a rejection.
type RolledHitPointGainRejection int

// The reasons a rolled gain is rejected.
const (
	// RolledHitPointGainBelowOne: the roll is below 1.
	RolledHitPointGainBelowOne RolledHitPointGainRejection = iota + 1
	// RolledHitPointGainAboveDieFaces: the roll is above the die's faces (any
	// roll, for an invalid die).
	RolledHitPointGainAboveDieFaces
)

// String returns "belowOne", "aboveDieFaces" or "unknown".
func (r RolledHitPointGainRejection) String() string {
	switch r {
	case RolledHitPointGainBelowOne:
		return "belowOne"
	case RolledHitPointGainAboveDieFaces:
		return "aboveDieFaces"
	}
	return "unknown"
}

// RolledHitPointGainError reports a roll outside its die. It matches
// srderr.ErrInvalidInput with errors.Is.
type RolledHitPointGainError struct {
	Value  int
	Die    content.HitPointDie
	Reason RolledHitPointGainRejection
}

// Error renders the error. The text is not a contract.
func (e *RolledHitPointGainError) Error() string {
	return "character: invalid rolled Hit Point gain " + strconv.Itoa(e.Value) + ": " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *RolledHitPointGainError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *RolledHitPointGainError with the same fields.
func (e *RolledHitPointGainError) Is(target error) bool {
	t, ok := target.(*RolledHitPointGainError)
	return ok && t != nil && e != nil && *t == *e
}

// NewRolledHitPointGain returns the gain from rolling value on die.
//
// It returns a *RolledHitPointGainError (matching srderr.ErrInvalidInput) for a
// value below 1 or above die.Faces(). It never panics.
func NewRolledHitPointGain(value int, die content.HitPointDie) (RolledHitPointGain, error) {
	switch {
	case value < 1:
		return RolledHitPointGain{}, &RolledHitPointGainError{Value: value, Die: die, Reason: RolledHitPointGainBelowOne}
	case value > die.Faces():
		return RolledHitPointGain{}, &RolledHitPointGainError{Value: value, Die: die, Reason: RolledHitPointGainAboveDieFaces}
	}
	return RolledHitPointGain{value: value, die: die}, nil
}

// MustRolledHitPointGain is NewRolledHitPointGain for known-valid input. It
// panics with the error NewRolledHitPointGain returns, and only then.
func MustRolledHitPointGain(value int, die content.HitPointDie) RolledHitPointGain {
	g, err := NewRolledHitPointGain(value, die)
	if err != nil {
		panic(err)
	}
	return g
}

// NewFixedHitPointGain returns the gain of the fixed value. It returns an error
// matching srderr.ErrInvalidInput for a value below 1. It never panics.
func NewFixedHitPointGain(value int) (FixedHitPointGain, error) {
	if value < 1 {
		return FixedHitPointGain{}, invalidInput("fixed Hit Point gain below 1")
	}
	return FixedHitPointGain{value: value}, nil
}

// Die returns the die rolled.
func (g RolledHitPointGain) Die() content.HitPointDie { return g.die }

// Value returns the number rolled.
func (g RolledHitPointGain) Value() int { return g.value }

// Value returns the fixed value.
func (g FixedHitPointGain) Value() int { return g.value }

// Amount returns the roll plus the Constitution modifier, at least 1.
func (g RolledHitPointGain) Amount(constitution ability.Modifier) int {
	return max(1, g.value+constitution.Value())
}

// Amount returns the fixed value plus the Constitution modifier, at least 1.
func (g FixedHitPointGain) Amount(constitution ability.Modifier) int {
	return max(1, g.value+constitution.Value())
}
