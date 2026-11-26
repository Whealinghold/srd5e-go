package combat

import "slices"

// IncreaseHitPointMaximumResult is the closed set of answers of IncreaseMaximum. Variants:
// IncreaseMaximumApplied, IncreaseMaximumRejected, IncreaseMaximumDuplicate,
// IncreaseMaximumLimitExceeded.
type IncreaseHitPointMaximumResult interface{ isIncreaseHitPointMaximumResult() }

// IncreaseMaximumApplied: the increase joined the ledger. Status is the creature after
// the call, Ledger the ledger with the new entry. Events holds, in order,
// HitPointMaximumIncreased and the events of the restoring (HitPointsRestored and
// Recovered); none when the effective total did not change.
type IncreaseMaximumApplied struct {
	status HitPointStatus
	ledger HitPointMaximumIncreases
	events []Event
}

// Status returns the Hit Point status after the call.
func (r IncreaseMaximumApplied) Status() HitPointStatus { return r.status }

// Ledger returns the ledger after the call.
func (r IncreaseMaximumApplied) Ledger() HitPointMaximumIncreases { return r.ledger }

// Events returns a copy of the events in the order they happened.
func (r IncreaseMaximumApplied) Events() []Event { return slices.Clone(r.events) }

// IncreaseMaximumRejected: the creature is Dead; status and ledger are unchanged.
// Construct it with a keyed literal.
type IncreaseMaximumRejected struct{ Rejection HitPointRejection }

// IncreaseMaximumDuplicate: the ledger already holds an increase with this effect
// id; nothing changed.
type IncreaseMaximumDuplicate struct{}

// IncreaseMaximumLimitExceeded: the ledger already holds 64 increases, the cap of
// active effects per creature (plan P10, result value limit_exceeded); nothing changed.
type IncreaseMaximumLimitExceeded struct{}

func (IncreaseMaximumApplied) isIncreaseHitPointMaximumResult()       {}
func (IncreaseMaximumRejected) isIncreaseHitPointMaximumResult()      {}
func (IncreaseMaximumDuplicate) isIncreaseHitPointMaximumResult()     {}
func (IncreaseMaximumLimitExceeded) isIncreaseHitPointMaximumResult() {}

// IncreaseMaximum adds increase to the ledger of the creature subject and raises its
// Hit Point maximum and current Hit Points by the change in the effective total
// (SRD 5.2.1, "Aid", p. 107; "Heroes' Feast", p. 140; "Combining Spell Effects",
// p. 106). Current Hit Points rise through Restore, so a creature at 0 Hit Points
// becomes Alive with Recovered. Temporary Hit Points are kept. A weaker or equal
// increase of a spell already held joins the ledger and changes nothing else.
//
// It decides in this order: Dead (IncreaseMaximumRejected), an effect id already in
// the ledger (IncreaseMaximumDuplicate), a full ledger (IncreaseMaximumLimitExceeded).
// It returns an error matching srderr.ErrInvalidInput for a zero subject, status or
// increase. It never panics and changes none of its inputs.
func IncreaseMaximum(subject CombatantID, status HitPointStatus, ledger HitPointMaximumIncreases, increase HitPointMaximumIncrease) (IncreaseHitPointMaximumResult, error) {
	if err := status.check(subject); err != nil {
		return nil, err
	}
	if increase.IsZero() {
		return nil, invalidInput("increase is the zero value")
	}
	switch {
	case status.isDead():
		return IncreaseMaximumRejected{CreatureIsDead}, nil
	case ledger.find(increase.id):
		return IncreaseMaximumDuplicate{}, nil
	case ledger.Len() >= maxMaximumIncreases:
		return IncreaseMaximumLimitExceeded{}, nil
	}
	next := ledger.with(increase)
	change := next.Total() - ledger.Total()
	if change == 0 {
		return IncreaseMaximumApplied{status, next, nil}, nil
	}
	raised := status.hp
	raised.maximum = satAdd(raised.maximum, change)
	tr, err := status.with(raised, status.life).Restore(subject, change, nil)
	if err != nil {
		return nil, err
	}
	// The creature is not Dead, so Restore cannot refuse.
	a := tr.(HitPointTransitionApplied)
	events := []Event{HitPointMaximumIncreased{restorationEventBase{subject}, raised.maximum, change}}
	for _, e := range a.events {
		events = append(events, e)
	}
	return IncreaseMaximumApplied{a.status, next, events}, nil
}
