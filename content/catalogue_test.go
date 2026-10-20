package content_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// plainEntry is a comparable synthetic Entry (fakeEntry is not comparable).
type plainEntry struct{ key content.Key }

func (e plainEntry) Key() content.Key { return e.key }

func plain(kind content.Kind, id string) plainEntry {
	return plainEntry{key: content.NewKey(kind, content.MustID(id))}
}

func asEntries(es ...plainEntry) []content.Entry {
	out := make([]content.Entry, len(es))
	for i, e := range es {
		out[i] = e
	}
	return out
}

func idsOf(entries []content.Entry) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, e.Key().ID().String())
	}
	return out
}

// Requirement: doc/specs/content-source-port.md FR-008, FR-009, FR-010, FR-016 -
// Lookup returns LookupFound with the entry of that key, and LookupNotFound
// carrying the requested key (never an entry of another kind) otherwise.
// Path: per case.
func TestCatalogueLookup(t *testing.T) {
	t.Parallel()
	c := content.MustCatalogue(asEntries(plain(content.KindSpell, "a"), plain(content.KindFeat, "b")))
	cases := []struct {
		name, path string
		key        content.Key
		want       content.LookupResult
	}{
		{"present", "happy", plain(content.KindSpell, "a").Key(), content.LookupFound{Entry: plain(content.KindSpell, "a")}},
		{"present under another kind", "happy", plain(content.KindFeat, "b").Key(), content.LookupFound{Entry: plain(content.KindFeat, "b")}},
		{"absent id", "unhappy (missing content)", plain(content.KindSpell, "zz").Key(), content.LookupNotFound{Key: plain(content.KindSpell, "zz").Key()}},
		{"id exists only under another kind", "unhappy (rule interaction)", plain(content.KindFeat, "a").Key(), content.LookupNotFound{Key: plain(content.KindFeat, "a").Key()}},
		{"zero key", "unhappy (invalid input)", content.Key{}, content.LookupNotFound{}},
	}
	for _, c2 := range cases {
		t.Run(c2.name, func(t *testing.T) {
			t.Parallel()
			if got := c.Lookup(c2.key); got != c2.want {
				t.Errorf("Lookup(%v) = %v, want %v [%s]", c2.key, got, c2.want, c2.path)
			}
		})
	}
	var zero content.Catalogue
	if got := zero.Lookup(plain(content.KindSpell, "a").Key()); got != (content.LookupNotFound{Key: plain(content.KindSpell, "a").Key()}) {
		t.Errorf("zero Catalogue Lookup = %v, want LookupNotFound [unhappy (empty case)]", got)
	}
}

// Requirement: doc/specs/content-source-port.md FR-011, FR-012, FR-041, FR-046
// (ordering table) - OfKind orders by ID in byte order, ignores other kinds,
// and is empty (not nil) when the kind has no entries.
// Path: per case.
func TestCatalogueOfKind(t *testing.T) {
	t.Parallel()
	sp := func(ids ...string) []content.Entry {
		var out []content.Entry
		for _, id := range ids {
			out = append(out, plain(content.KindSpell, id))
		}
		return out
	}
	cases := []struct {
		name, path string
		entries    []content.Entry
		kind       content.Kind
		want       []string
	}{
		{"already sorted", "happy", sp("a", "b", "c"), content.KindSpell, []string{"a", "b", "c"}},
		{"reverse insertion", "happy", sp("c", "b", "a"), content.KindSpell, []string{"a", "b", "c"}},
		{"digit before underscore before letter", "unhappy (boundary)", sp("ab", "a_b", "a1"), content.KindSpell, []string{"a1", "a_b", "ab"}},
		{"prefix first", "unhappy (boundary)", sp("ab", "a"), content.KindSpell, []string{"a", "ab"}},
		{"other kind ignored", "unhappy (rule interaction)", append(sp("a"), plain(content.KindFeat, "b")), content.KindFeat, []string{"b"}},
		{"single entry", "unhappy (empty and minimal)", sp("a"), content.KindSpell, []string{"a"}},
		{"same id under two kinds", "unhappy (rule interaction)", append(sp("a"), plain(content.KindFeat, "a")), content.KindFeat, []string{"a"}},
		{"kind with no entries", "unhappy (empty case)", sp("a"), content.KindMonster, []string{}},
		{"no entries at all", "unhappy (empty case)", nil, content.KindSpell, []string{}},
		{"invalid kind", "unhappy (invalid input)", sp("a"), content.Kind(0), []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cat, err := content.NewCatalogue(c.entries)
			if err != nil {
				t.Fatal(err)
			}
			got := cat.OfKind(c.kind)
			if got == nil || !slices.Equal(idsOf(got), c.want) {
				t.Errorf("OfKind(%v) ids = %v (nil: %v), want %v [%s]", c.kind, idsOf(got), got == nil, c.want, c.path)
			}
		})
	}
}

