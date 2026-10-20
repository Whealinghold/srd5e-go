package content_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func keys(s content.Set) []content.Key {
	var out []content.Key
	for _, e := range s.Entries() {
		out = append(out, e.Key())
	}
	return out
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-071 - Entries are
// ordered by Kind (declaration order) then ID (ordinal), whatever the input
// order; an equipment entry precedes a spell.
// Path: per case.
func TestSetOrder(t *testing.T) {
	t.Parallel()
	spellA, equipB, equipA, speciesZ := entry(content.KindSpell, "a"), entry(content.KindEquipment, "b"), entry(content.KindEquipment, "a"), entry(content.KindSpecies, "z")
	sorted := []content.Key{speciesZ.Key(), equipA.Key(), equipB.Key(), spellA.Key()}
	inputs := map[string][]content.Entry{
		"as sorted":  {speciesZ, equipA, equipB, spellA},
		"reversed":   {spellA, equipB, equipA, speciesZ},
		"shuffled":   {equipB, spellA, speciesZ, equipA},
		"id ordinal": {entry(content.KindFeat, "b"), entry(content.KindFeat, "a_1"), entry(content.KindFeat, "a")},
	}
	for name, in := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s, err := content.NewSet(in...)
			if err != nil {
				t.Fatal(err)
			}
			got, want := keys(s), sorted
			if name == "id ordinal" {
				want = []content.Key{in[2].Key(), in[1].Key(), in[0].Key()}
			}
			if !slices.Equal(got, want) {
				t.Errorf("order = %v, want %v [happy]", got, want)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-071 - a Set is
// immutable: mutating Entries() or the argument slice leaves it unchanged.
// Path: unhappy (immutability).
func TestSetCopies(t *testing.T) {
	t.Parallel()
	a, b := entry(content.KindSpell, "a"), entry(content.KindSpell, "b")
	in := []content.Entry{b, a}
	s := content.MustSet(in...)
	in[0] = entry(content.KindFeat, "x")
	out := s.Entries()
	out[0] = entry(content.KindFeat, "y")
	if !slices.Equal(keys(s), []content.Key{a.Key(), b.Key()}) {
		t.Errorf("set changed through a slice it handed out or took in: %v", keys(s))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-093 - NewSet rejects
// two entries with one key (the same id under two kinds is fine), a nil entry
// and an invalid key, each as a *SetError wrapping ErrInvalidInput; MustSet
// panics with it; the empty set is valid.
// Path: per case.
func TestNewSetErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		entries    []content.Entry
		want       content.SetProblem // 0 means accepted
		key        content.Key
	}{
		{"no entries", "unhappy (empty case)", nil, 0, content.Key{}},
		{"same id, two kinds", "unhappy (rule interaction)", []content.Entry{entry(content.KindFeat, "a"), entry(content.KindSpell, "a")}, 0, content.Key{}},
		{"duplicate key", "unhappy (invalid input)", []content.Entry{entry(content.KindSpell, "a"), entry(content.KindSpell, "b"), entry(content.KindSpell, "a", "other")}, content.SetDuplicateKey, entry(content.KindSpell, "a").Key()},
		{"nil entry", "unhappy (invalid input)", []content.Entry{entry(content.KindSpell, "a"), nil}, content.SetNilEntry, content.Key{}},
		{"nil entry pointer", "unhappy (invalid input)", []content.Entry{(*fakeEntry)(nil)}, content.SetNilEntry, content.Key{}},
		{"zero key", "unhappy (invalid input)", []content.Entry{fakeEntry{}}, content.SetInvalidKey, content.Key{}},
		{"invalid kind", "unhappy (invalid input)", []content.Entry{fakeEntry{key: content.NewKey(content.Kind(99), content.MustID("a"))}}, content.SetInvalidKey, content.NewKey(content.Kind(99), content.MustID("a"))},
		{"zero id", "unhappy (invalid input)", []content.Entry{fakeEntry{key: content.NewKey(content.KindSpell, content.ID{})}}, content.SetInvalidKey, content.NewKey(content.KindSpell, content.ID{})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := content.NewSet(c.entries...)
			panicked := panicValue(func() { content.MustSet(c.entries...) })
			if c.want == 0 {
				if err != nil || panicked != nil || len(s.Entries()) != len(c.entries) {
					t.Fatalf("got (%d entries, %v), panic %v; want accepted [%s]", len(s.Entries()), err, panicked, c.path)
				}
				return
			}
			var se *content.SetError
			if !errors.As(err, &se) || se.Problem != c.want || se.Key != c.key || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error = %v, want *SetError %v at %v [%s]", err, c.want, c.key, c.path)
			}
			if !errors.Is(err, &content.SetError{Problem: c.want}) || errors.Is(err, &content.SetError{Problem: c.want + 1}) || errors.Is(err, &content.SetError{Problem: c.want - 1}) {
				t.Errorf("SetError.Is must match by Problem only")
			}
			pe, ok := panicked.(error)
			if len(s.Entries()) != 0 || !ok || !errors.Is(pe, &content.SetError{Problem: c.want}) {
				t.Errorf("MustSet panic = %v, want the NewSet error; rejected set empty = %v", panicked, len(s.Entries()) == 0)
			}
		})
	}
	for i, w := range []string{"unknown", "nilEntry", "invalidKey", "duplicateKey", "unknown"} {
		if got := content.SetProblem(i).String(); got != w {
			t.Errorf("SetProblem(%d).String() = %q, want %q", i, got, w)
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-072, FR-097 - sets
// built from the same entries in any order are equal; any differing entry,
// field or length makes them unequal.
// Path: per case.
func TestSetEqual(t *testing.T) {
	t.Parallel()
	a, b := entry(content.KindSpell, "a", "x"), entry(content.KindSpell, "b")
	base := content.MustSet(a, b)
	cases := []struct {
		name, path string
		other      content.Set
		equal      bool
	}{
		{"same order", "happy", content.MustSet(a, b), true},
		{"other input order", "happy", content.MustSet(b, a), true},
		{"equal slice contents, new backing array", "happy", content.MustSet(entry(content.KindSpell, "a", "x"), b), true},
		{"differing field", "unhappy (rule interaction)", content.MustSet(entry(content.KindSpell, "a", "y"), b), false},
		{"differing key", "unhappy (rule interaction)", content.MustSet(a, entry(content.KindSpell, "c")), false},
		{"fewer entries", "unhappy (boundary)", content.MustSet(a), false},
		{"empty", "unhappy (empty case)", content.Set{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := base.Equal(c.other); got != c.equal || c.other.Equal(base) != c.equal {
				t.Errorf("Equal = %v, want %v [%s]", got, c.equal, c.path)
			}
		})
	}
	if !(content.Set{}).Equal(content.MustSet()) {
		t.Errorf("the zero Set must equal an empty NewSet")
	}
}
