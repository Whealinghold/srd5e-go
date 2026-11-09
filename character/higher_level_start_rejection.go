package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// HigherLevelStartRejection is the closed set of reasons a higher-level start is
// refused (SRD 5.2.1, "Starting at Higher Levels", p. 24). Variants:
// HigherLevelStartPlanLengthMismatch, HigherLevelStartCreationInvalid,
// HigherLevelStartLevelUpRejected, HigherLevelStartSubclassRejected,
// HigherLevelStartIncreaseRejected, HigherLevelStartIncreaseNotGranted,
// HigherLevelStartSubclassRequired.
type HigherLevelStartRejection interface{ isHigherLevelStartRejection() }

// HigherLevelStartPlanLengthMismatch is the rejection of a plan whose number of
// steps is not the starting level minus 1. Construct it with a keyed literal.
type HigherLevelStartPlanLengthMismatch struct{ ExpectedSteps, ActualSteps int }

// HigherLevelStartCreationInvalid is the rejection of starting arguments the
// level 1 Character refuses. Only the start builds one; the zero value lists
// nothing.
type HigherLevelStartCreationInvalid struct{ problems []CreationProblem }

// Problems returns a new slice of the problems New reported, in its order.
func (r HigherLevelStartCreationInvalid) Problems() []CreationProblem {
	return slices.Clone(r.problems)
}

// HigherLevelStartLevelUpRejected is the rejection of a step the aggregate's
// LevelUp refuses. AtLevel is the total level the step would have reached.
// Construct it with a keyed literal.
type HigherLevelStartLevelUpRejected struct {
	AtLevel   Level
	Rejection LevelUpRejection
}

// HigherLevelStartSubclassRejected is the rejection of a step whose subclass
// choice is refused. AtLevel is the total level the step would have reached.
// Construct it with a keyed literal.
type HigherLevelStartSubclassRejected struct {
	AtLevel   Level
	Rejection SubclassRejected
}

// HigherLevelStartIncreaseRejected is the rejection of a step whose ability score
// increase AbilityScoreIncreaser refuses. AtLevel is the total level the step
// would have reached. Construct it with a keyed literal.
type HigherLevelStartIncreaseRejected struct {
	AtLevel   Level
	Rejection AbilityScoreIncreaseRejection
}

// HigherLevelStartIncreaseNotGranted is the rejection of a step whose ability
// score increase matches no feat grant of the class level the step reaches.
// AtLevel is the total level the step would have reached. Construct it with a
// keyed literal.
type HigherLevelStartIncreaseNotGranted struct{ AtLevel Level }

// HigherLevelStartSubclassRequired is the rejection of a plan that leaves a class
// at or above its subclass level without a subclass. Only the start builds one.
type HigherLevelStartSubclassRequired struct{ classes []content.ClassID }

// Classes returns a new slice of the classes owed a subclass, in the order they
// first appear in the ClassLevels.
func (r HigherLevelStartSubclassRequired) Classes() []content.ClassID {
	return slices.Clone(r.classes)
}

func (HigherLevelStartPlanLengthMismatch) isHigherLevelStartRejection() {}
func (HigherLevelStartCreationInvalid) isHigherLevelStartRejection()    {}
func (HigherLevelStartLevelUpRejected) isHigherLevelStartRejection()    {}
func (HigherLevelStartSubclassRejected) isHigherLevelStartRejection()   {}
func (HigherLevelStartIncreaseRejected) isHigherLevelStartRejection()   {}
func (HigherLevelStartIncreaseNotGranted) isHigherLevelStartRejection() {}
func (HigherLevelStartSubclassRequired) isHigherLevelStartRejection()   {}

// creationInvalidFrom builds the rejection from the error New returned: the
// errors.Join of CreationProblem values.
func creationInvalidFrom(err error) HigherLevelStartCreationInvalid {
	var out []CreationProblem
	var join interface{ Unwrap() []error }
	parts := []error{err}
	if errors.As(err, &join) {
		parts = join.Unwrap()
	}
	for _, p := range parts {
		var cp CreationProblem
		if errors.As(p, &cp) {
			out = append(out, cp)
		}
	}
	return HigherLevelStartCreationInvalid{problems: out}
}
