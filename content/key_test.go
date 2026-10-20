package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// fakeEntry is a synthetic Entry. tags makes it uncomparable with ==.
type fakeEntry struct {
	key  content.Key
	tags []string
}

func (e fakeEntry) Key() content.Key { return e.key }

var _ content.Entry = fakeEntry{}

func entry(kind content.Kind, id string, tags ...string) fakeEntry {
	return fakeEntry{key: content.NewKey(kind, content.MustID(id)), tags: tags}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-145 - two keys are
// equal under == iff kind and id both match; the same id under two kinds is two
// keys and works as a map key.
// Path: per case.
func TestKeyEquality(t *testing.T) {
	t.Parallel()
	a := content.NewKey(content.KindSpell, content.MustID("a"))
	cases := []struct {
		name, path string
		other      content.Key
		equal      bool
	}{
		{"same kind and id", "unhappy (rule interaction)", content.NewKey(content.KindSpell, content.MustID("a")), true},
		{"same id, other kind", "unhappy (rule interaction)", content.NewKey(content.KindFeat, content.MustID("a")), false},
		{"same kind, other id", "unhappy (rule interaction)", content.NewKey(content.KindSpell, content.MustID("b")), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := a == c.other; got != c.equal {
				t.Errorf("== = %v, want %v [%s]", got, c.equal, c.path)
			}
			m := map[content.Key]bool{a: true}
			if m[c.other] != c.equal {
				t.Errorf("map lookup = %v, want %v", m[c.other], c.equal)
			}
		})
	}
	if a.Kind() != content.KindSpell || a.ID() != content.MustID("a") || a.IsZero() {
		t.Errorf("accessors of %v wrong", a)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-145 - the zero Key is
// the invalid key.
// Path: unhappy (empty case).
func TestKeyZero(t *testing.T) {
	t.Parallel()
	var k content.Key
	if !k.IsZero() || k != content.NewKey(0, content.ID{}) {
		t.Errorf("zero Key not zero")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-146 - Entry has one
// method, Key(); a value implementing it is usable as an Entry.
// Path: happy.
func TestEntryKey(t *testing.T) {
	t.Parallel()
	var e content.Entry = entry(content.KindFeat, "alert")
	if e.Key() != content.NewKey(content.KindFeat, content.MustID("alert")) {
		t.Errorf("Entry.Key() = %v", e.Key())
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-147 - MissingContent
// is comparable, its Key equals NewKey(kind, id) and equal parts are equal.
// Path: happy.
func TestMissingContent(t *testing.T) {
	t.Parallel()
	id := content.MustID("a")
	m := content.NewMissingContent(content.KindSpell, id)
	if m.Key() != content.NewKey(content.KindSpell, id) || m.Kind() != content.KindSpell || m.ID() != id {
		t.Errorf("MissingContent parts wrong: %v", m)
	}
	if m != content.NewMissingContent(content.KindSpell, id) || m == content.NewMissingContent(content.KindFeat, id) {
		t.Errorf("MissingContent must compare by value")
	}
	if (content.MissingContent{}).Key() != (content.Key{}) {
		t.Errorf("zero MissingContent must name nothing")
	}
}
