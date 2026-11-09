package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/social-interaction.md FR-006, FR-007, FR-042 - the five approaches
// map to Deception, Intimidation, Performance, Persuasion and Animal Handling; the ability
// is read from the skill (Charisma for four, Wisdom for Coax); an invalid approach has
// neither (SRD 5.2.1, "Influence Checks" table, p. 184; "Skills" table, p. 9).
// Path: per case.
func TestInfluenceApproach(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		a                       character.InfluenceApproach
		skill                   content.Skill
		ab                      ability.Ability
		str                     string
	}{
		{"deceive", "FR-006", "happy", character.Deceive, content.SkillDeception, ability.Charisma, "deceive"},
		{"intimidate", "FR-006", "happy", character.Intimidate, content.SkillIntimidation, ability.Charisma, "intimidate"},
		{"amuse", "FR-006", "happy", character.Amuse, content.SkillPerformance, ability.Charisma, "amuse"},
		{"persuade", "FR-006", "happy", character.Persuade, content.SkillPersuasion, ability.Charisma, "persuade"},
		{"coax", "FR-007", "happy", character.Coax, content.SkillAnimalHandling, ability.Wisdom, "coax"},
		{"zero", "FR-006", "unhappy (invalid input)", 0, 0, 0, "unknown"},
		{"past coax", "FR-006", "unhappy (boundary)", character.Coax + 1, 0, 0, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wantValid := c.skill != 0
			if c.a.Valid() != wantValid || c.a.Skill() != c.skill || c.a.Ability() != c.ab || c.a.String() != c.str {
				t.Errorf("approach %d = valid %v, %v, %v, %q [%s, %s]", int(c.a), c.a.Valid(), c.a.Skill(), c.a.Ability(), c.a.String(), c.requirement, c.path)
			}
			if c.a.Ability() != c.a.Skill().Ability() {
				t.Errorf("Ability() = %v, Skill().Ability() = %v [FR-042]", c.a.Ability(), c.a.Skill().Ability())
			}
		})
	}
}

// Requirement: doc/specs/social-interaction.md FR-008 - InfluenceComprehension is
// Understands or DoesNotUnderstand; the zero value is invalid (SRD 5.2.1, "Influence
// Checks" table, p. 184).
// Path: per case.
func TestInfluenceComprehension(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		c                       character.InfluenceComprehension
		valid                   bool
		str                     string
	}{
		{"understands", "FR-008", "happy", character.Understands, true, "understands"},
		{"does not understand", "FR-008", "happy", character.DoesNotUnderstand, true, "does_not_understand"},
		{"zero", "FR-008", "unhappy (invalid input)", 0, false, "unknown"},
		{"past the last", "FR-008", "unhappy (boundary)", character.DoesNotUnderstand + 1, false, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.c.Valid() != c.valid || c.c.String() != c.str {
				t.Errorf("comprehension %d = valid %v, %q [%s, %s]", int(c.c), c.c.Valid(), c.c.String(), c.requirement, c.path)
			}
		})
	}
}
