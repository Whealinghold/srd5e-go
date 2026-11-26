package combat

import "slices"

// DestinationSpace is what the host reports about the space a teleporter aims at.
// The engine has no map. The zero value is invalid.
type DestinationSpace int

// The facts about a destination space.
const (
	// Unoccupied: the space is unoccupied (SRD 5.2.1, "Unoccupied Space", p. 191).
	Unoccupied DestinationSpace = iota + 1
	// OccupiedByCreature: a creature occupies the space (p. 185).
	OccupiedByCreature
	// BlockedBySolidObstacle: a solid obstacle fills the space.
	BlockedBySolidObstacle
)

// String returns "unoccupied", "occupied_by_creature", "blocked_by_solid_obstacle" or "unknown".
func (d DestinationSpace) String() string {
	return enumName([]string{"unoccupied", "occupied_by_creature", "blocked_by_solid_obstacle"}, int(d))
}

// DestinationSight says whether the teleportation effect requires the teleporter to
// see the destination. The zero value is invalid.
type DestinationSight int

// The sight requirements of a teleportation effect.
const (
	// DestinationSightRequired: the teleporter must see the destination.
	DestinationSightRequired DestinationSight = iota + 1
	// DestinationSightNotRequired: sight is not required.
	DestinationSightNotRequired
)

// String returns "required", "not_required" or "unknown".
func (s DestinationSight) String() string {
	return enumName([]string{"required", "not_required"}, int(s))
}

// TouchedCreaturePolicy says whether the creatures the teleporter touches go along.
// The zero value is StayBehind, the rule's default.
type TouchedCreaturePolicy int

// The policies for touched creatures.
const (
	// StayBehind: a touched creature does not teleport (the default; SRD 5.2.1, "Teleportation", p. 190).
	StayBehind TouchedCreaturePolicy = iota
	// Accompany: the teleportation effect says touched creatures go along.
	Accompany
)

// String returns "stay_behind", "accompany" or "unknown".
func (p TouchedCreaturePolicy) String() string {
	switch p {
	case StayBehind:
		return "stay_behind"
	case Accompany:
		return "accompany"
	}
	return "unknown"
}

// TeleportFields are the inputs of NewTeleportCommand. Construct with a keyed literal.
type TeleportFields struct {
	// Teleporter is the creature that teleports.
	Teleporter CombatantID
	// Destination is what the host reports about the destination space.
	Destination DestinationSpace
	// Sight is whether the effect requires the teleporter to see the destination.
	Sight DestinationSight
	// DestinationSeen is whether the teleporter sees the destination.
	DestinationSeen bool
	// MovementCostFeet is the movement the effect states, 0 when it states none.
	MovementCostFeet int
	// Policy says whether touched creatures go along; the zero value is StayBehind.
	Policy TouchedCreaturePolicy
	// Touched are the creatures the teleporter touches, in order.
	Touched []CombatantID
}

// TeleportCommand is one teleportation: the facts the host supplies. Build it with
// NewTeleportCommand; the zero value is invalid. It holds a slice, so it is not
// comparable with ==.
type TeleportCommand struct {
	f  TeleportFields
	ok bool
}

// NewTeleportCommand returns the command. The touched list is copied. It returns an
// error matching srderr.ErrInvalidInput for a zero teleporter, an undefined
// destination, sight or policy, and a negative movement cost (a cost the effect
// states, SRD 5.2.1, "Teleportation", p. 190). A touched list holding the teleporter
// or a repeated id is not an error here: Resolve answers TeleportInvalidTouchedSet. More than 256 touched creatures is a LimitExceededError (plan P10), checked first.
// It never panics.
func NewTeleportCommand(f TeleportFields) (TeleportCommand, error) {
	if err := overCap("touched creatures", len(f.Touched), maxInputList); err != nil {
		return TeleportCommand{}, err
	}
	switch {
	case f.Teleporter.IsZero():
		return TeleportCommand{}, invalidInput("a teleportation needs a teleporter")
	case f.Destination < Unoccupied || f.Destination > BlockedBySolidObstacle:
		return TeleportCommand{}, invalidInput("a teleportation needs a destination")
	case f.Sight != DestinationSightRequired && f.Sight != DestinationSightNotRequired:
		return TeleportCommand{}, invalidInput("a teleportation needs a sight requirement")
	case f.Policy != StayBehind && f.Policy != Accompany:
		return TeleportCommand{}, invalidInput("undefined touched creature policy")
	case f.MovementCostFeet < 0:
		return TeleportCommand{}, invalidInput("a movement cost cannot be negative")
	}
	f.Touched = slices.Clone(f.Touched)
	return TeleportCommand{f, true}, nil
}

