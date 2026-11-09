package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The fixtures are synthetic: made-up ids and names with the numbers the
// spec's case tables print.

func gp(n int) content.Cost { return content.MustCost(n, content.CoinGP, 0) }

func price(c content.Cost) content.FixedCost { return content.MustFixedCost(c) }

func lbs(p float64) content.Weight { return content.MustWeight(p, "") }

func must[T any](v T, err error) T {
	if err != nil {
		panic(err) // a broken fixture, not a failure of the code under test
	}
	return v
}

func wantInvalid(t *testing.T, err error, what string) {
	t.Helper()
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("%s: error = %v, want srderr.ErrInvalidInput", what, err)
	}
}

func score(t *testing.T, n int) ability.Score {
	t.Helper()
	return must(ability.NewScore(n))
}

func modifierOf(t *testing.T, n int) ability.Modifier {
	t.Helper()
	// An ability score s has modifier (s-10)/2 rounded down; 10+2n gives n.
	s := 10 + 2*n
	if s < ability.MinScore {
		s = ability.MinScore
	}
	return score(t, s).Modifier()
}

func fullRule(t *testing.T, base int) content.BaseRule {
	t.Helper()
	return must(content.NewBaseRule(base, content.FullDexterity{}))
}

func cappedRule(t *testing.T, base, maximum int) content.BaseRule {
	t.Helper()
	return must(content.NewBaseRule(base, must(content.NewCappedDexterity(maximum))))
}

func noneRule(t *testing.T, base int) content.BaseRule {
	t.Helper()
	return must(content.NewBaseRule(base, content.NoDexterity{}))
}

// armorOf builds a body armor (or a Shield for ArmorShield) with the given
// Strength requirement (0 for none), Stealth flag, weight and price.
func armorOf(t *testing.T, id string, cat content.ArmorCategory, rule content.ArmorClassRule, strength int, stealth bool, pounds float64, cost content.Cost) content.Armor {
	t.Helper()
	return must(content.NewArmor(content.ItemID(id), id, content.SourceCitation{}, cat, rule, strength, stealth, lbs(pounds), price(cost)))
}

func shieldOf(t *testing.T) content.Armor {
	t.Helper()
	return armorOf(t, "shield_x", content.ArmorShield, must(content.NewBonusRule(2)), 0, false, 6, gp(10))
}

func mountOf(t *testing.T, id string, capacity int) content.Mount {
	t.Helper()
	return must(content.NewMount(content.ItemID(id), id, content.SourceCitation{}, capacity, price(gp(1))))
}

func levelOf(t *testing.T, n int) content.SpellLevel {
	t.Helper()
	return must(content.NewSpellLevel(n))
}
