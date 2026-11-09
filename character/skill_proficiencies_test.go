package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/derived-stats.md FR-016 - Expertise names only skills
// with proficiency; every offender is reported, in Skill order (SRD 5.2.1,
// "Expertise", p. 182).
// Path: unhappy (invalid input).
func TestNewSkillProficiencies(t *testing.T) {
	t.Parallel()
	p, err := character.NewSkillProficiencies(
		[]content.Skill{content.SkillAthletics, content.SkillStealth, content.SkillStealth},
		[]content.Skill{content.SkillStealth})
	if err != nil || !p.Proficient(content.SkillAthletics) || !p.HasExpertise(content.SkillStealth) ||
		!p.Proficient(content.SkillStealth) || p.HasExpertise(content.SkillAthletics) || p.Proficient(content.SkillArcana) {
		t.Fatalf("proficiencies = %+v, %v", p, err)
	}
	_, err = character.NewSkillProficiencies([]content.Skill{content.SkillAthletics},
		[]content.Skill{content.SkillSurvival, content.SkillAcrobatics, content.SkillAthletics, content.SkillSurvival})
	var se *character.SkillProficienciesError
	if !errors.As(err, &se) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("error = %v", err)
	}
	got := se.ExpertiseWithoutProficiency()
	if want := []content.Skill{content.SkillAcrobatics, content.SkillSurvival}; !slices.Equal(got, want) || len(got) == 0 {
		t.Errorf("offenders = %v, want %v", got, want)
	}
	got[0] = content.SkillArcana
	if se.ExpertiseWithoutProficiency()[0] != content.SkillAcrobatics {
		t.Error("ExpertiseWithoutProficiency exposes its slice")
	}
	if se.Error() == "" {
		t.Error("empty error text")
	}
}

// Requirement: doc/specs/derived-stats.md FR-016 - MustSkillProficiencies
// panics exactly when NewSkillProficiencies errors; an invalid skill is rejected.
// Path: unhappy (invalid input).
func TestMustSkillProficiencies(t *testing.T) {
	t.Parallel()
	if got := recovered(t, func() { character.MustSkillProficiencies(nil, nil) }); got != nil {
		t.Errorf("Must(nil, nil) panicked: %v", got)
	}
	if got := recovered(t, func() { character.MustSkillProficiencies(nil, []content.Skill{content.SkillArcana}) }); !errors.Is(got, srderr.ErrInvalidInput) {
		t.Errorf("Must panic = %v", got)
	}
	_, e1 := character.NewSkillProficiencies([]content.Skill{0}, nil)
	_, e2 := character.NewSkillProficiencies(nil, []content.Skill{99})
	wantInvalid(t, e1, "invalid proficient skill")
	wantInvalid(t, e2, "invalid expertise skill")
	var zero character.SkillProficiencies
	if zero.Proficient(0) || zero.HasExpertise(0) {
		t.Error("invalid skill reported as held")
	}
}
