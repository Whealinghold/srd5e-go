package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case tool-proficiency-and-crafting.md FR-034 (SRD 5.2.1,
// "Prerequisites for the Scribe", p. 103) - proficiency in Arcana (Expertise
// implies it) or with Calligrapher's Supplies; either suffices.
// Path: per case (unhappy rule interaction).
func TestQualifyScribe(t *testing.T) {
	t.Parallel()
	arcana := []content.Skill{content.SkillArcana}
	supplies := character.NewToolProficiencies("calligraphers_supplies")
	cases := []struct {
		name   string
		skills character.SkillProficiencies
		tools  character.ToolProficiencies
		want   bool
	}{
		{"Arcana only", skillsOf(arcana, nil), character.ToolProficiencies{}, true},
		{"Calligrapher's Supplies only", skillsOf(nil, nil), supplies, true},
		{"both", skillsOf(arcana, nil), supplies, true},
		{"neither", skillsOf([]content.Skill{content.SkillNature}, nil), character.NewToolProficiencies("thieves_tools"), false},
		{"Arcana Expertise", skillsOf(arcana, arcana), character.ToolProficiencies{}, true},
		{"nothing at all", character.SkillProficiencies{}, character.ToolProficiencies{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, qualified := character.QualifyScribe(c.skills, c.tools).(character.ScribeQualified)
			if qualified != c.want {
				t.Errorf("qualified = %v, want %v", qualified, c.want)
			}
		})
	}
}
