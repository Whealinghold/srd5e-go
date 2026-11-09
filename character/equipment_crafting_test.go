package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func cp(n int) content.Cost { return content.MustCost(n, content.CoinCP, 0) }

// Requirement: per case equipSpec FR-033, FR-035 (SRD 5.2.1, "Crafting
// Nonmagical Items", p. 103; package design Q15) - the "Crafting" rows: half
// the cost in Copper Pieces rounded down, and the ceiling of the cost in Copper
// Pieces over 1,000 as days.
// Path: per case (happy; unhappy boundary, rule interaction).
func TestRawMaterialsAndDays(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cost content.Cost
		raw  int
		days int
	}{
		{"Plate Armor 1,500 GP", gp(1500), 75000, 150}, {"Heavy Crossbow 50 GP", gp(50), 2500, 5},
		{"10 GP", gp(10), 500, 1}, {"20 GP", gp(20), 1000, 2}, {"45 GP", gp(45), 2250, 5},
		{"2 GP", gp(2), 100, 1}, {"15 GP", gp(15), 750, 2},
		{"10 GP 1 CP", cp(1001), 500, 2}, {"1 SP", content.MustCost(1, content.CoinSP, 0), 5, 1},
		{"1 CP", cp(1), 0, 1}, {"5 CP", cp(5), 2, 1}, {"5 SP kept as Copper", content.MustCost(5, content.CoinSP, 0), 25, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			raw, err := character.Crafting{}.RawMaterials(c.cost)
			if err != nil || raw != cp(c.raw) {
				t.Errorf("RawMaterials = %v, %v, want %d CP", raw, err, c.raw)
			}
			days, err := character.Crafting{}.Days(c.cost)
			if err != nil || days != c.days {
				t.Errorf("Days = %d, %v, want %d", days, err, c.days)
			}
		})
	}
}

// Requirement: equipSpec FR-034, FR-036 - a rated cost is an
// srderr.ErrInvalidInput error for both; a cost of 0 gives 0 CP raw materials
// but an error for days; a result beyond the largest Cost is an error.
// Path: unhappy (invalid input).
func TestCraftingInvalidCosts(t *testing.T) {
	t.Parallel()
	zero := cp(0)
	if raw, err := (character.Crafting{}).RawMaterials(zero); err != nil || raw != zero {
		t.Errorf("RawMaterials(0) = %v, %v", raw, err)
	}
	_, err := character.Crafting{}.Days(zero)
	wantInvalid(t, err, "Days(0)")
	rated := content.MustCost(5, content.CoinSP, content.CostPerDay)
	_, err = character.Crafting{}.RawMaterials(rated)
	wantInvalid(t, err, "RawMaterials(rated)")
	_, err = character.Crafting{}.Days(rated)
	wantInvalid(t, err, "Days(rated)")
	_, err = character.Crafting{}.RawMaterials(content.Cost{})
	wantInvalid(t, err, "RawMaterials(zero Cost)")
	_, err = character.Crafting{}.Days(content.Cost{})
	wantInvalid(t, err, "Days(zero Cost)")
	huge := content.MustCost(content.MaxCostAmount, content.CoinPP, 0)
	_, err = character.Crafting{}.RawMaterials(huge)
	wantInvalid(t, err, "RawMaterials(huge)")
	if d, err := (character.Crafting{}).Days(huge); err != nil || d != content.MaxCostAmount {
		t.Errorf("Days(huge) = %d, %v", d, err)
	}
	edge := content.MustCost(content.MaxCostAmount, content.CoinCP, 0)
	if raw, err := (character.Crafting{}).RawMaterials(edge); err != nil || raw != cp(content.MaxCostAmount/2) {
		t.Errorf("RawMaterials(largest CP) = %v, %v", raw, err)
	}
	if _, err := (character.Crafting{}).RawMaterials(content.MustCost(2*content.MaxCostAmount/200, content.CoinGP, 0)); err != nil {
		t.Errorf("largest halvable GP: %v", err)
	}
}
