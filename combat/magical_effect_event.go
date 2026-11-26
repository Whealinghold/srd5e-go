package combat

import "slices"

// MagicalEffectEvent is the closed family of events the magical-effect rules
// produce. Each is an immutable value with a constant snake_case Kind, past tense,
// whose Combatants is a fresh copy. The zero value of each variant is invalid; the
// operations build them. Variants: CurseRemoved, ShapeShifted, RevertedToTrueForm,
// CreatureTeleported. Each embeds magicalEffectEventBase.
type MagicalEffectEvent interface {
	Event
	magicalEffectEvent()
}

// magicalEffectEventBase carries the creature an event is about and puts the event
// in the Event and MagicalEffectEvent families.
type magicalEffectEventBase struct{ creature CombatantID }

func (magicalEffectEventBase) event()              {}
func (magicalEffectEventBase) magicalEffectEvent() {}

// Combatants returns a new slice holding the creature.
func (b magicalEffectEventBase) Combatants() []CombatantID { return []CombatantID{b.creature} }

func cloneMagicalEvents(e []MagicalEffectEvent) []MagicalEffectEvent { return slices.Clone(e) }

// CurseRemoved: a curse ended on the creature. Its Kind is curse_removed.
// Combatants is [creature].
type CurseRemoved struct {
	magicalEffectEventBase
	effect OngoingEffectID
	means  CurseRemovalMeans
}

// Kind returns "curse_removed".
func (CurseRemoved) Kind() string { return "curse_removed" }

// Effect returns the id of the removed curse.
func (e CurseRemoved) Effect() OngoingEffectID { return e.effect }

// Means returns what removed the curse.
func (e CurseRemoved) Means() CurseRemovalMeans { return e.means }

// ShapeShifted: the creature shifted shape by the effect. Its Kind is
// shape_shifted. Combatants is [creature].
type ShapeShifted struct {
	magicalEffectEventBase
	effect EffectSource
}

// Kind returns "shape_shifted".
func (ShapeShifted) Kind() string { return "shape_shifted" }

// Effect returns the effect that shifted the creature.
func (e ShapeShifted) Effect() EffectSource { return e.effect }

// RevertCause is why a creature returned to its true form. The zero value is not a
// cause.
type RevertCause int

// The causes of a return to true form.
const (
	// RevertCauseEnded: the effect ended or the creature reverted.
	RevertCauseEnded RevertCause = iota + 1
	// RevertCauseDied: the creature died (SRD 5.2.1, "Shape-Shifting", p. 187).
	RevertCauseDied
)

// String returns "ended", "died" or "unknown".
func (c RevertCause) String() string { return enumName([]string{"ended", "died"}, int(c)) }

// RevertedToTrueForm: the creature returned to its true form. Its Kind is
// reverted_to_true_form. Combatants is [creature].
type RevertedToTrueForm struct {
	magicalEffectEventBase
	cause RevertCause
}

// Kind returns "reverted_to_true_form".
func (RevertedToTrueForm) Kind() string { return "reverted_to_true_form" }

// Cause returns why the creature reverted.
func (e RevertedToTrueForm) Cause() RevertCause { return e.cause }

// TeleportArrival is where a teleporter arrived: at the destination or near it. The
// zero value is not an arrival.
type TeleportArrival int

// The arrivals of a teleportation.
const (
	// ArrivalAtDestination: the teleporter arrived at the destination.
	ArrivalAtDestination TeleportArrival = iota + 1
	// ArrivalNearDestination: the teleporter arrived at the nearest unoccupied space of its choice.
	ArrivalNearDestination
)

// String returns "at_destination", "near_destination" or "unknown".
func (a TeleportArrival) String() string {
	return enumName([]string{"at_destination", "near_destination"}, int(a))
}

// CreatureTeleported: a teleporter arrived, with the touched creatures that went
// with it. Its Kind is creature_teleported. Combatants is the teleporter, then the
// accompanying creatures in order. It holds a slice, so it is not comparable with ==.
type CreatureTeleported struct {
	magicalEffectEventBase
	accompanying []CombatantID
	arrival      TeleportArrival
}

// Kind returns "creature_teleported".
func (CreatureTeleported) Kind() string { return "creature_teleported" }

// Combatants returns a new slice: the teleporter, then the accompanying creatures.
func (e CreatureTeleported) Combatants() []CombatantID {
	return append([]CombatantID{e.creature}, e.accompanying...)
}

// Accompanying returns a copy of the creatures that teleported with the teleporter.
func (e CreatureTeleported) Accompanying() []CombatantID { return slices.Clone(e.accompanying) }

// Arrival returns whether the teleporter arrived at or near the destination.
func (e CreatureTeleported) Arrival() TeleportArrival { return e.arrival }