// TeleportCombatantResult is the closed set of answers of Teleportation.Resolve. Variants:
// TeleportArrivedAtDestination, TeleportArrivedNearDestination,
// TeleportDestinationNotSeen, TeleportInvalidTouchedSet.
type TeleportCombatantResult interface{ isTeleportCombatantResult() }

// teleportArrival is the part of an arrival result: who went along, the movement cost
// and the event.
type teleportArrival struct {
	teleporting []CombatantID
	costFeet    int
	events      []MagicalEffectEvent
}

// TeleportingCreatures returns a copy of the touched creatures that teleport (never
// the teleporter): all of them in the order given with Accompany, none with StayBehind.
func (a teleportArrival) TeleportingCreatures() []CombatantID { return slices.Clone(a.teleporting) }

// MovementCostFeet returns the movement the teleportation costs, 0 unless the effect states a cost.
func (a teleportArrival) MovementCostFeet() int { return a.costFeet }

// Events returns a copy of the events: one CreatureTeleported.
func (a teleportArrival) Events() []MagicalEffectEvent { return cloneMagicalEvents(a.events) }

// TeleportArrivedAtDestination: the teleporter arrived at the destination, an
// unoccupied space.
type TeleportArrivedAtDestination struct{ teleportArrival }

// TeleportArrivedNearDestination: the destination is occupied or blocked, so the
// teleporter arrives at the nearest unoccupied space of its choice. The host supplies
// that space; the engine has no map.
type TeleportArrivedNearDestination struct{ teleportArrival }

// TeleportDestinationNotSeen: the effect requires the teleporter to see the
// destination and it does not; no creature moves and there is no event.
type TeleportDestinationNotSeen struct{}

// TeleportInvalidTouchedSet: the touched list holds the teleporter or a repeated
// id; no creature moves and there is no event.
type TeleportInvalidTouchedSet struct{ offending []CombatantID }

// Offending returns a copy of the offending ids, each once, in the order they first
// offended.
func (r TeleportInvalidTouchedSet) Offending() []CombatantID { return slices.Clone(r.offending) }

func (TeleportArrivedAtDestination) isTeleportCombatantResult()   {}
func (TeleportArrivedNearDestination) isTeleportCombatantResult() {}
func (TeleportDestinationNotSeen) isTeleportCombatantResult()     {}
func (TeleportInvalidTouchedSet) isTeleportCombatantResult()      {}

// Teleportation is the stateless service for Teleportation. The zero value is ready
// to use. Safe for concurrent use.
type Teleportation struct{}

// Resolve resolves a teleportation (SRD 5.2.1, "Teleportation", p. 190). It decides
// in this order: a touched set holding the teleporter or a repeated id is
// TeleportInvalidTouchedSet; a destination the effect requires the teleporter to see
// and it does not is TeleportDestinationNotSeen, before placement is considered; an
// unoccupied destination is TeleportArrivedAtDestination; an occupied or blocked one
// is TeleportArrivedNearDestination. An arrival emits one CreatureTeleported.
// Teleporting costs no movement unless the command states a cost.
//
// It returns an error matching srderr.ErrInvalidInput, with the nil result, for a
// command not built by NewTeleportCommand. It does not change its argument and never
// panics.
func (Teleportation) Resolve(c TeleportCommand) (TeleportCombatantResult, error) {
	if !c.ok {
		return nil, invalidInput("a teleportation needs a command from NewTeleportCommand")
	}
	f := c.f
	var offending []CombatantID
	for i, id := range f.Touched {
		if (id == f.Teleporter || slices.Contains(f.Touched[:i], id)) && !slices.Contains(offending, id) {
			offending = append(offending, id)
		}
	}
	switch {
	case len(offending) > 0:
		return TeleportInvalidTouchedSet{offending}, nil
	case f.Sight == DestinationSightRequired && !f.DestinationSeen:
		return TeleportDestinationNotSeen{}, nil
	}
	var going []CombatantID
	if f.Policy == Accompany {
		going = slices.Clone(f.Touched)
	}
	arrival := ArrivalAtDestination
	if f.Destination != Unoccupied {
		arrival = ArrivalNearDestination
	}
	a := teleportArrival{going, f.MovementCostFeet, []MagicalEffectEvent{CreatureTeleported{magicalEffectEventBase{f.Teleporter}, going, arrival}}}
	if arrival == ArrivalAtDestination {
		return TeleportArrivedAtDestination{a}, nil
	}
	return TeleportArrivedNearDestination{a}, nil
}
