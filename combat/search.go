package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// SearchTarget is a row of the Search table: what a Search looks for. The zero value is invalid.
type SearchTarget int

// The rows of the Search table (SRD 5.2.1, "Search [Action]", p. 187).
const (
	// SearchTargetCreaturesStateOfMind: a creature's state of mind.
	SearchTargetCreaturesStateOfMind SearchTarget = iota + 1
	// SearchTargetCreaturesAilmentOrCauseOfDeath: a creature's ailment or cause of death.
	SearchTargetCreaturesAilmentOrCauseOfDeath
	// SearchTargetConcealedCreatureOrObject: a concealed creature or object.
	SearchTargetConcealedCreatureOrObject
	// SearchTargetTracksOrFood: tracks or food.
	SearchTargetTracksOrFood
)

// String returns the snake_case name of t or "unknown".
func (t SearchTarget) String() string {
	return enumName([]string{"creatures_state_of_mind", "creatures_ailment_or_cause_of_death", "concealed_creature_or_object", "tracks_or_food"}, int(t))
}

// SearchProximity says whether a search is near a hidden object. The zero value is invalid.
type SearchProximity int

// The proximities.
const (
	// SearchProximityNearTarget: the searcher is near the hidden object.
	SearchProximityNearTarget SearchProximity = iota + 1
	// SearchProximityNotNearTarget: the searcher is not.
	SearchProximityNotNearTarget
)

// SearchReveal says whether a search can reveal a hidden object. The zero value is invalid.
type SearchReveal int

// The reveals.
const (
	// SearchRevealPossible: a high enough total can reveal it.
	SearchRevealPossible SearchReveal = iota + 1
	// SearchRevealImpossible: no total can.
	SearchRevealImpossible
)

// String returns "possible", "impossible" or "unknown".
func (r SearchReveal) String() string { return enumName([]string{"possible", "impossible"}, int(r)) }

// SearchSkillAssessment is the closed set of answers of SearchRules.Assess.
// Variants: SuggestedSearchSkill, OffTableSearchSkill, InvalidSearchSkill.
type SearchSkillAssessment interface{ isSearchSkillAssessment() }

// SuggestedSearchSkill: the Search table suggests the skill for the target.
type SuggestedSearchSkill struct {
	Skill  content.Skill
	Target SearchTarget
}

// OffTableSearchSkill: a Wisdom skill the table does not name; the Game Master may accept it.
type OffTableSearchSkill struct{ Skill content.Skill }

// InvalidSearchSkill: the skill is not a Wisdom skill. Ability is the skill's ability
// (the zero ability for an undefined skill).
type InvalidSearchSkill struct {
	Skill   content.Skill
	Ability ability.Ability
}

func (SuggestedSearchSkill) isSearchSkillAssessment() {}
func (OffTableSearchSkill) isSearchSkillAssessment()  {}
func (InvalidSearchSkill) isSearchSkillAssessment()   {}

// SearchRules is the stateless domain service for the Search action (SRD 5.2.1,
// "Search [Action]", p. 187; "Finding Hidden Objects", p. 12). Pure, synchronous and
// silent; the zero value is ready to use. Safe for concurrent use.
type SearchRules struct{}

// Ability returns Wisdom, the ability of a Search check.
func (SearchRules) Ability() ability.Ability { return ability.Wisdom }

// SuggestedSkills returns a new slice of the skills the Search table suggests for the
// target: Insight, Medicine, Perception or Survival. It is empty for an undefined target.
func (SearchRules) SuggestedSkills(t SearchTarget) []content.Skill {
	switch t {
	case SearchTargetCreaturesStateOfMind:
		return []content.Skill{content.SkillInsight}
	case SearchTargetCreaturesAilmentOrCauseOfDeath:
		return []content.Skill{content.SkillMedicine}
	case SearchTargetConcealedCreatureOrObject:
		return []content.Skill{content.SkillPerception}
	case SearchTargetTracksOrFood:
		return []content.Skill{content.SkillSurvival}
	}
	return []content.Skill{}
}

// Assess classifies a skill for a Search: SuggestedSearchSkill for the four on the
// table, OffTableSearchSkill for any other Wisdom skill (accepted at the Game
// Master's discretion), InvalidSearchSkill for a skill whose ability is not Wisdom.
func (r SearchRules) Assess(s content.Skill) SearchSkillAssessment {
	if s.Ability() != r.Ability() {
		return InvalidSearchSkill{s, s.Ability()}
	}
	for t := SearchTargetCreaturesStateOfMind; t <= SearchTargetTracksOrFood; t++ {
		if r.SuggestedSkills(t)[0] == s {
			return SuggestedSearchSkill{s, t}
		}
	}
	return OffTableSearchSkill{s}
}

// CanReveal returns Impossible for a search that is not near a hidden object and
// Possible when it is (SRD 5.2.1, "Finding Hidden Objects", p. 12). Finding a hidden
// creature is unaffected.
func (SearchRules) CanReveal(p SearchProximity) SearchReveal {
	if p == SearchProximityNearTarget {
		return SearchRevealPossible
	}
	return SearchRevealImpossible
}
