package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ContainmentRejection says why the rules refuse a containment operation in the current state (SRD 5.2.1,
// "Gelatinous Cube", p. 287; "Purple Worm", p. 317; "Tarrasque", p. 332). It is the reason held by every
// ...Rejected result of the containment use cases. The zero value is not a rejection. Comparable and safe
// for concurrent use.
type ContainmentRejection int

// The rejections. The first five are checked, in this order, by every entry.
const (
	// ContainmentRejectionContainerIsContained: the container is itself held, and nested containment is not modelled.
	ContainmentRejectionContainerIsContained ContainmentRejection = iota + 1
	// ContainmentRejectionOccupantIsContainer: the occupant holds creatures of its own.
	ContainmentRejectionOccupantIsContainer
	// ContainmentRejectionOccupantAlreadyContained: the occupant is already held.
	ContainmentRejectionOccupantAlreadyContained
	// ContainmentRejectionNotGrappledByContainer: a Swallow needs the container to hold the target in a grapple.
	ContainmentRejectionNotGrappledByContainer
	// ContainmentRejectionOccupantGrappledElsewhere: an Engulf that grapples needs a creature that is not grappled.
	ContainmentRejectionOccupantGrappledElsewhere
	// ContainmentRejectionTooLarge: the creature is larger than the container can hold.
	ContainmentRejectionTooLarge
	// ContainmentRejectionNoRoom: the container's capacity is used up.
	ContainmentRejectionNoRoom
	// ContainmentRejectionNotContained: the creature is not held (or, for a damage report, not held alive in the container).
	ContainmentRejectionNotContained
	// ContainmentRejectionNoWayOut: the terms give the method no DC.
	ContainmentRejectionNoWayOut
	// ContainmentRejectionPullerIsContained: a creature in this container cannot pull.
	ContainmentRejectionPullerIsContained
	// ContainmentRejectionPullerOutOfReach: the puller is not within 5 feet.
	ContainmentRejectionPullerOutOfReach
	// ContainmentRejectionNoAction: the creature that acts cannot take actions.
	ContainmentRejectionNoAction
	// ContainmentRejectionNotTrapped: the creature is not trapped in a dead container.
	ContainmentRejectionNotTrapped
	// ContainmentRejectionInsufficientMovement: the creature has too little movement to climb out.
	ContainmentRejectionInsufficientMovement
)

// String returns the snake_case name, the srd5e.combat.result of a refused containment result, or "unknown".
func (r ContainmentRejection) String() string {
	return enumName([]string{"container_is_contained", "occupant_is_container", "occupant_already_contained",
		"not_grappled_by_container", "occupant_grappled_elsewhere", "too_large", "no_room", "not_contained",
		"no_way_out", "puller_is_contained", "puller_out_of_reach", "no_action", "not_trapped", "insufficient_movement"}, int(r))
}

// ContainmentProblem says why host input to the containment rules is unusable. It is the reason held by a
// ContainmentError and by every ...Invalid result of the containment use cases. The zero value is not a
// problem. Comparable and safe for concurrent use.
type ContainmentProblem int

// The problems.
const (
	// ContainmentProblemSameContainerAndOccupant: the container and the occupant are one creature.
	ContainmentProblemSameContainerAndOccupant ContainmentProblem = iota + 1
	// ContainmentProblemCircumstanceNotApplicable: the entry circumstance applies to the Engulfed mode only.
	ContainmentProblemCircumstanceNotApplicable
	// ContainmentProblemInvalidTerms: a value of the terms is unusable (the problem of a constructor error).
	ContainmentProblemInvalidTerms
	// ContainmentProblemMissingOccupantConditions: the map of conditions lacks an occupant the call affects.
	ContainmentProblemMissingOccupantConditions
	// ContainmentProblemUnknownOccupant: the map of conditions names a creature the call does not affect.
	ContainmentProblemUnknownOccupant
	// ContainmentProblemLimitExceeded: an entry would take the set past its cap of 256 records (plan P10).
	ContainmentProblemLimitExceeded
)

// String returns the snake_case name, the srd5e.combat.result of an invalid containment result, or "unknown".
func (p ContainmentProblem) String() string {
	return enumName([]string{"same_container_and_occupant", "circumstance_not_applicable", "invalid_terms",
		"missing_occupant_conditions", "unknown_occupant", valueResultLimitExceeded}, int(p))
}

// ContainmentError reports unusable containment input. It matches srderr.ErrInvalidInput with errors.Is
// and carries no display text.
type ContainmentError struct{ Problem ContainmentProblem }

// Error renders the error. The text is not a contract.
func (e *ContainmentError) Error() string {
	return "combat: invalid containment input: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ContainmentError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ContainmentError with the same Problem.
func (e *ContainmentError) Is(target error) bool {
	t, ok := target.(*ContainmentError)
	return ok && t != nil && e != nil && *t == *e
}

// ContainmentLeaveCause is how a creature stopped being held, one of exactly eight (SRD 5.2.1, pp. 263,
// 287, 317, 323, 332, 351). The zero value is invalid. Comparable and safe for concurrent use.
type ContainmentLeaveCause int

// The eight causes.
const (
	// ContainmentLeaveCauseEscaped: the creature escaped on its own.
	ContainmentLeaveCauseEscaped ContainmentLeaveCause = iota + 1
	// ContainmentLeaveCausePulledOut: another creature pulled it out.
	ContainmentLeaveCausePulledOut
	// ContainmentLeaveCauseRegurgitated: the container regurgitated it.
	ContainmentLeaveCauseRegurgitated
	// ContainmentLeaveCauseDisgorged: the container disgorged it after its first damage.
	ContainmentLeaveCauseDisgorged
	// ContainmentLeaveCauseContainerDied: the container died and the creature was released.
	ContainmentLeaveCauseContainerDied
	// ContainmentLeaveCauseEscapedCorpse: it climbed out of the dead container.
	ContainmentLeaveCauseEscapedCorpse
	// ContainmentLeaveCauseGrappleEnded: the grapple that held it ended.
	ContainmentLeaveCauseGrappleEnded
	// ContainmentLeaveCauseOccupantRemoved: the host removed it (dead, teleported or gone).
	ContainmentLeaveCauseOccupantRemoved
)

// String returns the snake_case name, the value of srd5e.combat.containment.end_reason, or "unknown".
func (c ContainmentLeaveCause) String() string {
	return enumName([]string{"escaped", "pulled_out", "regurgitated", "disgorged", "container_died", "escaped_corpse",
		"grapple_ended", "occupant_removed"}, int(c))
}

// Valid reports whether c is one of the eight causes.
func (c ContainmentLeaveCause) Valid() bool {
	return c >= ContainmentLeaveCauseEscaped && c <= ContainmentLeaveCauseOccupantRemoved
}

// leavesProne says whether a creature leaves with the Prone condition (SRD 5.2.1, "Giant Frog", p. 351;
// "Tarrasque", p. 332).
func (c ContainmentLeaveCause) leavesProne() bool {
	return c == ContainmentLeaveCauseRegurgitated || c == ContainmentLeaveCauseDisgorged || c == ContainmentLeaveCauseEscapedCorpse
}

// DamagedOnlyReason is why a failed entry save dealt its damage without holding the creature.
type DamagedOnlyReason int

// The three reasons.
const (
	// DamagedOnlyReasonTooLarge: the creature is above the maximum size.
	DamagedOnlyReasonTooLarge DamagedOnlyReason = iota + 1
	// DamagedOnlyReasonNoRoom: the container has no room left.
	DamagedOnlyReasonNoRoom
	// DamagedOnlyReasonGrappleImmune: the creature cannot be Grappled, so no grapple can hold it.
	DamagedOnlyReasonGrappleImmune
)

// String returns "too_large", "no_room", "grapple_immune" or "unknown".
func (r DamagedOnlyReason) String() string {
	return enumName([]string{"too_large", "no_room", "grapple_immune"}, int(r))
}

// ContainmentCover is the cover a held creature has against what is outside (SRD 5.2.1, "Cover", p. 179).
type ContainmentCover int

// The two covers.
const (
	// ContainmentCoverNone: no cover.
	ContainmentCoverNone ContainmentCover = iota + 1
	// ContainmentCoverTotal: Total Cover.
	ContainmentCoverTotal
)

// String returns "none", "total" or "unknown".
func (c ContainmentCover) String() string { return enumName([]string{"none", "total"}, int(c)) }

// ContainedRestrictions is what holding a creature restricts, read from the terms by
// ContainmentRules.RestrictionsOn. A creature that is not held has ContainmentCoverNone,
// SuffocationRuleNone and no other restriction. Build it with a keyed literal.
type ContainedRestrictions struct {
	// Cover is the cover against what is outside.
	Cover ContainmentCover
	// NoVerbalSpells says the creature cannot cast a spell with a Verbal component.
	NoVerbalSpells bool
	// NoTeleport says the creature cannot teleport.
	NoTeleport bool
	// Suffocation says whether the creature is suffocating.
	Suffocation SuffocationRule
}

// ContainmentEffects is what a contained result tells the host to do (FR-037): start choking per
// Suffocation, treat the creature as unable to speak a Verbal component or teleport, and give it Total
// Cover. Build it with a keyed literal.
type ContainmentEffects struct {
	// Suffocation says whether the creature starts suffocating.
	Suffocation SuffocationRule
	// NoVerbalSpells says the creature cannot cast a spell with a Verbal component.
	NoVerbalSpells bool
	// NoTeleport says the creature cannot teleport.
	NoTeleport bool
	// TotalCover says the creature has Total Cover against what is outside.
	TotalCover bool
}

// ContainmentDamageCause is why a ContainmentDamageDue is due.
type ContainmentDamageCause int

// The three causes.
const (
	// ContainmentDamageCauseEntry: the failed save of an entry.
	ContainmentDamageCauseEntry ContainmentDamageCause = iota + 1
	// ContainmentDamageCausePeriodic: the periodic damage of a turn.
	ContainmentDamageCausePeriodic
	// ContainmentDamageCausePull: the damage a puller takes from a successful pull.
	ContainmentDamageCausePull
)

// String returns "entry", "periodic", "pull" or "unknown".
func (c ContainmentDamageCause) String() string {
	return enumName([]string{"entry", "periodic", "pull"}, int(c))
}

// ContainmentDamageDue is damage the host rolls and applies through ApplyDamage (SRD 5.2.1, "Half Damage",
// p. 16); the containment rules never roll or apply it. Build it with a keyed literal; the terms are read
// with Terms.
type ContainmentDamageDue struct {
	// Container is the monster that holds, or held, the target.
	Container CombatantID
	// Target takes the damage: the occupant, or the puller of a pull.
	Target CombatantID
	terms  damageTerms
	halved bool
	cause  ContainmentDamageCause
}

// Terms returns a new slice of the dice and types to roll.
func (d ContainmentDamageDue) Terms() []DamageTerm { return d.terms.list() }

// Halved says whether the host halves the damage (rounded down).
func (d ContainmentDamageDue) Halved() bool { return d.halved }

// Cause says why the damage is due.
func (d ContainmentDamageDue) Cause() ContainmentDamageCause { return d.cause }

// ContainedPlacement is the placement fact of a contained result: the occupant is now in the container's
// space and moves with it. No distance is computed. Build it with a keyed literal.
type ContainedPlacement struct {
	// Container is the monster whose space the occupant is in.
	Container CombatantID
	// Occupant is the held creature.
	Occupant CombatantID
}

// RelocationDue says a creature that saved moves to an unoccupied space within WithinFeet (SRD 5.2.1,
// "Gelatinous Cube", p. 287). Build it with a keyed literal.
type RelocationDue struct {
	// Occupant is the creature that moves.
	Occupant CombatantID
	// WithinFeet is the distance within which the host places it.
	WithinFeet int
}

// ExitPlacement says where a creature that left a container is placed. Variants:
// NearestUnoccupiedSpace, LandingWithin. The host places the creature; no position is computed here.
type ExitPlacement interface{ isExitPlacement() }

// NearestUnoccupiedSpace: the creature enters the nearest unoccupied space (SRD 5.2.1, "Gelatinous
// Cube", p. 287).
type NearestUnoccupiedSpace struct{}

// LandingWithin: the creature lands in an unoccupied space within Feet (SRD 5.2.1, "Purple Worm", p. 317).
// Build it with a keyed literal.
type LandingWithin struct{ Feet int }

func (NearestUnoccupiedSpace) isExitPlacement() {}
func (LandingWithin) isExitPlacement()          {}

// ConditionImposition is what applying a condition to a held creature did: ConditionImpositionAdded,
// ConditionImpositionAlreadyHeld or ConditionImpositionImmune. The zero value is invalid.
type ConditionImposition int

// The three outcomes.
const (
	// ConditionImpositionAdded: the condition was added (and is removed again on release).
	ConditionImpositionAdded ConditionImposition = iota + 1
	// ConditionImpositionAlreadyHeld: the creature already had the condition, so it is never removed by the release.
	ConditionImpositionAlreadyHeld
	// ConditionImpositionImmune: the creature is immune to the condition.
	ConditionImpositionImmune
)

// String returns "added", "already_held", "immune" or "unknown".
func (c ConditionImposition) String() string {
	return enumName([]string{"added", "already_held", "immune"}, int(c))
}

// ImposedCondition reports one condition a contained result tried to apply. Build it with a keyed literal.
type ImposedCondition struct {
	// Kind is the condition.
	Kind content.ConditionKind
	// Result is what applying it did.
	Result ConditionImposition
}
