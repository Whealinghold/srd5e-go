package data_test

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var _ content.Source = (*data.BundledSource)(nil)

func spellKey(id string) content.Key { return content.NewKey(content.KindSpell, content.MustID(id)) }

// setOf builds a set of synthetic spell entries for the injected decoder.
func setOf(t *testing.T, ids ...string) content.Set {
	t.Helper()
	var entries []content.Entry
	for _, id := range ids {
		entries = append(entries, syntheticEntry{key: spellKey(id)})
	}
	set, err := content.NewSet(entries...)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

// syntheticEntry is an entry with no content, enough for the decoder double.
type syntheticEntry struct{ key content.Key }

func (e syntheticEntry) Key() content.Key { return e.key }

// Requirement: doc/specs/content-source-port.md FR-059, FR-060, FR-063 - an injected decoder sees exactly one decode
// call per text, in lexical name order whatever the map order, and the sets it returns are merged and served.
// Path: happy.
func TestBundledSourceDecodesEachTextOnce(t *testing.T) {
	t.Parallel()
	fake := contenttest.NewFakeDecoder().Return("A", setOf(t, "a1"), nil).Return("B", setOf(t, "b1", "b2"), nil)
	src, err := data.NewBundledSource(data.WithDocuments(map[string]string{"b.json": "B", "a.json": "A"}), data.WithDecoder(fake))
	if err != nil {
		t.Fatal(err)
	}
	var inputs []string
	for _, in := range fake.Inputs() {
		inputs = append(inputs, string(in))
	}
	if !slices.Equal(inputs, []string{"A", "B"}) {
		t.Errorf("decoder inputs = %v, want one call per text in name order", inputs)
	}
	got, err := src.ListKind(context.Background(), content.KindSpell)
	if err != nil || !slices.Equal(idsOf(got), []string{"a1", "b1", "b2"}) {
		t.Errorf("ListKind(spell) = %v, %v; want the three merged entries", idsOf(got), err)
	}
}

// Requirement: doc/specs/content-source-port.md FR-062, FR-022 - an empty index (empty or nil map) behaves as an empty
// source: no decode call, every lookup is not found and every list is empty.
// Path: unhappy (empty case).
func TestBundledSourceEmptyIndex(t *testing.T) {
	t.Parallel()
	for name, docs := range map[string]map[string]string{"empty": {}, "nil": nil} {
		fake := contenttest.NewFakeDecoder()
		src, err := data.NewBundledSource(data.WithDocuments(docs), data.WithDecoder(fake))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(fake.Inputs()) != 0 {
			t.Errorf("%s: decoder was called %d times, want 0", name, len(fake.Inputs()))
		}
		for k := content.KindSpecies; k.Valid(); k++ {
			if got, err := src.ListKind(context.Background(), k); err != nil || len(got) != 0 {
				t.Errorf("%s: ListKind(%s) = %v, %v; want empty", name, k, got, err)
			}
		}
		res, err := src.Lookup(context.Background(), spellKey("fire_bolt"))
		if _, ok := res.(content.LookupNotFound); err != nil || !ok {
			t.Errorf("%s: Lookup = %v, %v; want LookupNotFound", name, res, err)
		}
	}
}

// Requirement: doc/specs/content-source-port.md FR-062, FR-014 - building twice from the same texts gives equal,
// equally ordered results, and a list the caller clears does not change the source.
// Path: unhappy (determinism, immutability).
func TestBundledSourceDeterministicAndCopies(t *testing.T) {
	t.Parallel()
	a, b := newSlice(t), newSlice(t)
	for k := content.KindSpecies; k.Valid(); k++ {
		la, _ := a.ListKind(context.Background(), k)
		lb, _ := b.ListKind(context.Background(), k)
		if !reflect.DeepEqual(la, lb) {
			t.Errorf("ListKind(%s) differs between two builds", k)
		}
	}
	list, _ := a.ListKind(context.Background(), content.KindSpell)
	clear(list)
	again, _ := a.ListKind(context.Background(), content.KindSpell)
	if len(again) != 12 || again[0] == nil {
		t.Errorf("ListKind after clearing a returned list = %d entries, first %v; want the 12 spells intact", len(again), again[0])
	}
}

// Requirement: doc/specs/content-source-port.md FR-060, FR-059 - construction runs synchronously and leaves no
// background goroutine behind. Not parallel, so no other test of the package is running while it counts.
// Path: happy.
func TestBundledSourceStartsNoGoroutine(t *testing.T) {
	before := runtime.NumGoroutine()
	src := newSlice(t)
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines %d -> %d after NewBundledSource, want no new one", before, after)
	}
	runtime.KeepAlive(src)
}

