package combat_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (table 4.7b and FR-040; Source: SRD 5.2.1, "Mass Heal" p. 147)
// Path: per case.
func TestNewMassHealAllotment(t *testing.T) {
	t.Parallel()
	ids := func(n int) map[combat.CombatantID]int {
		m := map[combat.CombatantID]int{}
		for i := range n {
			m[gid(fmt.Sprintf("c%d", i))] = 0
		}
		return m
	}
	cases := []struct {
		name, requirement, path string
		in                      map[combat.CombatantID]int
		total                   int
		bad                     string
	}{
		{"a split of 400 and 300", "FR-040; table 4.7b Mass Heal split", "unhappy (boundary)", map[combat.CombatantID]int{gid("a"): 400, gid("b"): 300}, 700, ""},
		{"a total of 701", "FR-040; table 4.7b Mass Heal over", "unhappy (boundary)", map[combat.CombatantID]int{gid("a"): 400, gid("b"): 301}, 0, "mass_heal_total"},
		{"a single 700", "FR-040", "unhappy (boundary)", map[combat.CombatantID]int{gid("a"): 700}, 700, ""},
		{"a single 701", "FR-040", "unhappy (boundary)", map[combat.CombatantID]int{gid("a"): 701}, 0, "mass_heal_total"},
		{"a negative allotment", "FR-040; table 4.7b Mass Heal negative", "unhappy (invalid input)", map[combat.CombatantID]int{gid("a"): -1}, 0, "mass_heal_allotment"},
		{"a negative that would cancel an excess", "FR-040", "unhappy (invalid input)", map[combat.CombatantID]int{gid("a"): 800, gid("b"): -100}, 0, "mass_heal_allotment"},
		{"an allotment of 0", "FR-040; table 4.7b Mass Heal zero", "unhappy (boundary)", map[combat.CombatantID]int{gid("a"): 0}, 0, ""},
		{"an empty map", "FR-040", "unhappy (empty and minimal)", map[combat.CombatantID]int{}, 0, ""},
		{"a nil map", "FR-040", "unhappy (empty and minimal)", nil, 0, ""},
		{"an overflowing sum", "FR-040 (overflow)", "unhappy (boundary)", map[combat.CombatantID]int{gid("a"): math.MaxInt, gid("b"): math.MaxInt}, 0, "mass_heal_total"},
		{"256 creatures", "plan P10 (the cap)", "unhappy (boundary)", ids(256), 0, ""},
		{"257 creatures", "plan P10 (cap plus one)", "unhappy (boundary)", ids(257), 0, "mass_heal_creatures"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewMassHealAllotment(c.in)
			if c.bad != "" {
				var re *combat.RestorationRangeError
				if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &re) || re.Name != c.bad || got.Total() != 0 {
					t.Errorf("NewMassHealAllotment = %v, %v, want a range error for %s [%s; %s]", got.Total(), err, c.bad, c.requirement, c.path)
				}
				return
			}
			if err != nil || got.Total() != c.total {
				t.Errorf("NewMassHealAllotment = total %d, %v, want %d [%s; %s]", got.Total(), err, c.total, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-040 - the allotment copies its map on construction and returns a copy; a creature not in it has 0; the zero value is the
// empty allotment that heals no one; a blank combatant id is an error matching srderr.ErrInvalidInput.
// Path: unhappy (immutability).
func TestMassHealAllotmentCopies(t *testing.T) {
	t.Parallel()
	in := map[combat.CombatantID]int{gid("a"): 40, gid("b"): 0}
	a, err := combat.NewMassHealAllotment(in)
	if err != nil {
		t.Fatal(err)
	}
	in[gid("a")] = 600
	out := a.Allotments()
	out[gid("a")] = 1
	if a.For(gid("a")) != 40 || a.For(gid("b")) != 0 || a.For(gid("zzz")) != 0 || a.Total() != 40 || len(a.Allotments()) != 2 {
		t.Errorf("the allotment changed with its source or a returned map: %v", a.Allotments())
	}
	var zero combat.MassHealAllotment
	if zero.For(gid("a")) != 0 || zero.Total() != 0 || len(zero.Allotments()) != 0 {
		t.Error("the zero allotment is not empty")
	}
	if _, err := combat.NewMassHealAllotment(map[combat.CombatantID]int{"": 5}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank combatant id: %v, want an invalid input error", err)
	}
}
