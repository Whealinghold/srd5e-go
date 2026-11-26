package combat

import "slices"

// HitPointTransition is the outcome of a pure hit point operation. Variants:
// HitPointTransitionApplied, HitPointTransitionRefused.
type HitPointTransition interface{ isHitPointTransition() }

// HitPointTransitionApplied is an operation that took effect (possibly changing
// nothing): the new status, the ordered events, the per-instance resolutions
// for damage and the Hit Points actually regained for Healing. Only the
// operations build one.
type HitPointTransitionApplied struct {
	status      HitPointStatus
	events      []HitPointEvent
	resolutions []DamageResolution
	regained    int
}

// Status returns the new status.
func (t HitPointTransitionApplied) Status() HitPointStatus { return t.status }

// Events returns a copy of the events, in the order they happened.
func (t HitPointTransitionApplied) Events() []HitPointEvent { return slices.Clone(t.events) }

// Resolutions returns a copy of the per-instance damage resolutions; empty for
// an operation that is not ApplyDamage.
func (t HitPointTransitionApplied) Resolutions() []DamageResolution {
	return slices.Clone(t.resolutions)
}

// HitPointsRegained returns the Hit Points a Restore actually regained, and 0
// for every other operation.
func (t HitPointTransitionApplied) HitPointsRegained() int { return t.regained }

// HitPointTransitionRefused is an operation the rules refuse in the current
// state; the status is unchanged. Construct with a keyed literal.
type HitPointTransitionRefused struct{ Rejection HitPointRejection }

func (HitPointTransitionApplied) isHitPointTransition() {}
func (HitPointTransitionRefused) isHitPointTransition() {}

func applied(s HitPointStatus, events ...HitPointEvent) HitPointTransitionApplied {
	return HitPointTransitionApplied{status: s, events: events}
}
