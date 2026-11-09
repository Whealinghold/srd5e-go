package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// The bounds of a character level (SRD 5.2.1, "Level Advancement", p. 23).
const (
	// MinLevel is the lowest character level.
	MinLevel = 1
	// MaxLevel is the highest character level.
	MaxLevel = 20
)

// Level is a character's total level, 1 to 20 (SRD 5.2.1, "Level
// Advancement", Character Advancement table, p. 23). It is an immutable value,
// comparable with ==. The zero Level is invalid; see IsZero. Safe for
// concurrent use.
type Level struct{ n int }

// LevelRejection says why NewLevel rejected a value. The zero
// value is not a rejection.
type LevelRejection int

// The reasons a character level is rejected.
const (
	// LevelBelowMinimum: the value is below 1.
	LevelBelowMinimum LevelRejection = iota + 1
	// LevelAboveMaximum: the value is above 20.
	LevelAboveMaximum
)

// String returns "belowMinimum", "aboveMaximum" or "unknown".
func (r LevelRejection) String() string {
	switch r {
	case LevelBelowMinimum:
		return "belowMinimum"
	case LevelAboveMaximum:
		return "aboveMaximum"
	}
	return "unknown"
}

// LevelError reports a level outside 1 to 20. It matches
// srderr.ErrInvalidInput with errors.Is.
type LevelError struct {
	Value  int
	Reason LevelRejection
}

// Error renders the error. The text is not a contract.
func (e *LevelError) Error() string {
	return "character: invalid character level " + strconv.Itoa(e.Value) + ": " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *LevelError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *LevelError with the same Value and
// Reason.
func (e *LevelError) Is(target error) bool {
	t, ok := target.(*LevelError)
	return ok && t != nil && e != nil && *t == *e
}

// NewLevel returns the level n. It returns a *LevelError
// (matching srderr.ErrInvalidInput) for n outside 1 to 20. It never panics.
func NewLevel(n int) (Level, error) {
	switch {
	case n < MinLevel:
		return Level{}, &LevelError{Value: n, Reason: LevelBelowMinimum}
	case n > MaxLevel:
		return Level{}, &LevelError{Value: n, Reason: LevelAboveMaximum}
	}
	return Level{n: n}, nil
}

// MustLevel is NewLevel for constants known to be valid. It
// panics with the error NewLevel returns, and only then.
func MustLevel(n int) Level {
	l, err := NewLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level as a plain integer, or 0 for the zero Level.
func (l Level) Value() int { return l.n }

// IsZero reports whether l is the invalid zero Level.
func (l Level) IsZero() bool { return l.n == 0 }
