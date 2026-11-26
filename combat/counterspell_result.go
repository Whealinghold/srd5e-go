package combat

import (
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
)

// CounterspellAssessment is the closed result of Counterspell.Assess. Its variants
// are CounterspellEligible and CounterspellIneligible.
type CounterspellAssessment interface {
	isCounterspellAssessment()
}

// CounterspellEligible is an assessment that Counterspell can counter the cast. It
// carries the spell save DC of the counterspeller, which the caster's save must
// target (FR-041).
type CounterspellEligible struct {
	counterspeller CombatantID
	obs            CastObservation
	dc             character.SpellSaveDC
}

// CounterspellIneligible is an assessment that Counterspell cannot counter the
// cast: Refusal says why. Nothing is cast.
type CounterspellIneligible struct{ refusal CounterspellRefusal }

// Observation returns the cast Counterspell targets.
func (e CounterspellEligible) Observation() CastObservation { return e.obs }

// Dc returns the spell save DC of the counterspeller; the caster's save targets it.
func (e CounterspellEligible) Dc() character.SpellSaveDC { return e.dc }

// Refusal returns why the cast cannot be countered.
func (r CounterspellIneligible) Refusal() CounterspellRefusal { return r.refusal }

func (CounterspellEligible) isCounterspellAssessment()   {}
func (CounterspellIneligible) isCounterspellAssessment() {}

// CounterspellSave is the caster's Constitution save against Counterspell. It is a
// closed set: CounterspellRolledSave and CounterspellForcedFailure.
type CounterspellSave interface {
	isCounterspellSave()
}

// CounterspellRolledSave is a save the use case rolled through d20.ResolveCheck.
type CounterspellRolledSave struct{ check d20.CheckResult }

// CounterspellForcedFailure is a save that failed without a roll (the caster
// fails by choice or by rule).
type CounterspellForcedFailure struct{ failure d20.ForcedFailureResult }

// NewCounterspellRolledSave returns the save check. It never panics.
func NewCounterspellRolledSave(check d20.CheckResult) CounterspellRolledSave {
	return CounterspellRolledSave{check: check}
}

// NewCounterspellForcedFailure returns the forced failure. It never panics.
func NewCounterspellForcedFailure(failure d20.ForcedFailureResult) CounterspellForcedFailure {
	return CounterspellForcedFailure{failure: failure}
}

// Check returns the rolled check.
func (s CounterspellRolledSave) Check() d20.CheckResult { return s.check }

// Failure returns the forced failure.
func (s CounterspellForcedFailure) Failure() d20.ForcedFailureResult { return s.failure }

func (CounterspellRolledSave) isCounterspellSave()    {}
func (CounterspellForcedFailure) isCounterspellSave() {}

// WastedActionCost is the action a countered cast wasted (SRD 5.2.1, "Counterspell",
// p. 120). A cast of one minute or longer wastes none. The zero value is not a cost.
type WastedActionCost int

const (
	// WastedNone: the cast took a minute or longer, so no action is wasted.
	WastedNone WastedActionCost = iota + 1
	// WastedMagicAction: the cast took the Magic action.
	WastedMagicAction
	// WastedBonusAction: the cast took the Bonus Action.
	WastedBonusAction
	// WastedReaction: the cast took the Reaction.
	WastedReaction
)

// String returns "none", "magic_action", "bonus_action", "reaction" or "unknown".
func (c WastedActionCost) String() string {
	return enumName([]string{"none", "magic_action", "bonus_action", "reaction"}, int(c))
}

// CounterspellProblem says why a save cannot conclude Counterspell. The zero value is not a problem.
type CounterspellProblem int

const (
	// CounterspellNotASavingThrow: the save's kind is not a saving throw.
	CounterspellNotASavingThrow CounterspellProblem = iota + 1
	// CounterspellSaveTargetMismatch: the save's target is not the counterspeller's DC.
	CounterspellSaveTargetMismatch
	// CounterspellMissingInput: the eligible assessment is the zero value or the save is nil.
	CounterspellMissingInput
)

// String returns "not_a_saving_throw", "save_target_mismatch", "missing_input" or "unknown".
func (p CounterspellProblem) String() string {
	return enumName([]string{"not_a_saving_throw", "save_target_mismatch", "missing_input"}, int(p))
}

// CounterspellResult is the closed result of concluding Counterspell. Its variants
// are CounterspellCountered, CounterspellResistedResult and CounterspellInvalid.
type CounterspellResult interface {
	isCounterspellResult()
}

// CounterspellCountered is a cast that failed to cast: the caster failed the save.
// The host discards the uncommitted cast (no slot expended, no effect). SlotNotExpended
// is true when the cast came from a slot; PriorConcentrationEnds is true when the spell
// requires Concentration, which ends the caster's Concentration on another effect
// (SRD 5.2.1, "Concentration", p. 179); Wasted is the action the cast used.
type CounterspellCountered struct {
	event              SpellCountered
	slotNotExpended    bool
	priorConcentration bool
	wasted             WastedActionCost
}

// CounterspellResistedResult is a cast the caster's save resisted: the spell goes ahead.
type CounterspellResistedResult struct{ event CounterspellResisted }

// CounterspellInvalid is a conclusion the save given does not support: Problem says which.
type CounterspellInvalid struct{ problem CounterspellProblem }

// Event returns the SpellCountered event.
func (r CounterspellCountered) Event() SpellCountered { return r.event }

// SlotNotExpended reports whether the cast came from a spell slot, which is restored.
func (r CounterspellCountered) SlotNotExpended() bool { return r.slotNotExpended }

// PriorConcentrationEnds reports whether the countered spell required Concentration.
func (r CounterspellCountered) PriorConcentrationEnds() bool { return r.priorConcentration }

// Wasted returns the action the countered cast used.
func (r CounterspellCountered) Wasted() WastedActionCost { return r.wasted }

// Event returns the CounterspellResisted.
func (r CounterspellResistedResult) Event() CounterspellResisted { return r.event }

// Problem returns why the conclusion is invalid.
func (r CounterspellInvalid) Problem() CounterspellProblem { return r.problem }

func (CounterspellCountered) isCounterspellResult()      {}
func (CounterspellResistedResult) isCounterspellResult() {}
func (CounterspellInvalid) isCounterspellResult()        {}
