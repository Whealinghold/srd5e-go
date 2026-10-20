package content

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// DuplicateKeyError reports that two entries share one [Key]. It wraps
// srderr.ErrInvalidInput; find it with errors.As.
type DuplicateKeyError struct{ Key Key }

// Error describes the duplicate for logs.
func (e *DuplicateKeyError) Error() string { return "content: duplicate key" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *DuplicateKeyError) Unwrap() error { return srderr.ErrInvalidInput }

// Catalogue is an immutable, validated index of entries: it holds no duplicate
// key, answers Lookup as a [LookupResult] and OfKind as an ordered copy. It is
// pure and synchronous, and safe for concurrent use. The zero Catalogue is
// empty and usable. The same id under different kinds is allowed.
type Catalogue struct {
	index  map[Key]Entry
	byKind map[Kind][]Entry
}

// NewCatalogue reads entries once, copies them, and indexes them.
//
// It returns a *[DuplicateKeyError] when two entries share a key, and the
// *[SetError] of [NewSet] for a nil entry or an entry whose key is not valid;
// both wrap srderr.ErrInvalidInput. It never panics.
func NewCatalogue(entries []Entry) (Catalogue, error) {
	set, err := NewSet(entries...)
	if err != nil {
		var se *SetError
		if errors.As(err, &se) && se.Problem == SetDuplicateKey {
			return Catalogue{}, &DuplicateKeyError{Key: se.Key}
		}
		return Catalogue{}, err
	}
	all := set.Entries() // ordered by kind, then ID
	c := Catalogue{index: make(map[Key]Entry, len(all)), byKind: map[Kind][]Entry{}}
	for _, e := range all {
		k := e.Key()
		c.index[k] = e
		c.byKind[k.Kind()] = append(c.byKind[k.Kind()], e)
	}
	return c, nil
}

// MustCatalogue is [NewCatalogue] that panics with its error. Use it only for
// fixed, known-good entries.
func MustCatalogue(entries []Entry) Catalogue {
	c, err := NewCatalogue(entries)
	if err != nil {
		panic(err)
	}
	return c
}

// Lookup returns [LookupFound] for the entry with key, else [LookupNotFound]
// carrying key. An id present only under another kind is not found.
func (c Catalogue) Lookup(key Key) LookupResult {
	if e, ok := c.index[key]; ok {
		return LookupFound{Entry: e}
	}
	return LookupNotFound{Key: key}
}

// OfKind returns the entries of kind ordered by ID in byte order, as a copy;
// empty (not nil) when there are none.
func (c Catalogue) OfKind(kind Kind) []Entry {
	return append(make([]Entry, 0, len(c.byKind[kind])), c.byKind[kind]...)
}
