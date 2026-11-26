package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// moveScore is the ability score v; it fails the test for a value outside 1 to 30.
func moveScore(tb testing.TB, v int) ability.Score {
	tb.Helper()
	s, err := ability.NewScore(v)
	if err != nil {
		tb.Fatalf("NewScore(%d): %v", v, err)
	}
	return s
}

// moveArmor is a heavy armor with the Strength requirement given (Chain Mail needs 13).
func moveArmor(tb testing.TB, strength int) content.Armor {
	tb.Helper()
	rule, err := content.NewBaseRule(16, content.NoDexterity{})
	if err != nil {
		tb.Fatalf("NewBaseRule: %v", err)
	}
	cost, err := content.NewCost(75, content.CoinGP, 0)
	if err != nil {
		tb.Fatalf("NewCost: %v", err)
	}
	price, err := content.NewFixedCost(cost)
	if err != nil {
		tb.Fatalf("NewFixedCost: %v", err)
	}
	weight, err := content.NewWeight(55, "")
	if err != nil {
		tb.Fatalf("NewWeight: %v", err)
	}
	a, err := content.NewArmor("chain_mail", "Chain Mail", content.SourceCitation{}, content.ArmorHeavy, rule, strength, true, weight, price)
	if err != nil {
		tb.Fatalf("NewArmor: %v", err)
	}
	return a
}

// moveLimit is the LoadSpeedLimited assessment, the only input LimitSpeedsByLoad takes.
func moveLimit(tb testing.TB) character.LoadSpeedLimited {
	tb.Helper()
	a, err := character.NewCarryingCapacity(content.SizeMedium, moveScore(tb, 10)).Assess(200)
	if err != nil {
		tb.Fatalf("Assess: %v", err)
	}
	l, ok := a.(character.LoadSpeedLimited)
	if !ok {
		tb.Fatalf("Assess(200) = %#v, want LoadSpeedLimited", a)
	}
	return l
}

// moveSet is a ConditionSet holding the conditions.
func moveSet(tb testing.TB, conds ...combat.Condition) combat.ConditionSet {
	tb.Helper()
	return condSet(tb, conds)
}
