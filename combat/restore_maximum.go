package combat

import "slices"

// RestoreHitPointMaximumResult is the closed set of answers of RestoreMaximum. Variants:
// RestoreMaximumApplied, RestoreMaximumRejected, RestoreMaximumInvalidAmount.
type RestoreHitPointMaximumResult interface{ isRestoreHitPointMaximumResult() }

// RestoreMaximumApplied: the maximum rose by the amount, or nothing changed for an
// amount of 0. Events holds HitPointMaximumRestored, none for an amount of 0.
type RestoreMaximumApplied struct {
	status HitPointStatus
	events []Event
}

// Status returns the Hit Point status after the call.
func (r RestoreMaximumApplied) Status() HitPointStatus { return r.status }

// Events returns a copy of the events in the order they happened.
func (r RestoreMaximumApplied) Events() []Event { return slices.Clone(r.events) }

// RestoreMaximumRejected: the creature is Dead (a creature that died of a maximum of
// 0 stays Dead here; ReviveBySpell with RestoredMaximum is the way back); nothing
// changed. Construct it with a keyed literal.
type RestoreMaximumRejected struct{ Rejection HitPointRejection }

// RestoreMaximumInvalidAmount: the amount is negative; nothing changed.
type RestoreMaximumInvalidAmount struct{ Amount int }

func (RestoreMaximumApplied) isRestoreHitPointMaximumResult()       {}
func (RestoreMaximumRejected) isRestoreHitPointMaximumResult()      {}
func (RestoreMaximumInvalidAmount) isRestoreHitPointMaximumResult() {}

// RestoreMaximum ends a reduction of amount to the creature's Hit Point maximum, as
// Greater Restoration does (SRD 5.2.1, "Greater Restoration", p. 137). The maximum
// rises by amount (stopping at the largest int); current Hit Points, Temporary Hit
// Points and the LifeState are unchanged, since ending a reduction restores no Hit Points.
//
// It decides in this order: Dead (RestoreMaximumRejected), a negative amount
// (RestoreMaximumInvalidAmount), an amount of 0 (applied, no event). It returns an
// error matching srderr.ErrInvalidInput for a zero subject or status. It never
// panics and changes none of its inputs.
func RestoreMaximum(subject CombatantID, status HitPointStatus, amount int) (RestoreHitPointMaximumResult, error) {
	if err := status.check(subject); err != nil {
		return nil, err
	}
	switch {
	case status.isDead():
		return RestoreMaximumRejected{CreatureIsDead}, nil
	case amount < 0:
		return RestoreMaximumInvalidAmount{amount}, nil
	case amount == 0:
		return RestoreMaximumApplied{status, nil}, nil
	}
	tr, err := status.RestoreMaximum(subject, satAdd(status.hp.maximum, amount))
	if err != nil {
		return nil, err
	}
	// The creature is not Dead, so RestoreMaximum cannot refuse.
	a := tr.(HitPointTransitionApplied)
	var events []Event
	for _, e := range a.events {
		events = append(events, e)
	}
	return RestoreMaximumApplied{a.status, events}, nil
}
