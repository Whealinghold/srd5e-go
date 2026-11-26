package combat

import "slices"

// EndHitPointMaximumIncreaseResult is the closed set of answers of EndMaximumIncrease.
// Variants: EndMaximumIncreaseApplied, EndMaximumIncreaseNotPresent,
// EndMaximumIncreaseRejected.
type EndHitPointMaximumIncreaseResult interface{ isEndHitPointMaximumIncreaseResult() }

// EndMaximumIncreaseApplied: the entry left the ledger. Events holds, in order,
// HitPointMaximumIncreaseEnded and the events of ReduceMaximum (HitPointMaximumReduced
// and, when the maximum reached 0, Died); none when the effective total did not change.
type EndMaximumIncreaseApplied struct {
	status HitPointStatus
	ledger HitPointMaximumIncreases
	events []Event
}

// Status returns the Hit Point status after the call.
func (r EndMaximumIncreaseApplied) Status() HitPointStatus { return r.status }

// Ledger returns the ledger after the call.
func (r EndMaximumIncreaseApplied) Ledger() HitPointMaximumIncreases { return r.ledger }

// Events returns a copy of the events in the order they happened.
func (r EndMaximumIncreaseApplied) Events() []Event { return slices.Clone(r.events) }

// EndMaximumIncreaseNotPresent: the ledger holds no entry with that effect id;
// status and ledger are unchanged.
type EndMaximumIncreaseNotPresent struct{}

// EndMaximumIncreaseRejected: the creature is Dead; status and ledger are
// unchanged. Construct it with a keyed literal.
type EndMaximumIncreaseRejected struct{ Rejection HitPointRejection }

func (EndMaximumIncreaseApplied) isEndHitPointMaximumIncreaseResult()    {}
func (EndMaximumIncreaseNotPresent) isEndHitPointMaximumIncreaseResult() {}
func (EndMaximumIncreaseRejected) isEndHitPointMaximumIncreaseResult()   {}

// EndMaximumIncrease removes the entry effectID from the ledger and lowers the
// creature's Hit Point maximum by the drop in the effective total, through
// ReduceMaximum (SRD 5.2.1, "Hit Points", p. 183). Current Hit Points fall only to
// the new maximum; no damage is dealt, no Death Saving Throw fails and Temporary Hit
// Points stay. Ending the weaker entry of a spell changes the status not at all. A
// maximum that reaches 0 kills the creature ("Hit Point Maximum of 0", p. 17).
//
// It decides in this order: Dead (EndMaximumIncreaseRejected), an id not in the
// ledger (EndMaximumIncreaseNotPresent). It returns an error matching
// srderr.ErrInvalidInput for a zero subject or status. It never panics and changes
// none of its inputs.
func EndMaximumIncrease(subject CombatantID, status HitPointStatus, ledger HitPointMaximumIncreases, effectID OngoingEffectID) (EndHitPointMaximumIncreaseResult, error) {
	if err := status.check(subject); err != nil {
		return nil, err
	}
	switch {
	case status.isDead():
		return EndMaximumIncreaseRejected{CreatureIsDead}, nil
	case !ledger.find(effectID):
		return EndMaximumIncreaseNotPresent{}, nil
	}
	next := ledger.without(effectID)
	drop := ledger.Total() - next.Total()
	if drop == 0 {
		return EndMaximumIncreaseApplied{status, next, nil}, nil
	}
	tr, err := status.ReduceMaximum(subject, drop)
	if err != nil {
		return nil, err
	}
	// The creature is not Dead, so ReduceMaximum cannot refuse.
	a := tr.(HitPointTransitionApplied)
	lost := status.hp.current - a.status.hp.current
	events := []Event{HitPointMaximumIncreaseEnded{restorationEventBase{subject}, a.status.hp.maximum, drop, lost}}
	for _, e := range a.events {
		events = append(events, e)
	}
	return EndMaximumIncreaseApplied{a.status, next, events}, nil
}
