package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// BindingScene holds the facts BindingRules.Assess reads about an attempt to bind an unwilling
// creature with a Chain, Manacles or Rope (SRD 5.2.1, pp. 96, 98, 99). The engine models no
// map: the host reports the distance, the size and the willingness. Build it with a keyed
// literal.
type BindingScene struct {
	// Kind is how the creature is bound.
	Kind BindingKind
	// Willingness says whether the target is willing; the rules bind an unwilling creature.
	Willingness Willingness
	// DistanceFeet is the distance to the target, 0 or more. It is ignored for a Rope.
	DistanceFeet int
	// Target is the creature to bind.
	Target CombatantID
	// TargetSize is the target's size.
	TargetSize content.CreatureSize
	// TargetConditions is the target's effective conditions, in which implied kinds are already expanded.
	TargetConditions ConditionSet
	// Holds are the holds in force.
	Holds GearHolds
	// HoldID is the id the new hold would have.
	HoldID HoldID
}

// BindingRules is the rules of binding a creature with a Chain, Manacles or Rope. The zero
// value is ready to use. Pure; safe for concurrent use.
type BindingRules struct{}

// bindReachFeet is how near the target must be for a Chain or Manacles (SRD 5.2.1, "Chain",
// "Manacles", pp. 96, 98: within 5 feet of yourself).
const bindReachFeet = 5

// Assess decides whether the binding can be tried and, if so, returns the binder's check. The
// refusals, first that applies, in the order the package fixes (the SRD gives none):
// targetWilling, outOfReach (a Chain or Manacles beyond 5 feet; a Rope states no distance, so
// none is checked), sizeNotAllowed (Manacles take a Small or Medium creature only),
// targetNotHeldDown (the target is not Grappled, Incapacitated or Restrained, effective
// conditions counting what Unconscious, Paralyzed, Petrified and Stunned imply),
// targetAlreadyHeld (a hold of the same kind is on the target), idInUse. On a refusal the
// check is the zero HoldCheck.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a zero
// target or hold id, GearProblemNegativeDistance for a negative distance or
// GearProblemInvalidValue for a kind, willingness or size that is not valid. It never panics.
func (BindingRules) Assess(s BindingScene) (HoldCheck, GearRejection, error) {
	switch {
	case s.Target.IsZero() || s.HoldID == "":
		return HoldCheck{}, 0, gearInvalid(GearProblemBlankID)
	case !s.Kind.Valid() || !s.Willingness.Valid() || !s.TargetSize.Valid():
		return HoldCheck{}, 0, gearInvalid(GearProblemInvalidValue)
	case s.DistanceFeet < 0:
		return HoldCheck{}, 0, gearInvalid(GearProblemNegativeDistance)
	}
	check, refusal := s.assess()
	return check, refusal, nil
}

// assess is Assess for a scene known to be valid.
func (s BindingScene) assess() (HoldCheck, GearRejection) {
	hold := s.Kind.HoldKind()
	var refusal GearRejection
	switch {
	case s.Willingness == WillingnessWilling:
		refusal = GearRejectionTargetWilling
	case hold != HoldKindRope && s.DistanceFeet > bindReachFeet:
		refusal = GearRejectionOutOfReach
	case hold == HoldKindManacles && s.TargetSize != content.SizeSmall && s.TargetSize != content.SizeMedium:
		refusal = GearRejectionSizeNotAllowed
	case !heldDown(s.TargetConditions):
		refusal = GearRejectionTargetNotHeldDown
	case slices.ContainsFunc(s.Holds.HoldsOn(s.Target), func(h GearHold) bool { return h.kind == hold }):
		refusal = GearRejectionTargetAlreadyHeld
	case s.Holds.index(s.HoldID) >= 0:
		refusal = GearRejectionIDInUse
	}
	if refusal != 0 {
		return HoldCheck{}, refusal
	}
	check, _ := HoldProfileFor(hold).BindCheck()
	return check, 0
}

// heldDown reports whether the effective conditions include Grappled, Incapacitated or
// Restrained (SRD 5.2.1, "Chain", p. 96).
func heldDown(c ConditionSet) bool {
	return slices.ContainsFunc(c.Effective(), func(k content.ConditionKind) bool {
		return k == content.ConditionGrappled || k == content.ConditionIncapacitated || k == content.ConditionRestrained
	})
}
