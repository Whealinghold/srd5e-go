package character_test

import (
	"io/fs"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// hlsLevelSixRow is the Wizard Features table row of level 6 in the bundled document (the
// subclass feature) and hlsLevelSixASI the row of a Fighter-like class that repeats
// Ability Score Improvement there (SRD 5.2.1, Fighter Features table, p. 47).
const (
	hlsLevelSixRow = "\"level\": 6,\n          \"proficiencyBonus\": 3,\n          \"features\": [\n            {\n              \"kind\": \"subclassFeature\"\n            }\n          ],"
	hlsLevelSixASI = "\"level\": 6,\n          \"proficiencyBonus\": 3,\n          \"features\": [\n            {\n              \"featureId\": \"fighter_ability_score_improvement\"\n            }\n          ],"
)

// hlsFighter is the Fighter of Table SH: Hit Point Die d10, subclass level 3 with the Champion
// subclass, Ability Score Improvement at class levels 4 and 8 (the Wizard table) and again at 6,
// and an Epic Boon at 19.
func hlsFighter(t *testing.T) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "fighter", die: "D10", kind: "anyOf", abilities: []string{"strength", "dexterity"}, subclass: []string{"champion"},
		replace: [][2]string{{hlsLevelSixRow, hlsLevelSixASI}}})
}

// hlsSpecies is the bundled human species, a lookup target only.
func hlsSpecies(t *testing.T) content.Entry {
	t.Helper()
	raw, err := fs.ReadFile(data.Files(), "species.json")
	if err != nil {
		t.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range set.Entries() {
		if s, ok := e.(content.Species); ok && s.ID() == "human" {
			return s
		}
	}
	t.Fatal("no human species in the bundle")
	return nil
}

func hlsBackground(t *testing.T) content.Entry {
	t.Helper()
	return content.MustBackground("soldier", "Soldier", content.SourceCitation{}, content.BackgroundFields{
		AbilityScores:      []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution},
		Feat:               content.MustBackgroundFeatReference("feat_a", ""),
		SkillProficiencies: []content.Skill{content.SkillAthletics, content.SkillStealth},
		ToolProficiency:    must(content.NewSpecificToolProficiency("Tool")),
		EquipmentOptions:   content.NewEquipmentOptions(content.EquipmentOption{}, content.EquipmentOption{}),
	})
}

// hlsSource is a content source holding the human species, the soldier background and the given
// classes.
func hlsSource(t *testing.T, classes ...content.ClassDefinition) *contenttest.FakeSource {
	t.Helper()
	entries := []content.Entry{hlsSpecies(t), hlsBackground(t)}
	for _, c := range classes {
		entries = append(entries, c)
	}
	return contenttest.NewFakeSource(entries...)
}

// hlsScores are the scores of Table SH: Strength s, Constitution 14 (+2), the rest 10.
func hlsScores(t *testing.T, strength int) ability.Scores {
	t.Helper()
	return scoresOf(t, strength, 10, 14, 10, 10, 10)
}

// hlsStart is the level 1 character a plan is applied to, with the XP of the starting level.
func hlsStart(t *testing.T, class content.ClassDefinition, scores ability.Scores, level int) character.Character {
	t.Helper()
	return must(character.New(character.NewParams{
		ID: character.MustIdentifier("pc-1"), Class: class.ID(), HitPointDie: class.HitPointDie(), Species: "human", Background: "soldier",
		AbilityScores: scores, ExperiencePoints: character.HigherLevelStart{}.StartingExperience(character.MustLevel(level)),
	})).Character()
}

// hlsStep builds a domain step. A zero subclass and a zero increase mean none.
func hlsStep(t *testing.T, class content.ClassDefinition, gain character.HitPointGain, subclass string, inc character.AbilityScoreIncrease) character.HigherLevelStep {
	t.Helper()
	adv := must(character.NewAdvancementClass(class.ID(), class.HitPointDie(), class.PrimaryAbility()))
	return must(character.NewHigherLevelStep(adv, gain, content.SubclassID(subclass), inc))
}

func classFixedGain(class content.ClassDefinition) character.HitPointGain {
	return must(character.NewFixedHitPointGain(class.HitPointDie().Faces()/2 + 1))
}

func plusTwo(a ability.Ability) character.AbilityScoreIncrease {
	return must(character.NewAbilityScoreImprovementOneByTwo(a))
}

func classMap(defs ...content.ClassDefinition) map[content.ClassID]content.ClassDefinition {
	m := map[content.ClassID]content.ClassDefinition{}
	for _, d := range defs {
		m[d.ID()] = d
	}
	return m
}
