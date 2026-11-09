package character

import "strconv"

// SecondWindAvailability is the closed outcome of SecondWind.Availability.
// Variants: SecondWindAvailable, SecondWindUnavailable.
type SecondWindAvailability interface{ isSecondWindAvailability() }

// SecondWindAvailable: Second Wind can be used.
type SecondWindAvailable struct{}

// SecondWindUnavailable: Second Wind cannot be used. Construct it with a keyed literal.
type SecondWindUnavailable struct{ Reason FighterReason }

func (SecondWindAvailable) isSecondWindAvailability()   {}
func (SecondWindUnavailable) isSecondWindAvailability() {}

// SecondWindResolution is the outcome of using Second Wind: the d10 rolled, the
// Hit Points to regain and the state with one more use expended. The Hit Points
// are not capped by the Hit Point maximum, which the host does (SRD 5.2.1,
// "Damage and Healing", p. 17). Only SecondWind.Resolve builds one.
type SecondWindResolution struct {
	rolled, hitPoints int
	state             FighterState
}

// Rolled returns the d10 rolled.
func (r SecondWindResolution) Rolled() int { return r.rolled }

// HitPointsToRegain returns the d10 roll plus the Fighter level, uncapped.
func (r SecondWindResolution) HitPointsToRegain() int { return r.hitPoints }

// State returns the new state: one more Second Wind use expended.
func (r SecondWindResolution) State() FighterState { return r.state }

// SecondWind decides and resolves a use of Second Wind (SRD 5.2.1, "Level 1:
// Second Wind", p. 48). It is stateless and pure; the zero value is ready to use.
// Safe for concurrent use.
type SecondWind struct{}

// Availability reports whether the Fighter of stats' level can use Second Wind:
// SecondWindUnavailable with FighterNoUsesRemaining when every use is expended.
// The zero stats have no uses and are unavailable.
func (SecondWind) Availability(stats FighterLevelStats, state FighterState) SecondWindAvailability {
	if state.SecondWindRemaining(stats) < 1 {
		return SecondWindUnavailable{Reason: FighterNoUsesRemaining}
	}
	return SecondWindAvailable{}
}

// Resolve uses Second Wind with the d10 roll rolled: the Hit Points to regain are
// rolled plus the Fighter level of stats (not the total level), and one use is
// expended.
//
// It returns an error matching srderr.ErrInvalidInput for a rolled outside 1 to
// 10, for zero stats, or when Availability says unavailable. It never panics.
func (s SecondWind) Resolve(stats FighterLevelStats, state FighterState, rolled int) (SecondWindResolution, error) {
	if stats.IsZero() || rolled < 1 || rolled > 10 {
		return SecondWindResolution{}, invalidInput("Second Wind roll " + strconv.Itoa(rolled) + " outside 1 to 10, or zero stats")
	}
	if _, ok := s.Availability(stats, state).(SecondWindAvailable); !ok {
		return SecondWindResolution{}, invalidInput("Second Wind resolved with no use remaining")
	}
	state.secondWindExpended++
	return SecondWindResolution{rolled: rolled, hitPoints: rolled + stats.level.n, state: state}, nil
}

// TacticalShiftOffer is the closed result of FighterEffects.TacticalShift: the
// Fighter may or may not move after Second Wind. Variants: TacticalShiftOffered,
// NoTacticalShift.
type TacticalShiftOffer interface{ isTacticalShiftOffer() }

// TacticalShiftOffered: the Fighter may move up to half its Speed without
// provoking Opportunity Attacks. The offer carries the fraction, not a distance.
type TacticalShiftOffered struct{}

// NoTacticalShift: no move is offered.
type NoTacticalShift struct{}

func (TacticalShiftOffered) isTacticalShiftOffer() {}
func (NoTacticalShift) isTacticalShiftOffer()      {}

// HalfOf returns half of speed in feet, rounded down (SRD 5.2.1, "Round Down",
// p. 187): 30 gives 15, 25 gives 12, 0 gives 0. It returns an error matching
// srderr.ErrInvalidInput for a negative speed. It never panics.
func (TacticalShiftOffered) HalfOf(speed int) (int, error) {
	if speed < 0 {
		return 0, invalidInput("negative Speed")
	}
	return speed / 2, nil
}
