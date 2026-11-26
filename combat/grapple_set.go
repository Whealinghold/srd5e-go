package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// Grapples is the grapples in force, in the order they began (SRD 5.2.1,
// "Grappling", p. 182). It holds the invariants of the SRD: a part holds one
// creature, and a creature is held by one grapple. The zero Grapples is empty and
// valid. It holds a slice, so compare with Equal, not ==; it copies on
// construction and in every accessor. Safe for concurrent use.
type Grapples struct{ list []Grapple }

type partKey struct {
	grappler CombatantID
	part     GraspingPart
}

// NewGrapples returns the set holding gs, which is also the order they began in.
// It returns an error matching srderr.ErrInvalidInput (a *GrappleProblemError for
// a grapple NewGrapple would reject, otherwise a plain one) for two grapples with
// the same grappler and part, or with the same target, and a LimitExceededError for more than
// 256 grapples (plan P10). It copies gs and never panics.
func NewGrapples(gs []Grapple) (Grapples, error) {
	if err := overCap("grapples", len(gs), maxInputList); err != nil {
		return Grapples{}, err
	}
	parts, targets := make(map[partKey]struct{}, len(gs)), make(map[CombatantID]struct{}, len(gs))
	for _, g := range gs {
		if p, bad := g.problem(); bad {
			return Grapples{}, problemError(p)
		}
		k := partKey{g.Grappler, g.Part}
		if _, dup := parts[k]; dup {
			return Grapples{}, invalidInput("a part holds one creature")
		}
		if _, dup := targets[g.Target]; dup {
			return Grapples{}, invalidInput("a creature is held once")
		}
		parts[k], targets[g.Target] = struct{}{}, struct{}{}
	}
	return Grapples{slices.Clone(gs)}, nil
}

// MustGrapples is NewGrapples for literals known to be valid. It panics with the
// error NewGrapples returns, and only then.
func MustGrapples(gs []Grapple) Grapples {
	s, err := NewGrapples(gs)
	if err != nil {
		panic(err)
	}
	return s
}

// Len returns the number of grapples.
func (s Grapples) Len() int { return len(s.list) }

// All returns a copy of the grapples, in the order they began.
func (s Grapples) All() []Grapple { return slices.Clone(s.list) }

// Equal reports whether s and o hold the same grapples, whatever order they began
// in.
func (s Grapples) Equal(o Grapples) bool {
	if len(s.list) != len(o.list) {
		return false
	}
	for _, g := range s.list {
		if !slices.Contains(o.list, g) {
			return false
		}
	}
	return true
}

// IsHolding says whether the part of the grappler already holds a creature. The
// same part value on another grappler is a different part.
func (s Grapples) IsHolding(grappler CombatantID, part GraspingPart) PartState {
	for _, g := range s.list {
		if g.Grappler == grappler && g.Part == part {
			return PartStateHolding
		}
	}
	return PartStateNotHoldingGrapple
}

// CanTarget says whether the grappler may use the part to target the creature:
// forbidden when the part holds another creature, otherwise allowed (SRD 5.2.1,
// "Grappling", p. 182).
func (s Grapples) CanTarget(grappler CombatantID, part GraspingPart, target CombatantID) ConditionPermission {
	for _, g := range s.list {
		if g.Grappler == grappler && g.Part == part && g.Target != target {
			return ConditionPermissionForbidden
		}
	}
	return ConditionPermissionAllowed
}

// GrappleOfTarget is the answer of Grapples.GrappleOf. Variants: InGrapple,
// NotInGrapple.
type GrappleOfTarget interface{ isGrappleOfTarget() }

// InGrapple: a grapple holds the target. Read it with Grapple.
type InGrapple struct{ grapple Grapple }

// NotInGrapple: no grapple holds the target.
type NotInGrapple struct{}

func (InGrapple) isGrappleOfTarget()    {}
func (NotInGrapple) isGrappleOfTarget() {}

// Grapple returns the grapple that holds the target.
func (g InGrapple) Grapple() Grapple { return g.grapple }

// GrappleOf returns the grapple holding target, or NotInGrapple.
func (s Grapples) GrappleOf(target CombatantID) GrappleOfTarget {
	for _, g := range s.list {
		if g.Target == target {
			return InGrapple{g}
		}
	}
	return NotInGrapple{}
}

// HeldBy returns a new slice of the grapples the grappler holds, in the order
// they began; empty for a grappler holding none.
func (s Grapples) HeldBy(grappler CombatantID) []Grapple {
	out := []Grapple{}
	for _, g := range s.list {
		if g.Grappler == grappler {
			out = append(out, g)
		}
	}
	return out
}

// ExtraMovementCostPerFoot returns the extra feet each foot of movement costs the
// grappler dragging or carrying the creatures in carried: the sum over them of
// GrappleRules.MovementCost (SRD 5.2.1, "Grappled [Condition]", "Movable", p. 182).
// The sum is package design: the SRD states the cost in each Grappled creature's
// entry. It changes no grapple.
//
// It returns an error matching srderr.ErrInvalidInput, and 0, for an invalid
// grapplerSize, a creature listed twice, a creature the grappler does not hold, or
// a creature with no valid size in sizes, and a LimitExceededError for more than 256 carried
// (plan P10). It never panics.
func (s Grapples) ExtraMovementCostPerFoot(grappler CombatantID, grapplerSize content.CreatureSize,
	carried []CombatantID, sizes map[CombatantID]content.CreatureSize) (int, error) {
	if !grapplerSize.Valid() {
		return 0, invalidInput("invalid grappler size")
	}
	if err := overCap("carried creatures", len(carried), maxInputList); err != nil {
		return 0, err
	}
	seen := make(map[CombatantID]struct{}, len(carried))
	total := 0
	for _, id := range carried {
		if _, dup := seen[id]; dup {
			return 0, invalidInput("creature carried twice")
		}
		seen[id] = struct{}{}
		if g, ok := s.GrappleOf(id).(InGrapple); !ok || g.grapple.Grappler != grappler {
			return 0, invalidInput("creature is not held by the grappler")
		}
		size, ok := sizes[id]
		if !ok || !size.Valid() {
			return 0, invalidInput("carried creature has no size")
		}
		total += GrappleRules{}.MovementCost(grapplerSize, size)
	}
	return total, nil
}
