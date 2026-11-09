package character_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case equipSpec FR-026, FR-028 (SRD 5.2.1, "Barding", p. 100)
// - the "Barding" rows: four times the cost in the same coin, twice the weight,
// the printed coin kept, the Shield included.
// Path: per case (happy; unhappy boundary and rule interaction).
func TestDeriveBarding(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cat    content.ArmorCategory
		rule   content.ArmorClassRule
		cost   int
		pounds float64
		wantGP int
		wantLb float64
	}{
		{"Padded Armor", catL, fullRule(t, 11), 5, 8, 20, 16},
		{"Leather Armor", catL, fullRule(t, 11), 10, 10, 40, 20},
		{"Chain Shirt", catM, cappedRule(t, 13, 2), 50, 20, 200, 40},
		{"Half Plate", catM, cappedRule(t, 15, 2), 750, 40, 3000, 80},
		{"Plate Armor", catH, noneRule(t, 18), 1500, 65, 6000, 130},
		{"Shield", catS, must(content.NewBonusRule(2)), 10, 6, 40, 12},
		{"Studded Leather", catL, fullRule(t, 12), 45, 13, 180, 26},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := armorOf(t, "armor_x", c.cat, c.rule, 0, false, c.pounds, gp(c.cost))
			b, err := character.DeriveBarding(a)
			if err != nil {
				t.Fatal(err)
			}
			if b.Cost() != gp(c.wantGP) || b.Weight().Pounds() != c.wantLb {
				t.Errorf("barding = %v, %v lb, want %d GP, %v lb", b.Cost(), b.Weight().Pounds(), c.wantGP, c.wantLb)
			}
			again, _ := character.DeriveBarding(a)
			if again != b {
				t.Error("equal inputs should give barding equal under ==")
			}
		})
	}
}

// Requirement: equipSpec FR-026 - an invalid armor or a cost whose four-fold
// exceeds the largest Cost is an srderr.ErrInvalidInput error, not a wrong value.
// Path: unhappy (invalid input).
func TestDeriveBardingInvalid(t *testing.T) {
	t.Parallel()
	_, err := character.DeriveBarding(content.Armor{})
	wantInvalid(t, err, "zero armor")
	big := armorOf(t, "armor_x", catL, fullRule(t, 11), 0, false, 1, gp(content.MaxCostAmount/4+1))
	_, err = character.DeriveBarding(big)
	wantInvalid(t, err, "overflow")
	edge := armorOf(t, "armor_x", catL, fullRule(t, 11), 0, false, 1, gp(content.MaxCostAmount/4))
	if _, err := character.DeriveBarding(edge); err != nil {
		t.Errorf("largest four-foldable cost: %v", err)
	}
}

// Requirement: equipSpec FR-026, FR-028 - a weight whose double is not a
// weight is an srderr.ErrInvalidInput error, and the armor's weight note is not
// carried to the barding.
// Path: unhappy (invalid input, rule interaction).
func TestDeriveBardingWeight(t *testing.T) {
	t.Parallel()
	a := must(content.NewArmor("armor_x", "Armor", content.SourceCitation{}, catL, fullRule(t, 11), 0, false, must(content.NewWeight(math.MaxFloat64, "")), price(gp(1))))
	_, err := character.DeriveBarding(a)
	wantInvalid(t, err, "weight overflow")
	noted := must(content.NewArmor("armor_x", "Armor", content.SourceCitation{}, catL, fullRule(t, 11), 0, false, must(content.NewWeight(8, "full")), price(gp(1))))
	b, err := character.DeriveBarding(noted)
	if err != nil || b.Weight() != lbs(16) {
		t.Errorf("barding weight = %v, %v, want 16 lb. with no note", b.Weight(), err)
	}
}