// Requirement: doc/specs/content-source-port.md FR-061 - a text that does not decode returns a nil source and a
// *BundledInvalidError whose DecodeProblems equal the decoder's list and whose ReferenceProblems are empty, matching
// srderr.ErrInvalidInput; a dangling reference in another text is not examined because validation runs only when
// every document decoded.
// Path: unhappy (invalid bundle).
func TestBundledInvalidErrorFromACorruptedText(t *testing.T) {
	t.Parallel()
	species := embeddedDocuments(t)["species.json"] // dangling on its own: the spells are not in this index
	_, direct := content.JSONDecoder{}.Decode([]byte("not json"))
	src, err := data.NewBundledSource(data.WithDocuments(map[string]string{"a.json": "not json", "b.json": species}))
	var bad *data.BundledInvalidError
	if src != nil || !errors.As(err, &bad) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewBundledSource(corrupted) = (%v, %v), want nil and a *BundledInvalidError matching ErrInvalidInput", src, err)
	}
	if want := content.Problems(direct); !slices.Equal(bad.DecodeProblems, want) || len(want) != 1 || len(bad.ReferenceProblems) != 0 {
		t.Errorf("DecodeProblems = %v and ReferenceProblems = %v, want %v and none", bad.DecodeProblems, bad.ReferenceProblems, want)
	}
}

// Requirement: doc/specs/content-source-port.md FR-061 - DecodeProblems are ordered document by document in index order.
// Path: unhappy (invalid bundle: two failing documents).
func TestDecodeProblemsFollowTheIndexOrder(t *testing.T) {
	t.Parallel()
	_, first := content.JSONDecoder{}.Decode([]byte("not json"))
	_, second := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":2}`))
	_, err := data.NewBundledSource(data.WithDocuments(map[string]string{"b.json": `{"schemaVersion":2}`, "a.json": "not json"}))
	var bad *data.BundledInvalidError
	if !errors.As(err, &bad) {
		t.Fatalf("err = %v, want a *BundledInvalidError", err)
	}
	want := append(content.Problems(first), content.Problems(second)...)
	if !slices.Equal(bad.DecodeProblems, want) || len(want) != 2 || want[0] == want[1] {
		t.Errorf("DecodeProblems = %v, want %v (document a then b)", bad.DecodeProblems, want)
	}
}

// Requirement: doc/specs/content-source-port.md FR-061, FR-005 - the same key in two documents is a duplicate
// ReferenceProblem with empty DecodeProblems, never a *content.DuplicateKeyError and never a source.
// Path: unhappy (duplicate).
func TestDuplicateKeyAcrossDocuments(t *testing.T) {
	t.Parallel()
	spells := embeddedDocuments(t)["spells.json"]
	src, err := data.NewBundledSource(data.WithDocuments(map[string]string{"a.json": spells, "b.json": spells}))
	var bad *data.BundledInvalidError
	var dup *content.DuplicateKeyError
	if src != nil || !errors.As(err, &bad) || errors.As(err, &dup) || len(bad.DecodeProblems) != 0 {
		t.Fatalf("NewBundledSource(duplicate) = (%v, %v), want nil and a *BundledInvalidError with only reference problems", src, err)
	}
	if len(bad.ReferenceProblems) != 12 {
		t.Fatalf("ReferenceProblems = %d, want one per spell", len(bad.ReferenceProblems))
	}
	for _, p := range bad.ReferenceProblems {
		if p.Kind != content.ProblemDuplicate || p.Owner.Kind() != content.KindSpell || p.Target != p.Owner {
			t.Errorf("reference problem = %+v, want a duplicate of a spell key", p)
		}
	}
}

