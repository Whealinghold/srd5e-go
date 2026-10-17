package ability

import "github.com/Whealinghold/srd5e-go/srderr"

// Reason says why an ability value was rejected. The zero Reason is not a
// rejection.
type Reason int

// The rejection reasons. Adding one is a specification change.
const (
	// ReasonUnknownAbility: the text or value names no ability.
	ReasonUnknownAbility Reason = iota + 1
	// ReasonBelowMinimum: a score is below MinScore.
	ReasonBelowMinimum
	// ReasonAboveMaximum: a score is above MaxScore.
	ReasonAboveMaximum
	// ReasonMissingScore: a score is absent or is the zero Score.
	ReasonMissingScore
)

func reasonNames() []string {
	return []string{
		ReasonUnknownAbility: "unknown ability",
		ReasonBelowMinimum:   "below minimum",
		ReasonAboveMaximum:   "above maximum",
		ReasonMissingScore:   "missing score",
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
		return "ability: " + e.Reason.String()
	}
	return "ability: " + e.Reason.String() + ": " + e.Detail
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *InvalidError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is an *InvalidError with the same Reason.
func (e *InvalidError) Is(target error) bool {
	t, ok := target.(*InvalidError)
	return ok && t.Reason == e.Reason
}

func invalid(r Reason, detail string) error { return &InvalidError{Reason: r, Detail: detail} }
