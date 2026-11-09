package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func rogInput(t *testing.T, level int, entry character.RogueEntry) character.RogueChoicesInput {
	t.Helper()
	return character.RogueChoicesInput{
		Progression: rogProgression(t), Entry: entry, Level: rl(level),
		Skills:             []content.Skill{skillStealth, skillSleight, skillPerc, skillAcro},
		Expertise:          []content.Skill{skillStealth, skillSleight},
		Language:           content.LanguageElvish,
		PermittedLanguages: []content.Language{content.LanguageElvish, content.LanguageCommon},
		Equipment:          character.RogueEquipmentOptionA,
	}
}

// Requirement: doc/specs/class-rogue.md FR-022 - New validates only the choices the level requires and the Rogue rules own (skills,
// Expertise, language, equipment) and reports every invalid or missing choice, never only the first; the subclass, a feat and
// Weapon Mastery are listed by RequiredFor but not validated here.
// Path: per case (valid level 1, three independent problems, missing picks, levels needing no owned choice): happy / unhappy (invalid input).
func TestRogueChoicesNew(t *testing.T) {
	t.Parallel()
	first, multi := character.RogueFirstClass, character.RogueMulticlass
	P := func(p ...character.RogueChoiceProblem) []character.RogueChoiceProblem { return p }
	badSkills := rogInput(t, 1, first)
	badSkills.Skills = []content.Skill{skillArcana, skillArcana}
	badSkills.Expertise = []content.Skill{skillStealth}
	badSkills.Language = content.LanguageThievesCant
	missing := rogInput(t, 1, first)
	missing.Skills, missing.Expertise, missing.Language, missing.Equipment = nil, nil, 0, 0
	l6 := rogInput(t, 6, first)
	l6.Proficient, l6.ExpertiseUsed = []content.Skill{skillStealth, skillSleight, skillPerc}, []character.RogueExpertiseWindow{character.ExpertiseLevelOne}
	l6.Expertise = []content.Skill{skillPerc, skillStealth}
	l6.ExistingExpertise = []content.Skill{skillStealth, skillSleight}
	l6ok := l6
	l6ok.Expertise = []content.Skill{skillPerc, skillAcro}
	l6ok.Proficient = append(slices.Clone(l6ok.Proficient), skillAcro)
	for _, c := range []struct {
		name     string
		in       character.RogueChoicesInput
		problems []character.RogueChoiceProblem
	}{
		{"valid first class level 1", rogInput(t, 1, first), nil},
		{"valid multiclass level 1", func() character.RogueChoicesInput {
			i := rogInput(t, 1, multi)
			i.Skills = []content.Skill{skillSleight}
			i.Proficient = []content.Skill{skillStealth, skillSleight}
			return i
		}(), nil},
		{"valid level 6", l6ok, nil},
		{"subclass level needs nothing owned", rogInput(t, 3, first), nil},
		{"feat level needs nothing owned", rogInput(t, 4, first), nil},
		{"three independent problems", badSkills, P(
			character.RogueChoiceProblem{Kind: character.RogueChoiceSkills, Skill: character.RogueSkillWrongCount},
			character.RogueChoiceProblem{Kind: character.RogueChoiceSkills, Skill: character.RogueSkillDuplicate},
			character.RogueChoiceProblem{Kind: character.RogueChoiceSkills, Skill: character.RogueSkillNotInRogueList},
			character.RogueChoiceProblem{Kind: character.RogueChoiceExpertise, Expertise: character.RogueExpertiseWrongCount},
			character.RogueChoiceProblem{Kind: character.RogueChoiceExpertise, Expertise: character.RogueExpertiseNotProficient},
			character.RogueChoiceProblem{Kind: character.RogueChoiceLanguage, Language: character.RogueLanguageIsThievesCant},
			character.RogueChoiceProblem{Kind: character.RogueChoiceLanguage, Language: character.RogueLanguageNotPermitted},
		)},
		{"everything missing", missing, P(
			character.RogueChoiceProblem{Kind: character.RogueChoiceSkills, Missing: true},
			character.RogueChoiceProblem{Kind: character.RogueChoiceExpertise, Missing: true},
			character.RogueChoiceProblem{Kind: character.RogueChoiceLanguage, Missing: true},
			character.RogueChoiceProblem{Kind: character.RogueChoiceStartingEquipment, Missing: true},
		)},
		{"level 6 repeats an Expertise skill", l6, P(character.RogueChoiceProblem{Kind: character.RogueChoiceExpertise, Expertise: character.RogueExpertiseAlreadyHasExpertise})},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.RogueChoices{}.New(c.in)
			if c.problems != nil {
				if probs := rogProblems[character.RogueChoiceProblem](t, err); !slices.Equal(probs, c.problems) {
					t.Errorf("problems = %v, want %v", probs, c.problems)
				}
				return
			}
			if err != nil {
				t.Fatalf("New = %v", err)
			}
			if c.in.Level.Value() == 1 && c.in.Entry == first && (len(got.Skills().Skills()) != 4 || got.Equipment().Option() != character.RogueEquipmentOptionA || got.Language().Language() != content.LanguageElvish || got.Expertise().Window() != character.ExpertiseLevelOne) {
				t.Errorf("choices = %v %v %v %v", got.Skills().Skills(), got.Equipment().Option(), got.Language().Language(), got.Expertise().Window())
			}
			if c.in.Level.Value() == 6 && got.Expertise().Window() != character.ExpertiseLevelSix {
				t.Errorf("level 6 window = %v", got.Expertise().Window())
			}
			if c.in.Level.Value() >= 3 && c.in.Level.Value() != 6 && (len(got.Skills().Skills()) != 0 || got.Language().Language() != 0 || got.Equipment().Option() != 0) {
				t.Errorf("level %d validated a choice it did not require", c.in.Level.Value())
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-022 - an undefined entry, a zero level or the zero progression is a programmer error
// refused with ErrInvalidInput.
// Path: per case (unhappy: invalid input).
func TestRogueChoicesNewRefusesBadInput(t *testing.T) {
	t.Parallel()
	for name, mutate := range map[string]func(*character.RogueChoicesInput){
		"entry":       func(i *character.RogueChoicesInput) { i.Entry = 0 },
		"level":       func(i *character.RogueChoicesInput) { i.Level = character.RogueLevel{} },
		"progression": func(i *character.RogueChoicesInput) { i.Progression = character.RogueProgression{} },
	} {
		in := rogInput(t, 1, character.RogueFirstClass)
		mutate(&in)
		_, err := character.RogueChoices{}.New(in)
		wantInvalid(t, err, name)
	}
}

// Requirement: doc/specs/class-rogue.md FR-108, FR-110, decisions FR-024 - NewRogueCreationChoices validates the level 1 choices of a
// first-class Rogue (reporting every problem at once), implements ClassCreationChoices, names no content and seeds the RogueState
// under RogueStateKey with the level 1 Expertise window used and the language recorded, leaving other entries and its argument unchanged.
// Path: happy / unhappy (invalid input).
func TestRogueCreationChoices(t *testing.T) {
	t.Parallel()
	in := rogInput(t, 9, character.RogueMulticlass) // the entry and level are overridden: creation is a first-class level 1
	c, err := character.NewRogueCreationChoices(in)
	if err != nil {
		t.Fatalf("NewRogueCreationChoices = %v", err)
	}
	if len(c.Choices().Skills().Skills()) != 4 || len(c.ContentKeys()) != 0 {
		t.Errorf("choices = %v, keys %v", c.Choices().Skills().Skills(), c.ContentKeys())
	}
	var other character.FeatureStates
	other = must(character.SetFeatureState(other, character.FighterStateKey(), character.InitialFighterState()))
	seeded := c.SeedFeatureStates(other)
	state, ok, rerr := character.ReadFeatureState(seeded, character.RogueStateKey())
	if rerr != nil || !ok || state.OtherLanguage() != content.LanguageElvish || !slices.Equal(state.ExpertiseWindowsUsed(), []character.RogueExpertiseWindow{character.ExpertiseLevelOne}) ||
		state.StrokeOfLuck() != character.StrokeOfLuckStateAvailable || seeded.Len() != 2 || other.Len() != 1 {
		t.Errorf("seeded state = %v, %v, %v; states %d, argument %d", state, ok, rerr, seeded.Len(), other.Len())
	}
	var asInterface character.ClassCreationChoices = c
	if asInterface.SeedFeatureStates(character.FeatureStates{}).Len() != 1 {
		t.Error("the seeded states do not hold the Rogue state")
	}
	bad := rogInput(t, 1, character.RogueFirstClass)
	bad.Skills = nil
	z, err := character.NewRogueCreationChoices(bad)
	if got := rogProblems[character.RogueChoiceProblem](t, err); len(got) != 2 || !got[0].Missing || got[0].Kind != character.RogueChoiceSkills || got[1].Expertise != character.RogueExpertiseNotProficient {
		t.Errorf("problems = %v", got)
	}
	if z.SeedFeatureStates(character.FeatureStates{}).Len() != 1 { // the zero value seeds an initial state, never panics
		t.Error("the zero creation choices seeded nothing")
	}
}
