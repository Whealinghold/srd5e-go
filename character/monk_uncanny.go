package character

import "strconv"

// UncannyMetabolismResolution is the outcome of using Uncanny Metabolism: the die
// rolled, the Hit Points to regain, the Focus Points regained and the new state. The
// Hit Points are not capped by the Hit Point maximum, which the host does (SRD 5.2.1,
// "Damage and Healing", p. 17). Only UncannyMetabolism.Resolve builds one.
type UncannyMetabolismResolution struct {
	rolled, hitPoints, regained int
	state                       MonkState
}

// Rolled returns the Martial Arts die rolled.
func (r UncannyMetabolismResolution) Rolled() int { return r.rolled }

// HitPointsToRegain returns the Monk level plus the die rolled, uncapped.
func (r UncannyMetabolismResolution) HitPointsToRegain() int { return r.hitPoints }

// FocusPointsRegained returns how many expended Focus Points were regained.
func (r UncannyMetabolismResolution) FocusPointsRegained() int { return r.regained }

// State returns the new state: every Focus Point regained and the feature used until a Long Rest.
func (r UncannyMetabolismResolution) State() MonkState { return r.state }

// UncannyMetabolism decides and resolves a use of Uncanny Metabolism (SRD 5.2.1,
// "Level 2: Uncanny Metabolism", p. 51). It is stateless and pure; the zero value is
// ready to use. Safe for concurrent use.
type UncannyMetabolism struct{}

// Availability reports whether the Monk can use Uncanny Metabolism when
// rolledInitiative says the Monk rolled Initiative: MonkNotAMonk for zero stats,
// MonkLevelTooLow below level 2, MonkNotEligible when the trigger is not an
// Initiative roll, MonkAlreadyUsed when it was used since the last Long Rest.
func (UncannyMetabolism) Availability(stats MonkLevelStats, state MonkState, rolledInitiative bool) MonkAvailability {
	return monkAvailability(
		monkGate(stats, monkFocusLevel),
		monkIf(!rolledInitiative, MonkNotEligible),
		monkIf(state.uncannyUsed, MonkAlreadyUsed),
	)
}

// Resolve uses Uncanny Metabolism with the Martial Arts die roll rolled: every
// expended Focus Point is regained and the Hit Points to regain are the Monk level
// plus rolled.
//
// It returns an error matching srderr.ErrInvalidInput for a rolled outside the
// Martial Arts die, or when Availability says unavailable. It never panics.
func (u UncannyMetabolism) Resolve(stats MonkLevelStats, state MonkState, rolledInitiative bool, rolled int) (UncannyMetabolismResolution, error) {
	if stats.IsZero() || rolled < 1 || rolled > stats.martialArts.Sides() {
		return UncannyMetabolismResolution{}, invalidInput("Uncanny Metabolism roll " + strconv.Itoa(rolled) + " outside the Martial Arts die, or zero stats")
	}
	if _, ok := u.Availability(stats, state, rolledInitiative).(MonkAvailable); !ok {
		return UncannyMetabolismResolution{}, invalidInput("Uncanny Metabolism resolved while unavailable")
	}
	regained := min(state.focusExpended, stats.focusPoints)
	state.focusExpended, state.uncannyUsed = 0, true
	return UncannyMetabolismResolution{rolled: rolled, hitPoints: stats.level.n + rolled, regained: regained, state: state}, nil
}

// PerfectFocusResolution is the outcome of Perfect Focus: the Focus Points regained
// and the new state. Only PerfectFocus.Resolve builds one.
type PerfectFocusResolution struct {
	regained int
	state    MonkState
}

// FocusPointsRegained returns how many Focus Points were regained; 0 with 4 or more available.
func (r PerfectFocusResolution) FocusPointsRegained() int { return r.regained }

// State returns the new state.
func (r PerfectFocusResolution) State() MonkState { return r.state }

// PerfectFocus decides and resolves Perfect Focus (SRD 5.2.1, "Level 15: Perfect
// Focus", p. 52). It is stateless and pure; the zero value is ready to use. Safe for
// concurrent use.
type PerfectFocus struct{}

// perfectFocusFloor is the number of Focus Points Perfect Focus regains up to.
const perfectFocusFloor = 4

// Availability reports whether Perfect Focus applies when the Monk rolled Initiative
// (rolledInitiative) and did or did not use Uncanny Metabolism for that roll:
// MonkNotAMonk for zero stats, MonkLevelTooLow below level 15, MonkNotEligible when
// the trigger is not an Initiative roll or Uncanny Metabolism was used.
func (PerfectFocus) Availability(stats MonkLevelStats, rolledInitiative, usedUncannyMetabolism bool) MonkAvailability {
	return monkAvailability(
		monkGate(stats, monkPerfectFocusLevel),
		monkIf(!rolledInitiative || usedUncannyMetabolism, MonkNotEligible),
	)
}

// Resolve regains expended Focus Points until the Monk has 4, if the Monk has 3 or
// fewer available, and changes nothing with 4 or more.
//
// It returns an error matching srderr.ErrInvalidInput when Availability says
// unavailable. It never panics.
func (p PerfectFocus) Resolve(stats MonkLevelStats, state MonkState, rolledInitiative, usedUncannyMetabolism bool) (PerfectFocusResolution, error) {
	if _, ok := p.Availability(stats, rolledInitiative, usedUncannyMetabolism).(MonkAvailable); !ok {
		return PerfectFocusResolution{}, invalidInput("Perfect Focus resolved while unavailable")
	}
	available := state.FocusPoints(stats).Available()
	if available >= perfectFocusFloor {
		return PerfectFocusResolution{state: state}, nil
	}
	target := min(perfectFocusFloor, stats.focusPoints)
	state.focusExpended = stats.focusPoints - target
	return PerfectFocusResolution{regained: target - available, state: state}, nil
}

// Event returns the FocusPointsRegained event for the character id.
func (r PerfectFocusResolution) Event(id Identifier) MonkEvent {
	return FocusPointsRegained{id: id, count: r.regained}
}
