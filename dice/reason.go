package dice

import "github.com/Whealinghold/srd5e-go/srderr"

// Reason says why a dice value was rejected. The zero Reason is not a
// rejection.
type Reason int

// The rejection reasons. Adding one is a specification change.
const (
	// ReasonEmpty: the notation text was zero-length.
	ReasonEmpty Reason = iota + 1
	// ReasonMalformed: the notation text is not dice notation.
	ReasonMalformed
	// ReasonCountOutOfRange: the dice count is outside 1..MaxCount.
	ReasonCountOutOfRange
	// ReasonSidesOutOfRange: the die sides are outside MinSides..MaxSides.
	ReasonSidesOutOfRange
	// ReasonModifierOutOfRange: the modifier magnitude exceeds MaxModifier.
	ReasonModifierOutOfRange
	// ReasonValueOutOfRange: a face value is outside 1..sides of its die.
	ReasonValueOutOfRange
	// ReasonRollCountMismatch: a Result holds a different number of rolls
	// than its expression has dice.
	ReasonRollCountMismatch
	// ReasonDieMismatch: a roll in a Result is of a different die than its
	// expression.
	ReasonDieMismatch
	// ReasonSelectionOutOfRange: a count of dice to select is outside
	// 1..count.
	ReasonSelectionOutOfRange
)

func reasonNames() []string {
	return []string{
		ReasonEmpty:               "empty",
		ReasonMalformed:           "malformed",
		ReasonCountOutOfRange:     "count out of range",
		ReasonSidesOutOfRange:     "sides out of range",
		ReasonModifierOutOfRange:  "modifier out of range",
		ReasonValueOutOfRange:     "value out of range",
		ReasonRollCountMismatch:   "roll count mismatch",
		ReasonDieMismatch:         "die mismatch",
		ReasonSelectionOutOfRange: "selection out of range",
	}
}

// String returns the lowercase name of r, or "unknown" for an undefined value.
func (r Reason) String() string {
	if r < 1 || int(r) >= len(reasonNames()) {
		return "unknown"
	}
	return reasonNames()[r]
}

// InvalidError reports untrusted input rejected by this package.
//
// It matches srderr.ErrInvalidInput with errors.Is, and matches any other
// *InvalidError with the same Reason (Detail is ignored), so callers can write
// errors.Is(err, &dice.InvalidError{Reason: dice.ReasonMalformed}).
type InvalidError struct {
	Reason Reason
	// Detail is optional human-readable context; do not branch on it.
	Detail string
}

// Error renders the reason and detail. The text is not a contract.
func (e *InvalidError) Error() string {
	if e.Detail == "" {
		return "dice: " + e.Reason.String()
	}
	return "dice: " + e.Reason.String() + ": " + e.Detail
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *InvalidError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is an *InvalidError with the same Reason.
func (e *InvalidError) Is(target error) bool {
	t, ok := target.(*InvalidError)
	return ok && t.Reason == e.Reason
}

func invalid(r Reason, detail string) error { return &InvalidError{Reason: r, Detail: detail} }
