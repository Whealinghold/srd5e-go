package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const timeBonusAction = `{"kind":"bonusAction"}`

// Requirement: doc/specs/class-cleric.md FR-086 to FR-090, FR-093, FR-094, FR-096, FR-128, table 4.11.1 DI-01 to DI-14 (SRD 5.2.1, "Level 10: Divine Intervention", "Level 20: Greater Divine Intervention", p. 38) -
// a Cleric of level 10 may choose a Cleric-list spell of level 5 or lower (a cantrip too) whose chosen casting time is not a Reaction; Wish is allowed from level 20 only and rolls the grant's dice; the first reason is
// below_level_10, not_available, not_cleric_spell, spell_level_too_high, reaction_casting_time.
// Path: per case (DI-01 to DI-14): happy / unhappy (boundary, state error, rule interaction).
func TestDivineInterventionCheck(t *testing.T) {
	t.Parallel()
	rules := character.DefaultClericRuleData()
	cleric := []string{"cleric", "wizard"}
	reactionTimes := "[" + timeReaction + "]"
	bonusTimes := "[" + timeBonusAction + "]"
	avail, used := character.DivineInterventionState(character.DivineInterventionAvailable{}), character.DivineInterventionState(character.DivineInterventionUnavailableUntilLongRest{})
	for _, c := range []struct {
		label   string
		level   int
		state   character.DivineInterventionState
		spell   content.Spell
		reason  character.ClericReason
		granted bool
	}{
		{"DI-01", 10, avail, clcSpell(t, "cure_wounds", 1, cleric, ""), 0, false},
		{"DI-02", 10, avail, clcSpell(t, "sacred_flame", 0, cleric, ""), 0, false},
		{"DI-03", 10, avail, clcSpell(t, "flame_strike", 5, cleric, ""), 0, false},
		{"DI-04", 10, avail, clcSpell(t, "heal", 6, cleric, ""), character.ClericSpellLevelTooHigh, false},
		{"DI-05", 10, avail, clcSpell(t, "shield_of_x", 5, cleric, reactionTimes), character.ClericReactionCastingTime, false},
		{"DI-06", 10, avail, clcSpell(t, "healing_word", 1, cleric, bonusTimes), 0, false},
		{"DI-07", 10, avail, clcSpell(t, "fireball", 3, []string{"wizard"}, ""), character.ClericNotClericSpell, false},
		{"DI-08", 9, avail, clcSpell(t, "cure_wounds", 1, cleric, ""), character.ClericBelowLevel10, false},
		{"DI-09", 19, avail, clcSpell(t, "wish", 9, []string{"wizard"}, ""), character.ClericNotClericSpell, false},
		{"DI-10", 20, avail, clcSpell(t, "wish", 9, []string{"wizard"}, ""), 0, true},
		{"DI-11", 20, avail, clcSpell(t, "cure_wounds", 1, cleric, ""), 0, false},
		{"DI-12", 10, used, clcSpell(t, "cure_wounds", 1, cleric, ""), character.ClericNotAvailable, false},
		{"DI-13", 10, used, clcSpell(t, "heal", 6, cleric, ""), character.ClericNotAvailable, false},
		{"DI-14", 9, avail, clcSpell(t, "heal", 6, []string{"wizard"}, reactionTimes), character.ClericBelowLevel10, false},
		{"recharging is not available", 20, character.DivineInterventionRecharging{LongRestsRemaining: 3}, clcSpell(t, "cure_wounds", 1, cleric, ""), character.ClericNotAvailable, false},
		{"a granted spell that is a Reaction", 20, avail, clcSpell(t, "wish", 9, []string{"wizard"}, reactionTimes), character.ClericReactionCastingTime, true},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			v, err := character.DivineIntervention{}.Check(character.MustClericLevel(c.level), c.state, c.spell, nil, rules)
			if err != nil || v.Reason != c.reason || v.Granted != c.granted {
				t.Fatalf("= %+v, %v; want %v granted %v", v, err, c.reason, c.granted)
			}
			if c.granted && (v.Grant.Spell() != "wish" || v.Grant.MinLevel() != 20) {
				t.Errorf("grant = %+v", v.Grant)
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-089, FR-094 - the chosen casting time defaults to the first of the spell's casting times, must be one the spell lists, and a Reaction mode is refused; a non-SRD rule data grant
// names the spell with no literal in the rule logic.
// Path: per case (unhappy: rule interaction, invalid input).
func TestDivineInterventionCastingTimeAndHouseGrant(t *testing.T) {
	t.Parallel()
	modes := `[{"label":"Quick","kind":"bonusAction"},{"label":"Reactive","kind":"reaction","trigger":"which you take when hit"}]`
	spell := clcSpell(t, "guardian", 3, []string{"cleric"}, modes)
	rules := character.DefaultClericRuleData()
	avail := character.DivineInterventionState(character.DivineInterventionAvailable{})
	level := character.MustClericLevel(10)
	first, err := character.DivineIntervention{}.Check(level, avail, spell, nil, rules)
	if err != nil || first.Reason != 0 || first.CastingTime != (content.CastingTimeBonusAction{}) {
		t.Fatalf("default casting time = %+v, %v", first, err)
	}
	chosen := spell.CastingTimes().(content.SpellCastingTimesModes).Modes()[1].Time()
	v, err := character.DivineIntervention{}.Check(level, avail, spell, chosen, rules)
	if err != nil || v.Reason != character.ClericReactionCastingTime {
		t.Errorf("Reaction mode = %+v, %v", v, err)
	}
	_, err = character.DivineIntervention{}.Check(level, avail, spell, content.CastingTimeMinutes{}, rules)
	wantInvalid(t, err, "an unlisted casting time")
	_, err = character.DivineIntervention{}.Check(character.ClericLevel{}, avail, spell, nil, rules)
	wantInvalid(t, err, "zero level")
	_, err = character.DivineIntervention{}.Check(level, nil, spell, nil, rules)
	wantInvalid(t, err, "nil state")
	// A house grant: a different spell, unlocked at level 12 with 1d6.
	house := must(character.NewClericRuleData(nil, nil, []character.ExtraDivineInterventionGrant{must(character.NewExtraDivineInterventionGrant("plane_shift", 12, 1, dice.D6()))}))
	plane := clcSpell(t, "plane_shift", 7, []string{"wizard"}, "")
	for lvl, want := range map[int]character.ClericReason{11: character.ClericNotClericSpell, 12: 0} {
		v, err := character.DivineIntervention{}.Check(character.MustClericLevel(lvl), avail, plane, nil, house)
		if err != nil || v.Reason != want || (want == 0 && !v.Granted) {
			t.Errorf("house grant at level %d = %+v, %v", lvl, v, err)
		}
	}
}

// Requirement: doc/specs/class-cleric.md FR-094, FR-095, FR-097, table 4.11.2 GD-01, GD-04, GD-08 - the recharge state is the total of the grant's dice (2 to 8 for 2d4); a number of rolls other than the grant's
// dice or a roll outside the die's faces is invalid input.
// Path: per case (unhappy: boundary, invalid input).
func TestDivineInterventionRechargeAfterGrant(t *testing.T) {
	t.Parallel()
	grant := character.DefaultClericRuleData().ExtraDivineInterventionGrants()[0]
	for _, c := range []struct {
		label string
		rolls []int
		want  int
	}{{"GD-01", []int{1, 1}, 2}, {"GD-04", []int{4, 4}, 8}, {"GD-06", []int{2, 3}, 5}} {
		got, err := character.DivineIntervention{}.RechargeAfterGrant(grant, c.rolls)
		if err != nil || got.LongRestsRemaining != c.want {
			t.Errorf("%s: = %+v, %v", c.label, got, err)
		}
	}
	for name, rolls := range map[string][]int{"GD-08 a 5": {5, 1}, "a 0": {0, 1}, "one roll": {2}, "three rolls": {1, 1, 1}} {
		_, err := character.DivineIntervention{}.RechargeAfterGrant(grant, rolls)
		wantInvalid(t, err, name)
	}
	_, err := character.DivineIntervention{}.RechargeAfterGrant(character.ExtraDivineInterventionGrant{}, nil)
	wantInvalid(t, err, "zero grant")
}
