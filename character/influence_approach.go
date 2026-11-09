package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// InfluenceApproach is a row of the Influence Checks table: Deceive, Intimidate,
// Amuse, Persuade or Coax (SRD 5.2.1, "Influence [Action]", p. 184). Each names
// the skill the table suggests. The zero InfluenceApproach is invalid; see
// Valid. Comparable and safe for concurrent use.
type InfluenceApproach int

// The approaches of the Influence Checks table.
const (
	// Deceive is Deception, for a monster that understands you.
	Deceive InfluenceApproach = iota + 1
	// Intimidate is Intimidation.
	Intimidate
	// Amuse is Performance.
	Amuse
	// Persuade is Persuasion, for a monster that understands you.
	Persuade
	// Coax is Animal Handling, gently, on a Beast or Monstrosity.
	Coax
)

// Valid reports whether a is one of the five approaches.
func (a InfluenceApproach) Valid() bool { return a >= Deceive && a <= Coax }

// String returns the snake_case name of a ("deceive", "intimidate", "amuse",
// "persuade", "coax"), the value of the srd5e.influence.approach attribute, or
// "unknown" for an undefined value.
func (a InfluenceApproach) String() string {
	switch a {
	case Deceive:
		return "deceive"
	case Intimidate:
		return "intimidate"
	case Amuse:
		return "amuse"
	case Persuade:
		return "persuade"
	case Coax:
		return "coax"
	}
	return "unknown"
}

// Skill returns the skill of the approach's row, or the zero content.Skill for
// an invalid approach.
func (a InfluenceApproach) Skill() content.Skill {
	switch a {
	case Deceive:
		return content.SkillDeception
	case Intimidate:
		return content.SkillIntimidation
	case Amuse:
		return content.SkillPerformance
	case Persuade:
		return content.SkillPersuasion
	case Coax:
		return content.SkillAnimalHandling
	}
	return 0
}

// Ability returns the ability of the check, read from Skill().Ability(): Charisma
// for four approaches and Wisdom for Coax. It is the zero ability.Ability for an
// invalid approach.
func (a InfluenceApproach) Ability() ability.Ability { return a.Skill().Ability() }

// InfluenceComprehension says whether the monster understands the speaker (SRD
// 5.2.1, "Influence Checks" table, p. 184). The zero value is invalid.
// Comparable and safe for concurrent use.
type InfluenceComprehension int

// The comprehension states.
const (
	// Understands: the monster understands what is said to it.
	Understands InfluenceComprehension = iota + 1
	// DoesNotUnderstand: the monster does not.
	DoesNotUnderstand
)

// Valid reports whether c is Understands or DoesNotUnderstand.
func (c InfluenceComprehension) Valid() bool { return c == Understands || c == DoesNotUnderstand }

// String returns "understands" or "does_not_understand", or "unknown" for an
// undefined value.
func (c InfluenceComprehension) String() string {
	switch c {
	case Understands:
		return "understands"
	case DoesNotUnderstand:
		return "does_not_understand"
	}
	return "unknown"
}
