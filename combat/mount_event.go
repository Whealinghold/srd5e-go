package combat

import "github.com/Whealinghold/srd5e-go/content"

// MountEvent is the closed family of events the mounting use cases produce. Each is an
// immutable value comparable with ==, about the rider and the mount (its Combatants is
// exactly [rider, mount]) with a constant snake_case Kind. Variants: CreatureMounted,
// CreatureDismounted, FellOffMount. Each embeds mountEventBase. No function of this package
// returns a CombatantID; read the creatures from Combatants.
type MountEvent interface {
	Event
	mountEvent()
}

// mountEventBase carries the pair every mount event is about and puts the event in the
// Event and MountEvent families.
type mountEventBase struct{ rider, mount CombatantID }

func (mountEventBase) event()      {}
func (mountEventBase) mountEvent() {}

// Combatants returns a new slice [rider, mount].
func (b mountEventBase) Combatants() []CombatantID { return []CombatantID{b.rider, b.mount} }

// CreatureMounted: the rider mounted the creature, which cost the movement. Its Kind is
// creature_mounted.
type CreatureMounted struct {
	mountEventBase
	control    MountControl
	costFeet   int
	initiative MountInitiativeChange
	turn       MountTurn
}

// Kind returns "creature_mounted".
func (CreatureMounted) Kind() string { return "creature_mounted" }

// Control returns the control of the ride.
func (e CreatureMounted) Control() MountControl { return e.control }

// MovementCostFeet returns the movement the rider spends, which the caller charges.
func (e CreatureMounted) MovementCostFeet() int { return e.costFeet }

// InitiativeChange returns what the ride does to the mount's Initiative, for the host to apply.
func (e CreatureMounted) InitiativeChange() MountInitiativeChange { return e.initiative }

// Turn returns whose turn the mount moves on.
func (e CreatureMounted) Turn() MountTurn { return e.turn }

// CreatureDismounted: the rider dismounted, which cost the movement. Its Kind is
// creature_dismounted.
type CreatureDismounted struct {
	mountEventBase
	control  MountControl
	costFeet int
}

// Kind returns "creature_dismounted".
func (CreatureDismounted) Kind() string { return "creature_dismounted" }

// Control returns the control the ride had.
func (e CreatureDismounted) Control() MountControl { return e.control }

// MovementCostFeet returns the movement the rider spends, which the caller charges.
func (e CreatureDismounted) MovementCostFeet() int { return e.costFeet }

// FellOffMount: the rider failed the Falling Off save and landed Prone. Its Kind is
// fell_off_mount. The event applies nothing: the caller applies the condition it names.
type FellOffMount struct {
	mountEventBase
	control MountControl
	trigger FallingOffTrigger
	landing LandingSpace
}

// Kind returns "fell_off_mount".
func (FellOffMount) Kind() string { return "fell_off_mount" }

// Control returns the control the ride had.
func (e FellOffMount) Control() MountControl { return e.control }

// Trigger returns what called for the save.
func (e FellOffMount) Trigger() FallingOffTrigger { return e.trigger }

// Landing returns the landing space the host reported, unchanged.
func (e FellOffMount) Landing() LandingSpace { return e.landing }

// Condition returns the condition the rider lands with, Prone, for the caller to apply.
func (FellOffMount) Condition() content.ConditionKind { return content.ConditionProne }