// Requirement: doc/specs/content-source-port.md FR-061 - the problem slices of the error are copies: changing one does
// not affect another error built from the same texts, and a decoder error that holds no problem still yields one.
// Path: unhappy (immutability; a decoder that breaks its contract).
func TestBundledInvalidErrorCopiesAndContractBreak(t *testing.T) {
	t.Parallel()
	build := func() *data.BundledInvalidError {
		_, err := data.NewBundledSource(data.WithDocuments(map[string]string{"a.json": "not json"}))
		var bad *data.BundledInvalidError
		if !errors.As(err, &bad) {
			t.Fatalf("err = %v, want a *BundledInvalidError", err)
		}
		return bad
	}
	first, second := build(), build()
	first.DecodeProblems[0] = content.NewProblem("/changed", content.ProblemDuplicate)
	if second.DecodeProblems[0].Path() != "" {
		t.Errorf("a second error shares the slice of the first: %v", second.DecodeProblems)
	}
	if !strings.Contains(second.Error(), "1 decode problems, 0 reference problems") {
		t.Errorf("Error() = %q, want the problem counts", second.Error())
	}

	fake := contenttest.NewFakeDecoder().Return("x", content.Set{}, errors.New("no problems in here"))
	src, err := data.NewBundledSource(data.WithDocuments(map[string]string{"a.json": "x"}), data.WithDecoder(fake))
	var bad *data.BundledInvalidError
	if src != nil || !errors.As(err, &bad) || len(bad.DecodeProblems) != 1 || bad.DecodeProblems[0].Kind() != content.ProblemSyntax {
		t.Errorf("NewBundledSource(contract-breaking decoder) = (%v, %v), want nil and one syntax problem", src, err)
	}
}

// Requirement: doc/specs/content-source-port.md FR-059, FR-061 - nil options and a nil decoder are ignored, so the
// defaults serve the embedded documents; MustBundledSource returns the source for a good bundle and panics with the
// *BundledInvalidError for a bad one.
// Path: unhappy (invalid input: nil option) / happy.
func TestOptionsAndMust(t *testing.T) {
	t.Parallel()
	src, err := data.NewBundledSource(nil, data.WithDecoder(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := src.ListKind(context.Background(), content.KindSpecies); len(got) != 4 {
		t.Errorf("defaults serve %d species, want 4", len(got))
	}
	if data.MustBundledSource() == nil {
		t.Error("MustBundledSource() = nil")
	}
	func() {
		defer func() {
			var bad *data.BundledInvalidError
			if r, _ := recover().(error); !errors.As(r, &bad) {
				t.Errorf("MustBundledSource(bad) panic = %v, want a *BundledInvalidError", r)
			}
		}()
		data.MustBundledSource(data.WithDocuments(map[string]string{"a.json": "not json"}))
	}()
}

// Requirement: doc/specs/content-source-port.md FR-047, FR-027 - BundledSource runs RunSourceContract through an
// injected fake decoder (no failing backend) and RunReadOnlyContract over the real embedded documents with the
// entries decoded from them.
// Path: per case (the shared contract suites).
func TestBundledSourceContracts(t *testing.T) {
	t.Run("RunSourceContract over a fake decoder", func(t *testing.T) {
		contenttest.RunSourceContract(t, func(t *testing.T, entries []content.Entry) content.Source {
			t.Helper()
			set, err := content.NewSet(entries...)
			if err != nil {
				t.Fatal(err)
			}
			fake := contenttest.NewFakeDecoder().Return("doc", set, nil)
			src, err := data.NewBundledSource(data.WithDocuments(map[string]string{"doc.json": "doc"}), data.WithDecoder(fake))
			if err != nil {
				t.Fatal(err)
			}
			return src
		}, nil)
	})
	t.Run("RunReadOnlyContract over the embedded documents", func(t *testing.T) {
		var sets []content.Set
		for _, text := range embeddedDocuments(t) {
			set, err := content.JSONDecoder{}.Decode([]byte(text))
			if err != nil {
				t.Fatal(err)
			}
			sets = append(sets, set)
		}
		merged, err := content.ValidateSet(sets...)
		if err != nil {
			t.Fatal(err)
		}
		contenttest.RunReadOnlyContract(t, newSlice(t), merged.Entries())
	})
}
