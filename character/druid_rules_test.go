package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

var druRules = character.DruidRules{}

// Requirement: doc/specs/class-druid.md FR-052, FR-053, FR-054, FR-055 and table D (SRD 5.2.1, "Level 1: Primal Order", p. 42) - the Magician adds the Wisdom
// modifier, at least +1, to Intelligence (Arcana) and Intelligence (Nature) checks and to nothing else; the Warden and no order add nothing.
// Path: per case (happy / unhappy: boundary at the minimum, rule interaction).
func TestMagicianCheckBonus(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label string
		order character.PrimalOrder
		wis   int // Wisdom score
		a     ability.Ability
		skill content.Skill
		want  int
	}{
		{"score 1 (-5) gives the minimum", character.PrimalOrderMagician, 1, ability.Intelligence, content.SkillArcana, 1},
		{"score 8 (-1)", character.PrimalOrderMagician, 8, ability.Intelligence, content.SkillArcana, 1},
		{"score 10 (0)", character.PrimalOrderMagician, 10, ability.Intelligence, content.SkillNature, 1},
		{"score 12 (+1)", character.PrimalOrderMagician, 12, ability.Intelligence, content.SkillNature, 1},
		{"score 14 (+2)", character.PrimalOrderMagician, 14, ability.Intelligence, content.SkillArcana, 2},
		{"score 20 (+5)", character.PrimalOrderMagician, 20, ability.Intelligence, content.SkillNature, 5},
		{"score 30 (+10)", character.PrimalOrderMagician, 30, ability.Intelligence, content.SkillArcana, 10},
		{"Intelligence (History)", character.PrimalOrderMagician, 20, ability.Intelligence, content.SkillHistory, 0},
		{"Wisdom (Nature)", character.PrimalOrderMagician, 20, ability.Wisdom, content.SkillNature, 0},
		{"Wisdom (Perception)", character.PrimalOrderMagician, 20, ability.Wisdom, content.SkillPerception, 0},
		{"Warden", character.PrimalOrderWarden, 20, ability.Intelligence, content.SkillArcana, 0},
		{"no order", 0, 20, ability.Intelligence, content.SkillArcana, 0},
	} {
		if got := druRules.MagicianCheckBonus(c.order, score(t, c.wis).Modifier(), c.a, c.skill); got.Value() != c.want {
			t.Errorf("%s: bonus = %d, want %d", c.label, got.Value(), c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-079 (SRD 5.2.1, "Rules While Shape-Shifted", p. 43) - the retained statistics are the creature type, Hit Points, Hit
// Dice, the Intelligence, Wisdom and Charisma scores, class features, languages, feats, the skill and saving throw proficiencies and the Proficiency Bonus; every
// other statistic is replaced and the result is a copy.
// Path: happy.
func TestShapeShiftedStats(t *testing.T) {
	t.Parallel()
	s := druRules.ShapeShiftedStats()
	want := []character.RetainedStat{character.RetainedCreatureType, character.RetainedHitPoints, character.RetainedHitDice, character.RetainedIntelligenceScore,
		character.RetainedWisdomScore, character.RetainedCharismaScore, character.RetainedClassFeatures, character.RetainedLanguages, character.RetainedFeats,
		character.RetainedSkillProficiencies, character.RetainedSavingThrowProficiencies, character.RetainedProficiencyBonus}
	if got := s.Retained(); !slices.Equal(got, want) {
		t.Errorf("Retained = %v, want %v", got, want)
	}
	for _, r := range want {
		if s.Replaced(r) {
			t.Errorf("%v is replaced", r)
		}
	}
	if !s.Replaced(character.RetainedStat(99)) {
		t.Error("a statistic not retained is not replaced")
	}
	got := s.Retained()
	got[0] = 0
	if s.Retained()[0] != character.RetainedCreatureType {
		t.Error("Retained shares storage")
	}
}

// Requirement: doc/specs/class-druid.md FR-080, FR-081, FR-152 and table E (SRD 5.2.1, "Rules While Shape-Shifted", p. 43; "The Bonus Doesn't Stack", p. 8) - the
// shape-shifted modifier is the higher of the Druid's own (ability modifier plus the Proficiency Bonus once when proficient) and the stat block's modifier when it
// lists one; the bonus is the total-level bonus the caller passes.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestShapeShiftedModifier(t *testing.T) {
	t.Parallel()
	bonus := func(n int) character.ProficiencyBonus {
		return character.ProficiencyBonusFor(character.MustLevel(1 + (n-2)*4))
	}
	for _, c := range []struct {
		label  string
		mod    int
		proof  character.ShapeShiftedProficiency
		pb     int
		beast  int
		listed bool
		want   int
		source character.ShapeShiftedSource
	}{
		{"not proficient, none listed", 1, character.ShapeShiftedNotProficient, 2, 0, false, 1, character.ShapeShiftedSourceDruid},
		{"proficient, none listed", 2, character.ShapeShiftedProficient, 2, 0, false, 4, character.ShapeShiftedSourceDruid},
		{"stat block higher", 2, character.ShapeShiftedProficient, 3, 6, true, 6, character.ShapeShiftedSourceStatBlock},
		{"own higher", 2, character.ShapeShiftedProficient, 3, 4, true, 5, character.ShapeShiftedSourceDruid},
		{"equal", 2, character.ShapeShiftedProficient, 3, 5, true, 5, character.ShapeShiftedSourceDruid},
		{"not proficient, stat block listed", 0, character.ShapeShiftedNotProficient, 3, 3, true, 3, character.ShapeShiftedSourceStatBlock},
		{"proficient in both", 1, character.ShapeShiftedProficient, 2, 3, true, 3, character.ShapeShiftedSourceDruid},
		{"Druid 3 / Fighter 5 passes +3", 2, character.ShapeShiftedProficient, 3, 0, false, 5, character.ShapeShiftedSourceDruid},
		{"stat block ignored when unlisted", -3, character.ShapeShiftedNotProficient, 2, 9, false, -3, character.ShapeShiftedSourceDruid},
	} {
		got := druRules.ShapeShiftedModifier(modifierOf(t, c.mod), c.proof, bonus(c.pb), modifier.New(c.beast), c.listed)
		if got.Modifier.Value() != c.want || got.Source != c.source {
			t.Errorf("%s: %d from %v, want %d from %v", c.label, got.Modifier.Value(), got.Source, c.want, c.source)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-082, FR-083, FR-111, FR-155 (SRD 5.2.1, "Rules While Shape-Shifted", "Level 18: Beast Spells", p. 43) - not in a form every
// spell is allowed; in a form at Druid level 17 or below every spell is blocked with CannotCastInBeastForm; from level 18 only a spell with a costed or consumed Material
// component is blocked, with MaterialComponentRestricted, and a plain Material component is allowed.
// Path: per case (happy / unhappy: boundary at 17 and 18, state error).
func TestBeastFormCastCheck(t *testing.T) {
	t.Parallel()
	inForm := druState(t, 20, false, character.DruidStateFields{KnownForms: druIDs("wolf"), ActiveForm: "wolf"})
	free := character.DruidFeatureState{}
	plain := druSpell(t, "plain", 1, []string{"druid"}, "", verbalOnly())
	plainMaterial := druSpell(t, "plain_material", 1, []string{"druid"}, "", materialOnly("[]", false))
	costed := druSpell(t, "costed", 1, []string{"druid"}, "", materialOnly("[50]", false))
	consumed := druSpell(t, "consumed", 1, []string{"druid"}, "", materialOnly("[]", true))
	for _, c := range []struct {
		label string
		state character.DruidFeatureState
		level int
		spell content.Spell
		want  character.BeastFormCastCheck
	}{
		{"no form, level 2", free, 2, costed, character.BeastFormCastCheckAllowed{}},
		{"form, 17", inForm, 17, plain, character.BeastFormCastCheckBlocked{Reason: character.DruidCannotCastInBeastForm}},
		{"form, 17, costed", inForm, 17, costed, character.BeastFormCastCheckBlocked{Reason: character.DruidCannotCastInBeastForm}},
		{"form, 2", inForm, 2, plain, character.BeastFormCastCheckBlocked{Reason: character.DruidCannotCastInBeastForm}},
		{"form, 18, no Material", inForm, 18, plain, character.BeastFormCastCheckAllowed{}},
		{"form, 18, plain Material", inForm, 18, plainMaterial, character.BeastFormCastCheckAllowed{}},
		{"form, 18, costed", inForm, 18, costed, character.BeastFormCastCheckBlocked{Reason: character.DruidMaterialComponentRestricted}},
		{"form, 20, consumed", inForm, 20, consumed, character.BeastFormCastCheckBlocked{Reason: character.DruidMaterialComponentRestricted}},
	} {
		if got := druRules.BeastFormCastCheck(c.state, character.MustDruidLevel(c.level), c.spell); got != c.want {
			t.Errorf("%s: check = %#v, want %#v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-105, FR-106 and table F (SRD 5.2.1, "Level 7: Elemental Fury", p. 43) - with Potent Spellcasting a Druid cantrip, including one that
// counts as a Druid spell through a feature, gets the Wisdom modifier (0 and -1 are kept); a level 1+ spell and a cantrip of another class get NotADruidCantrip; with Primal Strike
// or no option chosen every spell gets FeatureNotChosen.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestPotentSpellcastingBonus(t *testing.T) {
	t.Parallel()
	potent := character.MustDruidFeatureState(character.DruidStateFields{ElementalFury: character.ElementalFuryPotentSpellcasting}, druFacts(t, 7, false), druTables(t))
	strike := character.MustDruidFeatureState(character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike}, druFacts(t, 7, false), druTables(t))
	cantrip := druSpell(t, "druid_cantrip", 0, []string{"druid"}, "", "")
	other := druSpell(t, "wizard_cantrip", 0, []string{"wizard"}, "", "")
	leveled := druSpell(t, "druid_spell", 1, []string{"druid"}, "", "")
	for _, c := range []struct {
		label string
		state character.DruidFeatureState
		spell content.Spell
		wis   int
		extra []content.SpellID
		want  character.PotentBonus
	}{
		{"Druid cantrip +3", potent, cantrip, 16, nil, character.PotentBonusApplied{Modifier: modifier.New(3)}},
		{"modifier 0", potent, cantrip, 10, nil, character.PotentBonusApplied{Modifier: modifier.New(0)}},
		{"modifier -1", potent, cantrip, 8, nil, character.PotentBonusApplied{Modifier: modifier.New(-1)}},
		{"Magician's extra cantrip +2", potent, other, 14, []content.SpellID{"wizard_cantrip"}, character.PotentBonusApplied{Modifier: modifier.New(2)}},
		{"cantrip of another class", potent, other, 16, []content.SpellID{"another"}, character.PotentBonusNotApplicable{Reason: character.DruidNotADruidCantrip}},
		{"level 1 Druid spell", potent, leveled, 16, nil, character.PotentBonusNotApplicable{Reason: character.DruidNotADruidCantrip}},
		{"level 1 spell counted as Druid", potent, leveled, 16, []content.SpellID{"druid_spell"}, character.PotentBonusNotApplicable{Reason: character.DruidNotADruidCantrip}},
		{"Primal Strike chosen", strike, cantrip, 16, nil, character.PotentBonusNotApplicable{Reason: character.DruidFeatureNotChosen}},
		{"no option", character.DruidFeatureState{}, cantrip, 16, nil, character.PotentBonusNotApplicable{Reason: character.DruidFeatureNotChosen}},
	} {
		if got := druRules.PotentSpellcastingBonus(c.state, c.spell, score(t, c.wis).Modifier(), c.extra); got != c.want {
			t.Errorf("%s: bonus = %#v, want %#v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-110 and table C (SRD 5.2.1, "Level 15: Improved Elemental Fury", p. 43) - from level 15 with Potent Spellcasting a distance range of
// 10 feet or more gains 300 feet (10 feet gives 310, 60 gives 360, 120 at level 20 gives 420, a range in miles is converted); Self, Touch, Sight, 5 feet and the other ranges are
// unchanged; at level 14 nothing changes; with Primal Strike chosen from level 15 the result is NotApplicable (FeatureNotChosen).
// Path: per case (happy / unhappy: boundary).
func TestImprovedPotentRange(t *testing.T) {
	t.Parallel()
	potent := character.MustDruidFeatureState(character.DruidStateFields{ElementalFury: character.ElementalFuryPotentSpellcasting}, druFacts(t, 15, false), druTables(t))
	strike := character.MustDruidFeatureState(character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike}, druFacts(t, 15, false), druTables(t))
	feet := func(n int) content.SpellRange { return must(content.NewSpellRangeDistance(n, content.DistanceFeet)) }
	for _, c := range []struct {
		label string
		state character.DruidFeatureState
		level int
		r     content.SpellRange
		want  character.RangeChange
	}{
		{"60 feet at 14", potent, 14, feet(60), character.RangeChangeUnchanged{}},
		{"60 feet at 15", potent, 15, feet(60), character.RangeChangeIncreased{NewFeet: 360}},
		{"exactly 10 feet", potent, 15, feet(10), character.RangeChangeIncreased{NewFeet: 310}},
		{"5 feet", potent, 15, feet(5), character.RangeChangeUnchanged{}},
		{"9 feet", potent, 15, feet(9), character.RangeChangeUnchanged{}},
		{"120 feet at 20", potent, 20, feet(120), character.RangeChangeIncreased{NewFeet: 420}},
		{"1 mile", potent, 15, must(content.NewSpellRangeDistance(1, content.DistanceMiles)), character.RangeChangeIncreased{NewFeet: 5580}},
		{"Self", potent, 15, content.NewSpellRangeSelf(), character.RangeChangeUnchanged{}},
		{"Touch", potent, 15, content.NewSpellRangeTouch(), character.RangeChangeUnchanged{}},
		{"Sight", potent, 15, content.NewSpellRangeSight(), character.RangeChangeUnchanged{}},
		{"Primal Strike chosen", strike, 15, feet(60), character.RangeChangeNotApplicable{Reason: character.DruidFeatureNotChosen}},
		{"Primal Strike chosen at 14", strike, 14, feet(60), character.RangeChangeUnchanged{}},
		{"no option at 15", character.DruidFeatureState{}, 15, feet(60), character.RangeChangeNotApplicable{Reason: character.DruidFeatureNotChosen}},
	} {
		if got := druRules.ImprovedPotentRange(c.state, character.MustDruidLevel(c.level), c.r); got != c.want {
			t.Errorf("%s: change = %#v, want %#v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-106 to FR-108, FR-107, D-7 and table C (SRD 5.2.1, "Level 7: Elemental Fury", "Level 15: Improved Elemental Fury", p. 43) - Primal Strike
// gives 1d8 (2d8 from level 15) of Cold, Fire, Lightning or Thunder on a hit with a weapon, in Wild Shape or not, and with a Beast form's attack in Wild Shape; a miss, a spell, another
// attack and a Beast form's attack outside Wild Shape are not eligible; with Potent Spellcasting or no option chosen the result is FeatureNotChosen, whatever the attack.
// Path: per case (happy / unhappy: boundary at 14 and 15, rule interaction).
func TestPrimalStrikeDice(t *testing.T) {
	t.Parallel()
	strike := func(level int, form bool) character.DruidFeatureState {
		f := character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike}
		if form {
			f.KnownForms, f.ActiveForm = druIDs("wolf"), "wolf"
		}
		return druState(t, level, false, f)
	}
	potent := druState(t, 7, false, character.DruidStateFields{ElementalFury: character.ElementalFuryPotentSpellcasting})
	hit, miss := rogAttack(t, true), rogAttack(t, false)
	damage := func(n int) character.PrimalStrikeOutcome {
		return character.PrimalStrikeOutcomeDamage{Damage: dice.MustExpression(n, dice.D8(), 0), Types: []content.DamageType{content.DamageCold, content.DamageFire, content.DamageLightning, content.DamageThunder}}
	}
	ineligible := character.PrimalStrikeOutcomeIneligible{Reason: character.DruidAttackNotEligible}
	notChosen := character.PrimalStrikeOutcomeIneligible{Reason: character.DruidFeatureNotChosen}
	for _, c := range []struct {
		label  string
		state  character.DruidFeatureState
		level  int
		source character.PrimalStrikeAttackSource
		attack d20.AttackRollResult
		want   character.PrimalStrikeOutcome
	}{
		{"weapon hit at 7", strike(7, false), 7, character.PrimalStrikeSourceWeapon, hit, damage(1)},
		{"weapon hit at 14", strike(14, false), 14, character.PrimalStrikeSourceWeapon, hit, damage(1)},
		{"weapon hit at 15", strike(15, false), 15, character.PrimalStrikeSourceWeapon, hit, damage(2)},
		{"weapon hit in Wild Shape (D-7)", strike(7, true), 7, character.PrimalStrikeSourceWeapon, hit, damage(1)},
		{"Beast attack in Wild Shape", strike(20, true), 20, character.PrimalStrikeSourceBeastForm, hit, damage(2)},
		{"Beast attack outside Wild Shape", strike(7, false), 7, character.PrimalStrikeSourceBeastForm, hit, ineligible},
		{"weapon miss", strike(7, false), 7, character.PrimalStrikeSourceWeapon, miss, ineligible},
		{"Beast miss in Wild Shape", strike(7, true), 7, character.PrimalStrikeSourceBeastForm, miss, ineligible},
		{"spell hit", strike(7, false), 7, character.PrimalStrikeSourceSpell, hit, ineligible},
		{"other hit", strike(7, true), 7, character.PrimalStrikeSourceOther, hit, ineligible},
		{"Potent chosen, weapon hit", potent, 7, character.PrimalStrikeSourceWeapon, hit, notChosen},
		{"Potent chosen, miss", potent, 7, character.PrimalStrikeSourceWeapon, miss, notChosen},
		{"no option", character.DruidFeatureState{}, 7, character.PrimalStrikeSourceWeapon, hit, notChosen},
	} {
		got := druRules.PrimalStrikeDice(c.state, character.MustDruidLevel(c.level), c.source, c.attack)
		if d, ok := got.(character.PrimalStrikeOutcomeDamage); ok {
			w := c.want.(character.PrimalStrikeOutcomeDamage)
			if d.Damage != w.Damage || !slices.Equal(d.Types, w.Types) {
				t.Errorf("%s: outcome = %#v, want %#v", c.label, got, c.want)
			}
		} else if got != c.want {
			t.Errorf("%s: outcome = %#v, want %#v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-123, FR-128 and table C, K (SRD 5.2.1, "Level 3: Land's Aid", "Level 6: Natural Recovery", p. 46) - Land's Aid is 2d6 below level 10, 3d6 from 10
// and 4d6 from 14 (boundaries 9, 10, 13, 14); the Natural Recovery budget is half the Druid level rounded up.
// Path: per case (unhappy: boundary).
func TestLandsAidDiceAndNaturalRecoveryBudget(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ level, dice, budget int }{{3, 2, 2}, {6, 2, 3}, {7, 2, 4}, {9, 2, 5}, {10, 3, 5}, {13, 3, 7}, {14, 4, 7}, {20, 4, 10}} {
		if got := druRules.LandsAidDice(character.MustDruidLevel(c.level)); got != dice.MustExpression(c.dice, dice.D6(), 0) {
			t.Errorf("LandsAidDice(%d) = %v, want %dd6", c.level, got, c.dice)
		}
		if got := druRules.NaturalRecoveryBudget(character.MustDruidLevel(c.level)); got != c.budget {
			t.Errorf("NaturalRecoveryBudget(%d) = %d, want %d", c.level, got, c.budget)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-072 to FR-074 and table B (SRD 5.2.1, "Level 2: Wild Shape", p. 42) - a form is eligible only if it is a Beast, its Challenge Rating is at most the
// maximum for the Druid level and it has no Fly Speed before level 8; when several reasons hold the first of NotABeast, ChallengeRatingTooHigh, FlySpeedNotAllowed is reported; below level 2 there
// is no tier.
// Path: per case (happy / unhappy: boundary, invalid input, rule interaction).
func TestFormEligibility(t *testing.T) {
	t.Parallel()
	tables := druTables(t)
	other := func(cr string, fly bool) character.BeastFormFacts {
		return must(character.NewBeastFormFacts("thing", content.CreatureTypeDragon, content.MustChallengeRating(cr), fly))
	}
	eligible := character.FormEligibilityEligible{}
	for _, c := range []struct {
		label string
		form  character.BeastFormFacts
		level int
		want  character.FormEligibility
	}{
		{"Beast at max CR", druForm("a", "1/4", false), 2, eligible},
		{"one CR step over", druForm("a", "1/2", false), 2, character.FormEligibilityIneligible{Reason: character.DruidChallengeRatingTooHigh}},
		{"CR 1/2 at level 3", druForm("a", "1/2", false), 3, character.FormEligibilityIneligible{Reason: character.DruidChallengeRatingTooHigh}},
		{"CR 1/2 at level 4", druForm("a", "1/2", false), 4, eligible},
		{"CR 1 at level 7", druForm("a", "1", false), 7, character.FormEligibilityIneligible{Reason: character.DruidChallengeRatingTooHigh}},
		{"CR 1 at level 8", druForm("a", "1", false), 8, eligible},
		{"flying below 8", druForm("a", "1/2", true), 7, character.FormEligibilityIneligible{Reason: character.DruidFlySpeedNotAllowed}},
		{"flying at 8", druForm("a", "1", true), 8, eligible},
		{"not a Beast", other("1/4", false), 8, character.FormEligibilityIneligible{Reason: character.DruidNotABeast}},
		{"not a Beast and too high", other("1", false), 2, character.FormEligibilityIneligible{Reason: character.DruidNotABeast}},
		{"too high and flying", druForm("a", "2", true), 4, character.FormEligibilityIneligible{Reason: character.DruidChallengeRatingTooHigh}},
		{"CR 0", druForm("a", "0", false), 2, eligible},
		{"level 1", druForm("a", "0", false), 1, character.FormEligibilityIneligible{Reason: character.DruidFeatureNotGranted}},
	} {
		if got := druRules.FormEligibility(c.form, tables, character.MustDruidLevel(c.level)); got != c.want {
			t.Errorf("%s: eligibility = %#v, want %#v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-044 to FR-047 (SRD 5.2.1, "Level 1: Druidic", p. 42) - a creature that knows Druidic spots a hidden message with no check and reads it; any other
// needs a DC 15 Intelligence (Investigation) check to spot it and cannot decipher it without magic, whatever the check's result: the two facts are independent.
// Path: per case (happy / unhappy: rule interaction).
func TestDruidicDetection(t *testing.T) {
	t.Parallel()
	known := druRules.DruidicDetection([]content.Language{content.LanguageCommon, content.LanguageDruidic})
	if _, ok := known.Presence.(character.DruidicPresenceAutomaticallySpotted); !ok || known.Content != character.DruidicContentReadable {
		t.Errorf("a Druidic speaker perceives %#v", known)
	}
	for _, langs := range [][]content.Language{nil, {content.LanguageCommon, content.LanguageElvish}} {
		d := druRules.DruidicDetection(langs)
		check, ok := d.Presence.(character.DruidicPresenceRequiresCheck)
		if !ok || check.DC != 15 || check.Ability != ability.Intelligence || check.Skill != content.SkillInvestigation || d.Content != character.DruidicContentNeedsMagic {
			t.Errorf("a stranger to Druidic (%v) perceives %#v", langs, d)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-044, FR-115, FR-040, FR-041, FR-120, FR-138 (SRD 5.2.1, "Level 1: Druidic", p. 42; "Level 3: Circle of the Land Spells", p. 46) - every Druid knows Druidic; Speak with Animals is always prepared from
// level 1 and, with the Circle of the Land and a land chosen, so are the Circle Spells of that land at the Druid level (read from the state each time: after the land changes none of the old land remains); Longevity is the ratio 10.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestDruidAlwaysPreparedLanguagesAndLongevity(t *testing.T) {
	t.Parallel()
	tables := druTables(t)
	if got := druRules.Languages(); !slices.Equal(got, []content.Language{content.LanguageDruidic}) {
		t.Errorf("Languages = %v", got)
	}
	got := druRules.Languages()
	got[0] = 0
	if druRules.Languages()[0] != content.LanguageDruidic || character.ArchdruidYearsPerYearAged != 10 {
		t.Error("Languages shares storage, or Longevity is not ten years per year")
	}
	ids := func(s ...string) []content.SpellID {
		out := make([]content.SpellID, len(s))
		for i, v := range s {
			out[i] = content.SpellID(v)
		}
		return out
	}
	for _, c := range []struct {
		label  string
		level  int
		circle bool
		land   character.LandType
		want   []content.SpellID
	}{
		{"level 1", 1, false, 0, ids("speak_with_animals")},
		{"circle, no land", 5, true, 0, ids("speak_with_animals")},
		{"land without the subclass cannot be", 5, false, 0, ids("speak_with_animals")},
		{"arid at 3", 3, true, character.LandArid, ids("speak_with_animals", "blur", "burning_hands", "fire_bolt")},
		{"arid at 4", 4, true, character.LandArid, ids("speak_with_animals", "blur", "burning_hands", "fire_bolt")},
		{"polar at 9", 9, true, character.LandPolar, ids("speak_with_animals", "fog_cloud", "hold_person", "ray_of_frost", "sleet_storm", "ice_storm", "cone_of_cold")},
	} {
		s := druState(t, c.level, c.circle, character.DruidStateFields{LandType: c.land})
		if got := druRules.AlwaysPreparedSpells(druFacts(t, c.level, c.circle), s, tables); !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, want %v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-131, FR-132, FR-157, FR-138 and table L (SRD 5.2.1, "Level 10: Nature's Ward", p. 46) - from level 10 with the Circle of the Land Nature's Ward gives immunity to the Poisoned condition and the Resistance of the current
// land (Fire, Cold, Lightning, Poison); with no land type chosen the immunity stays and the Resistance reason is LandTypeNotChosen; at level 9, or without the subclass, it gives nothing and is no error.
// Path: per case (happy / unhappy: boundary, missing content).
func TestNaturesWard(t *testing.T) {
	t.Parallel()
	tables := druTables(t)
	poisoned := []content.ConditionKind{content.ConditionPoisoned}
	for _, c := range []struct {
		label  string
		level  int
		circle bool
		land   character.LandType
		want   character.NaturesWardEffect
	}{
		{"arid", 10, true, character.LandArid, character.NaturesWardEffect{ImmuneTo: poisoned, Resistance: content.DamageFire}},
		{"polar", 14, true, character.LandPolar, character.NaturesWardEffect{ImmuneTo: poisoned, Resistance: content.DamageCold}},
		{"temperate", 20, true, character.LandTemperate, character.NaturesWardEffect{ImmuneTo: poisoned, Resistance: content.DamageLightning}},
		{"tropical", 10, true, character.LandTropical, character.NaturesWardEffect{ImmuneTo: poisoned, Resistance: content.DamagePoison}},
		{"no land", 10, true, 0, character.NaturesWardEffect{ImmuneTo: poisoned, ResistanceReason: character.DruidLandTypeNotChosen}},
		{"level 9", 9, true, character.LandArid, character.NaturesWardEffect{ImmuneTo: []content.ConditionKind{}}},
		{"no subclass", 10, false, 0, character.NaturesWardEffect{ImmuneTo: []content.ConditionKind{}}},
	} {
		got := druRules.NaturesWard(druFacts(t, c.level, c.circle), druState(t, c.level, c.circle, character.DruidStateFields{LandType: c.land}), tables)
		if !slices.Equal(got.ImmuneTo, c.want.ImmuneTo) || got.ImmuneTo == nil || got.Resistance != c.want.Resistance || got.ResistanceReason != c.want.ResistanceReason {
			t.Errorf("%s: %#v, want %#v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-054, FR-055, FR-090, FR-134 (SRD 5.2.1, "Level 1: Primal Order", "Level 2: Wild Companion", "Level 14: Nature's Sanctuary", pp. 42-46) - the Warden grants Martial weapons and Medium armor and the Magician and no order grant
// neither; the Wild Companion familiar is Fey; the Cube gives Half Cover.
// Path: per case (happy / unhappy: rule interaction).
func TestDruidOrderGrantsAndFacts(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		order         character.PrimalOrder
		martial, mail bool
	}{{character.PrimalOrderWarden, true, true}, {character.PrimalOrderMagician, false, false}, {0, false, false}} {
		if c.order.GrantsMartialWeapons() != c.martial || c.order.GrantsMediumArmor() != c.mail {
			t.Errorf("%v grants Martial %v and Medium armor %v, want %v and %v", c.order, c.order.GrantsMartialWeapons(), c.order.GrantsMediumArmor(), c.martial, c.mail)
		}
	}
	if (character.WildCompanionCast{}).FamiliarType() != content.CreatureTypeFey || !(character.NaturesSanctuaryCreated{}).HalfCover() {
		t.Error("the familiar is not Fey, or the Cube gives no Half Cover")
	}
}
