package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// AttitudeCircumstances returns the roll circumstances an attitude contributes
// to an Influence check: one Advantage for Friendly (SRD 5.2.1, "Friendly
// [Attitude]", p. 182), one Disadvantage for Hostile (p. 183), and none for
// Indifferent (p. 184) or an invalid attitude. The slice is a fresh copy.
func AttitudeCircumstances(a Attitude) []modifier.RollCircumstance {
	switch a {
	case Friendly:
		return []modifier.RollCircumstance{modifier.Advantage{}}
	case Hostile:
		return []modifier.RollCircumstance{modifier.Disadvantage{}}
	case Indifferent:
	}
	return []modifier.RollCircumstance{}
}

// IsSuggested reports whether the Influence Checks table suggests approach for
// target (SRD 5.2.1, p. 184): Intimidate and Amuse for any target, Deceive and
// Persuade when the target understands, Coax when it is a Beast or a
// Monstrosity. The table only suggests; the GM chooses the check.
func IsSuggested(approach InfluenceApproach, target InfluenceTarget) bool {
	switch approach {
	case Intimidate, Amuse:
		return true
	case Deceive, Persuade:
		return target.comprehension == Understands
	case Coax:
		return target.creatureType == content.CreatureTypeBeast || target.creatureType == content.CreatureTypeMonstrosity
	}
	return false
}

// DefaultDC returns the default Difficulty Class of an Influence check: 15 or
// the monster's Intelligence score, whichever is higher (SRD 5.2.1, "Influence
// [Action]", p. 184).
func DefaultDC(intelligence ability.Score) d20.DifficultyClass {
	return max(d20.DCMedium, d20.DifficultyClass(intelligence.Value()))
}
