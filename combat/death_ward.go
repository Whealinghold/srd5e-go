package combat

import "slices"

// DeathWard is the state of a Death Ward on one creature (SRD 5.2.1, "Death Ward",
// p. 122): active until it saves the creature once. The host holds one per warded
// creature and discards it when the spell's 8 hours pass. The zero value is
// WardActive. Comparable and safe for concurrent use.
type DeathWard int

// The states of a Death Ward.
const (
	// WardActive: the ward has not yet saved the creature.
	WardActive DeathWard = iota
	// WardSpent: the ward saved the creature and ended.
	WardSpent
)

// String returns "active", "spent" or "unknown".
func (w DeathWard) String() string {
	if w < WardActive {
		return "unknown"
	}
	return enumName([]string{"active", "spent"}, int(w)+1)
}

// GuardDamageWithDeathWardResult is the closed set of answers of ApplyDamageWithDeathWard
// and NegateInstantKill. Variants: GuardedDamageApplied, GuardedDamageRefused,
// GuardedDamageNotWarded.
type GuardDamageWithDeathWardResult interface{ isGuardDamageWithDeathWardResult() }

// GuardedDamageApplied: the call took effect. Status is the creature after it, Ward the
// ward after it, Resolutions the damage pipeline of the call and Events the events in
// order; a spent ward ends them with DeathWardSpent.
type GuardedDamageApplied struct {
	status      HitPointStatus
	ward        DeathWard
	resolutions []DamageResolution
	events      []Event
}

// Status returns the Hit Point status after the call.
func (r GuardedDamageApplied) Status() HitPointStatus { return r.status }

// Ward returns the Death Ward after the call.
func (r GuardedDamageApplied) Ward() DeathWard { return r.ward }

// Resolutions returns a copy of the damage resolutions, empty for NegateInstantKill.
func (r GuardedDamageApplied) Resolutions() []DamageResolution { return slices.Clone(r.resolutions) }

// Events returns a copy of the events in the order they happened.
func (r GuardedDamageApplied) Events() []Event { return slices.Clone(r.events) }

// GuardedDamageRefused: the creature is Dead, so the call was refused and the ward is
// unchanged. Construct it with a keyed literal.
type GuardedDamageRefused struct{ Rejection HitPointRejection }

// GuardedDamageNotWarded: NegateInstantKill found the ward already spent; nothing changed.
type GuardedDamageNotWarded struct{}

func (GuardedDamageApplied) isGuardDamageWithDeathWardResult()   {}
func (GuardedDamageRefused) isGuardDamageWithDeathWardResult()   {}
func (GuardedDamageNotWarded) isGuardDamageWithDeathWardResult() {}

// ApplyDamageWithDeathWard applies damage as HitPointStatus.ApplyDamage does, unless
// an active ward would be called on (SRD 5.2.1, "Death Ward", p. 122). If the
// damage would drop the creature to 0 Hit Points the ward is spent: the damage is applied
// again as a knock-out, so the creature has 1 Hit Point and stays Alive whatever its
// ZeroHitPointsRule (no Massive Damage, no Death Saving Throw failure), and DeathWardSpent
// replaces KnockedOut as the last event. Damage that leaves the creature above 0, damage
// to a creature already at 0, a knock-out, damage that deals nothing and a spent ward pass
// through unchanged. The ward guards only this operation: a maximum reduced to 0 or an
// Exhaustion death goes through other operations and is not guarded.
//
// A Dead creature gives GuardedDamageRefused. It returns an error matching
// srderr.ErrInvalidInput exactly where ApplyDamage does, and for a ward that is neither
// state. It never panics and changes none of its inputs.
func ApplyDamageWithDeathWard(subject CombatantID, status HitPointStatus, damages []Damage, adjustments []DamageAdjustment,
	delivery DamageDelivery, kind HitKind, unpreventable []Damage, ward DeathWard) (GuardDamageWithDeathWardResult, error) {
	if ward != WardActive && ward != WardSpent {
		return nil, invalidInput("unknown death ward state")
	}
	tr, err := status.ApplyDamage(subject, damages, adjustments, delivery, kind, unpreventable)
	if err != nil {
		return nil, err
	}
	a, ok := tr.(HitPointTransitionApplied)
	if !ok {
		return GuardedDamageRefused{tr.(HitPointTransitionRefused).Rejection}, nil
	}
	if ward == WardActive && has[DroppedToZeroHitPoints](a.events) {
		// The first call succeeded on the same inputs with a delivery that is valid, so the rerun cannot fail.
		rerun, _ := status.ApplyDamage(subject, damages, adjustments, MeleeAttackKnockOut, kind, unpreventable)
		a = rerun.(HitPointTransitionApplied)
		var events []Event
		for _, e := range a.events {
			if _, knocked := e.(KnockedOut); !knocked {
				events = append(events, e)
			}
		}
		events = append(events, DeathWardSpent{restorationEventBase{subject}, DropToZero})
		return GuardedDamageApplied{a.status, WardSpent, a.resolutions, events}, nil
	}
	var events []Event
	for _, e := range a.events {
		events = append(events, e)
	}
	return GuardedDamageApplied{a.status, ward, a.resolutions, events}, nil
}

// NegateInstantKill spends an active ward on an effect the host classifies as one
// that "would kill it instantly without dealing damage" (SRD 5.2.1, "Death Ward",
// p. 122): the status is unchanged, the ward is WardSpent and the event is DeathWardSpent
// with InstantKill. A spent ward gives GuardedDamageNotWarded; a Dead creature gives
// GuardedDamageRefused with the ward unchanged. The package does not decide which
// effects qualify.
//
// It returns an error matching srderr.ErrInvalidInput for a zero subject or status and a
// ward that is neither state. It never panics.
func NegateInstantKill(subject CombatantID, status HitPointStatus, ward DeathWard) (GuardDamageWithDeathWardResult, error) {
	if ward != WardActive && ward != WardSpent {
		return nil, invalidInput("unknown death ward state")
	}
	if err := status.check(subject); err != nil {
		return nil, err
	}
	switch {
	case status.isDead():
		return GuardedDamageRefused{CreatureIsDead}, nil
	case ward == WardSpent:
		return GuardedDamageNotWarded{}, nil
	}
	return GuardedDamageApplied{status, WardSpent, nil, []Event{DeathWardSpent{restorationEventBase{subject}, InstantKill}}}, nil
}
