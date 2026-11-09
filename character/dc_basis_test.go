package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/social-interaction.md FR-012 - DCBasis is DefaultRule or GMSet
// and the zero or an undefined value is invalid (SRD 5.2.1, "Influence [Action]", p. 184).
// Path: per case.
func TestDCBasis(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		b                       character.DCBasis
		valid                   bool
		str                     string
	}{
		{"default rule", "FR-012", "happy", character.DefaultRule, true, "default_rule"},
		{"GM set", "FR-012", "happy", character.GMSet, true, "gm_set"},
		{"zero", "FR-012", "unhappy (invalid input)", 0, false, "unknown"},
		{"past GM set", "FR-012", "unhappy (boundary)", character.GMSet + 1, false, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.b.Valid() != c.valid || c.b.String() != c.str {
				t.Errorf("DCBasis(%d) = valid %v, %q; want %v, %q [%s, %s]", int(c.b), c.b.Valid(), c.b.String(), c.valid, c.str, c.requirement, c.path)
			}
		})
	}
}
