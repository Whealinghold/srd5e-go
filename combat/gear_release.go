package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// ReleaseActor is who tries to free a creature from a hold. Variants: ActorIsHeld,
// ActorIsHelper.
type ReleaseActor interface{ isReleaseActor() }

// ActorIsHeld: the held creature acts for itself.
type ActorIsHeld struct{}

// ActorIsHelper: another creature acts. WithinReach is the host's fact: the helper is
// within reach of the hold (within 5 feet of a Net's target; within a Hunting Trap's reach).
// Construct with a keyed literal.
type ActorIsHelper struct{ WithinReach bool }

func (ActorIsHeld) isReleaseActor()   {}
func (ActorIsHelper) isReleaseActor() {}

// ReleaseRequest holds the facts HoldRules.Plan reads about an attempt to free a creature.
// Build it with a keyed literal.
type ReleaseRequest struct {
	// Hold is the hold to be freed from.
	Hold GearHold
	// Method is the way out tried.
	Method HoldReleaseMethod
	// Actor is who tries.
	Actor ReleaseActor
	// Actions says whether the actor can take actions (ConditionActionAvailability of its effective conditions).
	Actions ConditionActionAvailability
	// HasThievesTools says whether the actor has Thieves' Tools, read for picking a lock only.
	HasThievesTools bool
}

// HoldReleasePlan is what an accepted release attempt asks of the host: a check, or none for
// a key, and the cause the hold ends with on success. It is immutable and comparable. The
// zero value is invalid.
type HoldReleasePlan struct {
	check      HoldCheck
	needsCheck bool
	cause      HoldEndCause
}

// Check returns the check to make; ok is false for a key, which takes none.
func (p HoldReleasePlan) Check() (HoldCheck, bool) { return p.check, p.needsCheck }

// Cause returns how the hold ends on success.
func (p HoldReleasePlan) Cause() HoldEndCause { return p.cause }

// HoldRules is the rules of getting out of a hold (SRD 5.2.1, pp. 96-99). The zero value is
// ready to use. Pure; safe for concurrent use.
type HoldRules struct{}

// Available returns a new slice of the ways out of a hold of kind: escape for all, burst for
// Chain, Manacles and Rope, pick the lock and the key for Manacles. It is empty for a kind that
// is not valid.
func (HoldRules) Available(kind HoldKind) []HoldReleaseMethod {
	methods := []HoldReleaseMethod{}
	if !kind.Valid() {
		return methods
	}
	methods = append(methods, HoldReleaseMethodEscape)
	if _, ok := HoldProfileFor(kind).BurstCheck(); ok {
		methods = append(methods, HoldReleaseMethodBurst)
	}
	if kind == HoldKindManacles {
		methods = append(methods, HoldReleaseMethodPickLock, HoldReleaseMethodKey)
	}
	return methods
}

// PickLockCheck returns the Dexterity (Sleight of Hand) check, DC 15, to pick the Manacles'
// lock with Thieves' Tools (SRD 5.2.1, "Manacles", p. 98).
func (HoldRules) PickLockCheck() HoldCheck { return HoldCheck{content.SkillSleightOfHand, 15} }

// FailureDamage returns the damage a failed attempt to escape a hold deals to the held
// creature: 1 Piercing for a Hunting Trap (SRD 5.2.1, "Hunting Trap", pp. 97-98), none for any
// other kind.
func (HoldRules) FailureDamage(kind HoldKind) (Damage, bool) {
	if kind != HoldKindHuntingTrap {
		return Damage{}, false
	}
	return Damage{1, content.DamagePiercing}, true
}

// Plan decides whether the attempt can be made and what it asks. The refusals, first that
// applies: methodNotAvailable (FR-050), helperNotAllowed (a helper for the escape or burst of a
// Chain, Manacles or Rope), helperOutOfReach (a helper beyond reach of a Net or Hunting Trap),
// noAction (the actor cannot act, for the key too), toolsMissing (picking a lock without
// Thieves' Tools). A helper may pick the lock or use the key; its reach is the host's (Q-12).
// On a refusal the plan is the zero plan.
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for a
// zero hold, a method that is not valid, an actor that is nil or another type, or actions that
// are not valid. It never panics.
func (HoldRules) Plan(r ReleaseRequest) (HoldReleasePlan, GearRejection, error) {
	if r.Hold.IsZero() || !r.Method.Valid() || (r.Actions != ConditionActionAvailabilityAvailable && r.Actions != ConditionActionAvailabilityNone) {
		return HoldReleasePlan{}, 0, gearInvalid(GearProblemInvalidValue)
	}
	switch r.Actor.(type) {
	case ActorIsHeld, ActorIsHelper:
	default:
		return HoldReleasePlan{}, 0, gearInvalid(GearProblemInvalidValue)
	}
	plan, refusal := r.plan()
	return plan, refusal, nil
}

// plan is Plan for a request known to be valid.
func (r ReleaseRequest) plan() (HoldReleasePlan, GearRejection) {
	helper, isHelper := false, false
	if a, ok := r.Actor.(ActorIsHelper); ok {
		isHelper, helper = true, a.WithinReach
	}
	p := HoldProfileFor(r.Hold.kind)
	burst, canBurst := p.BurstCheck()
	var plan HoldReleasePlan
	switch r.Method {
	case HoldReleaseMethodEscape:
		plan = HoldReleasePlan{p.escape, true, HoldEndCauseEscaped}
	case HoldReleaseMethodBurst:
		plan = HoldReleasePlan{burst, canBurst, HoldEndCauseBurst}
	case HoldReleaseMethodPickLock:
		plan = HoldReleasePlan{HoldRules{}.PickLockCheck(), r.Hold.kind == HoldKindManacles, HoldEndCauseUnlocked}
	case HoldReleaseMethodKey:
		plan = HoldReleasePlan{cause: HoldEndCauseUnlocked}
	}
	switch {
	case r.Method == HoldReleaseMethodBurst && !canBurst,
		(r.Method == HoldReleaseMethodPickLock || r.Method == HoldReleaseMethodKey) && r.Hold.kind != HoldKindManacles:
		return HoldReleasePlan{}, GearRejectionMethodNotAvailable
	case isHelper && (r.Method == HoldReleaseMethodEscape || r.Method == HoldReleaseMethodBurst) && !p.helperMayRelease:
		return HoldReleasePlan{}, GearRejectionHelperNotAllowed
	case isHelper && (r.Method == HoldReleaseMethodEscape || r.Method == HoldReleaseMethodBurst) && !helper:
		return HoldReleasePlan{}, GearRejectionHelperOutOfReach
	case r.Actions == ConditionActionAvailabilityNone:
		return HoldReleasePlan{}, GearRejectionNoAction
	case r.Method == HoldReleaseMethodPickLock && !r.HasThievesTools:
		return HoldReleasePlan{}, GearRejectionToolsMissing
	}
	return plan, 0
}
