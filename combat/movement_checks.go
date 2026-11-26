package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// MovementChecks are the checks of the conditions on a move or a posture change that the turn tracker
// cannot make because it records the Speeds of its profile, which go stale after a condition change
// (combat-action-resolution.md FR-057 to FR-061, FR-124). It is pure and stateless; the zero value is
// usable. Each method returns a nil rejection when the request is allowed, and an error matching
// srderr.ErrInvalidInput for a negative Speed.
type MovementChecks struct{}

// isEffectivelyProne reports whether the set holds Prone, explicitly or by implication (Unconscious).
func isEffectivelyProne(set ConditionSet) bool {
	return slices.Contains(set.Effective(), content.ConditionProne)
}

// speedForbidden reports whether the conditions force the walking Speed to 0.
func speedForbidden(set ConditionSet, walkingSpeedFeet int) (bool, error) {
	sp, err := (ConditionEffects{}).Speed(set, walkingSpeedFeet)
	return sp.Increase == SpeedIncreaseForbidden, err
}

// Move checks a move along segments: a Speed the conditions force to 0, then a Prone mover with a
// segment that is not a crawl (SRD 5.2.1, "Prone [Condition]", p. 186). walkingSpeedFeet is the Speed
// of the tracker's profile; only whether the conditions zero it matters.
func (MovementChecks) Move(set ConditionSet, walkingSpeedFeet int, segments []MoveSegment) (ResolutionRejection, error) {
	zero, err := speedForbidden(set, walkingSpeedFeet)
	if err != nil {
		return nil, err
	}
	var rej ResolutionRejection
	switch {
	case zero:
		rej = MovementRestrictedRejection{MovementRestrictionSpeedZeroedByCondition}
	case isEffectivelyProne(set) && slices.ContainsFunc(segments, func(s MoveSegment) bool { return s.Mode() != MovementModeCrawl }):
		rej = MovementRestrictedRejection{MovementRestrictionProneAllowsOnlyCrawl}
	}
	return rej, nil
}

// DropProne checks Drop Prone: a mover already effectively Prone, then a Speed the conditions force to 0
// (SRD 5.2.1, "Dropping Prone", p. 14).
func (MovementChecks) DropProne(set ConditionSet, walkingSpeedFeet int) (ResolutionRejection, error) {
	zero, err := speedForbidden(set, walkingSpeedFeet)
	if err != nil {
		return nil, err
	}
	var rej ResolutionRejection
	switch {
	case isEffectivelyProne(set):
		rej = PostureMismatchRejection{PostureMismatchAlreadyProne}
	case zero:
		rej = MovementRestrictedRejection{MovementRestrictionSpeedZeroedByCondition}
	}
	return rej, nil
}

// StandUp checks standing up: a mover that is not effectively Prone, then one that can only crawl
// because its Speed is 0. It returns the ProneMovement the tracker takes the cost from when allowed.
func (MovementChecks) StandUp(set ConditionSet, baseWalkingSpeedFeet int) (ProneMovement, ResolutionRejection, error) {
	if !isEffectivelyProne(set) {
		if baseWalkingSpeedFeet < 0 {
			return nil, nil, invalidInput("a stand up needs a walking Speed of 0 or more")
		}
		return nil, PostureMismatchRejection{PostureMismatchNotProne}, nil
	}
	m, err := (ConditionEffects{}).StandUp(set, baseWalkingSpeedFeet)
	if err != nil {
		return nil, nil, err
	}
	if _, crawlOnly := m.(CrawlOnly); crawlOnly {
		return nil, MovementRestrictedRejection{MovementRestrictionCannotRightItself}, nil
	}
	return m, nil, nil
}
