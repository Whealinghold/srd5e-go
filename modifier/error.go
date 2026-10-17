package modifier

import "github.com/Whealinghold/srd5e-go/srderr"

// Reason says why a value was rejected. The zero Reason is not a rejection.
type Reason int

// The rejection reasons. Adding one is a specification change.
const (
	// ReasonRollCountMismatch: the number of d20 values does not match the
	// roll mode.
	ReasonRollCountMismatch Reason = iota + 1
	// ReasonFaceOutOfRange: a d20 value is outside 1..20.
	ReasonFaceOutOfRange
	// ReasonUnknownRollMode: the text or value names no roll mode.
	ReasonUnknownRollMode
)

func reasonNames() []string {
	return []string{
		ReasonRollCountMismatch: "roll count mismatch",
		ReasonFaceOutOfRange:    "face out of range",
		ReasonUnknownRollMode:   "unknown roll mode",
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
// *InvalidError with the same Reason (Detail is ignored).
type InvalidError struct {
	Reason Reason
	// Detail is optional human-readable context; do not branch on it.
	Detail string
}

// Error renders the reason and detail. The text is not a contract.
func (e *InvalidError) Error() string {
	if e.Detail == "" {
		return "modifier: " + e.Reason.String()
	}
	return "modifier: " + e.Reason.String() + ": " + e.Detail
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *InvalidError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is an *InvalidError with the same Reason.
func (e *InvalidError) Is(target error) bool {
	t, ok := target.(*InvalidError)
	return ok && t.Reason == e.Reason
}

func invalid(r Reason, detail string) error { return &InvalidError{Reason: r, Detail: detail} }
