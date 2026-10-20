package content

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// SetProblem says why [NewSet] rejected its entries. The zero value is not a
// rejection.
type SetProblem int

// The set rejection reasons.
const (
	// SetNilEntry: an entry is nil.
	SetNilEntry SetProblem = iota + 1
	// SetInvalidKey: an entry's key has an invalid kind or the zero id.
	SetInvalidKey
	// SetDuplicateKey: two entries have the same key.
	SetDuplicateKey
)

// String returns the lowercase name of p, or "unknown" for an undefined value.
func (p SetProblem) String() string {
	switch p {
	case SetNilEntry:
		return "nilEntry"
	case SetInvalidKey:
		return "invalidKey"
	case SetDuplicateKey:
		return "duplicateKey"
	}
	return "unknown"
}

// SetError reports entries rejected by [NewSet]. Key is the offending key
// (the zero Key for SetNilEntry). It matches srderr.ErrInvalidInput with
// errors.Is.
type SetError struct {
	Problem SetProblem
	Key     Key
}

// Error renders the problem. The text is not a contract.
func (e *SetError) Error() string { return "content: invalid set: " + e.Problem.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *SetError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *SetError with the same Problem.
func (e *SetError) Is(target error) bool {
	t, ok := target.(*SetError)
	return ok && t.Problem == e.Problem
}

// Set is an immutable collection of entries with distinct keys, ordered by
// [Kind] (declaration order) then id (ordinal), whatever the order they were
// given in. The zero Set is empty and valid. A Set checks only keys, never
// references between entries. A Set holds a slice, so it is not comparable
// with ==; use [Set.Equal]. Safe for concurrent use.
type Set struct{ entries []Entry }

// NewSet returns the set of entries, ordered by key.
//
// It returns a *[SetError] wrapping srderr.ErrInvalidInput for a nil entry or a pointer entry,
// an entry whose key is invalid, and two entries with one key. It never
// panics. The argument slice is copied.
func NewSet(entries ...Entry) (Set, error) {
	sorted := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if isNilEntry(e) {
			return Set{}, &SetError{Problem: SetNilEntry}
		}
		if !e.Key().valid() {
			return Set{}, &SetError{Problem: SetInvalidKey, Key: e.Key()}
		}
		sorted = append(sorted, e)
	}
	slices.SortStableFunc(sorted, compareKeys)
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Key() == sorted[i-1].Key() {
			return Set{}, &SetError{Problem: SetDuplicateKey, Key: sorted[i].Key()}
		}
	}
	return Set{entries: sorted}, nil
}

// compareKeys orders entries by kind (declaration order) then id (ordinal).
func compareKeys(a, b Entry) int {
	return cmp.Or(cmp.Compare(a.Key().Kind(), b.Key().Kind()), cmp.Compare(a.Key().ID().String(), b.Key().ID().String()))
}

// MustSet is like [NewSet] but panics with its error when the entries are
// invalid.
func MustSet(entries ...Entry) Set {
	s, err := NewSet(entries...)
	if err != nil {
		panic(err)
	}
	return s
}

// Entries returns the entries ordered by kind then id, as a new slice that the
// caller may modify without affecting s.
func (s Set) Entries() []Entry { return slices.Clone(s.entries) }

// Equal reports whether s and o hold the same entries in the same order.
//
// shortcut: entries are compared by their %#v rendering, which is exact for
// the immutable value entries the Entry contract requires but would follow
// pointers by address; give entries an explicit equality if one ever holds a
// pointer.
func (s Set) Equal(o Set) bool {
	return slices.EqualFunc(s.entries, o.entries, func(a, b Entry) bool {
		return fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b)
	})
}

// isNilEntry reports whether e is nil or a pointer (nil or not): an Entry is an
// immutable value, and a typed nil would panic on its first method call. The
// type name is read with %T because the package may not use reflection.
func isNilEntry(e Entry) bool {
	return e == nil || strings.HasPrefix(fmt.Sprintf("%T", e), "*")
}
