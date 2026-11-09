package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/d20"
)

// TacticalMindAvailability is the closed outcome of TacticalMind.Availability.
// Variants: TacticalMindAvailable, TacticalMindUnavailable.
type TacticalMindAvailability interface{ isTacticalMindAvailability() }

// TacticalMindAvailable: Tactical Mind can be used on the failed check.
type TacticalMindAvailable struct{}

// TacticalMindUnavailable: Tactical Mind cannot be used. Construct it with a keyed literal.
type TacticalMindUnavailable struct{ Reason FighterReason }

func (TacticalMindAvailable) isTacticalMindAvailability()   {}
func (TacticalMindUnavailable) isTacticalMindAvailability() {}

// TacticalMindResolution is the closed outcome of TacticalMind.Resolve.
// Variants: TacticalMindResolutionSucceeded, TacticalMindResolutionStillFailed.
type TacticalMindResolution interface{ isTacticalMindResolution() }

// TacticalMindResolutionSucceeded: the d10 turned the check into a success and a
// Second Wind use is expended. Only TacticalMind.Resolve builds one.
type TacticalMindResolutionSucceeded struct{ outcome tacticalMindOutcome }

// TacticalMindResolutionStillFailed: the check still fails and no use is
// expended. Only TacticalMind.Resolve builds one.
type TacticalMindResolutionStillFailed struct{ outcome tacticalMindOutcome }

type tacticalMindOutcome struct {
	rolled int
	check  d20.CheckResult
	state  FighterState
}

// Rolled returns the d10 rolled.
func (r TacticalMindResolutionSucceeded) Rolled() int { return r.outcome.rolled }

// Check returns the ability check with the d10 added to its total; its Roll is the original.
func (r TacticalMindResolutionSucceeded) Check() d20.CheckResult { return r.outcome.check }

// State returns the new state: one Second Wind use expended.
func (r TacticalMindResolutionSucceeded) State() FighterState { return r.outcome.state }

// Rolled returns the d10 rolled.
func (r TacticalMindResolutionStillFailed) Rolled() int { return r.outcome.rolled }

// Check returns the ability check with the d10 added to its total; its Roll is the original.
func (r TacticalMindResolutionStillFailed) Check() d20.CheckResult { return r.outcome.check }

// State returns the state, unchanged.
func (r TacticalMindResolutionStillFailed) State() FighterState { return r.outcome.state }

func (TacticalMindResolutionSucceeded) isTacticalMindResolution()   {}
func (TacticalMindResolutionStillFailed) isTacticalMindResolution() {}

// TacticalMind decides and resolves a use of Tactical Mind (SRD 5.2.1, "Level 2:
// Tactical Mind", p. 48). It is stateless and pure; the zero value is ready to use.
// Safe for concurrent use.
type TacticalMind struct{}

// Availability reports whether the Fighter can use Tactical Mind on check. The
// first reason that holds is reported: FighterFeatureNotGained below Fighter level
// 2, FighterNotAnAbilityCheck, FighterCheckNotFailed, FighterNoUsesRemaining.
func (TacticalMind) Availability(stats FighterLevelStats, state FighterState, check d20.CheckResult) TacticalMindAvailability {
	switch {
	case stats.level.n < 2:
		return TacticalMindUnavailable{Reason: FighterFeatureNotGained}
	case check.Kind() != d20.AbilityCheck:
		return TacticalMindUnavailable{Reason: FighterNotAnAbilityCheck}
	case check.Outcome() != d20.Failure:
		return TacticalMindUnavailable{Reason: FighterCheckNotFailed}
	case state.SecondWindRemaining(stats) < 1:
		return TacticalMindUnavailable{Reason: FighterNoUsesRemaining}
	}
	return TacticalMindAvailable{}
}

// Resolve adds the d10 roll rolled to check's total. Meeting or exceeding the DC
// is TacticalMindResolutionSucceeded and expends one Second Wind use; otherwise
// TacticalMindResolutionStillFailed and nothing is expended. The d20 is not
// rerolled, so only the total changes, with Advantage or Disadvantage alike
// (SRD 5.2.1, "Interactions with Rerolls", p. 8).
//
// It returns an error matching srderr.ErrInvalidInput for a rolled outside 1 to
// 10, for zero stats or when Availability says unavailable. It never panics.
func (t TacticalMind) Resolve(stats FighterLevelStats, state FighterState, check d20.CheckResult, rolled int) (TacticalMindResolution, error) {
	if stats.IsZero() || rolled < 1 || rolled > 10 {
		return nil, invalidInput("Tactical Mind roll " + strconv.Itoa(rolled) + " outside 1 to 10, or zero stats")
	}
	if _, ok := t.Availability(stats, state, check).(TacticalMindAvailable); !ok {
		return nil, invalidInput("Tactical Mind resolved while unavailable")
	}
	adjusted, err := check.AdjustedBy(rolled)
	if err != nil {
		return nil, err
	}
	if adjusted.Outcome() == d20.Success {
		state.secondWindExpended++
		return TacticalMindResolutionSucceeded{tacticalMindOutcome{rolled: rolled, check: adjusted, state: state}}, nil
	}
	return TacticalMindResolutionStillFailed{tacticalMindOutcome{rolled: rolled, check: adjusted, state: state}}, nil
}
