package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// RogueProblemsError is the error a Rogue constructor returns for untrusted
// input: every problem found, in check order, never only the first (SRD 5.2.1,
// "Rogue Features", p. 62). P is the closed problem enum of the operation, for
// example CunningStrikeProblem. It matches srderr.ErrInvalidInput with
// errors.Is and carries no display text. Read the problems with errors.As to a
// *RogueProblemsError[P]. Immutable; safe for concurrent use.
type RogueProblemsError[P comparable] struct {
	operation string
	problems  []P
}

// Problems returns the problems found as a new slice, in check order.
func (e *RogueProblemsError[P]) Problems() []P { return slices.Clone(e.problems) }

// Error renders the error. The text is not a contract.
func (e *RogueProblemsError[P]) Error() string {
	return "character: invalid " + e.operation + ": " + srderr.ErrInvalidInput.Error()
}

// Unwrap returns srderr.ErrInvalidInput.
func (*RogueProblemsError[P]) Unwrap() error { return srderr.ErrInvalidInput }

// rogueProblems returns the error for the operation when problems is not empty, and nil otherwise.
func rogueProblems[P comparable](operation string, problems []P) error {
	if len(problems) == 0 {
		return nil
	}
	return &RogueProblemsError[P]{operation: operation, problems: problems}
}

// addOnce appends p to list unless it is already there.
func addOnce[P comparable](list []P, p P) []P {
	if slices.Contains(list, p) {
		return list
	}
	return append(list, p)
}
