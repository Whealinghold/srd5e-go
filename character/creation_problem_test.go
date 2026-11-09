package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/character-creation.md FR-094, FR-073, FR-074 - a problem carries typed data: the step, the offending skill, tool,
// letter, ability and counts, and the rejection of the plan or a feat.
// Path: per case.
func TestCreateCharacterProblemData(t *testing.T) {
	t.Parallel()
	one := func(mutate func(*character.CreateCharacterCommand)) character.CreateCharacterProblem {
		cmd := crCommand()
		mutate(&cmd)
		r, ok := crMust(t, cmd).(character.CreateCharacterRejected)
		if !ok || len(r.Problems()) != 1 {
			t.Fatalf("result = %#v", r)
		}
		return r.Problems()[0]
	}
	if p := one(func(c *character.CreateCharacterCommand) { c.Options, c.Plan = startingLevel(3), c.Plan[:0] }); p.Step() != character.CreationStepStartingLevel || p.Expected() != 2 || p.Actual() != 0 {
		t.Errorf("plan length problem = %+v", p)
	}
	if p := one(func(c *character.CreateCharacterCommand) {
		c.ClassSkills = []content.Skill{content.SkillAthletics, content.SkillArcana}
	}); p.Step() != character.CreationStepClass || p.Skill() != content.SkillArcana {
		t.Errorf("skill problem = %+v", p)
	}
	if p := one(func(c *character.CreateCharacterCommand) {
		c.ClassEquipment = character.OptionC
		c.ClassID = "wizard"
		c.ClassSkills = []content.Skill{content.SkillArcana, content.SkillHistory}
	}); p.Letter() != character.OptionC || p.Kind() != character.ClassEquipmentOptionUnknown {
		t.Errorf("letter problem = %+v", p)
	}
	if p := one(func(c *character.CreateCharacterCommand) {
		c.BackgroundIncrease = character.MustPlusTwoPlusOne(ability.Charisma, ability.Strength)
	}); p.Ability() != ability.Charisma || p.Step() != character.CreationStepBackground {
		t.Errorf("ability problem = %+v", p)
	}
	if p := one(func(c *character.CreateCharacterCommand) {
		c.Languages = []content.Language{content.LanguageAbyssal, content.LanguageElvish}
	}); p.Language() != content.LanguageAbyssal {
		t.Errorf("language problem = %+v", p)
	}
	if p := one(func(c *character.CreateCharacterCommand) { c.SpeciesID = "human" }); p.Kind() != character.SizeRequired || p.Name() != "" || p.FeatRejection() != nil || p.PlanRejection() != nil {
		t.Errorf("size problem = %+v", p)
	}
	if p := one(func(c *character.CreateCharacterCommand) {
		c.Options, c.Plan = startingLevel(2), []character.StartingLevelStep{{ClassID: "wizard", HitPointChoice: character.TakeFixedValue{}}}
	}); p.PlanRejection() == nil {
		t.Errorf("plan rejection problem = %+v", p)
	}
	if p := one(func(c *character.CreateCharacterCommand) {
		c.SpeciesGrants = must(character.NewSpeciesGrantSelections(0, "savage_attacker", nil))
	}); p.FeatRejection() == nil {
		t.Errorf("feat rejection problem = %+v", p)
	}
}

// Requirement: doc/specs/character-creation.md FR-005 - problems are ordered by step and then by kind, an unresolved placeholder last in
// its step, the list has no repeats, and the same command gives the same list.
// Path: unhappy (rule interaction, determinism).
func TestCreateCharacterProblemsAreOrderedAndStable(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.ClassID, cmd.ClassSkills = "monk", []content.Skill{content.SkillAcrobatics, content.SkillArcana}
	cmd.BackgroundTool, cmd.BackgroundEquipment = nil, character.OptionC
	cmd.SpeciesID = "human"
	want := []character.CreateCharacterProblemKind{character.ClassSkillNotOffered, character.ClassToolCountWrong, character.EquipmentPlaceholderUnresolved,
		character.BackgroundToolMissing, character.BackgroundEquipmentOptionUnknown, character.SizeRequired}
	for range 3 {
		if got := crKinds(t, crMust(t, cmd)); !reflect.DeepEqual(got, want) {
			t.Errorf("kinds = %v, want %v", got, want)
		}
	}
}

// Requirement: doc/specs/character-creation.md FR-094, section 5 - the kinds are numbered 1 to 31 and a value outside is invalid and
// "unknown", including the first value past the last.
// Path: unhappy (boundary).
func TestCreateCharacterProblemKindBounds(t *testing.T) {
	t.Parallel()
	for _, k := range []character.CreateCharacterProblemKind{0, -1, 32, 33, 99} {
		if k.Valid() || k.String() != "unknown" {
			t.Errorf("kind %d: Valid %v, String %q", k, k.Valid(), k.String())
		}
	}
	if !character.ClassSkillCountWrong.Valid() || !character.LevelAdvancementRejected.Valid() || character.LevelAdvancementRejected != 31 || character.ClassSkillCountWrong != 1 {
		t.Error("the first or last kind is not valid or not numbered 1 and 31")
	}
}

// Requirement: doc/specs/character-creation.md FR-005 - when the checks find problems out of kind order (a background tool missing, an
// unresolved placeholder, then a refused feat), the result lists them in kind order with the placeholder last.
// Path: unhappy (rule interaction).
func TestCreateCharacterProblemOrderDiffersFromDiscoveryOrder(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.BackgroundTool, cmd.BackgroundFeatChoices = nil, clericInitiate()
	want := []character.CreateCharacterProblemKind{character.BackgroundToolMissing, character.BackgroundFeatRejected, character.EquipmentPlaceholderUnresolved}
	if got := crKinds(t, crMust(t, cmd)); !reflect.DeepEqual(got, want) {
		t.Errorf("kinds = %v, want %v", got, want)
	}
}
