package combat

// ResourceState says whether a per-turn resource is still available. The zero value
// is invalid.
type ResourceState int

// The two states.
const (
	ResourceAvailable ResourceState = iota + 1
	ResourceSpent
)

// String returns "available", "spent" or "unknown".
func (s ResourceState) String() string { return enumName([]string{"available", "spent"}, int(s)) }

// DisengageState says whether the combatant took Disengage this turn. The zero value
// is invalid.
type DisengageState int

// The two states.
const (
	// Disengaged: the combatant took Disengage (an action, a Bonus Action or a feature) this turn.
	Disengaged DisengageState = iota + 1
	// NotDisengaged: it did not.
	NotDisengaged
)

// String returns "disengaged", "not_disengaged" or "unknown".
func (s DisengageState) String() string {
	return enumName([]string{"disengaged", "not_disengaged"}, int(s))
}

// TurnResources are what the active combatant has spent this turn: the feet moved, the
// Dash extra, the action, the Bonus Action and the free object interaction, and
// whether it Disengaged. The movement left on a speed is derived from these and the
// profile, never stored (initiative-and-actions.md FR-115). Comparable with ==;
// immutable. The zero value is invalid.
type TurnResources struct {
	moved, dash                int
	action, bonus, interaction ResourceState
	disengage                  DisengageState
}

// freshTurnResources are the resources at the start of a turn: nothing moved, no Dash
// extra, the action and the interaction available, no Bonus Action to take and not
// Disengaged (FR-034). The Bonus Action is "available" here and is gated by the grant.
func freshTurnResources() TurnResources {
	return TurnResources{0, 0, ResourceAvailable, ResourceAvailable, ResourceAvailable, NotDisengaged}
}

// MovedFeet returns the feet moved this turn.
func (r TurnResources) MovedFeet() int { return r.moved }

// DashExtraFeet returns the movement Dash added this turn.
func (r TurnResources) DashExtraFeet() int { return r.dash }

// Action returns the state of the turn's action.
func (r TurnResources) Action() ResourceState { return r.action }

// BonusAction returns the state of the turn's Bonus Action.
func (r TurnResources) BonusAction() ResourceState { return r.bonus }

// Interaction returns the state of the free object interaction.
func (r TurnResources) Interaction() ResourceState { return r.interaction }

// Disengage returns whether the combatant took Disengage this turn.
func (r TurnResources) Disengage() DisengageState { return r.disengage }
