package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
)

// scenarioEntry is a synthetic Entry; no SRD content is involved.
type scenarioEntry struct{ key content.Key }

func (e scenarioEntry) Key() content.Key { return e.key }

func scenarioSpell(id string) scenarioEntry {
	return scenarioEntry{content.NewKey(content.KindSpell, content.MustID(id))}
}

// Requirement: doc/specs/content-source-port.md FR-043 - an InMemorySource is
// built, a present and an absent key are looked up, ListKind is called, and the
// LookupResult is switched over exhaustively with no default.
// Path: happy.
func TestInMemorySourceServesLookupsAndLists(t *testing.T) {
	t.Parallel()
	b, a := scenarioSpell("b"), scenarioSpell("a")
	src, err := content.NewInMemorySource([]content.Entry{b, a})
	if err != nil {
		t.Fatalf("NewInMemorySource: %v", err)
	}
	ctx := context.Background()

	absent := content.NewKey(content.KindSpell, content.MustID("zz"))
	for _, key := range []content.Key{a.Key(), absent} {
		res, err := src.Lookup(ctx, key)
		if err != nil {
			t.Fatalf("Lookup(%v): %v", key, err)
		}
		switch r := res.(type) {
		case content.LookupFound:
			if key != a.Key() || r.Entry != content.Entry(a) {
				t.Errorf("Lookup(%v) found %v, want the entry a", key, r.Entry)
			}
		case content.LookupNotFound:
			if key != absent || r.Key != absent || r.Missing().Key() != absent {
				t.Errorf("Lookup(%v) not found for %v, want the requested key", key, r.Key)
			}
		}
	}

	list, err := src.ListKind(ctx, content.KindSpell)
	if err != nil || len(list) != 2 || list[0] != content.Entry(a) || list[1] != content.Entry(b) {
		t.Errorf("ListKind(spell) = (%v, %v), want [a b] in ID order", list, err)
	}
	if none, err := src.ListKind(ctx, content.KindMonster); err != nil || len(none) != 0 {
		t.Errorf("ListKind(monster) = (%v, %v), want empty", none, err)
	}
}

// Requirement: doc/specs/content-source-port.md FR-029, FR-043 - the shipped
// double and the real source answer the same through the public API.
// Path: happy.
func TestFakeSourceAgreesWithInMemorySource(t *testing.T) {
	t.Parallel()
	entries := []content.Entry{scenarioSpell("b"), scenarioSpell("a")}
	real, err := content.NewInMemorySource(entries)
	if err != nil {
		t.Fatal(err)
	}
	fake := contenttest.NewFakeSource(entries...)
	ctx := context.Background()
	want, _ := real.ListKind(ctx, content.KindSpell)
	got, err := fake.ListKind(ctx, content.KindSpell)
	if err != nil || len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("fake ListKind = (%v, %v), want %v", got, err, want)
	}
	if q := fake.Queries(); len(q) != 1 || q[0] != (content.ListKindQuery{Kind: content.KindSpell}) {
		t.Errorf("fake recorded %v, want one ListKindQuery", q)
	}
}
