package contenttest_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
)

// backend is a store that can fail, the kind of thing a real adapter wraps.
type backend interface {
	get(key content.Key) (content.Entry, bool, error)
	all(kind content.Kind) ([]content.Entry, error)
}

type memBackend struct {
	catalogue content.Catalogue
	err       error // when set, every call fails with it
}

func (b memBackend) get(key content.Key) (content.Entry, bool, error) {
	if b.err != nil {
		return nil, false, b.err
	}
	found, ok := b.catalogue.Lookup(key).(content.LookupFound)
	return found.Entry, ok, nil
}

func (b memBackend) all(kind content.Kind) ([]content.Entry, error) {
	return b.catalogue.OfKind(kind), b.err
}

// wrappingSource is the test-only adapter of source-port FR-027: it wraps a
// backend and turns its failures into *content.SourceError.
type wrappingSource struct{ b backend }

var _ content.Source = wrappingSource{}

func (s wrappingSource) Lookup(_ context.Context, key content.Key) (content.LookupResult, error) {
	e, ok, err := s.b.get(key)
	if err != nil {
		return nil, &content.SourceError{Query: content.LookupQuery{Key: key}, Err: err}
	}
	if !ok {
		return content.LookupNotFound{Key: key}, nil
	}
	return content.LookupFound{Entry: e}, nil
}

func (s wrappingSource) ListKind(_ context.Context, kind content.Kind) ([]content.Entry, error) {
	list, err := s.b.all(kind)
	if err != nil {
		return nil, &content.SourceError{Query: content.ListKindQuery{Kind: kind}, Err: err}
	}
	return list, nil
}

// Requirement: doc/specs/content-source-port.md FR-027, FR-029, FR-047 - the
// contract suite passes InMemorySource, FakeSource (no failing group) and the
// test-only wrapping adapter (with the failing group).
// Path: happy.
func TestSourceContractRegistrations(t *testing.T) {
	t.Run("InMemorySource", func(t *testing.T) {
		contenttest.RunSourceContract(t, func(t *testing.T, entries []content.Entry) content.Source {
			t.Helper()
			s, err := content.NewInMemorySource(entries)
			if err != nil {
				t.Fatal(err)
			}
			return s
		}, nil)
	})
	t.Run("FakeSource", func(t *testing.T) {
		contenttest.RunSourceContract(t, func(_ *testing.T, entries []content.Entry) content.Source {
			return contenttest.NewFakeSource(entries...)
		}, func(_ *testing.T, entries []content.Entry, backendErr error) content.Source {
			return contenttest.NewFakeSource(entries...).FailAll(backendErr)
		})
	})
	t.Run("wrapping adapter", func(t *testing.T) {
		contenttest.RunSourceContract(t, func(_ *testing.T, entries []content.Entry) content.Source {
			return wrappingSource{b: memBackend{catalogue: content.MustCatalogue(entries)}}
		}, func(_ *testing.T, entries []content.Entry, backendErr error) content.Source {
			return wrappingSource{b: memBackend{catalogue: content.MustCatalogue(entries), err: backendErr}}
		})
	})
}

// Requirement: doc/specs/content-source-port.md FR-058 - the read-only suite
// passes a conforming fixed source.
// Path: happy.
func TestReadOnlyContractPassesFixedSource(t *testing.T) {
	entries := []content.Entry{
		entryOf(content.KindSpell, "b"), entryOf(content.KindSpell, "a"), entryOf(content.KindFeat, "a"),
	}
	src, err := content.NewInMemorySource(entries)
	if err != nil {
		t.Fatal(err)
	}
	contenttest.RunReadOnlyContract(t, src, entries)
	contenttest.RunReadOnlyContract(t, contenttest.NewFakeSource(), nil)
}
