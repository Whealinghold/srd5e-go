package character

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// SkillProficiencies is the skills a character is proficient in and, among
// them, those with Expertise (SRD 5.2.1, "Skills", p. 9; "Expertise", p. 182).
// It is an immutable value, comparable with ==. The zero value holds no
// proficiency. Safe for concurrent use.
type SkillProficiencies struct{ proficient, expertise uint32 }

// SkillProficienciesError reports Expertise in skills without proficiency. It
// holds at least one skill. It matches srderr.ErrInvalidInput with errors.Is.
type SkillProficienciesError struct {
	expertiseWithoutProficiency []content.Skill
}

// ExpertiseWithoutProficiency returns a new slice of every skill that has
// Expertise but no proficiency, in Skill order.
func (e *SkillProficienciesError) ExpertiseWithoutProficiency() []content.Skill {
	return append([]content.Skill(nil), e.expertiseWithoutProficiency...)
}

// Error renders the error. The text is not a contract.
func (e *SkillProficienciesError) Error() string {
	names := make([]string, len(e.expertiseWithoutProficiency))
	for i, s := range e.expertiseWithoutProficiency {
		names[i] = s.String()
	}
	return "character: Expertise without proficiency in " + strings.Join(names, ", ")
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *SkillProficienciesError) Unwrap() error { return srderr.ErrInvalidInput }

// NewSkillProficiencies returns the proficiencies in proficient with Expertise
// in expertise. Order and repeats are irrelevant.
//
// It returns a *SkillProficienciesError (matching srderr.ErrInvalidInput) that
// holds every skill in expertise that is not in proficient, and an error
// matching srderr.ErrInvalidInput for an invalid skill. It never panics.
func NewSkillProficiencies(proficient, expertise []content.Skill) (SkillProficiencies, error) {
	var p, x uint32
	for _, s := range proficient {
		if !s.Valid() {
			return SkillProficiencies{}, invalidInput("invalid skill")
		}
		p |= 1 << s
	}
	for _, s := range expertise {
		if !s.Valid() {
			return SkillProficiencies{}, invalidInput("invalid skill")
		}
		x |= 1 << s
	}
	if missing := x &^ p; missing != 0 {
		var skills []content.Skill
		for s := content.SkillAcrobatics; s <= content.SkillSurvival; s++ {
			if missing&(1<<s) != 0 {
				skills = append(skills, s)
			}
		}
		return SkillProficiencies{}, &SkillProficienciesError{expertiseWithoutProficiency: skills}
	}
	return SkillProficiencies{proficient: p, expertise: x}, nil
}

// MustSkillProficiencies is NewSkillProficiencies for known-valid input. It
// panics with the error NewSkillProficiencies returns, and only then.
func MustSkillProficiencies(proficient, expertise []content.Skill) SkillProficiencies {
	s, err := NewSkillProficiencies(proficient, expertise)
	if err != nil {
		panic(err)
	}
	return s
}

// Proficient reports whether the character is proficient in s (Expertise
// included).
func (p SkillProficiencies) Proficient(s content.Skill) bool {
	return s.Valid() && p.proficient&(1<<s) != 0
}

// HasExpertise reports whether the character has Expertise in s.
func (p SkillProficiencies) HasExpertise(s content.Skill) bool {
	return s.Valid() && p.expertise&(1<<s) != 0
}
