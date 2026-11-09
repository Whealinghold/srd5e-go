package character

import "github.com/Whealinghold/srd5e-go/content"

// HigherLevelStep is one level of a higher-level start: the class the level is
// gained in, the Hit Point gain, and optionally the subclass chosen at the class
// level the step reaches and an ability score increase (SRD 5.2.1, "Starting at
// Higher Levels", "Creating Your Character", p. 24). It is immutable. It holds
// an AbilityScoreIncrease, which holds a map, so it is not comparable with ==.
// The zero HigherLevelStep is invalid; build it with NewHigherLevelStep. Safe
// for concurrent use.
type HigherLevelStep struct {
	class    AdvancementClass
	gain     HitPointGain
	subclass content.SubclassID
	increase AbilityScoreIncrease
}

// NewHigherLevelStep returns the step gaining a level in class with gain. The
// zero subclass id and the zero increase mean "none".
//
// It returns an error matching srderr.ErrInvalidInput for a zero class, a nil
// gain, or a non-empty subclass id that is not valid. It never panics.
func NewHigherLevelStep(class AdvancementClass, gain HitPointGain, subclass content.SubclassID, increase AbilityScoreIncrease) (HigherLevelStep, error) {
	if class.IsZero() || gain == nil || (subclass != "" && !subclass.Valid()) {
		return HigherLevelStep{}, invalidInput("higher-level step with a zero class, a nil gain or an invalid subclass")
	}
	return HigherLevelStep{class: class, gain: gain, subclass: subclass, increase: increase}, nil
}

// Class returns the class the level is gained in.
func (s HigherLevelStep) Class() AdvancementClass { return s.class }

// HitPointGain returns the Hit Point gain of the level.
func (s HigherLevelStep) HitPointGain() HitPointGain { return s.gain }

// Subclass returns the subclass chosen with the step and true, or false for none.
func (s HigherLevelStep) Subclass() (content.SubclassID, bool) { return s.subclass, s.subclass != "" }

// Increase returns the ability score increase of the step and true, or false for
// none.
func (s HigherLevelStep) Increase() (AbilityScoreIncrease, bool) {
	return s.increase, !s.increase.IsZero()
}

// IsZero reports whether s is the invalid zero HigherLevelStep.
func (s HigherLevelStep) IsZero() bool { return s.class.IsZero() }

// StartingLevelStep is one step of a StartAtHigherLevelCommand: the class the
// level is gained in, how its Hit Points are decided, and optionally the subclass
// chosen at the class level reached and an ability score increase. The zero
// SubclassID and the zero AbilityScoreIncrease mean "none". Construct it with a
// keyed literal. It holds a map through AbilityScoreIncrease, so it is not
// comparable with ==.
type StartingLevelStep struct {
	ClassID              content.ClassID
	HitPointChoice       HitPointChoice
	SubclassID           content.SubclassID
	AbilityScoreIncrease AbilityScoreIncrease
}

// ConsumedFeatGrant reports an Ability Score Improvement (or Epic Boon) a start
// step used: the class, the class level the step reached and the id of the
// feature entry that grants it. A host keeps the reported grants beside the
// character and treats a grant as used when it appears here or is counted by
// TakenFeats.TakenFrom, so the same grant is never offered twice. Construct it
// with a keyed literal. It is comparable with ==.
type ConsumedFeatGrant struct {
	Class      content.ClassID
	ClassLevel int
	Feature    content.FeatureID
}
