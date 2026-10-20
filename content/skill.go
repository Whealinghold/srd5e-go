package content

import "github.com/Whealinghold/srd5e-go/ability"

// Skill is one of the 18 skills, in the order of the Skills table (SRD 5.2.1,
// "Skills", p. 9). The zero Skill is invalid. Comparable and safe for
// concurrent use.
type Skill int

// The skills, named for their printed names.
const (
	SkillAcrobatics Skill = iota + 1
	SkillAnimalHandling
	SkillArcana
	SkillAthletics
	SkillDeception
	SkillHistory
	SkillInsight
	SkillIntimidation
	SkillInvestigation
	SkillMedicine
	SkillNature
	SkillPerception
	SkillPerformance
	SkillPersuasion
	SkillReligion
	SkillSleightOfHand
	SkillStealth
	SkillSurvival
)

func skillNames() []string {
	return []string{
		"Acrobatics", "Animal Handling", "Arcana", "Athletics", "Deception", "History",
		"Insight", "Intimidation", "Investigation", "Medicine", "Nature", "Perception",
		"Performance", "Persuasion", "Religion", "Sleight of Hand", "Stealth", "Survival",
	}
}

// Valid reports whether s is one of the 18 skills.
func (s Skill) Valid() bool { return enumValid(skillNames(), int(s)) }

// String returns the printed name of s, or "unknown" for an undefined value.
func (s Skill) String() string { return enumString(skillNames(), int(s)) }

// Ability returns the ability the Skills table prints for s, or the zero
// ability.Ability when s is invalid.
func (s Skill) Ability() ability.Ability {
	switch s {
	case SkillAcrobatics, SkillSleightOfHand, SkillStealth:
		return ability.Dexterity
	case SkillAnimalHandling, SkillInsight, SkillMedicine, SkillPerception, SkillSurvival:
		return ability.Wisdom
	case SkillArcana, SkillHistory, SkillInvestigation, SkillNature, SkillReligion:
		return ability.Intelligence
	case SkillAthletics:
		return ability.Strength
	case SkillDeception, SkillIntimidation, SkillPerformance, SkillPersuasion:
		return ability.Charisma
	}
	return 0 // equivalent mutant: replaces the literal 0 with 0
}

// ParseSkill returns the skill spelled exactly s, such as "Sleight of Hand".
// Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseSkill(s string) (Skill, error) {
	v, err := parseEnum("Skill", skillNames(), s)
	return Skill(v), err
}
