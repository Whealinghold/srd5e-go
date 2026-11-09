package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// SavingThrowModifier is the modifier of one saving throw (SRD 5.2.1, "Saving
// Throws", p. 7) with the breakdown that produced it. Its value is the
// breakdown's total. It is immutable; it holds a slice, so it is not comparable
// with ==. The zero value is invalid. Safe for concurrent use.
type SavingThrowModifier struct {
	ability   ability.Ability
	breakdown ModifierBreakdown
}

// Ability returns the ability of the save.
func (m SavingThrowModifier) Ability() ability.Ability { return m.ability }

// Value returns the modifier to add to the save, the total of the breakdown.
func (m SavingThrowModifier) Value() int { return m.breakdown.Total().Value() }

// Breakdown returns the terms that make up the value.
func (m SavingThrowModifier) Breakdown() ModifierBreakdown { return m.breakdown }

// SkillModifier is the modifier of one skill (SRD 5.2.1, "Skills", p. 9) with
// the breakdown that produced it. Its value is the breakdown's total. It is
// immutable; it holds a slice, so it is not comparable with ==. The zero value
// is invalid. Safe for concurrent use.
type SkillModifier struct {
	skill     content.Skill
	breakdown ModifierBreakdown
}

// Skill returns the skill.
func (m SkillModifier) Skill() content.Skill { return m.skill }

// Value returns the modifier to add to a check of the skill, the total of the
// breakdown.
func (m SkillModifier) Value() int { return m.breakdown.Total().Value() }

// Breakdown returns the terms that make up the value.
func (m SkillModifier) Breakdown() ModifierBreakdown { return m.breakdown }

// PassivePerception is the Passive Perception score (SRD 5.2.1, "Passive
// Perception", p. 22, p. 186) with the breakdown that produced it. Its value is
// the breakdown's total. It is immutable; it holds a slice, so it is not
// comparable with ==. The zero value is a score of 0. Safe for concurrent use.
type PassivePerception struct{ breakdown ModifierBreakdown }

// Value returns the score, the total of the breakdown.
func (p PassivePerception) Value() int { return p.breakdown.Total().Value() }

// Breakdown returns the terms that make up the value.
func (p PassivePerception) Breakdown() ModifierBreakdown { return p.breakdown }