// Requirement: doc/specs/content-source-port.md FR-013, FR-023, FR-046 - the
// catalogue copies its input and returns copies: mutating the caller's slice or
// a returned slice changes nothing.
// Path: unhappy (immutability).
func TestCatalogueIsImmutable(t *testing.T) {
	t.Parallel()
	in := asEntries(plain(content.KindSpell, "b"), plain(content.KindSpell, "a"))
	c := content.MustCatalogue(in)
	clear(in)
	if got := idsOf(c.OfKind(content.KindSpell)); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("after clearing the input, OfKind = %v, want [a b]", got)
	}
	if _, ok := c.Lookup(plain(content.KindSpell, "a").Key()).(content.LookupFound); !ok {
		t.Error("after clearing the input, Lookup(a) is not found")
	}
	out := c.OfKind(content.KindSpell)
	clear(out)
	if got := idsOf(c.OfKind(content.KindSpell)); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("after clearing a returned slice, OfKind = %v, want [a b]", got)
	}
}

// Requirement: doc/specs/content-source-port.md FR-021 - a duplicate key is a
// *DuplicateKeyError carrying that key and wrapping ErrInvalidInput; MustCatalogue
// panics with it. A nil entry or invalid key is a *SetError, also invalid input.
// Path: per case.
func TestNewCatalogueRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	dup := plain(content.KindSpell, "a")
	cases := []struct {
		name, path string
		entries    []content.Entry
		wantKey    content.Key
		wantSet    content.SetProblem // 0: expect *DuplicateKeyError
	}{
		{"duplicate key", "unhappy (invalid input)", asEntries(dup, plain(content.KindSpell, "b"), dup), dup.Key(), 0},
		{"nil entry", "unhappy (invalid input)", []content.Entry{nil}, content.Key{}, content.SetNilEntry},
		{"zero key", "unhappy (invalid input)", asEntries(plainEntry{}), content.Key{}, content.SetInvalidKey},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.NewCatalogue(c.entries)
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error = %v, want one wrapping ErrInvalidInput", err)
			}
			var dk *content.DuplicateKeyError
			var se *content.SetError
			switch {
			case c.wantSet == 0 && (!errors.As(err, &dk) || dk.Key != c.wantKey):
				t.Errorf("error = %v, want *DuplicateKeyError for %v", err, c.wantKey)
			case c.wantSet != 0 && (!errors.As(err, &se) || se.Problem != c.wantSet):
				t.Errorf("error = %v, want *SetError %v", err, c.wantSet)
			}
			if got.OfKind(content.KindSpell) == nil || len(got.OfKind(content.KindSpell)) != 0 {
				t.Errorf("rejected Catalogue is not empty")
			}
			defer func() {
				if r, _ := recover().(error); !errors.Is(r, srderr.ErrInvalidInput) {
					t.Errorf("MustCatalogue panicked with %v, want an error wrapping ErrInvalidInput", r)
				}
			}()
			content.MustCatalogue(c.entries)
			t.Error("MustCatalogue did not panic")
		})
	}
}

// Requirement: doc/specs/content-source-port.md FR-041, FR-046 - the same id
// under different kinds is accepted and both entries are found.
// Path: unhappy (rule interaction).
func TestCatalogueAcceptsSameIDUnderTwoKinds(t *testing.T) {
	t.Parallel()
	c, err := content.NewCatalogue(asEntries(plain(content.KindSpell, "a"), plain(content.KindFeat, "a")))
	if err != nil {
		t.Fatalf("NewCatalogue error = %v, want nil", err)
	}
	for _, k := range []content.Kind{content.KindSpell, content.KindFeat} {
		if got := c.Lookup(plain(k, "a").Key()); got != (content.LookupFound{Entry: plain(k, "a")}) {
			t.Errorf("Lookup(%v a) = %v", k, got)
		}
	}
}
