package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-schema-and-decoder.md FR-142 - Kind declares
// its members in a pinned order (species 1 to monster 8; poison, trap and
// contagion arrive with their rows in Phase 5), the zero value is invalid and
// String is the wire spelling; there is no animal kind.
// Path: per case.
func TestKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		kind       content.Kind
		ordinal    int
		wire       string
		valid      bool
	}{
		{"species", "happy", content.KindSpecies, 1, "species", true},
		{"background", "happy", content.KindBackground, 2, "background", true},
		{"character class", "happy", content.KindCharacterClass, 3, "characterClass", true},
		{"feat", "happy", content.KindFeat, 4, "feat", true},
		{"equipment", "happy", content.KindEquipment, 5, "equipment", true},
		{"spell", "happy", content.KindSpell, 6, "spell", true},
		{"magic item", "happy", content.KindMagicItem, 7, "magicItem", true},
		{"monster", "happy", content.KindMonster, 8, "monster", true},
		{"zero is invalid", "unhappy (boundary)", content.Kind(0), 0, "unknown", false},
		{"first undeclared (poison arrives in Phase 5)", "unhappy (boundary)", content.Kind(9), 9, "unknown", false},
		{"negative", "unhappy (boundary)", content.Kind(-1), -1, "unknown", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if int(c.kind) != c.ordinal {
				t.Errorf("ordinal = %d, want %d [%s]", int(c.kind), c.ordinal, c.path)
			}
			if got := c.kind.String(); got != c.wire {
				t.Errorf("String() = %q, want %q", got, c.wire)
			}
			if got := c.kind.Valid(); got != c.valid {
				t.Errorf("Valid() = %v, want %v", got, c.valid)
			}
		})
	}
}
