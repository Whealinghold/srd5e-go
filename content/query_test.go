package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-source-port.md FR-057 - Query is closed over
// LookupQuery and ListKindQuery (an exhaustive switch with no default); variants
// compare by value and are unequal to each other.
// Path: per case.
func TestQueryVariants(t *testing.T) {
	t.Parallel()
	k := plain(content.KindSpell, "a").Key()
	cases := []struct {
		name, path string
		a, b       content.Query
		equal      bool
	}{
		{"same lookup", "happy", content.LookupQuery{Key: k}, content.LookupQuery{Key: k}, true},
		{"lookup other key", "unhappy (rule interaction)", content.LookupQuery{Key: k}, content.LookupQuery{Key: plain(content.KindFeat, "a").Key()}, false},
		{"same list", "happy", content.ListKindQuery{Kind: content.KindSpell}, content.ListKindQuery{Kind: content.KindSpell}, true},
		{"list other kind", "unhappy (rule interaction)", content.ListKindQuery{Kind: content.KindSpell}, content.ListKindQuery{Kind: content.KindFeat}, false},
		{"lookup vs list", "unhappy (rule interaction)", content.LookupQuery{}, content.ListKindQuery{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.a == c.b; got != c.equal {
				t.Errorf("%v == %v is %v, want %v [%s]", c.a, c.b, got, c.equal, c.path)
			}
		})
	}
	for _, q := range []content.Query{content.LookupQuery{Key: k}, content.ListKindQuery{Kind: content.KindSpell}} {
		switch v := q.(type) {
		case content.LookupQuery:
			if v.Key != k {
				t.Errorf("LookupQuery key = %v", v.Key)
			}
		case content.ListKindQuery:
			if v.Kind != content.KindSpell {
				t.Errorf("ListKindQuery kind = %v", v.Kind)
			}
		}
	}
}
