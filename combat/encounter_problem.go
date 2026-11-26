package combat

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// EncounterProblem is one reason Encounter.Start rejected its input. Variants:
// NoCombatantsProblem, MissingSideProblem, DuplicateCombatantIDProblem,
// CombatantNotInTurnOrderProblem, TurnOrderEntryWithoutCombatantProblem, CombatantLimitProblem.
type EncounterProblem interface{ isEncounterProblem() }

// NoCombatantsProblem: the combatant list is empty.
type NoCombatantsProblem struct{}

// MissingSideProblem: no combatant is on Side.
type MissingSideProblem struct{ Side Side }

// DuplicateCombatantIDProblem: a second combatant has the id ID.
type DuplicateCombatantIDProblem struct{ ID CombatantID }

// CombatantNotInTurnOrderProblem: the combatant ID is not in the turn order.
type CombatantNotInTurnOrderProblem struct{ ID CombatantID }

// TurnOrderEntryWithoutCombatantProblem: the turn order names ID but no combatant has it.
type TurnOrderEntryWithoutCombatantProblem struct{ ID CombatantID }

// CombatantLimitProblem: the combatants or the turn order hold more than 256 combatants (plan P10). It is the
// only problem reported then: the other checks do not run, and the result value is limit_exceeded.
type CombatantLimitProblem struct{}

func (NoCombatantsProblem) isEncounterProblem()                   {}
func (MissingSideProblem) isEncounterProblem()                    {}
func (DuplicateCombatantIDProblem) isEncounterProblem()           {}
func (CombatantNotInTurnOrderProblem) isEncounterProblem()        {}
func (TurnOrderEntryWithoutCombatantProblem) isEncounterProblem() {}
func (CombatantLimitProblem) isEncounterProblem()                 {}

// EncounterProblemError is one EncounterProblem as an error. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type EncounterProblemError struct{ problem EncounterProblem }

// Problem returns the problem.
func (e *EncounterProblemError) Problem() EncounterProblem { return e.problem }

// Error renders the error. The text is not a contract.
func (e *EncounterProblemError) Error() string {
	return "combat: invalid encounter: " + typeName(e.problem)
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *EncounterProblemError) Unwrap() error { return srderr.ErrInvalidInput }

// ProblemsOf returns the EncounterProblems an error from Encounter.Start joins, in
// the order found. It returns nil for an error that holds none (and for nil).
func ProblemsOf(err error) []EncounterProblem {
	var out []EncounterProblem
	var walk func(error)
	walk = func(e error) {
		if e == nil {
			return
		}
		if h, ok := e.(interface{ Problem() EncounterProblem }); ok {
			out = append(out, h.Problem())
			return
		}
		if j, ok := e.(interface{ Unwrap() []error }); ok {
			for _, c := range j.Unwrap() {
				walk(c)
			}
			return
		}
		walk(errors.Unwrap(e))
	}
	walk(err)
	return out
}

// joinEncounterProblems joins the problems into one error, nil for none.
func joinEncounterProblems(problems []EncounterProblem) error {
	errs := make([]error, len(problems))
	for i, p := range problems {
		errs[i] = &EncounterProblemError{p}
	}
	return errors.Join(errs...)
}
