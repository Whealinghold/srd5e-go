package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

var searchR combat.SearchRules

// Requirement: doc/specs/hiding-searching-and-senses.md FR-074 (SRD 5.2.1, "Search [Action]", p. 187) - the Search check is Wisdom.
// Path: happy.
func TestSearchAbility(t *testing.T) {
	t.Parallel()
	if searchR.Ability() != ability.Wisdom {
		t.Errorf("Ability = %v", searchR.Ability())
	}
}

// Requirement: per case (doc/specs/hiding-searching-and-senses.md 4.11; SRD 5.2.1, "Search [Action]", table, p. 187)
// Path: per case.
func TestSearchSuggestedSkills(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement string
		target            combat.SearchTarget
		want              content.Skill
	}{
		{"state of mind", "FR-075", combat.SearchTargetCreaturesStateOfMind, content.SkillInsight},
		{"ailment or cause of death", "FR-076", combat.SearchTargetCreaturesAilmentOrCauseOfDeath, content.SkillMedicine},
		{"concealed creature or object", "FR-077", combat.SearchTargetConcealedCreatureOrObject, content.SkillPerception},
		{"tracks or food", "FR-078", combat.SearchTargetTracksOrFood, content.SkillSurvival},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := searchR.SuggestedSkills(c.target)
			if !slices.Equal(got, []content.Skill{c.want}) {
				t.Errorf("SuggestedSkills(%v) = %v, want %v [%s]", c.target, got, c.want, c.requirement)
			}
			got[0] = 0
			if searchR.SuggestedSkills(c.target)[0] != c.want {
				t.Error("SuggestedSkills is not a fresh slice [FR-100]")
			}
			if a, ok := searchR.Assess(c.want).(combat.SuggestedSearchSkill); !ok || a.Target != c.target || a.Skill != c.want {
				t.Errorf("Assess(%v) = %v", c.want, a)
			}
		})
	}
	if got := searchR.SuggestedSkills(0); got == nil || len(got) != 0 {
		t.Errorf("SuggestedSkills(undefined) = %#v, want empty", got)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-079, FR-080, FR-081 - Animal Handling is off the table, Arcana, Stealth
// and Persuasion are invalid with their ability, an undefined skill is invalid, and across the 18 skills exactly four are
// suggested, one off table and thirteen invalid.
// Path: unhappy (boundary).
func TestSearchAssessSkills(t *testing.T) {
	t.Parallel()
	if a := searchR.Assess(content.SkillAnimalHandling); a != (combat.OffTableSearchSkill{Skill: content.SkillAnimalHandling}) {
		t.Errorf("Assess(Animal Handling) = %v", a)
	}
	for skill, want := range map[content.Skill]ability.Ability{content.SkillArcana: ability.Intelligence, content.SkillStealth: ability.Dexterity, content.SkillPersuasion: ability.Charisma, 0: 0} {
		if a := searchR.Assess(skill); a != (combat.InvalidSearchSkill{Skill: skill, Ability: want}) {
			t.Errorf("Assess(%v) = %v, want invalid with %v", skill, a, want)
		}
	}
	var suggested, off, invalid int
	for s := content.SkillAcrobatics; s <= content.SkillSurvival; s++ {
		switch searchR.Assess(s).(type) {
		case combat.SuggestedSearchSkill:
			suggested++
		case combat.OffTableSearchSkill:
			off++
		case combat.InvalidSearchSkill:
			invalid++
		}
	}
	if suggested != 4 || off != 1 || invalid != 13 {
		t.Errorf("partition = %d, %d, %d, want 4, 1, 13", suggested, off, invalid)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-082 (SRD 5.2.1, "Finding Hidden Objects", p. 12) - a search nowhere near a
// hidden object reveals nothing, whatever the total; near it, a reveal is possible.
// Path: unhappy (state error).
func TestSearchCanReveal(t *testing.T) {
	t.Parallel()
	if searchR.CanReveal(combat.SearchProximityNearTarget) != combat.SearchRevealPossible ||
		searchR.CanReveal(combat.SearchProximityNotNearTarget) != combat.SearchRevealImpossible ||
		searchR.CanReveal(0) != combat.SearchRevealImpossible {
		t.Error("CanReveal does not follow the proximity")
	}
	got := []string{combat.SearchRevealPossible.String(), combat.SearchRevealImpossible.String(), combat.SearchReveal(0).String(),
		combat.SearchTargetTracksOrFood.String(), combat.SearchTarget(0).String()}
	if want := []string{"possible", "impossible", "unknown", "tracks_or_food", "unknown"}; !slices.Equal(got, want) {
		t.Errorf("strings = %v, want %v", got, want)
	}
}
