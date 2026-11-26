package combat

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// CombatantID is the id of a creature in a fight, supplied by the host (the
// module never generates ids). It is a comparable value and is not trimmed or
// case-folded. The zero CombatantID (the empty string) is invalid; see IsZero.
// Safe for concurrent use.
type CombatantID string

// CombatantIDRejection says why NewCombatantID rejected a value. The zero value
// is not a rejection.
type CombatantIDRejection int

// CombatantIDEmpty: the value is empty or only whitespace.
const CombatantIDEmpty CombatantIDRejection = 1

// String returns "empty" or "unknown".
func (r CombatantIDRejection) String() string {
	if r == CombatantIDEmpty {
		return "empty"
	}
	return "unknown"
}

// CombatantIDError reports a rejected CombatantID. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type CombatantIDError struct{ Reason CombatantIDRejection }

// Error renders the error. The text is not a contract.
func (e *CombatantIDError) Error() string {
	return "combat: invalid combatant id: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *CombatantIDError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *CombatantIDError with the same Reason.
func (e *CombatantIDError) Is(target error) bool {
	t, ok := target.(*CombatantIDError)
	return ok && t != nil && e != nil && *t == *e
}

// NewCombatantID returns value unchanged as a CombatantID. It returns a
// *CombatantIDError (matching srderr.ErrInvalidInput) with CombatantIDEmpty
// when strings.TrimSpace(value) is empty. It never panics.
func NewCombatantID(value string) (CombatantID, error) {
	if strings.TrimSpace(value) == "" {
		return "", &CombatantIDError{Reason: CombatantIDEmpty}
	}
	return CombatantID(value), nil
}

// MustCombatantID is NewCombatantID for literals known to be valid. It panics
// with the error NewCombatantID returns, and only then.
func MustCombatantID(value string) CombatantID {
	id, err := NewCombatantID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// IsZero reports whether id is the invalid zero CombatantID.
func (id CombatantID) IsZero() bool { return id == "" }
