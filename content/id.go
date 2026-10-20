package content

import "github.com/Whealinghold/srd5e-go/srderr"

// MaxIDLength is the longest id, in bytes, that [ParseID] accepts.
const MaxIDLength = 64

// IDProblem says why [ParseID] rejected a string. The zero IDProblem is not
// a rejection.
type IDProblem int

// The id rejection reasons, listed in the order they are checked.
const (
	// IDEmpty: the string is empty.
	IDEmpty IDProblem = iota + 1
	// IDTooLong: the string is longer than [MaxIDLength] bytes.
	IDTooLong
	// IDInvalidFormat: the string does not match [a-z][a-z0-9_]*.
	IDInvalidFormat
)

// String returns the wire spelling of p, or "unknown" for an undefined value.
func (p IDProblem) String() string {
	switch p {
	case IDEmpty:
		return "empty"
	case IDTooLong:
		return "tooLong"
	case IDInvalidFormat:
		return "invalidFormat"
	}
	return "unknown"
}

// IDError reports a string rejected as an [ID]. It matches
// srderr.ErrInvalidInput with errors.Is and carries the rejected value for
// diagnosis; do not branch on the rendered text.
type IDError struct {
	Problem IDProblem
	Value   string
}

// Error renders the problem. The text is not a contract.
func (e *IDError) Error() string { return "content: invalid id: " + e.Problem.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *IDError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is an *IDError with the same Problem.
func (e *IDError) Is(target error) bool {
	t, ok := target.(*IDError)
	return ok && t.Problem == e.Problem
}

// ID names one content entry within its [Kind]: lowercase letters, digits and
// underscores, starting with a letter, at most [MaxIDLength] bytes. An ID can
// only be built by [ParseID] or [MustID], so no conversion bypasses the check.
// The zero ID is invalid (see [ID.IsZero]). ID is comparable, usable as a map
// key and safe for concurrent use.
type ID struct{ v string }

// ParseID validates s and returns it as an [ID].
//
// It returns a *[IDError] wrapping srderr.ErrInvalidInput for the empty string
// (IDEmpty), for a string over [MaxIDLength] bytes (IDTooLong) and for any
// other string that does not match [a-z][a-z0-9_]* (IDInvalidFormat), checked
// in that order. It never panics.
func ParseID(s string) (ID, error) {
	if s == "" { // equivalent mutant: a string is never < "", so == and <= agree
		return ID{}, &IDError{Problem: IDEmpty, Value: s}
	}
	if len(s) > MaxIDLength {
		return ID{}, &IDError{Problem: IDTooLong, Value: s}
	}
	for i := 0; i < len(s); i++ { // equivalent mutant: i starts at 0 and steps by one, so < and != agree
		c := s[i]
		letter := c >= 'a' && c <= 'z'
		if !letter && (i == 0 || (c < '0' || c > '9') && c != '_') { // equivalent mutant: i == 0 and i <= 0 agree because i is never negative
			return ID{}, &IDError{Problem: IDInvalidFormat, Value: s}
		}
	}
	return ID{v: s}, nil
}

// MustID is like [ParseID] but panics with the ParseID error when s is
// invalid. Use it for literals known to be valid.
func MustID(s string) ID {
	id, err := ParseID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the id text, or "" for the zero ID.
func (i ID) String() string { return i.v }

// IsZero reports whether i is the invalid zero ID.
func (i ID) IsZero() bool { return i.v == "" } // equivalent mutant: == and <= agree for strings against ""
