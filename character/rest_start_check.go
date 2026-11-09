package character

import "slices"

// longRestSpacingHours is how long a creature must wait after finishing a Long
// Rest before starting another (SRD 5.2.1, "Long Rest", p. 185).
const longRestSpacingHours = 16

// maxExhaustionLevel is the Exhaustion level at which a creature dies (SRD 5.2.1,
// "Exhaustion", p. 181).
const maxExhaustionLevel = 6

// RestStartBlocker is a reason a rest may not start. Variants:
// RestStartNoHitPoints, RestStartExhaustionLevelSix,
// RestStartTooSoonAfterLongRest.
type RestStartBlocker interface{ isRestStartBlocker() }

// RestStartNoHitPoints: the creature has 0 Hit Points (SRD 5.2.1, "Short Rest",
// p. 187; "Long Rest", p. 185).
type RestStartNoHitPoints struct{}

// RestStartExhaustionLevelSix: the creature is at Exhaustion level 6, which is
// death (SRD 5.2.1, "Exhaustion", p. 181).
type RestStartExhaustionLevelSix struct{}

// RestStartTooSoonAfterLongRest: a Long Rest finished fewer than 16 hours ago.
type RestStartTooSoonAfterLongRest struct{ hoursStillToWait int }

// HoursStillToWait returns 16 minus the hours since the previous Long Rest
// finished, at least 1.
func (b RestStartTooSoonAfterLongRest) HoursStillToWait() int { return b.hoursStillToWait }

func (RestStartNoHitPoints) isRestStartBlocker()          {}
func (RestStartExhaustionLevelSix) isRestStartBlocker()   {}
func (RestStartTooSoonAfterLongRest) isRestStartBlocker() {}

// RestStartDecision is the outcome of CheckRestStart. Variants: RestMayStart,
// RestMayNotStart.
type RestStartDecision interface{ isRestStartDecision() }

// RestMayStart: nothing blocks the rest.
type RestMayStart struct{}

// RestMayNotStart: at least one blocker applies.
type RestMayNotStart struct{ blockers []RestStartBlocker }

// Blockers returns a new slice of every blocker, once each, in the order
// RestStartNoHitPoints, RestStartExhaustionLevelSix,
// RestStartTooSoonAfterLongRest.
func (d RestMayNotStart) Blockers() []RestStartBlocker { return slices.Clone(d.blockers) }

func (RestMayStart) isRestStartDecision()    {}
func (RestMayNotStart) isRestStartDecision() {}

// CheckRestStart decides whether a rest of kind may start with the values at the
// start of the rest: a creature needs at least 1 current Hit Point, must not be
// at Exhaustion level 6, and may not start a Long Rest fewer than 16 hours after
// finishing the previous one (SRD 5.2.1, "Short Rest", p. 187; "Long Rest",
// p. 185). A Short Rest is never blocked by spacing. A nil spacing is
// NoLongRestFinishedYet.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid kind,
// negative currentHitPoints or an exhaustionLevel outside 0 to 6. It never
// panics.
func CheckRestStart(kind RestKind, currentHitPoints, exhaustionLevel int, spacing LongRestSpacing) (RestStartDecision, error) {
	if !kind.Valid() || currentHitPoints < 0 || exhaustionLevel < 0 || exhaustionLevel > maxExhaustionLevel {
		return nil, invalidInput("rest start check with an invalid kind, Hit Points or Exhaustion level")
	}
	var blockers []RestStartBlocker
	if currentHitPoints == 0 {
		blockers = append(blockers, RestStartNoHitPoints{})
	}
	if exhaustionLevel == maxExhaustionLevel {
		blockers = append(blockers, RestStartExhaustionLevelSix{})
	}
	if ago, ok := spacing.(LongRestFinishedHoursAgo); ok && kind == LongRest && ago.hours < longRestSpacingHours {
		blockers = append(blockers, RestStartTooSoonAfterLongRest{hoursStillToWait: longRestSpacingHours - ago.hours})
	}
	if len(blockers) > 0 {
		return RestMayNotStart{blockers: blockers}, nil
	}
	return RestMayStart{}, nil
}
