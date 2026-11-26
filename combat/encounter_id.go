package combat

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// EncounterID is the id of an Encounter, supplied by the host (the module never
// generates ids). It is a comparable value, not trimmed or case-folded, and a
// distinct type from CombatantID. The zero EncounterID (the empty string) is
// invalid; see IsZero. Safe for concurrent use.
type EncounterID string

// EncounterIDRejection says why ParseEncounterID rejected a value. The zero value
// is not a rejection.
type EncounterIDRejection int

// EncounterIDEmpty: the value is empty or only whitespace.
const EncounterIDEmpty EncounterIDRejection = 1

// String returns "empty" or "unknown".
func (r EncounterIDRejection) String() string {
	if r == EncounterIDEmpty {
		return "empty"
	}
	return "unknown"
}

// EncounterIDError reports a rejected EncounterID. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type EncounterIDError struct{ Reason EncounterIDRejection }

// Error renders the error. The text is not a contract.
func (e *EncounterIDError) Error() string {
	return "combat: invalid encounter id: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *EncounterIDError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *EncounterIDError with the same Reason.
func (e *EncounterIDError) Is(target error) bool {
	t, ok := target.(*EncounterIDError)
	return ok && t != nil && e != nil && *t == *e
}

// ParseEncounterID returns value unchanged as an EncounterID. It returns a
// *EncounterIDError (matching srderr.ErrInvalidInput) with EncounterIDEmpty when
// strings.TrimSpace(value) is empty. It never panics.
func ParseEncounterID(value string) (EncounterID, error) {
	if strings.TrimSpace(value) == "" {
		return "", &EncounterIDError{Reason: EncounterIDEmpty}
	}
	return EncounterID(value), nil
}

// MustEncounterID is ParseEncounterID for literals known to be valid. It panics
// with the error ParseEncounterID returns, and only then.
func MustEncounterID(value string) EncounterID {
	id, err := ParseEncounterID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// IsZero reports whether id is the invalid zero EncounterID.
func (id EncounterID) IsZero() bool { return id == "" }
