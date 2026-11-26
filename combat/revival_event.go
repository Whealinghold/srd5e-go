package combat

// RevivalEvent is the closed family of events the return to life produces. It is
// an immutable value about one creature with a constant snake_case Kind, past
// tense, whose Combatants is a fresh copy. The zero value is invalid; Revive builds
// it. Variants: CreatureReturnedToLife. Each embeds revivalEventBase.
type RevivalEvent interface {
	Event
	revivalEvent()
}

// revivalEventBase carries the creature an event is about and puts the event in
// the Event and RevivalEvent families.
type revivalEventBase struct{ creature CombatantID }

func (revivalEventBase) event()        {}
func (revivalEventBase) revivalEvent() {}

// Combatants returns a new slice holding the creature.
func (b revivalEventBase) Combatants() []CombatantID { return []CombatantID{b.creature} }

// CreatureReturnedToLife: a Dead creature returned to life with the Hit Points the
// revival effect determined, after capping at its Hit Point maximum (SRD 5.2.1,
// "Dead", p. 180). Its Kind is creature_returned_to_life. Combatants is [creature].
// Comparable with ==.
type CreatureReturnedToLife struct {
	revivalEventBase
	hitPoints int
}

// Kind returns "creature_returned_to_life".
func (CreatureReturnedToLife) Kind() string { return "creature_returned_to_life" }

// HitPoints returns the creature's current Hit Points after the return.
func (e CreatureReturnedToLife) HitPoints() int { return e.hitPoints }
