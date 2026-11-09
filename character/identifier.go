package character

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// Identifier is the id of a Character, supplied by the host (the module never
// generates ids). It is an immutable value, comparable with ==. The zero
// Identifier is invalid; see IsZero. Safe for concurrent use.
type Identifier struct{ v string }

// IdentifierReason says why NewIdentifier rejected a value. The zero value is
// not a reason.
type IdentifierReason int

// IdentifierEmpty: the value is empty or only whitespace.
const IdentifierEmpty IdentifierReason = 1

// String returns "empty" or "unknown".
func (r IdentifierReason) String() string {
	if r == IdentifierEmpty {
		return "empty"
	}
	return "unknown"
}

// IdentifierError reports a rejected Character id. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type IdentifierError struct{ Reason IdentifierReason }

// Error renders the error. The text is not a contract.
func (e *IdentifierError) Error() string { return "character: invalid id: " + e.Reason.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *IdentifierError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *IdentifierError with the same Reason.
func (e *IdentifierError) Is(target error) bool {
	t, ok := target.(*IdentifierError)
	return ok && t != nil && e != nil && *t == *e
}

// NewIdentifier returns the id s. It returns a *IdentifierError (matching
// srderr.ErrInvalidInput) with IdentifierEmpty for an empty or whitespace-only
// s. It never panics.
func NewIdentifier(s string) (Identifier, error) {
	if strings.TrimSpace(s) == "" {
		return Identifier{}, &IdentifierError{Reason: IdentifierEmpty}
	}
	return Identifier{v: s}, nil
}

// MustIdentifier is NewIdentifier for constants known to be valid. It panics
// with the error NewIdentifier returns, and only then.
func MustIdentifier(s string) Identifier {
	id, err := NewIdentifier(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the id as the host supplied it.
func (id Identifier) String() string { return id.v }

// IsZero reports whether id is the invalid zero Identifier.
func (id Identifier) IsZero() bool { return id.v == "" }
