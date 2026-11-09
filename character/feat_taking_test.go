package character_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func fightingStyleFeat(id string) content.Feat {
	return featOf(id, content.FeatFightingStyle, false)
}

// Requirement: per case
// Path: per case.
func TestTakeFeatGrantScopes(t *testing.T) {
	t.Parallel()
	c := charAt(t, 8)
	no := character.NoRequiredSpellList{}
	cleric := character.RequiredSpellListIs{Class: "cleric"}
	classFeat := character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}
	for _, c2 := range []struct {
		name, requirement, path string
		feat                    content.Feat
		grant                   character.FeatGrant
		choices                 character.FeatChoices
		want                    character.FeatRejection // nil: accepted
	}{
		{"Acolyte, Magic Initiate on the Cleric list", featsSpec + ` FR-004, FR-076, Table G; SRD 5.2.1, "Acolyte", p. 83`, "happy", initiateFeat(), grantOf(character.BackgroundFeat{Background: "acolyte"}, character.SpecifiedFeat{Feat: "magic_initiate", Required: cleric}), clericInitiate(), nil},
		{"Acolyte, Magic Initiate on the Wizard list", featsSpec + " FR-076, Table G", "unhappy (invalid input)", initiateFeat(), grantOf(character.BackgroundFeat{Background: "acolyte"}, character.SpecifiedFeat{Feat: "magic_initiate", Required: cleric}), wizardInitiate(),
			character.FeatChoicesInvalid{}},
		{"Sage, Magic Initiate on the Wizard list", featsSpec + " FR-004, Table G", "happy", initiateFeat(), grantOf(character.BackgroundFeat{Background: "sage"}, character.SpecifiedFeat{Feat: "magic_initiate", Required: character.RequiredSpellListIs{Class: "wizard"}}), wizardInitiate(), nil},
		{"a specified feat with no required list", featsSpec + " FR-076", "happy", initiateFeat(), grantOf(character.BackgroundFeat{Background: "acolyte"}, character.SpecifiedFeat{Feat: "magic_initiate", Required: no}), clericInitiate(), nil},
		{"no required list places no constraint", featsSpec + " FR-076", "happy", initiateFeat(), grantOf(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin}), clericInitiate(), nil},
		{"Criminal, Alert", featsSpec + " FR-004, Table G", "happy", alertFeat(), grantOf(character.BackgroundFeat{Background: "criminal"}, character.SpecifiedFeat{Feat: "alert", Required: no}), character.NoFeatChoices{}, nil},
		{"Soldier, Savage Attacker", featsSpec + " FR-004, Table G", "happy", savageFeat(), grantOf(soldierSrc, character.SpecifiedFeat{Feat: "savage_attacker", Required: no}), character.NoFeatChoices{}, nil},
		{"Criminal slot, Savage Attacker", featsSpec + " FR-003, Table G", "unhappy (invalid input)", savageFeat(), grantOf(character.BackgroundFeat{Background: "criminal"}, character.SpecifiedFeat{Feat: "alert", Required: no}), character.NoFeatChoices{}, character.FeatOutsideGrant{Feat: "savage_attacker"}},
		{"Sage, Magic Initiate on the Cleric list", featsSpec + " FR-076, Table G", "unhappy (invalid input)", initiateFeat(), grantOf(character.BackgroundFeat{Background: "sage"}, character.SpecifiedFeat{Feat: "magic_initiate", Required: character.RequiredSpellListIs{Class: "wizard"}}), clericInitiate(),
			character.FeatChoicesInvalid{}},
		{"Soldier slot, Alert", featsSpec + " FR-003, Table G", "unhappy (invalid input)", alertFeat(), grantOf(soldierSrc, character.SpecifiedFeat{Feat: "savage_attacker", Required: no}), character.NoFeatChoices{}, character.FeatOutsideGrant{Feat: "alert"}},
		{"custom background, an Origin feat", featsSpec + " FR-001, Table G", "happy", alertFeat(), grantOf(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin}), character.NoFeatChoices{}, nil},
		{"Human Versatile, an Origin feat", featsSpec + " FR-001, Table G", "happy", savageFeat(), grantOf(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin}), character.NoFeatChoices{}, nil},
		{"Versatile slot, an Epic Boon", featsSpec + " FR-002, Table G", "unhappy (invalid input)", boonFeat("boon_of_fate"), grantOf(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin}), singleOf(ability.Wisdom), character.FeatOutsideGrant{Feat: "boon_of_fate"}},
		{"Lessons of the First Ones, an Origin feat", featsSpec + " FR-001, Table G", "happy", alertFeat(), grantOf(character.EldritchInvocationFeat{Option: "lessons_of_the_first_ones"}, character.FeatFromCategory{Category: content.FeatOrigin}), character.NoFeatChoices{}, nil},
		{"Fighting Style slot, a Fighting Style feat", featsSpec + " FR-001, Table G", "happy", fightingStyleFeat("two_weapon_fighting"), grantOf(classFeat, character.FeatFromCategory{Category: content.FeatFightingStyle}), character.NoFeatChoices{}, nil},
		{"Fighting Style slot, a General feat", featsSpec + " FR-002, Table G", "unhappy (invalid input)", grapplerFeat(), grantOf(classFeat, character.FeatFromCategory{Category: content.FeatFightingStyle}), singleOf(ability.Strength), character.FeatOutsideGrant{Feat: "grappler"}},
		{"Ability Score Improvement slot, a General feat", featsSpec + " FR-001, Table G", "happy", asiFeat(), anyGrant(), must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0)), nil},
		{"Ability Score Improvement slot, an Origin feat", featsSpec + " FR-001, Table G", "happy", alertFeat(), anyGrant(), character.NoFeatChoices{}, nil},
		{"Bonus Feats at Level 20, any feat", featsSpec + " FR-001, Table G", "happy", savageFeat(), grantOf(character.BonusFeat{}, character.AnyFeat{}), character.NoFeatChoices{}, nil},
	} {
		t.Run(c2.name, func(t *testing.T) {
			t.Parallel()
			tk, err := character.FeatTakingRules{}.Take(c, character.TakenFeats{}, c2.feat, c2.grant, c2.choices, fightingStyleHeld(t), miSpells(t))
			if err != nil {
				t.Fatal(err)
			}
			switch want := c2.want.(type) {
			case nil:
				if _, ok := tk.(character.FeatTakingChanged); !ok {
					t.Errorf("= %#v, want a change [%s]", tk, c2.requirement)
				}
			case character.FeatChoicesInvalid:
				got, ok := tk.(character.FeatTakingRejected).Rejection.(character.FeatChoicesInvalid)
				if !ok || !slices.Equal(got.Problems(), []character.FeatChoiceProblem{{Name: character.FeatChoiceSpellList, Kind: character.FeatProblemListNotAllowed}}) {
					t.Errorf("= %#v [%s]", tk, c2.requirement)
				}
			default:
				if got := tk.(character.FeatTakingRejected).Rejection; got != want {
					t.Errorf("= %#v, want %#v [%s]", got, want, c2.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-001 (SRD 5.2.1, "Parts of a Feat", Category, p. 87) - an AnyFeat grant accepts a feat of each of the four categories.
// Path: happy.
func TestTakeFeatAnyFeatAcceptsEveryCategory(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20)
	held := must(character.NewHeldFeatures("fighter_fighting_style"))
	for _, tc := range []struct {
		feat    content.Feat
		choices character.FeatChoices
	}{
		{alertFeat(), character.NoFeatChoices{}},
		{asiFeat(), must(character.NewAbilityScoreImprovementChoices(character.ImprovementTwoByOne, ability.Strength, ability.Dexterity))},
		{archeryFeat(), character.NoFeatChoices{}},
		{boonFeat("boon_of_truesight"), singleOf(ability.Wisdom)},
	} {
		ch := takeOK(t, c, character.TakenFeats{}, tc.feat, anyGrant(), tc.choices, held)
		if got := ch.Feats().All(); len(got) != 1 || got[0].Category() != tc.feat.FeatCategory() || got[0].Feat() != tc.feat.ID() {
			t.Errorf("%s: %v", tc.feat.ID(), got)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestTakeFeatRepeatability(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20)
	held := fightingStyleHeld(t)
	versatile := grantOf(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin})
	soldier := grantOf(soldierSrc, character.SpecifiedFeat{Feat: "savage_attacker", Required: character.NoRequiredSpellList{}})
	has := heldFeats(t, savageFeat(), character.NoFeatChoices{})
	skilled2 := skilledOf(character.SkilledSkill{Skill: content.SkillArcana}, character.SkilledSkill{Skill: content.SkillHistory}, character.SkilledSkill{Skill: content.SkillNature})
	skilled1 := skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledSkill{Skill: content.SkillPerception}, character.SkilledSkill{Skill: content.SkillStealth})
	hasSkilled := heldFeats(t, skilledFeat(), skilled1)
	style := featOf("archery", content.FeatFightingStyle, true, featureReq("Fighting Style Feature"))
	hasStyle := heldFeats(t, style, character.NoFeatChoices{})
	for _, tc := range []struct {
		name, requirement, path string
		feats                   character.TakenFeats
		feat                    content.Feat
		grant                   character.FeatGrant
		choices                 character.FeatChoices
		rejected                bool
	}{
		{"Savage Attacker from Soldier, then from Versatile", featsSpec + " FR-005 (SRD 5.2.1, \"Parts of a Feat\", Repeatable, p. 87)", "unhappy (state error)", has, savageFeat(), versatile, character.NoFeatChoices{}, true},
		{"the same source twice", featsSpec + " FR-005", "unhappy (state error)", has, savageFeat(), soldier, character.NoFeatChoices{}, true},
		{"a repeatable feat again", featsSpec + " FR-006", "happy", hasSkilled, skilledFeat(), versatile, skilled2, false},
		{"a Fighting Style feat supplied as repeatable", featsSpec + " FR-007", "unhappy (rule interaction)", hasStyle, style, anyGrant(), character.NoFeatChoices{}, false},
		{"a Fighting Style feat as the content source defines it", featsSpec + " FR-007", "unhappy (rule interaction)", heldFeats(t, archeryFeat(), character.NoFeatChoices{}), archeryFeat(), anyGrant(), character.NoFeatChoices{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tk, err := character.FeatTakingRules{}.Take(c, tc.feats, tc.feat, tc.grant, tc.choices, held, miSpells(t))
			if err != nil {
				t.Fatal(err)
			}
			_, rejected := tk.(character.FeatTakingRejected)
			if rejected != tc.rejected {
				t.Errorf("= %#v, rejected %v, want %v [%s]", tk, rejected, tc.rejected, tc.requirement)
			}
			if rejected && tk.(character.FeatTakingRejected).Rejection != (character.FeatNotRepeatable{Feat: tc.feat.ID()}) {
				t.Errorf("rejection = %#v [%s]", tk, tc.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestTakeFeatChoiceShapes(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20, 14, 14, 10, 10, 10, 10)
	held := must(character.NewHeldFeatures("fighter_fighting_style", "wizard_spellcasting"))
	shape := func(k character.FeatChoiceProblemKind) []character.FeatChoiceProblem {
		return []character.FeatChoiceProblem{{Name: character.FeatChoiceFeatChoicesShape, Kind: k}}
	}
	ability1 := func(k character.FeatChoiceProblemKind) []character.FeatChoiceProblem {
		return []character.FeatChoiceProblem{{Name: character.FeatChoiceIncreasedAbility, Kind: k}}
	}
	skilled := skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledSkill{Skill: content.SkillPerception}, character.SkilledSkill{Skill: content.SkillStealth})
	asi := must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	for _, tc := range []struct {
		name, requirement, path string
		feat                    content.Feat
		choices                 character.FeatChoices
		want                    []character.FeatChoiceProblem // nil: accepted
	}{
		{"Alert takes none", featsSpec + " FR-017, Table C", "happy", alertFeat(), character.NoFeatChoices{}, nil},
		{"Savage Attacker takes none", featsSpec + " FR-017, Table C", "happy", savageFeat(), character.NoFeatChoices{}, nil},
		{"Magic Initiate", featsSpec + " FR-017, Table C", "happy", initiateFeat(), clericInitiate(), nil},
		{"Skilled", featsSpec + " FR-017, Table C", "happy", skilledFeat(), skilled, nil},
		{"Ability Score Improvement", featsSpec + " FR-017, Table C", "happy", asiFeat(), asi, nil},
		{"Grappler, Strength", featsSpec + " FR-023, Table C", "happy", grapplerFeat(), singleOf(ability.Strength), nil},
		{"Grappler, Dexterity", featsSpec + " FR-023, Table C", "happy", grapplerFeat(), singleOf(ability.Dexterity), nil},
		{"Archery takes none", featsSpec + " FR-017, Table C", "happy", archeryFeat(), character.NoFeatChoices{}, nil},
		{"Defense takes none", featsSpec + " FR-017, Table C", "happy", defenseFeat(), character.NoFeatChoices{}, nil},
		{"Great Weapon Fighting takes none", featsSpec + " FR-017, Table C", "happy", fightingStyleFeat("great_weapon_fighting"), character.NoFeatChoices{}, nil},
		{"Two-Weapon Fighting takes none", featsSpec + " FR-017, Table C", "happy", fightingStyleFeat("two_weapon_fighting"), character.NoFeatChoices{}, nil},
		{"Boon of Combat Prowess, any ability", featsSpec + " FR-024, Table C", "happy", boonFeat("boon_of_combat_prowess"), singleOf(ability.Charisma), nil},
		{"Boon of Dimensional Travel, any ability", featsSpec + " FR-024, Table C", "happy", boonFeat("boon_of_dimensional_travel"), singleOf(ability.Constitution), nil},
		{"Boon of Fate, any ability", featsSpec + " FR-024, Table C", "happy", boonFeat("boon_of_fate"), singleOf(ability.Wisdom), nil},
		{"Boon of the Night Spirit, any ability", featsSpec + " FR-024, Table C", "happy", boonFeat("boon_of_the_night_spirit"), singleOf(ability.Intelligence), nil},
		{"Boon of Truesight, any ability", featsSpec + " FR-024, Table C", "happy", boonFeat("boon_of_truesight"), singleOf(ability.Strength), nil},
		{"Boon of Irresistible Offense, Strength", featsSpec + " FR-024, Table C", "happy", boonFeat("boon_of_irresistible_offense"), singleOf(ability.Strength), nil},
		{"Boon of Irresistible Offense, Dexterity", featsSpec + " FR-024, Table C", "happy", boonFeat("boon_of_irresistible_offense"), singleOf(ability.Dexterity), nil},
		{"Boon of Spell Recall, Intelligence", featsSpec + " FR-024, Table C", "happy", recallFeat(), singleOf(ability.Intelligence), nil},
		{"Boon of Spell Recall, Wisdom", featsSpec + " FR-024, Table C", "happy", recallFeat(), singleOf(ability.Wisdom), nil},
		{"Boon of Spell Recall, Charisma", featsSpec + " FR-024, Table C", "happy", recallFeat(), singleOf(ability.Charisma), nil},
		{"Grappler, none supplied", featsSpec + " FR-017, Table C", "unhappy (invalid input)", grapplerFeat(), character.NoFeatChoices{}, shape(character.FeatProblemMissing)},
		{"Skilled, none supplied", featsSpec + " FR-017, Table C", "unhappy (invalid input)", skilledFeat(), character.NoFeatChoices{}, shape(character.FeatProblemMissing)},
		{"Alert with a single increase", featsSpec + " FR-017, Table C", "unhappy (invalid input)", alertFeat(), singleOf(ability.Strength), shape(character.FeatProblemUnexpected)},
		{"Skilled with Magic Initiate choices", featsSpec + " FR-017, Table C", "unhappy (invalid input)", skilledFeat(), clericInitiate(), shape(character.FeatProblemUnexpected)},
		{"Magic Initiate with Skilled choices", featsSpec + " FR-017, Table C", "unhappy (invalid input)", initiateFeat(), skilled, shape(character.FeatProblemUnexpected)},
		{"Ability Score Improvement with a single increase", featsSpec + " FR-017, Table C", "unhappy (invalid input)", asiFeat(), singleOf(ability.Strength), shape(character.FeatProblemUnexpected)},
		{"Grappler with the improvement", featsSpec + " FR-017, Table C", "unhappy (invalid input)", grapplerFeat(), asi, shape(character.FeatProblemUnexpected)},
		{"Grappler, Constitution", featsSpec + " FR-023, Table C", "unhappy (invalid input)", grapplerFeat(), singleOf(ability.Constitution), ability1(character.FeatProblemAbilityNotAllowed)},
		{"Boon of Irresistible Offense, Wisdom", featsSpec + " FR-024, Table C", "unhappy (invalid input)", boonFeat("boon_of_irresistible_offense"), singleOf(ability.Wisdom), ability1(character.FeatProblemAbilityNotAllowed)},
		{"Boon of Spell Recall, Strength", featsSpec + " FR-024, Table C", "unhappy (invalid input)", recallFeat(), singleOf(ability.Strength), ability1(character.FeatProblemAbilityNotAllowed)},
		{"the zero improvement", featsSpec + " FR-020", "unhappy (invalid input)", asiFeat(), character.AbilityScoreImprovementChoices{}, ability1(character.FeatProblemAbilityNotAllowed)},
		{"the zero Skilled choices", featsSpec + " FR-038", "unhappy (invalid input)", skilledFeat(), character.SkilledChoices{}, []character.FeatChoiceProblem{{Name: character.FeatChoiceSkilledPicks, Kind: character.FeatProblemMissing}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tk, err := character.FeatTakingRules{}.Take(c, character.TakenFeats{}, tc.feat, anyGrant(), tc.choices, held, miSpells(t))
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == nil {
				if _, ok := tk.(character.FeatTakingChanged); !ok {
					t.Errorf("= %#v, want a change [%s]", tk, tc.requirement)
				}
				return
			}
			got, ok := tk.(character.FeatTakingRejected).Rejection.(character.FeatChoicesInvalid)
			if !ok || !slices.Equal(got.Problems(), tc.want) {
				t.Errorf("= %#v, want %v [%s]", tk, tc.want, tc.requirement)
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-018 - a feat with no behaviour (anything but the 17 of Table C) is FeatBehaviourUnknown carrying the id, and nothing changes.
// Path: unhappy (missing content).
func TestTakeFeatWithoutBehaviour(t *testing.T) {
	t.Parallel()
	c := charAt(t, 8)
	feats := heldFeats(t, alertFeat(), character.NoFeatChoices{})
	tk, err := character.FeatTakingRules{}.Take(c, feats, featOf("house_feat", content.FeatGeneral, false), anyGrant(), character.NoFeatChoices{}, character.HeldFeatures{}, nil)
	if got := tk.(character.FeatTakingRejected).Rejection; err != nil || got != (character.FeatBehaviourUnknown{Feat: "house_feat"}) || got.String() != "feat_behaviour_unknown" {
		t.Errorf("= %#v, %v", tk, err)
	}
}

// Requirement: per case
// Path: per case.
func TestTakeFeatAbilityScoreIncreases(t *testing.T) {
	t.Parallel()
	asiOne := func(a ability.Ability) character.AbilityScoreImprovementChoices {
		return must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, a, 0))
	}
	for _, tc := range []struct {
		name, requirement, path string
		feat                    content.Feat
		choices                 character.FeatChoices
		scores                  []int
		want                    map[ability.Ability]int // scores after; nil: rejected
		max                     int
	}{
		{"ASI one by two", featsSpec + ` FR-020, Table A; SRD 5.2.1, "Ability Score Improvement", p. 87`, "happy", asiFeat(), asiOne(ability.Strength), []int{18, 10, 10, 10, 10, 10}, map[ability.Ability]int{ability.Strength: 20}, 0},
		{"ASI two by one", featsSpec + " FR-020, Table A", "happy", asiFeat(), must(character.NewAbilityScoreImprovementChoices(character.ImprovementTwoByOne, ability.Dexterity, ability.Constitution)), []int{10, 19, 10, 10, 10, 10}, map[ability.Ability]int{ability.Dexterity: 20, ability.Constitution: 11}, 0},
		{"ASI over the maximum", featsSpec + " FR-079, Table A", "unhappy (boundary)", asiFeat(), asiOne(ability.Strength), []int{19, 10, 10, 10, 10, 10}, nil, 20},
		{"Grappler", featsSpec + " FR-023, Table A", "happy", grapplerFeat(), singleOf(ability.Dexterity), []int{13, 19, 10, 10, 10, 10}, map[ability.Ability]int{ability.Dexterity: 20}, 0},
		{"Grappler at the limit", featsSpec + " FR-079, Table A", "unhappy (boundary)", grapplerFeat(), singleOf(ability.Strength), []int{20, 13, 10, 10, 10, 10}, nil, 20},
		{"Boon below 30", featsSpec + " FR-024, Table A", "unhappy (boundary)", boonFeat("boon_of_fate"), singleOf(ability.Constitution), []int{10, 10, 29, 10, 10, 10}, map[ability.Ability]int{ability.Constitution: 30}, 0},
		{"Boon at 30", featsSpec + " FR-079, Table A", "unhappy (boundary)", boonFeat("boon_of_fate"), singleOf(ability.Constitution), []int{10, 10, 30, 10, 10, 10}, nil, 30},
		{"Boon above 20", featsSpec + " FR-024, Table A", "happy", boonFeat("boon_of_truesight"), singleOf(ability.Wisdom), []int{10, 10, 10, 10, 20, 10}, map[ability.Ability]int{ability.Wisdom: 21}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := charAt(t, 20, tc.scores...)
			before, feats := c, character.TakenFeats{}
			tk, err := character.FeatTakingRules{}.Take(c, feats, tc.feat, anyGrant(), tc.choices, character.HeldFeatures{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !c.Equal(before) || feats.Len() != 0 {
				t.Error("Take altered its inputs")
			}
			if tc.want == nil {
				inc, ok := tk.(character.FeatTakingRejected).Rejection.(character.FeatIncreaseRejected)
				over, isOver := inc.Rejection.(character.IncreaseExceedsMaximum)
				if !ok || !isOver || over.MaximumScore() != tc.max || len(over.Abilities()) != 1 {
					t.Errorf("= %#v [%s]", tk, tc.requirement)
				}
				return
			}
			got := tk.(character.FeatTakingChanged).Character()
			for a, want := range tc.want {
				if got.AbilityScores().Score(a).Value() != want {
					t.Errorf("%v = %d, want %d [%s]", a, got.AbilityScores().Score(a).Value(), want, tc.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-026, Table A "event order" (SRD 5.2.1, "Level Advancement", p. 23) - the events of a taken feat are FeatTaken first, then every AbilityScoreChanged, then at most one HitPointMaximumChanged; Take alters none of its inputs and gives the same result on every call (FR-058, FR-064).
// Path: happy.
func TestTakeFeatEventOrder(t *testing.T) {
	t.Parallel()
	c := charAt(t, 8, 10, 10, 17, 10, 10, 10)
	choices := must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Constitution, 0))
	first := takeOK(t, c, character.TakenFeats{}, asiFeat(), anyGrant(), choices, character.HeldFeatures{})
	second := takeOK(t, c, character.TakenFeats{}, asiFeat(), anyGrant(), choices, character.HeldFeatures{})
	events := first.Events()
	if len(events) != 3 {
		t.Fatalf("events = %#v", events)
	}
	taken, ok := events[0].(character.FeatTaken)
	if !ok || taken.Feat() != "ability_score_improvement" || taken.Category() != content.FeatGeneral || taken.CharacterID() != c.ID() || taken.Source() != anyGrant().Source() {
		t.Errorf("first event = %#v", events[0])
	}
	if _, ok := events[1].(character.AbilityScoreChanged); !ok {
		t.Errorf("second event = %#v", events[1])
	}
	if _, ok := events[2].(character.HitPointMaximumChanged); !ok {
		t.Errorf("third event = %#v", events[2])
	}
	if !reflect.DeepEqual(first.Events(), second.Events()) || !first.Character().Equal(second.Character()) || !first.Feats().Equal(second.Feats()) {
		t.Error("the same inputs must give the same result")
	}
	first.Events()[0] = nil
	if first.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
	if !first.ProficiencyGrants().IsZero() || !first.GrantedSpells().IsZero() {
		t.Error("an Ability Score Improvement grants no proficiency or spell")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-013 (SRD 5.2.1, "Parts of a Feat", Prerequisite, p. 87) - prerequisites are read before the feat's own benefits, so Grappler's increase never counts toward its own prerequisite; a later fall in a score or level does not remove the feat (FR-014).
// Path: unhappy (rule interaction).
func TestTakeFeatPrerequisiteTiming(t *testing.T) {
	t.Parallel()
	c := charAt(t, 4, 12, 12, 10, 10, 10, 10)
	rj := takeRejected(t, c, character.TakenFeats{}, grapplerFeat(), anyGrant(), singleOf(ability.Strength), character.HeldFeatures{})
	unmet, ok := rj.(character.FeatPrerequisiteNotMet)
	if !ok || len(unmet.Unmet()) != 1 || rj.String() != "feat_prerequisite_not_met" {
		t.Fatalf("rejection = %#v", rj)
	}
	unmet.Unmet()[0] = nil
	if unmet.Unmet()[0] == nil {
		t.Error("Unmet must return a copy")
	}
	// FR-014: the feat taken at level 4 is still held, with its effects, when the state passed in is at level 1.
	feats := takeOK(t, charAt(t, 4, 13, 13, 10, 10, 10, 10), character.TakenFeats{}, grapplerFeat(), anyGrant(), singleOf(ability.Strength), character.HeldFeatures{}).Feats()
	_ = takeOK(t, charAt(t, 4, 14, 14, 10, 10, 10, 10), feats, alertFeat(), anyGrant(), character.NoFeatChoices{}, character.HeldFeatures{})
	if !feats.Holds("grappler") || len((character.FeatEffects{}).Effects(feats)) != 3 {
		t.Error("a taken feat keeps its benefits")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-015, FR-016 (SRD 5.2.1, "Fighting Style Feats", pp. 87-88) - an unmet prerequisite gives FeatPrerequisiteNotMet with the clauses, no change and no event; a Fighting Style feat through an Ability Score Improvement slot needs the Fighting Style feature.
// Path: unhappy (rule interaction).
func TestTakeFeatFightingStyleThroughAnySlot(t *testing.T) {
	t.Parallel()
	c := charAt(t, 4)
	if rj := takeRejected(t, c, character.TakenFeats{}, archeryFeat(), anyGrant(), character.NoFeatChoices{}, character.HeldFeatures{}); unmetKinds(rj.(character.FeatPrerequisiteNotMet).Unmet())[0] != "feature" {
		t.Errorf("rejection = %#v", rj)
	}
	takeOK(t, c, character.TakenFeats{}, archeryFeat(), anyGrant(), character.NoFeatChoices{}, fightingStyleHeld(t))
}

// Requirement: per case
// Path: per case.
func TestTakeFeatRejectionOrder(t *testing.T) {
	t.Parallel()
	held := fightingStyleHeld(t)
	asiOne := must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	soldier := grantOf(soldierSrc, character.SpecifiedFeat{Feat: "savage_attacker", Required: character.NoRequiredSpellList{}})
	cases := []struct {
		name, requirement, path string
		c                       character.Character
		feats                   character.TakenFeats
		feat                    content.Feat
		grant                   character.FeatGrant
		choices                 character.FeatChoices
		want                    character.FeatRejection
	}{
		{"no behaviour and outside the grant", featsSpec + " FR-060, Table O", "unhappy (rule interaction)", charAt(t, 20), character.TakenFeats{}, featOf("house_feat", content.FeatGeneral, false), soldier, character.NoFeatChoices{}, character.FeatBehaviourUnknown{Feat: "house_feat"}},
		{"outside the grant and already held", featsSpec + " FR-060, Table O", "unhappy (rule interaction)", charAt(t, 20), heldFeats(t, alertFeat(), character.NoFeatChoices{}), alertFeat(), soldier, character.NoFeatChoices{}, character.FeatOutsideGrant{Feat: "alert"}},
		{"already held and level too low", featsSpec + " FR-060, Table O", "unhappy (rule interaction)", charAt(t, 3), heldFeats(t, asiFeat(), asiOne), asiFeat(), anyGrant(), asiOne, character.FeatNotRepeatable{Feat: "ability_score_improvement"}},
		{"level too low and a missing choice", featsSpec + " FR-060, Table O", "unhappy (rule interaction)", charAt(t, 3), character.TakenFeats{}, asiFeat(), anyGrant(), character.NoFeatChoices{}, nil},
		{"a wrong choice and a score above the maximum", featsSpec + " FR-060, Table O", "unhappy (rule interaction)", charAt(t, 20, 20, 10, 10, 10, 10, 10), character.TakenFeats{}, grapplerFeat(), anyGrant(), singleOf(ability.Constitution), nil},
		{"only the maximum exceeded", featsSpec + " FR-060, Table O", "unhappy (rule interaction)", charAt(t, 20, 20, 10, 10, 10, 10, 10), character.TakenFeats{}, asiFeat(), anyGrant(), asiOne, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rj := takeRejected(t, tc.c, tc.feats, tc.feat, tc.grant, tc.choices, held)
			switch want := tc.want.(type) {
			case nil:
				switch tc.name {
				case "level too low and a missing choice":
					if _, ok := rj.(character.FeatPrerequisiteNotMet); !ok {
						t.Errorf("rejection = %#v [%s]", rj, tc.requirement)
					}
				case "a wrong choice and a score above the maximum":
					// Strength 20 does not meet nothing here: Grappler needs level 4 and a 13, and a wrong ability is the choice problem
					if _, ok := rj.(character.FeatChoicesInvalid); !ok {
						t.Errorf("rejection = %#v [%s]", rj, tc.requirement)
					}
				default:
					if _, ok := rj.(character.FeatIncreaseRejected); !ok {
						t.Errorf("rejection = %#v [%s]", rj, tc.requirement)
					}
				}
			default:
				if rj != want {
					t.Errorf("rejection = %#v, want %#v [%s]", rj, want, tc.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-027 to FR-030, Table M (SRD 5.2.1, "Magic Initiate", p. 87) - taking Magic Initiate reports every problem of the list, the ability, the cantrips and the level 1 spell, and refuses a second one on the same list.
// Path: per case.
func TestTakeFeatMagicInitiateChoices(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20)
	grant := grantOf(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
	p := func(n character.FeatChoiceName, k character.FeatChoiceProblemKind) character.FeatChoiceProblem {
		return character.FeatChoiceProblem{Name: n, Kind: k}
	}
	holdsCleric := heldFeats(t, initiateFeat(), clericInitiate())
	for _, tc := range []struct {
		name, requirement, path string
		feats                   character.TakenFeats
		choices                 character.MagicInitiateChoices
		want                    []character.FeatChoiceProblem
	}{
		{"valid Cleric", featsSpec + " Table M", "happy", character.TakenFeats{}, clericInitiate(), nil},
		{"valid Wizard", featsSpec + " Table M", "happy", character.TakenFeats{}, wizardInitiate(), nil},
		{"list Bard", featsSpec + " FR-027, Table M", "unhappy (invalid input)", character.TakenFeats{}, miOf("bard", ability.Charisma, []string{"vicious_mockery", "light"}, "bless"), []character.FeatChoiceProblem{
			p(character.FeatChoiceSpellList, character.FeatProblemListNotAllowed), p(character.FeatChoiceCantrips, character.FeatProblemNotOnSpellList), p(character.FeatChoiceLevelOneSpell, character.FeatProblemNotOnSpellList)}},
		{"ability Strength", featsSpec + " FR-027, Table M", "unhappy (invalid input)", character.TakenFeats{}, miOf("cleric", ability.Strength, []string{"guidance", "light"}, "bless"), []character.FeatChoiceProblem{p(character.FeatChoiceSpellcastingAbility, character.FeatProblemAbilityNotAllowed)}},
		{"ability Intelligence, Wisdom and Charisma are allowed", featsSpec + " FR-027", "happy", character.TakenFeats{}, miOf("druid", ability.Charisma, []string{"guidance", "shillelagh"}, "cure_wounds"), nil},
		{"one cantrip", featsSpec + " FR-028, Table M", "unhappy (boundary)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"guidance"}, "bless"), []character.FeatChoiceProblem{p(character.FeatChoiceCantrips, character.FeatProblemWrongCount)}},
		{"three cantrips", featsSpec + " FR-028, Table M", "unhappy (boundary)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"guidance", "light", "mending"}, "bless"), []character.FeatChoiceProblem{p(character.FeatChoiceCantrips, character.FeatProblemWrongCount)}},
		{"two equal cantrips", featsSpec + " FR-028, Table M", "unhappy (boundary)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"light", "light"}, "bless"), []character.FeatChoiceProblem{p(character.FeatChoiceCantrips, character.FeatProblemDuplicate)}},
		{"a level 1 spell among the cantrips", featsSpec + " FR-028, Table M", "unhappy (boundary)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"guidance", "cure_wounds"}, "bless"), []character.FeatChoiceProblem{p(character.FeatChoiceCantrips, character.FeatProblemNotACantrip)}},
		{"a cantrip as the level 1 spell", featsSpec + " FR-029, Table M", "unhappy (boundary)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"guidance", "light"}, "mending"), []character.FeatChoiceProblem{p(character.FeatChoiceLevelOneSpell, character.FeatProblemNotLevelOne)}},
		{"a level 2 spell as the level 1 spell", featsSpec + " FR-029, Table M", "unhappy (boundary)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"guidance", "light"}, "hold_person"), []character.FeatChoiceProblem{p(character.FeatChoiceLevelOneSpell, character.FeatProblemNotLevelOne)}},
		{"an off-list cantrip", featsSpec + " FR-028, Table M", "unhappy (invalid input)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"guidance", "fire_bolt"}, "bless"), []character.FeatChoiceProblem{p(character.FeatChoiceCantrips, character.FeatProblemNotOnSpellList)}},
		{"an off-list level 1 spell", featsSpec + " FR-029, Table M", "unhappy (invalid input)", character.TakenFeats{}, miOf("cleric", ability.Wisdom, []string{"guidance", "light"}, "shield"), []character.FeatChoiceProblem{p(character.FeatChoiceLevelOneSpell, character.FeatProblemNotOnSpellList)}},
		{"a second Magic Initiate on another list", featsSpec + " FR-030, Table M", "happy", holdsCleric, wizardInitiate(), nil},
		{"a second Magic Initiate on the same list", featsSpec + " FR-030, Table M", "unhappy (rule interaction)", holdsCleric, miOf("cleric", ability.Intelligence, []string{"mending", "light"}, "cure_wounds"), []character.FeatChoiceProblem{p(character.FeatChoiceSpellList, character.FeatProblemListAlreadyUsed)}},
		{"overlapping spells on another list", featsSpec + " FR-030, Table M", "unhappy (rule interaction)", holdsCleric, miOf("druid", ability.Wisdom, []string{"guidance", "shillelagh"}, "cure_wounds"), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tk, err := character.FeatTakingRules{}.Take(c, tc.feats, initiateFeat(), grant, tc.choices, character.HeldFeatures{}, miSpells(t))
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == nil {
				ch, ok := tk.(character.FeatTakingChanged)
				if !ok {
					t.Fatalf("= %#v, want a change [%s]", tk, tc.requirement)
				}
				g := ch.GrantedSpells()
				if g.List() != tc.choices.List() || g.Ability() != tc.choices.Ability() || g.LevelOneSpell() != tc.choices.LevelOneSpell() || !slices.Equal(g.Cantrips(), tc.choices.Cantrips()) {
					t.Errorf("granted spells = %+v", g)
				}
				return
			}
			got, ok := tk.(character.FeatTakingRejected).Rejection.(character.FeatChoicesInvalid)
			if !ok || !slices.Equal(got.Problems(), tc.want) {
				t.Errorf("= %#v, want %v [%s]", tk, tc.want, tc.requirement)
			}
		})
	}
	// a spell the caller did not resolve is a caller error, not a rule outcome
	if _, err := (character.FeatTakingRules{}).Take(c, character.TakenFeats{}, initiateFeat(), grant, clericInitiate(), character.HeldFeatures{}, map[content.SpellID]content.Spell{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unresolved spell: %v", err)
	}
	spells := miSpells(t)
	delete(spells, "bless")
	if _, err := (character.FeatTakingRules{}).Take(c, character.TakenFeats{}, initiateFeat(), grant, clericInitiate(), character.HeldFeatures{}, spells); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unresolved level 1 spell: %v", err)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-031, FR-040, FR-041, Table S (SRD 5.2.1, "Skilled", p. 87) - a taken Skilled exposes its three picks as ProficiencyGrants, even a pick already held, and a second Skilled is accepted with its own picks.
// Path: per case.
func TestTakeFeatSkilledGrants(t *testing.T) {
	t.Parallel()
	c := charAt(t, 8)
	picks := skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledTool{Tool: "thieves_tools"}, character.SkilledTool{Tool: "herbalism_kit"})
	grant := grantOf(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
	first := takeOK(t, c, character.TakenFeats{}, skilledFeat(), grant, picks, character.HeldFeatures{})
	g := first.ProficiencyGrants()
	if !slices.Equal(g.Skills(), []content.Skill{content.SkillInsight}) || !slices.Equal(g.Tools(), []content.ItemID{"thieves_tools", "herbalism_kit"}) || g.IsZero() {
		t.Fatalf("grants = %+v", g)
	}
	g.Skills()[0] = 0
	g.Tools()[0] = ""
	if first.ProficiencyGrants().Skills()[0] != content.SkillInsight || first.ProficiencyGrants().Tools()[0] != "thieves_tools" {
		t.Error("the grants must be copies")
	}
	second := takeOK(t, c, first.Feats(), skilledFeat(), grant, skilledOf(character.SkilledSkill{Skill: content.SkillArcana}, character.SkilledSkill{Skill: content.SkillHistory}, character.SkilledSkill{Skill: content.SkillInsight}), character.HeldFeatures{})
	all := second.Feats().ProficiencyGrants()
	if second.Feats().Len() != 2 || len(all.Skills()) != 4 || len(all.Tools()) != 2 {
		t.Errorf("both Skilled feats grant: %+v", all)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-066, FR-058, FR-059 - Take refuses a zero character, feat or grant and nil choices with an ErrInvalidInput error and never mutates its inputs.
// Path: unhappy (invalid input).
func TestTakeFeatInvalidArguments(t *testing.T) {
	t.Parallel()
	c := charAt(t, 4)
	for name, run := range map[string]func() (character.FeatTaking, error){
		"zero character": func() (character.FeatTaking, error) {
			return character.FeatTakingRules{}.Take(character.Character{}, character.TakenFeats{}, alertFeat(), anyGrant(), character.NoFeatChoices{}, character.HeldFeatures{}, nil)
		},
		"zero feat": func() (character.FeatTaking, error) {
			return character.FeatTakingRules{}.Take(c, character.TakenFeats{}, content.Feat{}, anyGrant(), character.NoFeatChoices{}, character.HeldFeatures{}, nil)
		},
		"zero grant": func() (character.FeatTaking, error) {
			return character.FeatTakingRules{}.Take(c, character.TakenFeats{}, alertFeat(), character.FeatGrant{}, character.NoFeatChoices{}, character.HeldFeatures{}, nil)
		},
		"nil choices": func() (character.FeatTaking, error) {
			return character.FeatTakingRules{}.Take(c, character.TakenFeats{}, alertFeat(), anyGrant(), nil, character.HeldFeatures{}, nil)
		},
	} {
		if got, err := run(); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, got, err)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestTakeFeatMagicInitiateListsAndInstances(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20)
	grant := grantOf(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
	listProblem := []character.FeatChoiceProblem{{Name: character.FeatChoiceSpellList, Kind: character.FeatProblemListNotAllowed}}
	for _, list := range []string{"paladin", "ranger", "sorcerer", "wizards"} {
		t.Run("list "+list, func(t *testing.T) {
			t.Parallel()
			rj := takeRejected(t, c, character.TakenFeats{}, initiateFeat(), grant, miOf(list, ability.Wisdom, []string{"guidance", "light"}, "bless"), character.HeldFeatures{})
			got, ok := rj.(character.FeatChoicesInvalid)
			if !ok || len(got.Problems()) == 0 || got.Problems()[0] != listProblem[0] {
				t.Errorf("rejection = %#v, want the list refused first [%s]", rj, featsSpec+" FR-027, Table M")
			}
		})
	}
	// the same list is refused wherever its first instance stands among the held feats
	held := heldFeats(t, initiateFeat(), wizardInitiate(), initiateFeat(), clericInitiate())
	rj := takeRejected(t, c, held, initiateFeat(), grant, miOf("cleric", ability.Intelligence, []string{"mending", "light"}, "cure_wounds"), character.HeldFeatures{})
	got, ok := rj.(character.FeatChoicesInvalid)
	if !ok || len(got.Problems()) != 1 || got.Problems()[0] != (character.FeatChoiceProblem{Name: character.FeatChoiceSpellList, Kind: character.FeatProblemListAlreadyUsed}) {
		t.Errorf("a second Cleric Magic Initiate: %#v [%s]", rj, featsSpec+" FR-030")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-066 - a cantrip or level 1 spell of a Magic Initiate missing from the resolved spells is a caller error, whichever of them it is.
// Path: unhappy (invalid input).
func TestTakeFeatMagicInitiateEachSpellMustBeResolved(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20)
	grant := grantOf(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
	for _, missing := range []content.SpellID{"guidance", "light", "bless"} {
		spells := miSpells(t)
		delete(spells, missing)
		if got, err := (character.FeatTakingRules{}).Take(c, character.TakenFeats{}, initiateFeat(), grant, clericInitiate(), character.HeldFeatures{}, spells); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s unresolved: %v, %v", missing, got, err)
		}
	}
}
