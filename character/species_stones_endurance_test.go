package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-038, Table G "Stone roll cases", "Stone zero and negative damage" (SRD 5.2.1, "Stone's Endurance", p. 85; interpretation Q5) - the reduction is the roll plus the Constitution modifier but never below 0, and the damage left never falls below 0.
// Path: per case.
func TestStonesEnduranceReduce(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                      string
		roll, mod, damage         int
		wantReduction, wantRemain int
	}{
		{"roll 6, Con +2, damage 20", 6, 2, 20, 8, 12},
		{"roll 12, Con +3, damage 10", 12, 3, 10, 15, 0},
		{"roll 1, Con -3, damage 5", 1, -3, 5, 0, 5},
		{"roll 1, Con -5, damage 9", 1, -5, 9, 0, 9},
		{"damage 0", 7, 2, 0, 9, 0},
		{"reduction equals the damage", 4, 2, 6, 6, 0},
		{"reduction one below the damage", 4, 2, 7, 6, 1},
		{"a negative damage counts as 0", 4, 2, -3, 6, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, left := character.StonesEnduranceRule{}.Reduce(c.roll, c.mod, c.damage)
			if r != c.wantReduction || left != c.wantRemain {
				t.Errorf("Reduce(%d, %d, %d) = %d, %d; want %d, %d", c.roll, c.mod, c.damage, r, left, c.wantReduction, c.wantRemain)
			}
		})
	}
}
