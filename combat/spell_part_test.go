package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/dispel-and-counterspell.md FR-017 - a Prismatic Wall layer is 1 to 7 (SRD 5.2.1, "Prismatic Wall", p. 155; boundary and invalid input).
// Path: per case.
func TestNewPrismaticLayerAcceptsOneToSeven(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		n                       int
		wantErr                 bool
	}{
		{"layer 0 is refused", "FR-017", "unhappy (boundary)", 0, true},
		{"layer 1 is red", "FR-017", "happy", 1, false},
		{"layer 7 is violet", "FR-017", "happy", 7, false},
		{"layer 8 is refused", "FR-017", "unhappy (boundary)", 8, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			l, err := combat.NewPrismaticLayer(c.n)
			if c.wantErr {
				if !errors.Is(err, srderr.ErrInvalidInput) {
					t.Fatalf("NewPrismaticLayer(%d) error = %v, want ErrInvalidInput [%s]", c.n, err, c.requirement)
				}
				return
			}
			if err != nil || !l.Valid() || int(l) != c.n {
				t.Fatalf("NewPrismaticLayer(%d) = %v, %v [%s]", c.n, l, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-017 - the zero SpellPart names the whole spell or its parent (SRD 5.2.1, "Guards and Wards", p. 138).
// Path: happy.
func TestSpellPartZeroIsWholeSpell(t *testing.T) {
	t.Parallel()
	var p combat.SpellPart
	if !p.IsWhole() {
		t.Fatal("zero SpellPart IsWhole() = false, want true [FR-017]")
	}
	if combat.SpellPartGuardsAndWardsEffect().IsWhole() {
		t.Fatal("a Guards and Wards effect IsWhole() = true, want false [FR-017]")
	}
}
