package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/class-cleric.md FR-075, FR-076, FR-128, table 4.14.2 RJ-14 (SRD 5.2.1, "Level 7: Blessed Strikes", p. 38) - the choice is made at level 7 or higher, once, with exactly one option;
// the first reason is below_level_7, already_chosen, option_missing, both_options, unknown_option; the choice is stored with a ClericOptionChosen event and the input is unchanged.
// Path: per case: happy / unhappy (boundary, state error, invalid input).
func TestChooseBlessedStrikes(t *testing.T) {
	t.Parallel()
	held := clcWithChoices(t, clcCleric(t, 7), clcStrikes(character.BlessedStrikesDivineStrike))
	for _, c := range []struct {
		name   string
		char   character.Character
		names  []string
		reason character.ClericReason
		option character.BlessedStrikesOption
	}{
		{"Divine Strike", clcCleric(t, 7), []string{"divine_strike"}, 0, character.BlessedStrikesDivineStrike},
		{"Potent Spellcasting at 20", clcCleric(t, 20), []string{"potent_spellcasting"}, 0, character.BlessedStrikesPotentSpellcasting},
		{"RJ-14 level 6, both named", clcCleric(t, 6), []string{"divine_strike", "potent_spellcasting"}, character.ClericBelowLevel7, 0},
		{"already chosen, none named", held, nil, character.ClericAlreadyChosen, 0},
		{"missing", clcCleric(t, 7), nil, character.ClericOptionMissing, 0},
		{"both", clcCleric(t, 7), []string{"divine_strike", "potent_spellcasting"}, character.ClericBothOptions, 0},
		{"unknown", clcCleric(t, 7), []string{"smite"}, character.ClericUnknownOption, 0},
		{"Fighter 5 / Cleric 7", clcChar(t, cl("fighter", 5), cl("cleric", 7)), []string{"divine_strike"}, 0, character.BlessedStrikesDivineStrike},
		{"Fighter 7 / Cleric 6", clcChar(t, cl("fighter", 7), cl("cleric", 6)), []string{"divine_strike"}, character.ClericBelowLevel7, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := character.ChooseBlessedStrikes(c.char, c.names)
			if err != nil {
				t.Fatal(err)
			}
			if c.reason != 0 {
				if r, ok := res.(character.ChooseBlessedStrikesRejected); !ok || r.Reason != c.reason {
					t.Errorf("result = %#v, want rejected %v", res, c.reason)
				}
				return
			}
			r, ok := res.(character.ChooseBlessedStrikesChosen)
			if !ok || r.Choices().BlessedStrikes() != c.option {
				t.Fatalf("result = %#v", res)
			}
			stored, _, _ := character.ReadFeatureState(r.Character().FeatureStates(), character.ClericChoicesKey())
			ev, isEvent := r.Events()[0].(character.ClericOptionChosen)
			if stored != r.Choices() || !isEvent || ev.Feature() != "blessed_strikes" || ev.Option() != c.option.String() {
				t.Errorf("stored %+v, events %v", stored, r.Events())
			}
			if _, held, _ := character.ReadFeatureState(c.char.FeatureStates(), character.ClericChoicesKey()); held && c.char.FeatureStates().Len() == 0 {
				t.Error("the input Character was altered")
			}
			r.Events()[0] = nil
			if r.Events()[0] == nil {
				t.Error("Events returned a shared slice")
			}
		})
	}
	res, err := character.ChooseBlessedStrikes(fgtFighter(t, 9), []string{"divine_strike"})
	if _, ok := res.(character.ChooseBlessedStrikesNotACleric); err != nil || !ok {
		t.Errorf("Fighter = %#v, %v", res, err)
	}
	_, err = character.ChooseBlessedStrikes(character.Character{}, nil)
	wantInvalid(t, err, "zero character")
}

func clcHolding(t *testing.T, level int, o character.BlessedStrikesOption) character.Character {
	t.Helper()
	return clcWithChoices(t, clcCleric(t, level), clcStrikes(o))
}

