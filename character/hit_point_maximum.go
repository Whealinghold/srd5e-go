package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// HitPointMaximum is a Hit Point maximum, at least 1 (SRD 5.2.1, "Hit Points",
// p. 22; "Gaining a Level", p. 23). It is the stored base: the reported maximum
// adds bonuses at read time and never stores them. It is an immutable value,
// comparable with ==. The zero HitPointMaximum is invalid; see IsZero. Safe for
// concurrent use.
type HitPointMaximum struct{ n int }

// HitPointMaximumRejection says why NewHitPointMaximum rejected a value. The
// zero value is not a rejection.
type HitPointMaximumRejection int

// HitPointMaximumBelowMinimum: the value is below 1.
const HitPointMaximumBelowMinimum HitPointMaximumRejection = 1

// String returns "belowMinimum" or "unknown".
func (r HitPointMaximumRejection) String() string {
	if r == HitPointMaximumBelowMinimum {
		return "belowMinimum"
	}
	return "unknown"
}

// HitPointMaximumError reports a Hit Point maximum below 1. It matches
// srderr.ErrInvalidInput with errors.Is.
type HitPointMaximumError struct {
	Value  int
	Reason HitPointMaximumRejection
}

// Error renders the error. The text is not a contract.
func (e *HitPointMaximumError) Error() string {
	return "character: invalid Hit Point maximum " + strconv.Itoa(e.Value) + ": " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *HitPointMaximumError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *HitPointMaximumError with the same Value and
// Reason.
func (e *HitPointMaximumError) Is(target error) bool {
	t, ok := target.(*HitPointMaximumError)
	return ok && t != nil && e != nil && *t == *e
}

// NewHitPointMaximum returns the maximum n. It returns a *HitPointMaximumError
// (matching srderr.ErrInvalidInput) for n below 1. It never panics.
func NewHitPointMaximum(n int) (HitPointMaximum, error) {
	if n < 1 {
		return HitPointMaximum{}, &HitPointMaximumError{Value: n, Reason: HitPointMaximumBelowMinimum}
	}
	return HitPointMaximum{n: n}, nil
}

// MustHitPointMaximum is NewHitPointMaximum for constants known to be valid. It
// panics with the error NewHitPointMaximum returns, and only then.
func MustHitPointMaximum(n int) HitPointMaximum {
	m, err := NewHitPointMaximum(n)
	if err != nil {
		panic(err)
	}
	return m
}

// Value returns the maximum as a plain integer, or 0 for the zero value.
func (m HitPointMaximum) Value() int { return m.n }

// IsZero reports whether m is the invalid zero HitPointMaximum.
func (m HitPointMaximum) IsZero() bool { return m.n == 0 }

// Plus returns a new maximum n higher. The type never holds less than 1, so a
// result below 1 is raised to 1; the zero value stays zero.
func (m HitPointMaximum) Plus(n int) HitPointMaximum {
	if m.IsZero() {
		return m
	}
	return HitPointMaximum{n: max(1, m.n+n)}
}
