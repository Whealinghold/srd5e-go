package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/casting-and-concentration.md FR-078, FR-079, FR-134, FR-135, Table K - the DC is the greater of 10
// and half the damage rounded down, to a maximum of 30; an amount below 1 is an error.
// Path: per case.
func TestConcentrationSaveDCForDamage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, req, path string
		damage          int
		dc              int // 0: an error
	}{
		{"K1 1 damage", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary)", 1, 10},
		{"K2 20 damage", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary)", 20, 10},
		{"K3 21 damage rounds down", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary, rounds down)", 21, 10},
		{"K4 22 damage", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary)", 22, 11},
		{"K5 59 damage", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary)", 59, 29},
		{"K6 60 damage", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary)", 60, 30},
		{"K7 61 damage", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary)", 61, 30},
		{"K12 62 damage is the first capped value", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary)", 62, 30},
		{"K8 200 damage", `SRD 5.2.1, "Concentration", p. 179`, "unhappy (boundary, maximum)", 200, 30},
		{"K11 7 damage, all absorbed by Temporary Hit Points, still counts", "FR-135 (package interpretation of p. 18 and p. 179)", "unhappy (rule interaction)", 7, 10},
		{"K9 zero damage", "FR-134", "unhappy (invalid input)", 0, 0},
		{"K10 negative damage", "FR-134", "unhappy (invalid input)", -1, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.ConcentrationSaveDCForDamage(c.damage)
			if c.dc == 0 {
				if !errors.Is(err, srderr.ErrInvalidInput) || got != (character.ConcentrationSaveDC{}) {
					t.Errorf("ConcentrationSaveDCForDamage(%d) = %v, %v, want ErrInvalidInput [%s]", c.damage, got, err, c.req)
				}
				return
			}
			if err != nil || got.Value() != c.dc || got.DifficultyClass() != d20.DifficultyClass(c.dc) {
				t.Errorf("ConcentrationSaveDCForDamage(%d) = %d, %v, want %d [%s]", c.damage, got.Value(), err, c.dc, c.req)
			}
		})
	}
}
