package character

import (
	"slices"
	"testing"
)

// Requirement: doc/specs/character-creation.md FR-005 - sortProblems orders by step and then by kind, an unresolved placeholder last in either
// of the two steps that report it, keeps the order of equal ones and drops repeats, whatever order the checks found them in.
// Path: unhappy (rule interaction, determinism).
func TestSortProblemsOrdersByStepAndKind(t *testing.T) {
	t.Parallel()
	p := func(step CreationStep, kind CreateCharacterProblemKind, name string) CreateCharacterProblem {
		return CreateCharacterProblem{step: step, kind: kind, name: name}
	}
	in := []CreateCharacterProblem{
		p(CreationStepSpecies, SizeRequired, ""), p(CreationStepClass, EquipmentPlaceholderUnresolved, ""), p(CreationStepClass, ClassEquipmentOptionUnknown, ""),
		p(CreationStepBackground, EquipmentPlaceholderUnresolved, ""), p(CreationStepClass, ClassSkillNotOffered, "b"), p(CreationStepBackground, BackgroundFeatRejected, ""),
		p(CreationStepClass, ClassSkillCountWrong, ""), p(CreationStepClass, ClassSkillNotOffered, "a"), p(CreationStepClass, ClassSkillNotOffered, "b"),
		p(CreationStepBackground, BackgroundToolMissing, ""),
	}
	want := []CreateCharacterProblem{
		p(CreationStepClass, ClassSkillCountWrong, ""), p(CreationStepClass, ClassSkillNotOffered, "b"), p(CreationStepClass, ClassSkillNotOffered, "a"),
		p(CreationStepClass, ClassEquipmentOptionUnknown, ""), p(CreationStepClass, EquipmentPlaceholderUnresolved, ""),
		p(CreationStepBackground, BackgroundToolMissing, ""), p(CreationStepBackground, BackgroundFeatRejected, ""), p(CreationStepBackground, EquipmentPlaceholderUnresolved, ""),
		p(CreationStepSpecies, SizeRequired, ""),
	}
	got := sortProblems(in)
	if !slices.EqualFunc(got, want, CreateCharacterProblem.Equal) {
		t.Errorf("sortProblems = %v, want %v", got, want)
	}
	if len(in) != 10 {
		t.Error("sortProblems altered its input")
	}
}

// Requirement: doc/specs/character-creation.md FR-093 - Equal compares every field, including the feat and plan rejections by their value.
// Path: per case.
func TestCreateCharacterProblemEqual(t *testing.T) {
	t.Parallel()
	base := CreateCharacterProblem{step: CreationStepBackground, kind: BackgroundFeatRejected}
	withFeat := func(r FeatRejection) CreateCharacterProblem { q := base; q.feat = r; return q }
	withPlan := func(r HigherLevelStartRejection) CreateCharacterProblem { q := base; q.plan = r; return q }
	for _, c := range []struct {
		name string
		a, b CreateCharacterProblem
		want bool
	}{
		{"same", base, base, true},
		{"same feat rejection", withFeat(FeatNotRepeatable{}), withFeat(FeatNotRepeatable{}), true},
		{"different feat rejections", withFeat(FeatNotRepeatable{}), withFeat(FeatOutsideGrant{}), false},
		{"a feat rejection against none", withFeat(FeatNotRepeatable{}), base, false},
		{"none against a feat rejection", base, withFeat(FeatNotRepeatable{}), false},
		{"different plan rejections", withPlan(HigherLevelStartSubclassRequired{}), withPlan(HigherLevelStartPlanLengthMismatch{}), false},
		{"a plan rejection against none", withPlan(HigherLevelStartSubclassRequired{}), base, false},
		{"none against a plan rejection", base, withPlan(HigherLevelStartSubclassRequired{}), false},
		{"different kinds", base, CreateCharacterProblem{step: CreationStepBackground, kind: BackgroundToolMissing}, false},
		{"different counts", CreateCharacterProblem{step: CreationStepLanguages, kind: LanguageCountWrong, expected: 2, actual: 1}, CreateCharacterProblem{step: CreationStepLanguages, kind: LanguageCountWrong, expected: 2, actual: 3}, false},
	} {
		if got := c.a.Equal(c.b); got != c.want {
			t.Errorf("%s: Equal = %v, want %v", c.name, got, c.want)
		}
	}
}
