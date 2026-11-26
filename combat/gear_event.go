package combat

// GearEvent is the closed family of events the gear rules produce. Each is an immutable
// value comparable with ==, with a constant snake_case Kind, past tense. Combatants is a
// fresh copy of the creatures concerned, the deployer first, without duplicates. Variants:
// DeployedGearPlaced, DeployedGearRecovered, PouredOilLit, PouredOilBurnedOut,
// HuntingTrapSprung, HuntingTrapReset, GearHoldStarted, GearHoldRestraintChanged,
// GearHoldEnded. Each embeds gearEventBase. Kind is the event's name, so the kind of gear or
// hold an event is about is read through GearKind or HoldKind.
type GearEvent interface {
	Event
	gearEvent()
}

// gearEventBase carries the creatures an event is about, the first being the deployer (or
// the held creature) and the second, when not zero and not the first, the other creature
// concerned, and puts the event in the Event and GearEvent families.
type gearEventBase struct{ first, second CombatantID }

func (gearEventBase) event()     {}
func (gearEventBase) gearEvent() {}

// Combatants returns a new slice: the first creature and, if it is another one, the second.
func (b gearEventBase) Combatants() []CombatantID {
	if b.second.IsZero() || b.second == b.first {
		return []CombatantID{b.first}
	}
	return []CombatantID{b.first, b.second}
}

// DeployedGearPlaced: gear was set out. Its Kind is deployed_gear_placed.
type DeployedGearPlaced struct {
	gearEventBase
	id   DeployedGearID
	kind DeployedGearKind
}

// Kind returns "deployed_gear_placed".
func (DeployedGearPlaced) Kind() string { return "deployed_gear_placed" }

// GearID returns the id of the gear.
func (e DeployedGearPlaced) GearID() DeployedGearID { return e.id }

// GearKind returns the kind of the gear.
func (e DeployedGearPlaced) GearKind() DeployedGearKind { return e.kind }

// DeployedGearRecovered: Caltrops or Ball Bearings were gathered up. Its Kind is
// deployed_gear_recovered.
type DeployedGearRecovered struct {
	gearEventBase
	id   DeployedGearID
	kind DeployedGearKind
}

// Kind returns "deployed_gear_recovered".
func (DeployedGearRecovered) Kind() string { return "deployed_gear_recovered" }

// GearID returns the id of the gear.
func (e DeployedGearRecovered) GearID() DeployedGearID { return e.id }

// GearKind returns the kind of the gear.
func (e DeployedGearRecovered) GearKind() DeployedGearKind { return e.kind }

// PouredOilLit: poured Oil was lit and burns through the end of a turn. Its Kind is
// poured_oil_lit. Combatants is [deployer, lighter].
type PouredOilLit struct {
	gearEventBase
	id           DeployedGearID
	burnsThrough TurnStamp
}

// Kind returns "poured_oil_lit".
func (PouredOilLit) Kind() string { return "poured_oil_lit" }

// GearID returns the id of the gear.
func (e PouredOilLit) GearID() DeployedGearID { return e.id }

// BurnsThrough returns the end of which turn the oil burns through.
func (e PouredOilLit) BurnsThrough() TurnStamp { return e.burnsThrough }

// PouredOilBurnedOut: burning Oil burned out and is gone. Its Kind is poured_oil_burned_out.
// Combatants is [deployer, lighter].
type PouredOilBurnedOut struct {
	gearEventBase
	id DeployedGearID
}

// Kind returns "poured_oil_burned_out".
func (PouredOilBurnedOut) Kind() string { return "poured_oil_burned_out" }

// GearID returns the id of the gear.
func (e PouredOilBurnedOut) GearID() DeployedGearID { return e.id }

// HuntingTrapSprung: a creature stepped on a Hunting Trap's plate and it snapped shut.
// Its Kind is hunting_trap_sprung. Combatants is [deployer, creature].
type HuntingTrapSprung struct {
	gearEventBase
	id DeployedGearID
}

// Kind returns "hunting_trap_sprung".
func (HuntingTrapSprung) Kind() string { return "hunting_trap_sprung" }

// GearID returns the id of the gear.
func (e HuntingTrapSprung) GearID() DeployedGearID { return e.id }

// HuntingTrapReset: a sprung, empty Hunting Trap was set again. Its Kind is
// hunting_trap_reset.
type HuntingTrapReset struct {
	gearEventBase
	id DeployedGearID
}

// Kind returns "hunting_trap_reset".
func (HuntingTrapReset) Kind() string { return "hunting_trap_reset" }

// GearID returns the id of the gear.
func (e HuntingTrapReset) GearID() DeployedGearID { return e.id }

// GearHoldStarted: a hold began on a creature. Its Kind is gear_hold_started. Combatants is
// [held creature].
type GearHoldStarted struct {
	gearEventBase
	id   HoldID
	kind HoldKind
}

// Kind returns "gear_hold_started".
func (GearHoldStarted) Kind() string { return "gear_hold_started" }

// HoldID returns the id of the hold.
func (e GearHoldStarted) HoldID() HoldID { return e.id }

// HoldKind returns the kind of the hold.
func (e GearHoldStarted) HoldKind() HoldKind { return e.kind }

// GearHoldRestraintChanged: a Manacles hold was attached to, or taken off, a fixed chain
// or hook. Its Kind is gear_hold_restraint_changed. Combatants is [held creature].
type GearHoldRestraintChanged struct {
	gearEventBase
	id        HoldID
	restraint HoldRestraint
}

// Kind returns "gear_hold_restraint_changed".
func (GearHoldRestraintChanged) Kind() string { return "gear_hold_restraint_changed" }

// HoldID returns the id of the hold.
func (e GearHoldRestraintChanged) HoldID() HoldID { return e.id }

// Restraint returns the restraint the hold now has.
func (e GearHoldRestraintChanged) Restraint() HoldRestraint { return e.restraint }

// GearHoldEnded: a hold ended. Its Kind is gear_hold_ended. Combatants is [held creature].
type GearHoldEnded struct {
	gearEventBase
	id    HoldID
	kind  HoldKind
	cause HoldEndCause
}

// Kind returns "gear_hold_ended".
func (GearHoldEnded) Kind() string { return "gear_hold_ended" }

// HoldID returns the id of the hold.
func (e GearHoldEnded) HoldID() HoldID { return e.id }

// HoldKind returns the kind of the hold that ended.
func (e GearHoldEnded) HoldKind() HoldKind { return e.kind }

// Cause returns why the hold ended.
func (e GearHoldEnded) Cause() HoldEndCause { return e.cause }
