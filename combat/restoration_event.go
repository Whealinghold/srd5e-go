package combat

// RestorationEvent is the closed family of events the revival and restoration
// spells produce on their own: HitPointMaximumIncreased, HitPointMaximumIncreaseEnded
// and DeathWardSpent. Each is an immutable value about one creature with a constant
// snake_case Kind, past tense, whose Combatants is a fresh copy. The zero value is
// invalid. (HitPointMaximumRestored, which RestoreMaximum emits, belongs to
// HitPointEvent.) Each embeds restorationEventBase.
type RestorationEvent interface {
	Event
	restorationEvent()
}

// restorationEventBase carries the creature an event is about and puts the event in
// the Event and RestorationEvent families.
type restorationEventBase struct{ creature CombatantID }

func (restorationEventBase) event()            {}
func (restorationEventBase) restorationEvent() {}

// Combatants returns a new slice holding the creature.
func (b restorationEventBase) Combatants() []CombatantID { return []CombatantID{b.creature} }

// HitPointMaximumIncreased: a spell raised a creature's Hit Point maximum (SRD 5.2.1,
// "Aid", p. 107; "Heroes' Feast", p. 140). Its Kind is hit_point_maximum_increased.
// Combatants is [creature]. Comparable with ==.
type HitPointMaximumIncreased struct {
	restorationEventBase
	maximum, increase int
}

// Kind returns "hit_point_maximum_increased".
func (HitPointMaximumIncreased) Kind() string { return "hit_point_maximum_increased" }

// Maximum returns the Hit Point maximum after the increase.
func (e HitPointMaximumIncreased) Maximum() int { return e.maximum }

// Increase returns how much the maximum rose.
func (e HitPointMaximumIncreased) Increase() int { return e.increase }

// HitPointMaximumIncreaseEnded: a spell's increase to a creature's Hit Point maximum
// ended and the maximum fell. It is not damage. Its Kind is
// hit_point_maximum_increase_ended. Combatants is [creature]. Comparable with ==.
type HitPointMaximumIncreaseEnded struct {
	restorationEventBase
	maximum, drop, hitPointsLost int
}

// Kind returns "hit_point_maximum_increase_ended".
func (HitPointMaximumIncreaseEnded) Kind() string { return "hit_point_maximum_increase_ended" }

// Maximum returns the Hit Point maximum after the increase ended.
func (e HitPointMaximumIncreaseEnded) Maximum() int { return e.maximum }

// Drop returns how much the maximum fell.
func (e HitPointMaximumIncreaseEnded) Drop() int { return e.drop }

// HitPointsLost returns the current Hit Points removed to fit the new maximum, 0 when none.
func (e HitPointMaximumIncreaseEnded) HitPointsLost() int { return e.hitPointsLost }

// DeathWardTrigger is why Death Ward was spent (SRD 5.2.1, "Death Ward", p. 122).
// The zero value is not a trigger.
type DeathWardTrigger int

// The triggers of Death Ward.
const (
	// DropToZero: damage would have dropped the creature to 0 Hit Points.
	DropToZero DeathWardTrigger = iota + 1
	// InstantKill: an effect would have killed the creature instantly without damage.
	InstantKill
)

// String returns "drop_to_zero", "instant_kill" or "unknown".
func (t DeathWardTrigger) String() string {
	return enumName([]string{"drop_to_zero", "instant_kill"}, int(t))
}

// DeathWardSpent: Death Ward saved a creature and ended. Its Kind is
// death_ward_spent. Combatants is [creature]. Comparable with ==.
type DeathWardSpent struct {
	restorationEventBase
	reason DeathWardTrigger
}

// Kind returns "death_ward_spent".
func (DeathWardSpent) Kind() string { return "death_ward_spent" }

// Reason returns what the ward saved the creature from.
func (e DeathWardSpent) Reason() DeathWardTrigger { return e.reason }
