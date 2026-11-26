package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// SaveFailure is a failed saving throw that Legendary Resistance can turn into a success.
// Variants: RolledSaveFailure, ForcedSaveFailure.
type SaveFailure interface{ isSaveFailure() }

// RolledSaveFailure is a saving throw that was rolled. Construct with a keyed literal; the
// zero value holds the zero result and is rejected.
type RolledSaveFailure struct{ Result d20.CheckResult }

// ForcedSaveFailure is a saving throw that failed without a roll, chosen or automatic (SRD
// 5.2.1, "Saving Throws", p. 7). Construct with a keyed literal; the zero value holds the zero
// result and is rejected.
type ForcedSaveFailure struct{ Result d20.ForcedFailureResult }

func (RolledSaveFailure) isSaveFailure() {}
func (ForcedSaveFailure) isSaveFailure() {}

// LegendaryResistanceRefusal says why Legendary Resistance was refused. The zero value is
// invalid. Comparable and safe for concurrent use.
type LegendaryResistanceRefusal int

const (
	// LegendaryResistanceNotTracked: the trait's part has no entry in the LimitedUseState.
	LegendaryResistanceNotTracked LegendaryResistanceRefusal = iota + 1
	// LegendaryResistanceNotAFailure: the rolled saving throw succeeded.
	LegendaryResistanceNotAFailure
	// LegendaryResistanceExhausted: no use remains.
	LegendaryResistanceExhausted
)

// String returns "not_tracked", "not_a_failure", "exhausted" or "unknown".
func (r LegendaryResistanceRefusal) String() string {
	return enumName([]string{"not_tracked", "not_a_failure", "exhausted"}, int(r))
}

// LegendaryResistanceOutcome is the closed set of outcomes of UseLegendaryResistance.
// Variants: LegendaryResistanceApplied, LegendaryResistanceRefused.
type LegendaryResistanceOutcome interface{ isLegendaryResistanceOutcome() }

// LegendaryResistanceApplied: the creature chose to succeed. It holds the new
// LimitedUseState, the original failure (unedited), the effective outcome and the events.
type LegendaryResistanceApplied struct {
	state   LimitedUseState
	failure SaveFailure
	events  []Event
}

// State returns the LimitedUseState with one use spent.
func (a LegendaryResistanceApplied) State() LimitedUseState { return a.state }

// Failure returns the saving throw that failed, unchanged.
func (a LegendaryResistanceApplied) Failure() SaveFailure { return a.failure }

// EffectiveOutcome returns d20.Success: the saving throw counts as passed.
func (LegendaryResistanceApplied) EffectiveOutcome() d20.Outcome { return d20.Success }

// Events returns a copy of the events in the order they happened: LimitedUsePartSpent,
// then LegendaryResistanceUsed.
func (a LegendaryResistanceApplied) Events() []Event { return slices.Clone(a.events) }

// LegendaryResistanceRefused: Legendary Resistance cannot be used; the state is unchanged
// and there is no event. Construct with a keyed literal.
type LegendaryResistanceRefused struct{ Reason LegendaryResistanceRefusal }

func (LegendaryResistanceApplied) isLegendaryResistanceOutcome() {}
func (LegendaryResistanceRefused) isLegendaryResistanceOutcome() {}

// UseLegendaryResistance turns a failed saving throw of subject into a success, spending a
// use of the trait's part (SRD 5.2.1, "Aboleth", p. 258; "Saving Throws", p. 7). A rolled
// failure and a forced one (chosen or automatic) are accepted alike. Whether to use it is
// the host's choice. state is not changed.
//
// It returns LegendaryResistanceRefused, with state unchanged, for the first that applies:
// the part has no entry in state (LegendaryResistanceNotTracked), the rolled saving throw
// succeeded (LegendaryResistanceNotAFailure), no use remains (LegendaryResistanceExhausted).
// It returns an error matching srderr.ErrInvalidInput for a zero trait, state or subject, a
// state that belongs to another combatant, a nil or zero failure, or a failure whose kind is
// not d20.SavingThrow. It never panics.
func UseLegendaryResistance(state LimitedUseState, trait LegendaryResistanceTrait, subject CombatantID, failure SaveFailure) (LegendaryResistanceOutcome, error) {
	if trait.IsZero() || state.IsZero() || subject.IsZero() || state.combatant != subject {
		return nil, invalidInput("legendary resistance needs a trait, the state of the subject and a subject")
	}
	var rolledSuccess bool
	switch f := failure.(type) {
	case RolledSaveFailure:
		if f.Result.Kind() != d20.SavingThrow {
			return nil, invalidInput("legendary resistance covers saving throws only")
		}
		rolledSuccess = f.Result.Outcome() == d20.Success
	case ForcedSaveFailure:
		if f.Result.Kind() != d20.SavingThrow {
			return nil, invalidInput("legendary resistance covers saving throws only")
		}
	default:
		return nil, invalidInput("legendary resistance needs a failed saving throw")
	}
	availability := state.AvailabilityOf(trait.part)
	switch {
	case availability == UseNotLimited:
		return LegendaryResistanceRefused{LegendaryResistanceNotTracked}, nil
	case rolledSuccess:
		return LegendaryResistanceRefused{LegendaryResistanceNotAFailure}, nil
	case availability == UseExhausted:
		return LegendaryResistanceRefused{LegendaryResistanceExhausted}, nil
	}
	spent, ok := state.Spend(trait.part).(LimitedUseSpent)
	if !ok { // unreachable: the part is tracked and has a use left
		return nil, invalidInput("legendary resistance could not spend a use")
	}
	var events []Event
	for _, e := range spent.Transition.Events() {
		events = append(events, e)
	}
	events = append(events, LegendaryResistanceUsed{monsterTraitEventBase{subject}})
	return LegendaryResistanceApplied{spent.Transition.State(), failure, events}, nil
}
