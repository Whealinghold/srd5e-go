package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// effectsOf takes feat (with choices) at level 20 and returns its declared effects.
func featEffectsOf(t *testing.T, feat content.Feat, choices character.FeatChoices) []character.FeatEffect {
	t.Helper()
	return character.FeatEffects{}.Effects(heldFeats(t, feat, choices))
}

func fatePrompt() character.UseRenewedBy {
	return must(character.NewUseRenewedBy(character.RenewOnInitiativeRolled, character.RenewOnShortRestFinished, character.RenewOnLongRestFinished))
}

// Requirement: per case
// Path: per case.
func TestFeatEffectsOfEachFeat(t *testing.T) {
	t.Parallel()
	dim := [2]content.LightLevel{content.LightDim, content.LightDarkness}
	defense, err := character.NewFlatBonus(modifier.New(1), character.FeatureFlatBonusSource{Feature: "defense"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, requirement, path string
		feat                    content.Feat
		choices                 character.FeatChoices
		want                    []character.FeatEffect
	}{
		{"Alert", featsSpec + ` FR-044, Table E; SRD 5.2.1, "Alert", p. 87`, "happy", alertFeat(), character.NoFeatChoices{}, []character.FeatEffect{
			character.InitiativeProficiency{}, character.InitiativeSwap{Timing: character.AfterRollingInitiative, Partner: character.WillingAllyInSameCombat, BlockedByCondition: content.ConditionIncapacitated}}},
		{"Magic Initiate declares none", featsSpec + " FR-044, Table E", "happy", initiateFeat(), clericInitiate(), nil},
		{"Savage Attacker", featsSpec + " FR-044, FR-049, Table E", "happy", savageFeat(), character.NoFeatChoices{}, []character.FeatEffect{character.RollDamageTwice{Trigger: character.TriggerWeaponHit, Limit: character.OncePerTurn{}}}},
		{"Skilled declares none", featsSpec + " FR-044, Table E", "happy", skilledFeat(), skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledSkill{Skill: content.SkillArcana}, character.SkilledSkill{Skill: content.SkillHistory}), nil},
		{"Ability Score Improvement", featsSpec + " FR-044, Table E", "happy", asiFeat(), must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0)), nil},
		{"Grappler", featsSpec + " FR-044, FR-049, Table E", "happy", grapplerFeat(), singleOf(ability.Strength), []character.FeatEffect{
			character.UnarmedStrikeDamageAndGrapple{Limit: character.OncePerTurn{}}, character.AdvantageAgainstGrappledByYou{}, character.GrappleMovementDiscount{TargetSize: character.SameSizeOrSmaller}}},
		{"Archery", featsSpec + " FR-044, Table E", "happy", archeryFeat(), character.NoFeatChoices{}, []character.FeatEffect{character.AttackRollBonus{Bonus: modifier.New(2), WeaponType: content.WeaponRanged}}},
		{"Defense", featsSpec + " FR-044, Table E", "happy", defenseFeat(), character.NoFeatChoices{}, []character.FeatEffect{
			character.ArmoredArmorClassBonus{Bonus: defense, WhileWearing: [3]content.ArmorCategory{content.ArmorLight, content.ArmorMedium, content.ArmorHeavy}}}},
		{"Great Weapon Fighting", featsSpec + " FR-044, FR-047, Table E", "happy", fightingStyleFeat("great_weapon_fighting"), character.NoFeatChoices{}, []character.FeatEffect{
			character.DamageDieMinimum{Minimum: 3, AppliesToRolledAtMost: 2, WeaponType: content.WeaponMelee, Hands: character.TwoHands, AnyOfProperties: [2]content.WeaponProperty{content.WeaponPropertyTwoHanded, content.WeaponPropertyVersatile}}}},
		{"Two-Weapon Fighting", featsSpec + " FR-044, Table E", "happy", fightingStyleFeat("two_weapon_fighting"), character.NoFeatChoices{}, []character.FeatEffect{character.ExtraAttackAbilityModifierToDamage{RequiredProperty: content.WeaponPropertyLight}}},
		{"Boon of Combat Prowess", featsSpec + " FR-044, FR-049, Table E", "happy", boonFeat("boon_of_combat_prowess"), singleOf(ability.Strength), []character.FeatEffect{character.MissBecomesHit{Limit: character.OnceUntilStartOfYourNextTurn{}}}},
		{"Boon of Dimensional Travel", featsSpec + " FR-044, Table E", "happy", boonFeat("boon_of_dimensional_travel"), singleOf(ability.Strength), []character.FeatEffect{
			character.BlinkSteps{DistanceFeet: 30, After: [2]character.FeatTriggerAction{character.TriggerAttackAction, character.TriggerMagicAction}, Destination: character.UnoccupiedSpaceYouCanSee}}},
		{"Boon of Fate", featsSpec + " FR-044, FR-050, Table E", "happy", boonFeat("boon_of_fate"), singleOf(ability.Strength), []character.FeatEffect{
			character.ImproveFate{RangeFeet: 60, Dice: dice.MustExpression(2, dice.D4(), 0), Use: fatePrompt()}}},
		{"Boon of Irresistible Offense", featsSpec + " FR-044, FR-048, Table E", "happy", boonFeat("boon_of_irresistible_offense"), singleOf(ability.Dexterity), []character.FeatEffect{
			character.IgnoreResistance{DamageTypes: [3]content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing}},
			character.OverwhelmingStrike{Ability: ability.Dexterity, Trigger: character.TriggerNatural20OnAttackRoll, DamageType: character.SameAsAttackDamageType{}}}},
		{"Boon of Spell Recall", featsSpec + " FR-044, FR-053, Table E", "happy", recallFeat(), singleOf(ability.Wisdom), []character.FeatEffect{
			character.FreeCasting{SlotLevelsFrom: character.MustSlotLevel(1), SlotLevelsTo: character.MustSlotLevel(4), Die: dice.MustExpression(1, dice.D4(), 0)}}},
		{"Boon of the Night Spirit", featsSpec + " FR-044, Table E", "happy", boonFeat("boon_of_the_night_spirit"), singleOf(ability.Strength), []character.FeatEffect{
			character.MergeWithShadows{Requires: dim, Grants: content.ConditionInvisible, Cost: character.TriggerBonusAction, EndsAfter: [3]character.FeatTriggerAction{character.TriggerAction, character.TriggerBonusAction, character.TriggerReaction}},
			character.ShadowyForm{Requires: dim, ResistanceToAllExcept: [2]content.DamageType{content.DamagePsychic, content.DamageRadiant}}}},
		{"Boon of Truesight", featsSpec + " FR-044, Table E", "happy", boonFeat("boon_of_truesight"), singleOf(ability.Strength), []character.FeatEffect{character.TruesightSense{RangeFeet: 60}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := featEffectsOf(t, c.feat, c.choices)
			if len(got) != len(c.want) {
				t.Fatalf("effects = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("effect %d = %#v, want %#v [%s]", i, got[i], c.want[i], c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-044, Table E "no feat held" - a character with no feats has no effects, and the effects follow the order of the feats held.
// Path: unhappy (empty case).
func TestFeatEffectsOrder(t *testing.T) {
	t.Parallel()
	if got := (character.FeatEffects{}).Effects(character.TakenFeats{}); len(got) != 0 {
		t.Errorf("effects = %v", got)
	}
	feats := heldFeats(t, savageFeat(), character.NoFeatChoices{}, alertFeat(), character.NoFeatChoices{})
	got := (character.FeatEffects{}).Effects(feats)
	if len(got) != 3 || got[0] != (character.RollDamageTwice{Trigger: character.TriggerWeaponHit, Limit: character.OncePerTurn{}}) || got[1] != (character.InitiativeProficiency{}) {
		t.Errorf("effects = %v", got)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-045 (SRD 5.2.1, "Alert", Initiative Proficiency, p. 87) - Alert's initiative contribution is the Proficiency Bonus of the character's level, and zero without Alert.
// Path: per case.
func TestFeatEffectsInitiativeModifier(t *testing.T) {
	t.Parallel()
	alert := heldFeats(t, alertFeat(), character.NoFeatChoices{})
	for _, c := range []struct{ level, want int }{{1, 2}, {4, 2}, {5, 3}, {8, 3}, {9, 4}, {12, 4}, {13, 5}, {16, 5}, {17, 6}, {20, 6}} {
		pb := charAt(t, c.level).ProficiencyBonus()
		if got := (character.FeatEffects{}).InitiativeModifier(alert, pb); got != modifier.New(c.want) {
			t.Errorf("level %d: %v, want +%d", c.level, got, c.want)
		}
		if got := (character.FeatEffects{}).InitiativeModifier(character.TakenFeats{}, pb); got != modifier.New(0) {
			t.Errorf("level %d without Alert: %v", c.level, got)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestFeatEffectsArmorClassBonuses(t *testing.T) {
	t.Parallel()
	defense := heldFeats(t, defenseFeat(), character.NoFeatChoices{})
	for _, c := range []struct {
		name, requirement, path string
		feats                   character.TakenFeats
		worn                    []content.ArmorCategory
		want                    int
	}{
		{"Light armor", featsSpec + ` FR-046; SRD 5.2.1, "Defense", p. 88`, "unhappy (rule interaction)", defense, []content.ArmorCategory{content.ArmorLight}, 1},
		{"Medium armor", featsSpec + " FR-046", "unhappy (rule interaction)", defense, []content.ArmorCategory{content.ArmorMedium}, 1},
		{"Heavy armor", featsSpec + " FR-046", "unhappy (rule interaction)", defense, []content.ArmorCategory{content.ArmorHeavy}, 1},
		{"Light armor and a Shield", featsSpec + " FR-046", "unhappy (rule interaction)", defense, []content.ArmorCategory{content.ArmorLight, content.ArmorShield}, 1},
		{"a Shield alone", featsSpec + " FR-046", "unhappy (rule interaction)", defense, []content.ArmorCategory{content.ArmorShield}, 0},
		{"nothing worn", featsSpec + " FR-046", "unhappy (rule interaction)", defense, nil, 0},
		{"Heavy armor without Defense", featsSpec + " FR-046", "unhappy (rule interaction)", character.TakenFeats{}, []content.ArmorCategory{content.ArmorHeavy}, 0},
		{"Heavy armor with other feats held", featsSpec + " FR-046", "unhappy (rule interaction)", heldFeats(t, alertFeat(), character.NoFeatChoices{}, savageFeat(), character.NoFeatChoices{}), []content.ArmorCategory{content.ArmorHeavy}, 0},
		{"Heavy armor with Defense among other feats", featsSpec + " FR-046", "unhappy (rule interaction)", heldFeats(t, alertFeat(), character.NoFeatChoices{}, defenseFeat(), character.NoFeatChoices{}, savageFeat(), character.NoFeatChoices{}), []content.ArmorCategory{content.ArmorHeavy}, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.FeatEffects{}.ArmorClassBonuses(c.feats, c.worn)
			if len(got) != c.want {
				t.Fatalf("bonuses = %v, want %d [%s]", got, c.want, c.requirement)
			}
			if c.want == 1 {
				src, ok := got[0].Source().(character.FeatureFlatBonusSource)
				if got[0].Modifier() != modifier.New(1) || !ok || src.Feature != "defense" {
					t.Errorf("bonus = %#v", got[0])
				}
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-048 (SRD 5.2.1, "Boon of Irresistible Offense", Overwhelming Strike, p. 88) - the extra damage is the current score of the ability the feat increased.
// Path: happy.
func TestOverwhelmingStrikeExtraDamage(t *testing.T) {
	t.Parallel()
	scores := scoresOf(t, 20, 17, 10, 10, 10, 10)
	if got := (character.OverwhelmingStrike{Ability: ability.Dexterity}).ExtraDamage(scores); got != 17 {
		t.Errorf("Dexterity = %d", got)
	}
	if got := (character.OverwhelmingStrike{Ability: ability.Strength}).ExtraDamage(scores); got != 20 {
		t.Errorf("Strength = %d", got)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-049, FR-050 - UseRenewedBy is the set of renewal moments, comparable, and refuses an empty or invalid set.
// Path: per case.
func TestUseRenewedBy(t *testing.T) {
	t.Parallel()
	u := must(character.NewUseRenewedBy(character.RenewOnLongRestFinished))
	if !u.Renews(character.RenewOnLongRestFinished) || u.Renews(character.RenewOnShortRestFinished) || u.Renews(character.RenewOnInitiativeRolled) || u.Renews(0) || u.Renews(4) {
		t.Error("Renews must report exactly the moments of the set")
	}
	if _, err := character.NewUseRenewedBy(); err == nil {
		t.Error("an empty set must be refused")
	}
	if _, err := character.NewUseRenewedBy(character.RenewOnLongRestFinished, 9); err == nil {
		t.Error("an invalid moment must be refused")
	}
	if u != must(character.NewUseRenewedBy(character.RenewOnLongRestFinished, character.RenewOnLongRestFinished)) {
		t.Error("a set compares by its moments")
	}
}

// Requirement: doc/specs/feats-behaviour.md section 5 - the enums of the declared effects print their camel-case names and "unknown" outside them.
// Path: per case.
func TestFeatEffectEnumStrings(t *testing.T) {
	t.Parallel()
	for _, c := range [][2]string{
		{character.TriggerAttackAction.String(), "attackAction"}, {character.TriggerMagicAction.String(), "magicAction"}, {character.TriggerBonusAction.String(), "bonusAction"},
		{character.TriggerReaction.String(), "reaction"}, {character.TriggerAction.String(), "action"}, {character.FeatTriggerAction(0).String(), "unknown"},
		{character.TwoHands.String(), "twoHands"}, {character.HandsUsed(2).String(), "unknown"},
		{character.AfterRollingInitiative.String(), "afterRollingInitiative"}, {character.FeatInitiativeTiming(0).String(), "unknown"},
		{character.WillingAllyInSameCombat.String(), "willingAllyInSameCombat"}, {character.FeatSwapPartner(2).String(), "unknown"},
		{character.TriggerWeaponHit.String(), "weaponHit"}, {character.TriggerNatural20OnAttackRoll.String(), "natural20OnAttackRoll"}, {character.FeatTrigger(3).String(), "unknown"},
		{character.SameSizeOrSmaller.String(), "sameSizeOrSmaller"}, {character.FeatSizeRelation(0).String(), "unknown"},
		{character.UnoccupiedSpaceYouCanSee.String(), "unoccupiedSpaceYouCanSee"}, {character.FeatDestination(2).String(), "unknown"},
		{character.RenewOnInitiativeRolled.String(), "initiativeRolled"}, {character.RenewOnShortRestFinished.String(), "shortRestFinished"}, {character.RenewOnLongRestFinished.String(), "longRestFinished"}, {character.FeatRenewal(0).String(), "unknown"},
	} {
		if c[0] != c[1] {
			t.Errorf("String() = %q, want %q", c[0], c[1])
		}
	}
	if !slices.Contains([]character.FeatDamageTypeRule{character.SameAsAttackDamageType{}, character.AttackDamageTypeIs{Type: content.DamageFire}}, character.FeatDamageTypeRule(character.AttackDamageTypeIs{Type: content.DamageFire})) {
		t.Error("a specific damage type compares by its type")
	}
}