// Requirement: doc/specs/class-cleric.md FR-077, FR-078, FR-079, FR-082, FR-085, FR-128, table 4.14.2 RJ-16 and RJ-17 (SRD 5.2.1, "Level 7: Blessed Strikes" (Divine Strike), "Level 14: Improved Blessed
// Strikes", p. 38) - Divine Strike adds 1d8 (2d8 from level 14) of the chosen Necrotic or Radiant damage once a turn to a weapon hit; it does not apply when the turn is spent, for a spell attack, for another
// damage type, below level 7, with no option or with Potent Spellcasting; the first reason is the one reported.
// Path: per case: happy / unhappy (boundary, state error, rule interaction).
func TestResolveBlessedStrikesDivineStrike(t *testing.T) {
	t.Parallel()
	hit := character.DivineStrikeFacts{WeaponAttack: true, DamageType: content.DamageRadiant}
	for _, c := range []struct {
		name   string
		char   character.Character
		facts  character.DivineStrikeFacts
		reason character.ClericReason
		dice   dice.Expression
	}{
		{"level 7", clcHolding(t, 7, character.BlessedStrikesDivineStrike), hit, 0, dice.MustExpression(1, dice.D8(), 0)},
		{"level 13", clcHolding(t, 13, character.BlessedStrikesDivineStrike), hit, 0, dice.MustExpression(1, dice.D8(), 0)},
		{"level 14", clcHolding(t, 14, character.BlessedStrikesDivineStrike), character.DivineStrikeFacts{WeaponAttack: true, DamageType: content.DamageNecrotic}, 0, dice.MustExpression(2, dice.D8(), 0)},
		{"turn spent", clcHolding(t, 9, character.BlessedStrikesDivineStrike), character.DivineStrikeFacts{WeaponAttack: true, TurnSpent: true, DamageType: content.DamageRadiant}, character.ClericTurnSpent, dice.Expression{}},
		{"RJ-17 turn spent and a spell attack", clcHolding(t, 9, character.BlessedStrikesDivineStrike), character.DivineStrikeFacts{TurnSpent: true, DamageType: content.DamageRadiant}, character.ClericTurnSpent, dice.Expression{}},
		{"spell attack", clcHolding(t, 9, character.BlessedStrikesDivineStrike), character.DivineStrikeFacts{DamageType: content.DamageRadiant}, character.ClericNotWeaponAttack, dice.Expression{}},
		{"Fire damage", clcHolding(t, 9, character.BlessedStrikesDivineStrike), character.DivineStrikeFacts{WeaponAttack: true, DamageType: content.DamageFire}, character.ClericDamageTypeNotAllowed, dice.Expression{}},
		{"level 6", clcCleric(t, 6), hit, character.ClericBelowLevel7, dice.Expression{}},
		{"no option chosen", clcCleric(t, 7), hit, character.ClericNoOptionChosen, dice.Expression{}},
		{"RJ-16 Potent Spellcasting held", clcHolding(t, 7, character.BlessedStrikesPotentSpellcasting), hit, character.ClericOtherOptionChosen, dice.Expression{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := character.ResolveBlessedStrikes(c.char, c.facts)
			if err != nil {
				t.Fatal(err)
			}
			if c.reason != 0 {
				if r, ok := res.(character.ResolveBlessedStrikesNotApplicable); !ok || r.Reason != c.reason {
					t.Errorf("result = %#v, want not applicable %v", res, c.reason)
				}
				return
			}
			r, ok := res.(character.ResolveBlessedStrikesResolved)
			expr, dt, has := r.ExtraDamage()
			_, hasBonus := r.BonusDamage()
			_, hasTemp := r.TemporaryHitPoints()
			if !ok || !has || expr != c.dice || dt != c.facts.DamageType || hasBonus || hasTemp || r.TemporaryHitPointsReason() != 0 {
				t.Errorf("result = %#v", res)
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-080 to FR-085, FR-128, table 4.10.1 PS-01 to PS-12 (SRD 5.2.1, "Level 7: Blessed Strikes" (Potent Spellcasting), "Level 14: Improved Blessed Strikes", p. 38) - Potent Spellcasting
// adds the literal Wisdom modifier to Cleric cantrip damage; from level 14 it offers Temporary Hit Points equal to twice the modifier (0 for a modifier of 0 or less) within 60 feet; a spell that is not a Cleric cantrip, a
// holder of Divine Strike, a cantrip that dealt no damage, level 13 and a recipient at 61 feet give the reasons of FR-084.
// Path: per case (PS-01 to PS-12): happy / unhappy (boundary, rule interaction).
func TestResolveBlessedStrikesPotentSpellcasting(t *testing.T) {
	t.Parallel()
	flame := clcSpell(t, "sacred_flame", 0, []string{"cleric"}, "")
	offList := clcSpell(t, "fire_bolt", 0, []string{"wizard"}, "")
	level1 := clcSpell(t, "bless", 1, []string{"cleric"}, "")
	for _, c := range []struct {
		label      string
		level      int
		wis        int
		spell      content.Spell
		dealt      bool
		dist       int
		reason     character.ClericReason
		bonus      int
		temp       int
		tempOffer  bool
		tempReason character.ClericReason
	}{
		{"PS-01", 7, 3, flame, true, 0, 0, 3, 0, false, character.ClericBelowLevel14}, {"PS-02", 7, 0, flame, true, 0, 0, 0, 0, false, character.ClericBelowLevel14},
		{"PS-03", 7, -1, flame, true, 0, 0, -1, 0, false, character.ClericBelowLevel14}, {"PS-04", 7, 3, offList, true, 0, character.ClericNotClericCantrip, 0, 0, false, 0},
		{"PS-05", 7, 3, level1, true, 0, character.ClericNotClericCantrip, 0, 0, false, 0}, {"PS-06", 13, 3, flame, true, 0, 0, 3, 0, false, character.ClericBelowLevel14},
		{"PS-07", 14, 3, flame, true, 0, 0, 3, 6, true, 0}, {"PS-08", 14, 1, flame, true, 0, 0, 1, 2, true, 0}, {"PS-09", 14, 0, flame, true, 0, 0, 0, 0, true, 0},
		{"PS-10", 14, -2, flame, true, 0, 0, -2, 0, true, 0}, {"PS-11", 14, 3, flame, false, 0, character.ClericNoDamageDealt, 0, 0, false, 0},
		{"PS-12", 20, 5, flame, true, 0, 0, 5, 10, true, 0}, {"60 feet", 14, 3, flame, true, 60, 0, 3, 6, true, 0},
		{"61 feet", 14, 3, flame, true, 61, 0, 3, 0, false, character.ClericRecipientOutOfRange},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			res, err := character.ResolveBlessedStrikes(clcHolding(t, c.level, character.BlessedStrikesPotentSpellcasting),
				character.PotentSpellcastingFacts{Spell: c.spell, DamageDealt: c.dealt, Wisdom: modifierOf(t, c.wis), RecipientDistanceFeet: c.dist})
			if err != nil {
				t.Fatal(err)
			}
			if c.reason != 0 {
				if r, ok := res.(character.ResolveBlessedStrikesNotApplicable); !ok || r.Reason != c.reason {
					t.Errorf("result = %#v, want not applicable %v", res, c.reason)
				}
				return
			}
			r, ok := res.(character.ResolveBlessedStrikesResolved)
			bonus, hasBonus := r.BonusDamage()
			temp, offered := r.TemporaryHitPoints()
			_, _, hasExtra := r.ExtraDamage()
			if !ok || !hasBonus || bonus != c.bonus || offered != c.tempOffer || temp != c.temp || r.TemporaryHitPointsReason() != c.tempReason || hasExtra {
				t.Errorf("result = %#v", res)
			}
		})
	}
	// FR-082, FR-084: a holder of Divine Strike gets nothing from a cantrip.
	res, _ := character.ResolveBlessedStrikes(clcHolding(t, 14, character.BlessedStrikesDivineStrike), character.PotentSpellcastingFacts{Spell: flame, DamageDealt: true})
	if r, ok := res.(character.ResolveBlessedStrikesNotApplicable); !ok || r.Reason != character.ClericOtherOptionChosen {
		t.Errorf("Divine Strike holder cast = %#v", res)
	}
}

// Requirement: doc/specs/class-cleric.md FR-122 - a character with no Cleric levels is ResolveBlessedStrikesNotACleric; a zero Character, nil facts, facts of no known kind, a foreign state under the key and a negative
// recipient distance are errors matching srderr.ErrInvalidInput.
// Path: per case (unhappy: state error, invalid input).
func TestResolveBlessedStrikesNotAClericAndInvalid(t *testing.T) {
	t.Parallel()
	hit := character.DivineStrikeFacts{WeaponAttack: true, DamageType: content.DamageRadiant}
	res, err := character.ResolveBlessedStrikes(fgtFighter(t, 9), hit)
	if _, ok := res.(character.ResolveBlessedStrikesNotACleric); err != nil || !ok {
		t.Errorf("Fighter = %#v, %v", res, err)
	}
	_, err = character.ResolveBlessedStrikes(character.Character{}, hit)
	wantInvalid(t, err, "zero character")
	_, err = character.ResolveBlessedStrikes(clcHolding(t, 7, character.BlessedStrikesDivineStrike), nil)
	wantInvalid(t, err, "nil facts")
	foreign := must(character.WithFeatureState(clcCleric(t, 7), character.MustFeatureStateKey[character.ChosenSubclass]("cleric_choices"), must(character.NewChosenSubclass("life_domain"))))
	_, err = character.ResolveBlessedStrikes(foreign, hit)
	wantInvalid(t, err, "foreign state")
	_, err = character.ResolveBlessedStrikes(clcHolding(t, 14, character.BlessedStrikesPotentSpellcasting),
		character.PotentSpellcastingFacts{Spell: clcSpell(t, "sacred_flame", 0, []string{"cleric"}, ""), DamageDealt: true, RecipientDistanceFeet: -1})
	wantInvalid(t, err, "negative distance")
}
