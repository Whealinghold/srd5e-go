package d20

import "github.com/Whealinghold/srd5e-go/modifier"

// CheckAttempt is how an ability check is attempted: by a [RollAttempt] or an
// [AutomaticFailure]. The variants are values; handle them with a type
// switch.
type CheckAttempt interface {
	checkAttempt()
}

// SaveAttempt is how a saving throw is attempted: by a [RollAttempt], a
// [ChooseToFail] or an [AutomaticFailure]. The variants are values; handle
// them with a type switch. A creature may choose to fail a saving throw
// ("Saving Throws", SRD 5.2.1, p. 7), so [ChooseToFail] is a SaveAttempt only.
type SaveAttempt interface {
	saveAttempt()
}

// RollAttempt rolls the dice. It is the only attempt an attack roll takes.
// Construct with keyed literals.
type RollAttempt struct {
	// Modifiers are added to the number rolled.
	Modifiers RollModifiers
	// Circumstances are the Advantage and Disadvantage sources of the roll;
	// the roll mode is resolved from them once by modifier.Resolve. The
	// resolver reads but never changes the slice.
	Circumstances []modifier.RollCircumstance
}

// ChooseToFail is a saving throw the creature fails on purpose, without
// rolling.
type ChooseToFail struct{}

// AutomaticFailure is a check or saving throw that fails automatically from
// the situation, without rolling. The caller decides that the situation
// applies.
type AutomaticFailure struct{}

func (RollAttempt) checkAttempt()      {}
func (RollAttempt) saveAttempt()       {}
func (AutomaticFailure) checkAttempt() {}
func (AutomaticFailure) saveAttempt()  {}
func (ChooseToFail) saveAttempt()      {}
