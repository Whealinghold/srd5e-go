package content_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/content-source-port.md FR-007 (compile-time) - the
// InMemorySource satisfies Source, and the methods take ctx first and return
// (LookupResult, error) and ([]Entry, error).
// Path: happy.
var (
	_ content.Source                                                   = (*content.InMemorySource)(nil)
	_ func(context.Context, content.Key) (content.LookupResult, error) = (*content.InMemorySource)(nil).Lookup
	_ func(context.Context, content.Kind) ([]content.Entry, error)     = (*content.InMemorySource)(nil).ListKind
)

// Requirement: doc/specs/content-source-port.md FR-020, FR-021, FR-023 -
// NewInMemorySource rejects duplicates with the catalogue's errors and returns
// no source, and copies its input.
// Path: unhappy (invalid input).
func TestNewInMemorySource(t *testing.T) {
	t.Parallel()
	a := plain(content.KindSpell, "a")
	src, err := content.NewInMemorySource(asEntries(a, a))
	var dk *content.DuplicateKeyError
	if src != nil || !errors.As(err, &dk) || dk.Key != a.Key() || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewInMemorySource(duplicate) = (%v, %v), want nil and *DuplicateKeyError", src, err)
	}

	in := asEntries(a)
	src, err = content.NewInMemorySource(in)
	if err != nil {
		t.Fatal(err)
	}
	clear(in)
	res, err := src.Lookup(context.Background(), a.Key())
	if err != nil || res != (content.LookupFound{Entry: a}) {
		t.Errorf("Lookup after clearing the input = (%v, %v), want the entry", res, err)
	}
}

// Requirement: doc/specs/content-source-port.md FR-022, FR-024 - the zero
// InMemorySource and one built from no entries are empty, and neither ever
// returns an error.
// Path: unhappy (empty case).
func TestInMemorySourceEmpty(t *testing.T) {
	t.Parallel()
	built, err := content.NewInMemorySource(nil)
	if err != nil {
		t.Fatal(err)
	}
	key := plain(content.KindSpell, "a").Key()
	for name, src := range map[string]content.Source{"built": built, "zero": &content.InMemorySource{}} {
		res, err := src.Lookup(context.Background(), key)
		if err != nil || res != (content.LookupNotFound{Key: key}) {
			t.Errorf("%s: Lookup = (%v, %v), want LookupNotFound", name, res, err)
		}
		list, err := src.ListKind(context.Background(), content.KindSpell)
		if err != nil || list == nil || len(list) != 0 {
			t.Errorf("%s: ListKind = (%v, %v), want empty and nil error", name, list, err)
		}
	}
}

// Requirement: doc/specs/content-source-port.md FR-024 - an already-cancelled
// context does not change the result: the source does not consult ctx and
// returns no error.
// Path: unhappy (cancellation).
func TestInMemorySourceIgnoresCancelledContext(t *testing.T) {
	t.Parallel()
	a, b := plain(content.KindSpell, "a"), plain(content.KindSpell, "b")
	src, err := content.NewInMemorySource(asEntries(b, a))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := src.Lookup(ctx, a.Key())
	if err != nil || res != (content.LookupFound{Entry: a}) {
		t.Errorf("Lookup with a cancelled ctx = (%v, %v), want the entry and nil error", res, err)
	}
	list, err := src.ListKind(ctx, content.KindSpell)
	if err != nil || !slices.Equal(idsOf(list), []string{"a", "b"}) {
		t.Errorf("ListKind with a cancelled ctx = (%v, %v), want [a b] and nil error", idsOf(list), err)
	}
}

// Requirement: doc/specs/content-source-port.md FR-024 - an InMemorySource is
// safe for concurrent use; run under -race.
// Path: happy.
func TestInMemorySourceConcurrentUse(t *testing.T) {
	t.Parallel()
	a, b := plain(content.KindSpell, "a"), plain(content.KindFeat, "b")
	src, err := content.NewInMemorySource(asEntries(a, b))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			if res, err := src.Lookup(ctx, a.Key()); err != nil || res != (content.LookupFound{Entry: a}) {
				t.Errorf("Lookup = (%v, %v)", res, err)
			}
			list, err := src.ListKind(ctx, content.KindFeat)
			if err != nil || !slices.Equal(idsOf(list), []string{"b"}) {
				t.Errorf("ListKind = (%v, %v)", idsOf(list), err)
			}
			clear(list)
		}()
	}
	wg.Wait()
}
