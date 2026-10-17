package d20

import "github.com/Whealinghold/srd5e-go/srderr"

// Reason says why a value or request was rejected.
type Reason int

// The reasons an [InvalidError] carries.
const (
	// ReasonIndexOutOfRange: a die index is outside the roll.
	ReasonIndexOutOfRange Reason = iota + 1
	// ReasonFaceOutOfRange: a d20 face is outside 1 to 20.
	ReasonFaceOutOfRange
	// ReasonUnknownKind: the kind is not one of the three kinds.
	ReasonUnknownKind
	// ReasonKindMismatch: an attack roll was paired with a Difficulty Class
	// or a check with an Armor Class.
	ReasonKindMismatch
	// ReasonCriticalRange: the critical range is not 18, 19 or 20.
	ReasonCriticalRange
	// ReasonForcedFailureNotAllowed: the SRD gives no rule for this forced
	// failure (an attack roll, or an ability check chosen to fail).
	ReasonForcedFailureNotAllowed
	// ReasonUnknownReason: the forced failure reason is not valid.
	ReasonUnknownReason
	// ReasonUnknownDecision: the reroll decision is nil or not a variant.
	ReasonUnknownDecision
	// ReasonZeroRoll: the zero Roll was used where a rolled one is needed.
	ReasonZeroRoll
	// ReasonUnknownAbility: the command's ability label is not valid.
	ReasonUnknownAbility
	// ReasonMissingAttempt: the command has no attempt, or an unknown one.
	ReasonMissingAttempt
	// ReasonUnknownCommand: the command is nil or not a command variant.
	ReasonUnknownCommand
	// ReasonMissingRoller: the resolver was built without a roller.
	ReasonMissingRoller
)

func reasonNames() []string {
	return []string{
		ReasonIndexOutOfRange:         "index out of range",
		ReasonFaceOutOfRange:          "face out of range",
		ReasonUnknownKind:             "unknown kind",
		ReasonKindMismatch:            "kind mismatch",
		ReasonCriticalRange:           "critical range out of range",
		ReasonForcedFailureNotAllowed: "forced failure not allowed",
		ReasonUnknownReason:           "unknown forced failure reason",
		ReasonUnknownDecision:         "unknown reroll decision",
		ReasonZeroRoll:                "zero roll",
		ReasonUnknownAbility:          "unknown ability",
		ReasonMissingAttempt:          "missing attempt",
		ReasonUnknownCommand:          "unknown command",
		ReasonMissingRoller:           "missing roller",
	}
}

// String returns the reason in words, or "unknown" for an undefined reason.
func (r Reason) String() string {
	if r < 1 || int(r) >= len(reasonNames()) {
		return "unknown"
	}
	return reasonNames()[r]
}

// InvalidError reports input this package rejects. It matches
// [srderr.ErrInvalidInput] with errors.Is, and another *InvalidError with the
// same Reason.
type InvalidError struct {
	Reason Reason
	Detail string
}

// Error returns "d20: <reason>" with the detail appended when present.
func (e *InvalidError) Error() string {
	if e.Detail == "" {
		return "d20: " + e.Reason.String()
	}
	return "d20: " + e.Reason.String() + ": " + e.Detail
}

// Unwrap returns [srderr.ErrInvalidInput].
func (e *InvalidError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is an *InvalidError with the same Reason.
func (e *InvalidError) Is(target error) bool {
	t, ok := target.(*InvalidError)
	return ok && t.Reason == e.Reason
}

func invalid(r Reason, detail string) error { return &InvalidError{Reason: r, Detail: detail} }
