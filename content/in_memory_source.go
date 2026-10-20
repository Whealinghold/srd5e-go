package content

import "context"

var _ Source = (*InMemorySource)(nil)

// InMemorySource is a [Source] over a fixed [Catalogue]. It never fails, never
// starts background work and does not consult ctx: an in-memory lookup cannot
// block, the parameter exists to satisfy the port. It is immutable and safe for
// concurrent use. The zero InMemorySource is an empty source.
type InMemorySource struct{ catalogue Catalogue }

// NewInMemorySource builds a source from entries, which it reads once and
// copies; later changes to the slice do not affect it.
//
// It returns the errors of [NewCatalogue] (a *[DuplicateKeyError], or a
// *[SetError] for a nil entry or invalid key), all wrapping
// srderr.ErrInvalidInput. It never panics.
func NewInMemorySource(entries []Entry) (*InMemorySource, error) {
	c, err := NewCatalogue(entries)
	if err != nil {
		return nil, err
	}
	return &InMemorySource{catalogue: c}, nil
}

// Lookup answers from memory per [Source.Lookup]; the error is always nil.
func (s *InMemorySource) Lookup(_ context.Context, key Key) (LookupResult, error) {
	return s.catalogue.Lookup(key), nil
}

// ListKind answers from memory per [Source.ListKind]; the error is always nil.
func (s *InMemorySource) ListKind(_ context.Context, kind Kind) ([]Entry, error) {
	return s.catalogue.OfKind(kind), nil
}
