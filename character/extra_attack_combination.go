package character

import (
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ExtraAttackCountError reports an invalid input of ExtraAttackCombination:
// the Parameter name ("counts") and the InvalidValue (the count below 1, or 0
// for an empty collection). It matches srderr.ErrInvalidInput with errors.Is.
type ExtraAttackCountError struct {
	Parameter    string
	InvalidValue int
}

// Error renders the error. The text is not a contract.
func (e *ExtraAttackCountError) Error() string {
	return "character: invalid extra attack count: " + e.Parameter
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ExtraAttackCountError) Unwrap() error { return srderr.ErrInvalidInput }

// ExtraAttackCombination returns the number of attacks a multiclass character
// makes with the Attack action from Extra Attack, given the count each class
// gives (1 for a class without the feature). The features do not stack: the
// result is the largest count, so two classes that each give 2 give 2 and a
// class that gives 3 or 4 keeps its count (SRD 5.2.1, "Multiclassing", "Extra
// Attack", p. 25). The order of counts does not matter and counts is not
// altered.
//
// It returns an *ExtraAttackCountError, matching srderr.ErrInvalidInput, for an
// empty counts or any count below 1. It never panics.
func ExtraAttackCombination(counts []int) (int, error) {
	if len(counts) == 0 {
		return 0, &ExtraAttackCountError{Parameter: "counts"}
	}
	largest := 0
	for _, n := range counts {
		if n < 1 {
			return 0, &ExtraAttackCountError{Parameter: "counts", InvalidValue: n}
		}
		largest = max(largest, n)
	}
	return largest, nil
}
