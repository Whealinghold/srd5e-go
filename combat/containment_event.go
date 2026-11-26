package combat

// ContainmentEvent is the closed family of events the containment rules produce. Each is an immutable
// value comparable with ==, about an occupant and its container (its Combatants is exactly
// [occupant, container]) with a constant snake_case Kind, past tense. Variants: OccupantContained,
// OccupantReleased, ContainerTrapped. Each embeds containmentEventBase. There are no accessors for the
// two ids, because no function of this package returns a CombatantID; read them from Combatants.
type ContainmentEvent interface {
	Event
	containmentEvent()
}

// containmentEventBase carries the two creatures every containment event is about and puts the event in
// the Event and ContainmentEvent families.
type containmentEventBase struct{ occupant, container CombatantID }

func (containmentEventBase) event()            {}
func (containmentEventBase) containmentEvent() {}

// Combatants returns a new slice [occupant, container].
func (b containmentEventBase) Combatants() []CombatantID {
	return []CombatantID{b.occupant, b.container}
}

// OccupantContained: a creature is now held by a container in the mode. Its Kind is occupant_contained.
type OccupantContained struct {
	containmentEventBase
	mode ContainmentMode
}

// Kind returns "occupant_contained".
func (OccupantContained) Kind() string { return "occupant_contained" }

// Mode returns how the creature is held.
func (e OccupantContained) Mode() ContainmentMode { return e.mode }

// OccupantReleased: a creature stopped being held, for the cause. Its Kind is occupant_released.
type OccupantReleased struct {
	containmentEventBase
	cause ContainmentLeaveCause
}

// Kind returns "occupant_released".
func (OccupantReleased) Kind() string { return "occupant_released" }

// Cause returns how the creature stopped being held.
func (e OccupantReleased) Cause() ContainmentLeaveCause { return e.cause }

// ContainerTrapped: the container died and the creature stays inside until it has used feet of movement
// to climb out. Its Kind is container_trapped.
type ContainerTrapped struct {
	containmentEventBase
	feet int
}

// Kind returns "container_trapped".
func (ContainerTrapped) Kind() string { return "container_trapped" }

// Feet returns the movement the creature must spend to climb out.
func (e ContainerTrapped) Feet() int { return e.feet }
