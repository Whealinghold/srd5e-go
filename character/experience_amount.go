package character

import "github.com/Whealinghold/srd5e-go/srderr"

// ExperienceAmount is a number of Experience Points awarded at once: a positive
// whole number. It is an immutable value, comparable with ==. The zero
// ExperienceAmount is invalid; see IsZero. Safe for concurrent use.
type ExperienceAmount struct{ n int }

// ExperienceAmountReason says why NewExperienceAmount rejected a value. The
// zero value is not a reason.
type ExperienceAmountReason int

// The reasons an award is rejected.
const (
	// ExperienceAmountZero: the value is 0.
	ExperienceAmountZero ExperienceAmountReason = iota + 1
	// ExperienceAmountNegative: the value is below 0.
	ExperienceAmountNegative
)

// String returns "zero", "negative" or "unknown".
func (r ExperienceAmountReason) String() string {
	switch r {
	case ExperienceAmountZero:
		return "zero"
	case ExperienceAmountNegative:
		return "negative"
	}
	return "unknown"
}

// ExperienceAmountError reports a rejected award. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type ExperienceAmountError struct{ Reason ExperienceAmountReason }

// Error renders the error. The text is not a contract.
func (e *ExperienceAmountError) Error() string {
	return "character: invalid Experience Points award: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ExperienceAmountError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ExperienceAmountError with the same Reason.
func (e *ExperienceAmountError) Is(target error) bool {
	t, ok := target.(*ExperienceAmountError)
	return ok && t != nil && e != nil && *t == *e
}

// NewExperienceAmount returns the award n. It returns a *ExperienceAmountError
// (matching srderr.ErrInvalidInput) for 0 or a negative n. It never panics.
func NewExperienceAmount(n int) (ExperienceAmount, error) {
	switch {
	case n == 0:
		return ExperienceAmount{}, &ExperienceAmountError{Reason: ExperienceAmountZero}
	case n < 0:
		return ExperienceAmount{}, &ExperienceAmountError{Reason: ExperienceAmountNegative}
	}
	return ExperienceAmount{n: n}, nil
}

// MustExperienceAmount is NewExperienceAmount for constants known to be valid.
// It panics with the error NewExperienceAmount returns, and only then.
func MustExperienceAmount(n int) ExperienceAmount {
	a, err := NewExperienceAmount(n)
	if err != nil {
		panic(err)
	}
	return a
}

// Value returns the award as a plain integer, or 0 for the zero value.
func (a ExperienceAmount) Value() int { return a.n }

// IsZero reports whether a is the invalid zero ExperienceAmount.
func (a ExperienceAmount) IsZero() bool { return a.n == 0 }
