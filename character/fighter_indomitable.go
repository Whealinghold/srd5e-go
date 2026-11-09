package character

import (
	"github.com/Whealinghold/srd5e-go/d20"
)

// IndomitableAvailability is the closed outcome of Indomitable.Availability.
// Variants: IndomitableAvailable, IndomitableUnavailable.
type IndomitableAvailability interface{ isIndomitableAvailability() }

// IndomitableAvailable: Indomitable can be used on the failed saving throw.
type IndomitableAvailable struct{}

// IndomitableUnavailable: Indomitable cannot be used. Construct it with a keyed literal.
type IndomitableUnavailable struct{ Reason FighterReason }

func (IndomitableAvailable) isIndomitableAvailability()   {}
func (IndomitableUnavailable) isIndomitableAvailability() {}

// IndomitableResolution is the closed outcome of Indomitable.Resolve. Variants:
// IndomitableResolutionSucceeded, IndomitableResolutionStillFailed.
type IndomitableResolution interface{ isIndomitableResolution() }

// IndomitableResolutionSucceeded: the new total meets or exceeds the DC. Only
// Indomitable.Resolve builds one.
type IndomitableResolutionSucceeded struct{ outcome indomitableOutcome }

// IndomitableResolutionStillFailed: the new total is still below the DC; the use
// is expended all the same. Only Indomitable.Resolve builds one.
type IndomitableResolutionStillFailed struct{ outcome indomitableOutcome }

type indomitableOutcome struct {
	rolled int
	check  d20.CheckResult
	state  FighterState
}

// Rolled returns the d20 rolled, which must be used.
func (r IndomitableResolutionSucceeded) Rolled() int { return r.outcome.rolled }

// Check returns the recomputed saving throw: the used die of the new Roll, the
// modifiers and the Fighter level.
func (r IndomitableResolutionSucceeded) Check() d20.CheckResult { return r.outcome.check }

// State returns the new state: one Indomitable use expended.
func (r IndomitableResolutionSucceeded) State() FighterState { return r.outcome.state }

// Rolled returns the d20 rolled, which must be used.
func (r IndomitableResolutionStillFailed) Rolled() int { return r.outcome.rolled }

// Check returns the recomputed saving throw: the used die of the new Roll, the
// modifiers and the Fighter level.
func (r IndomitableResolutionStillFailed) Check() d20.CheckResult { return r.outcome.check }

// State returns the new state: one Indomitable use expended.
func (r IndomitableResolutionStillFailed) State() FighterState { return r.outcome.state }

func (IndomitableResolutionSucceeded) isIndomitableResolution()   {}
func (IndomitableResolutionStillFailed) isIndomitableResolution() {}

// Indomitable decides and resolves a use of Indomitable (SRD 5.2.1, "Level 9:
// Indomitable", p. 48). It is stateless and pure; the zero value is ready to use.
// Safe for concurrent use.
type Indomitable struct{}

// Availability reports whether the Fighter can reroll save, replacing the die
// at index replacedDie of its Roll. The first reason that holds is reported:
// FighterFeatureNotGained below Fighter level 9, FighterNotASavingThrow,
// FighterDieNotRolled, FighterSaveNotFailed, FighterNoUsesRemaining.
func (Indomitable) Availability(stats FighterLevelStats, state FighterState, save d20.CheckResult, replacedDie int) IndomitableAvailability {
	switch {
	case stats.level.n < 9:
		return IndomitableUnavailable{Reason: FighterFeatureNotGained}
	case save.Kind() != d20.SavingThrow:
		return IndomitableUnavailable{Reason: FighterNotASavingThrow}
	case replacedDie < 0 || replacedDie >= len(save.Roll().Dice()):
		return IndomitableUnavailable{Reason: FighterDieNotRolled}
	case save.Outcome() != d20.Failure:
		return IndomitableUnavailable{Reason: FighterSaveNotFailed}
	case state.IndomitableRemaining(stats) < 1:
		return IndomitableUnavailable{Reason: FighterNoUsesRemaining}
	}
	return IndomitableAvailable{}
}

// Resolve replaces the die at replacedDie with the d20 roll rolled (only that die;
// the other keeps its value and the used die follows the Roll's mode), recomputes
// the save with its modifiers and adds the Fighter level of stats (not the total
// level). One use is expended whether the new total succeeds or not (SRD 5.2.1,
// "Interactions with Rerolls", p. 8).
//
// It returns an error matching srderr.ErrInvalidInput for zero stats or when
// Availability says unavailable, and for a rolled outside 1 to 20 (refused by
// d20.Roll.ReplaceDie). It never panics.
func (i Indomitable) Resolve(stats FighterLevelStats, state FighterState, save d20.CheckResult, replacedDie, rolled int) (IndomitableResolution, error) {
	if _, ok := i.Availability(stats, state, save, replacedDie).(IndomitableAvailable); !ok {
		return nil, invalidInput("Indomitable resolved while unavailable")
	}
	roll, err := save.Roll().ReplaceDie(replacedDie, rolled)
	if err != nil {
		return nil, err
	}
	again, err := d20.ResolveCheck(d20.SavingThrow, roll, save.Modifiers(), save.Target())
	if err != nil {
		return nil, err
	}
	adjusted, err := again.AdjustedBy(stats.level.n)
	if err != nil {
		return nil, err
	}
	state.indomitableExpended++
	out := indomitableOutcome{rolled: rolled, check: adjusted, state: state}
	if adjusted.Outcome() == d20.Success {
		return IndomitableResolutionSucceeded{out}, nil
	}
	return IndomitableResolutionStillFailed{out}, nil
}
