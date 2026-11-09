package character

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestShapeOf(t *testing.T) {
	t.Parallel()
	strDex := []ability.Ability{ability.Strength, ability.Dexterity}
	all := ability.All()
	longRest := UseRenewedBy{mask: 1 << RenewOnLongRestFinished}
	every := UseRenewedBy{mask: 1<<RenewOnInitiativeRolled | 1<<RenewOnShortRestFinished | 1<<RenewOnLongRestFinished}
	for _, c := range []struct {
		id          content.FeatID
		choices     choiceShape
		abilities   []ability.Ability
		maximum     int
		renewedBy   UseRenewedBy
		increases   bool
		requirement string
	}{
		{"alert", shapeNone, nil, 0, UseRenewedBy{}, false, "doc/specs/feats-behaviour.md Table C no choices, Origin"},
		{"savage_attacker", shapeNone, nil, 0, UseRenewedBy{}, false, "doc/specs/feats-behaviour.md Table C no choices, Origin"},
		{"magic_initiate", shapeMagicInitiate, nil, 0, longRest, false, "Table C magic initiate; FR-032"},
		{"skilled", shapeSkilled, nil, 0, UseRenewedBy{}, false, "Table C skilled"},
		{"ability_score_improvement", shapeImprovement, nil, 20, UseRenewedBy{}, true, "Table C ability score improvement"},
		{"grappler", shapeSingle, strDex, 20, UseRenewedBy{}, true, "Table C grappler"},
		{"archery", shapeNone, nil, 0, UseRenewedBy{}, false, "Table C Fighting Style feats"},
		{"defense", shapeNone, nil, 0, UseRenewedBy{}, false, "Table C Fighting Style feats"},
		{"great_weapon_fighting", shapeNone, nil, 0, UseRenewedBy{}, false, "Table C Fighting Style feats"},
		{"two_weapon_fighting", shapeNone, nil, 0, UseRenewedBy{}, false, "Table C Fighting Style feats"},
		{"boon_of_combat_prowess", shapeSingle, all, 30, UseRenewedBy{}, true, "Table C Epic Boon, any ability"},
		{"boon_of_dimensional_travel", shapeSingle, all, 30, UseRenewedBy{}, true, "Table C Epic Boon, any ability"},
		{"boon_of_the_night_spirit", shapeSingle, all, 30, UseRenewedBy{}, true, "Table C Epic Boon, any ability"},
		{"boon_of_truesight", shapeSingle, all, 30, UseRenewedBy{}, true, "Table C Epic Boon, any ability"},
		{"boon_of_fate", shapeSingle, all, 30, every, true, "Table C Epic Boon, any ability; FR-050"},
		{"boon_of_irresistible_offense", shapeSingle, strDex, 30, UseRenewedBy{}, true, "Table C boon of irresistible offense"},
		{"boon_of_spell_recall", shapeSingle, []ability.Ability{ability.Intelligence, ability.Wisdom, ability.Charisma}, 30, UseRenewedBy{}, true, "Table C boon of spell recall"},
	} {
		got, ok := shapeOf(c.id)
		if !ok || got.choices != c.choices || !slices.Equal(got.abilities, c.abilities) || got.maximum != c.maximum || got.renewedBy != c.renewedBy || got.givesIncrease() != c.increases {
			t.Errorf("shapeOf(%s) = %+v, %v [%s]", c.id, got, ok, c.requirement)
		}
	}
	for _, id := range []content.FeatID{"house_feat", "", "Alert"} {
		if _, ok := shapeOf(id); ok {
			t.Errorf("shapeOf(%q) knows a feat with no behaviour (FR-018)", id)
		}
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-017, FR-061, Table C - a shape accepts exactly the one type of choices it names, and nil is none of them.
// Path: per case.
func TestFeatShapeMatches(t *testing.T) {
	t.Parallel()
	choices := map[choiceShape]FeatChoices{
		shapeNone: NoFeatChoices{}, shapeSingle: SingleAbilityIncrease{ability: ability.Strength}, shapeImprovement: AbilityScoreImprovementChoices{},
		shapeMagicInitiate: MagicInitiateChoices{}, shapeSkilled: SkilledChoices{},
	}
	for shape := shapeNone; shape <= shapeSingle; shape++ {
		for other, c := range choices {
			if got := (featShape{choices: shape}).matches(c); got != (shape == other) {
				t.Errorf("shape %d matches %T = %v", shape, c, got)
			}
		}
		if (featShape{choices: shape}).matches(nil) {
			t.Errorf("shape %d matches nil choices", shape)
		}
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-017, FR-061 - the zero shape and a shape outside the five accept no choices and give no increase.
// Path: unhappy (boundary).
func TestFeatShapeOutsideTheFiveMatchesNothing(t *testing.T) {
	t.Parallel()
	for _, shape := range []choiceShape{0, -1, 6, 99} {
		for _, c := range []FeatChoices{NoFeatChoices{}, SingleAbilityIncrease{}, AbilityScoreImprovementChoices{}, MagicInitiateChoices{}, SkilledChoices{}} {
			if (featShape{choices: shape}).matches(c) {
				t.Errorf("shape %d matches %T", shape, c)
			}
		}
		if (featShape{choices: shape}).givesIncrease() {
			t.Errorf("shape %d gives an increase", shape)
		}
	}
}
