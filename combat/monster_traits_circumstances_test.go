package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func mtMagicTrait(tb testing.TB) combat.MagicResistanceTrait {
	tb.Helper()
	return luMust(combat.NewMagicResistanceTrait(mtRef(tb, 0)))
}

// Requirement: doc/specs/monster-traits.md FR-020, FR-022 - Advantage against a spell or magical effect, nothing otherwise, nothing for a zero trait or an invalid source.
// Path: per case.
func TestMagicResistanceCircumstances(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		trait combat.MagicResistanceTrait
		src   combat.SaveSourceKind
		want  []modifier.RollCircumstance
	}{
		{"spell", mtMagicTrait(t), combat.SpellOrMagicalEffect, []modifier.RollCircumstance{modifier.Advantage{}}},
		{"other", mtMagicTrait(t), combat.OtherSaveSource, nil},
		{"invalid source", mtMagicTrait(t), 0, nil},
		{"zero trait", combat.MagicResistanceTrait{}, combat.SpellOrMagicalEffect, nil},
	}
	for _, c := range cases {
		got := combat.MagicResistanceCircumstances(c.trait, c.src)
		if len(got) != len(c.want) || !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if combat.SpellOrMagicalEffect.String() != "spell_or_magical_effect" || combat.OtherSaveSource.String() != "other" || combat.SaveSourceKind(0).String() != "unknown" || combat.SaveSourceKind(0).Valid() {
		t.Error("source names")
	}
}

// Requirement: doc/specs/monster-traits.md FR-021 - the result feeds modifier.Resolve unchanged: with a Disadvantage the mode is normal, with a second Advantage it is advantage.
// Path: unhappy (rule interaction).
func TestMagicResistanceWithResolve(t *testing.T) {
	t.Parallel()
	adv := combat.MagicResistanceCircumstances(mtMagicTrait(t), combat.SpellOrMagicalEffect)
	if got := modifier.Resolve(append(slices.Clone(adv), modifier.Disadvantage{})...); got != modifier.Normal {
		t.Errorf("with Disadvantage = %v", got)
	}
	if got := modifier.Resolve(append(slices.Clone(adv), modifier.Advantage{})...); got != modifier.WithAdvantage {
		t.Errorf("with a second Advantage = %v", got)
	}
	if got := modifier.Resolve(append(slices.Clone(adv), modifier.Advantage{}, modifier.Disadvantage{}, modifier.Disadvantage{})...); got != modifier.Normal {
		t.Errorf("Advantage and Disadvantage together = %v", got)
	}
}

// Requirement: doc/specs/monster-traits.md FR-070 - Pack Tactics: Advantage when at least one capable ally is 0 to 5 feet from the target (PT-1 rows), an error for a negative distance.
// Path: per case.
func TestPackTacticsCircumstances(t *testing.T) {
	t.Parallel()
	tr := luMust(combat.NewPackTacticsTrait(mtRef(t, 0)))
	adv := []modifier.RollCircumstance{modifier.Advantage{}}
	cases := []struct {
		name   string
		trait  combat.PackTacticsTrait
		allies []combat.AllyFact
		want   []modifier.RollCircumstance
		bad    bool
	}{
		{"ally at 5", tr, []combat.AllyFact{{DistanceFeet: 5}}, adv, false},
		{"ally at 6", tr, []combat.AllyFact{{DistanceFeet: 6}}, nil, false},
		{"ally at 0", tr, []combat.AllyFact{{DistanceFeet: 0}}, adv, false},
		{"ally incapacitated", tr, []combat.AllyFact{{DistanceFeet: 5, Incapacitated: true}}, nil, false},
		{"one capable of two", tr, []combat.AllyFact{{DistanceFeet: 5, Incapacitated: true}, {DistanceFeet: 5}}, adv, false},
		{"capable ally far, near ally incapacitated", tr, []combat.AllyFact{{DistanceFeet: 10}, {DistanceFeet: 1, Incapacitated: true}}, nil, false},
		{"no allies", tr, nil, nil, false},
		{"empty allies", tr, []combat.AllyFact{}, nil, false},
		{"ally at -1", tr, []combat.AllyFact{{DistanceFeet: -1}}, nil, true},
		{"negative after a qualifying ally", tr, []combat.AllyFact{{DistanceFeet: 1}, {DistanceFeet: -1}}, nil, true},
		{"zero trait", combat.PackTacticsTrait{}, []combat.AllyFact{{DistanceFeet: 1}}, nil, false},
	}
	for _, c := range cases {
		got, err := combat.PackTacticsCircumstances(c.trait, c.allies)
		if c.bad != isInvalid(err) || (err != nil && got != nil) || (err == nil && !slices.Equal(got, c.want)) {
			t.Errorf("%s: got %v, %v want %v (bad %v)", c.name, got, err, c.want, c.bad)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-071 - Sunlight Sensitivity covers checks and attack rolls but not saving throws, Weakness covers all three, only in sunlight.
// Path: per case.
func TestSunlightPenaltyCircumstances(t *testing.T) {
	t.Parallel()
	sens := luMust(combat.NewSunlightPenaltyTrait(mtRef(t, 0), combat.SunlightPenaltyAbilityChecksAndAttackRolls))
	weak := luMust(combat.NewSunlightPenaltyTrait(mtRef(t, 0), combat.SunlightPenaltyAllD20Tests))
	dis := []modifier.RollCircumstance{modifier.Disadvantage{}}
	cases := []struct {
		name  string
		trait combat.SunlightPenaltyTrait
		kind  d20.Kind
		sun   bool
		want  []modifier.RollCircumstance
	}{
		{"sensitivity check", sens, d20.AbilityCheck, true, dis},
		{"sensitivity attack", sens, d20.AttackRoll, true, dis},
		{"sensitivity save", sens, d20.SavingThrow, true, nil},
		{"weakness save", weak, d20.SavingThrow, true, dis},
		{"weakness check", weak, d20.AbilityCheck, true, dis},
		{"weakness attack", weak, d20.AttackRoll, true, dis},
		{"not in sunlight", weak, d20.AttackRoll, false, nil},
		{"invalid kind", weak, 0, true, nil},
		{"kind past the last", weak, 4, true, nil},
		{"zero trait", combat.SunlightPenaltyTrait{}, d20.AttackRoll, true, nil},
	}
	for _, c := range cases {
		if got := combat.SunlightPenaltyCircumstances(c.trait, c.kind, c.sun); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
