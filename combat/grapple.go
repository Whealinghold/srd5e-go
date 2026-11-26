package combat

import "github.com/Whealinghold/srd5e-go/d20"

// GrappleUnarmedStrikeRangeFeet is the range of a grapple begun by an Unarmed
// Strike: 5 feet. The SRD never defines "the grapple's range"; p. 190 only says
// the attack's target is "within 5 feet of you". A host may pass another range,
// such as a stat block's.
const GrappleUnarmedStrikeRangeFeet = 5

// Grapple is one creature holding another with one part (SRD 5.2.1, "Grappling",
// p. 182). Build it with NewGrapple, or with a keyed literal and let NewGrapples
// and Grapples.Begin check it; the zero value is invalid. It is comparable with ==
// on all five fields. The ids are exported fields because no function of this
// package returns a CombatantID. Safe for concurrent use.
type Grapple struct {
	// Grappler holds Target.
	Grappler CombatantID
	// Target is the Grappled creature.
	Target CombatantID
	// Part is the part Grappler holds Target with.
	Part GraspingPart
	// EscapeDC is the Difficulty Class of every escape attempt. The SRD bounds
	// no DC, so it is held as given.
	EscapeDC d20.DifficultyClass
	// RangeFeet is the distance in whole feet beyond which the grapple ends. It
	// is 0 or more.
	RangeFeet int
}

// NewGrapple returns the grapple. It returns a *GrappleProblemError (matching
// srderr.ErrInvalidInput) carrying, in this order, GrappleProblemMissingValue
// for a blank id or an invalid part, GrappleProblemSameGrapplerAndTarget when the
// ids are equal, and GrappleProblemNegativeRange for a negative range. It never
// panics.
func NewGrapple(grappler, target CombatantID, part GraspingPart, escapeDC d20.DifficultyClass, rangeFeet int) (Grapple, error) {
	g := Grapple{grappler, target, part, escapeDC, rangeFeet}
	if p, bad := g.problem(); bad {
		return Grapple{}, problemError(p)
	}
	return g, nil
}

// MustGrapple is NewGrapple for literals known to be valid. It panics with the
// error NewGrapple returns, and only then.
func MustGrapple(grappler, target CombatantID, part GraspingPart, escapeDC d20.DifficultyClass, rangeFeet int) Grapple {
	g, err := NewGrapple(grappler, target, part, escapeDC, rangeFeet)
	if err != nil {
		panic(err)
	}
	return g
}

// problem returns the first problem of g in the order of NewGrapple.
func (g Grapple) problem() (GrappleProblem, bool) {
	switch {
	case g.Grappler.IsZero() || g.Target.IsZero() || !validPart(g.Part):
		return GrappleProblemMissingValue, true
	case g.Grappler == g.Target:
		return GrappleProblemSameGrapplerAndTarget, true
	case g.RangeFeet < 0:
		return GrappleProblemNegativeRange, true
	}
	return 0, false
}

// Condition returns the Grappled condition the target is given, whose source is
// the grappler. The host applies it; Grapples never touches a ConditionSet.
func (g Grapple) Condition() Grappled { return Grappled{Grappler: g.Grappler} }
