package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RerolledDieError reports the argument NewRerolledDie refused. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type RerolledDieError struct {
	Argument string // "newValue" or "position"
	Value    int
}

// Error renders the error. The text is not a contract.
func (e *RerolledDieError) Error() string {
	return "character: invalid rerolled die " + e.Argument + " " + strconv.Itoa(e.Value)
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *RerolledDieError) Unwrap() error { return srderr.ErrInvalidInput }

// RerolledDie is the new roll of one die replaced by Heroic Inspiration. The
// player must use NewValue: the original value is not offered, and the new roll
// is used even when it is lower (SRD 5.2.1, "Heroic Inspiration", p. 8). It is
// immutable and comparable; the zero RerolledDie is invalid. Safe for
// concurrent use.
type RerolledDie struct {
	die                dice.Die
	position, newValue int
}

// NewRerolledDie returns the rerolled die. It returns a *RerolledDieError for a
// newValue outside 1 to die.Sides() (always, for the zero Die) and for a
// negative position. It never panics.
func NewRerolledDie(die dice.Die, position, newValue int) (RerolledDie, error) {
	if newValue < 1 || newValue > die.Sides() {
		return RerolledDie{}, &RerolledDieError{Argument: "newValue", Value: newValue}
	}
	if position < 0 {
		return RerolledDie{}, &RerolledDieError{Argument: "position", Value: position}
	}
	return RerolledDie{die: die, position: position, newValue: newValue}, nil
}

// MustRerolledDie is NewRerolledDie for arguments known to be valid. It panics
// with the *RerolledDieError otherwise.
func MustRerolledDie(die dice.Die, position, newValue int) RerolledDie {
	r, err := NewRerolledDie(die, position, newValue)
	if err != nil {
		panic(err)
	}
	return r
}

// Die returns the die that was rerolled.
func (r RerolledDie) Die() dice.Die { return r.die }

// Position returns the index of the die among those rolled.
func (r RerolledDie) Position() int { return r.position }

// NewValue returns the face of the new roll, which must be used.
func (r RerolledDie) NewValue() int { return r.newValue }
