package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-050, table R - each face of the Reincarnate table (R1 to R10): 1 rolls again, 2 to 10 give dragonborn, dwarf, elf, gnome, goliath, halfling, human, orc and tiefling (SRD 5.2.1, "Reincarnate", p. 158); a face outside 1 to 10 is ErrFaceOutOfRange.
// Path: per case.
func TestReincarnationSpeciesFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		face                    int
		want                    content.SpeciesID
		again                   bool
	}{
		{"R1 roll again", "FR-050", "unhappy (boundary)", 1, "", true},
		{"R2 dragonborn", "FR-050", "unhappy (boundary)", 2, "dragonborn", false},
		{"R3 dwarf", "FR-050", "happy", 3, "dwarf", false},
		{"R4 elf", "FR-050", "happy", 4, "elf", false},
		{"R5 gnome", "FR-050", "happy", 5, "gnome", false},
		{"R6 goliath", "FR-050", "happy", 6, "goliath", false},
		{"R7 halfling", "FR-050", "happy", 7, "halfling", false},
		{"R8 human", "FR-050", "happy", 8, "human", false},
		{"R9 orc", "FR-050", "happy", 9, "orc", false},
		{"R10 tiefling", "FR-050", "unhappy (boundary)", 10, "tiefling", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, again, err := combat.ReincarnationSpeciesFor(c.face)
			if err != nil || got != c.want || again != c.again {
				t.Errorf("ReincarnationSpeciesFor(%d) = %q, %v, %v, want %q, %v [%s%s]", c.face, got, again, err, c.want, c.again, otSpec, c.requirement)
			}
		})
	}
	for _, face := range []int{0, 11, -1} {
		if _, _, err := combat.ReincarnationSpeciesFor(face); !errors.Is(err, combat.ErrFaceOutOfRange) {
			t.Errorf("ReincarnationSpeciesFor(%d) = %v, want ErrFaceOutOfRange", face, err)
		}
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-054 - both outcomes say that the new species' traits replace the old ones and apply nothing.
// Path: happy.
func TestReincarnationOutcomesReplaceTraits(t *testing.T) {
	t.Parallel()
	if !(combat.ReincarnationSpeciesRolled{}).ReplacesSpeciesTraits() || !(combat.ReincarnationSpeciesChosen{}).ReplacesSpeciesTraits() {
		t.Error("a reincarnated creature gets the new species' traits in place of the old ones")
	}
	if combat.NotHumanoid.String() != "not_humanoid" || combat.DeadTooLong.String() != "dead_too_long" || combat.SpellTableRefusal(0).String() != "unknown" {
		t.Error("refusal names")
	}
}
