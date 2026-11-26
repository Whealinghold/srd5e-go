package combat

import (
	"context"
	"fmt"
)

// ReviewGrapplesCommand is the input of ReviewGrapples.Run: the grapples in force
// and what the host reports about the grapples it chooses to check. Build it with a
// keyed literal.
type ReviewGrapplesCommand struct {
	Grapples     Grapples
	Observations []GrappleObservation
}

// ReviewGrapplesResult is the closed set of outcomes of ReviewGrapples.Run. Variants:
// ReviewGrapplesApplied, ReviewGrapplesInvalid.
type ReviewGrapplesResult interface{ isReviewGrapplesResult() }

// ReviewGrapplesApplied: the review is done. The host passes each ending's target
// and cause to GrappleRules.EndOnSet.
type ReviewGrapplesApplied struct {
	grapples Grapples
	endings  []GrappleEnding
}

// Grapples returns the set without the grapples that ended.
func (r ReviewGrapplesApplied) Grapples() Grapples { return r.grapples }

// Endings returns a copy of the endings, in the order the grapples began; empty when
// nothing ended.
func (r ReviewGrapplesApplied) Endings() []GrappleEnding {
	return append([]GrappleEnding{}, r.endings...)
}

// ReviewGrapplesInvalid: an observation names a grapple the set does not hold, or two
// observations name one grapple. Construct it with a keyed literal.
type ReviewGrapplesInvalid struct{ Problem GrappleProblem }

func (ReviewGrapplesApplied) isReviewGrapplesResult() {}
func (ReviewGrapplesInvalid) isReviewGrapplesResult() {}

// ReviewGrapples is the use case that ends the grapples whose grappler is Incapacitated
// (or dead) or whose target is beyond the range (SRD 5.2.1, "Grappling", "Ending a
// Grapple", p. 182). It rolls no die. Create it with NewReviewGrapples; the zero value
// is silent and works. Safe for concurrent use.
type ReviewGrapples struct{ grappleUse }

// NewReviewGrapples returns the use case. Options: WithTelemetry, WithClock.
func NewReviewGrapples(opts ...Option) ReviewGrapples { return ReviewGrapples{newGrappleUse(opts)} }

// Run reviews the grapples with Grapples.Review. It records the shared signals plus,
// for an applied result, one srd5e.combat.grapple.ended per ending and
// srd5e.combat.grapple.in_force.
//
// It returns ctx.Err() unwrapped, before any signal, for a cancelled ctx. An unknown or
// duplicate observation is a ReviewGrapplesInvalid result, never an error.
func (u ReviewGrapples) Run(ctx context.Context, cmd ReviewGrapplesCommand) (ReviewGrapplesResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationReviewGrapples, spanReviewGrapples}
	res, err := run(ctx, u.in, op, func(context.Context) (ReviewGrapplesResult, error) {
		switch t := cmd.Grapples.Review(cmd.Observations).(type) {
		case GrappleTransitionApplied:
			return ReviewGrapplesApplied{t.grapples, t.endings}, nil
		case GrappleTransitionInvalid:
			return ReviewGrapplesInvalid(t), nil
		default:
			panic(fmt.Errorf("combat: unreachable grapple transition %T", t))
		}
	}, func(r ReviewGrapplesResult) description {
		switch v := r.(type) {
		case ReviewGrapplesApplied:
			return description{Class: resultApplied, Name: valueResultApplied}
		case ReviewGrapplesInvalid:
			return invalidGrapple(v.Problem)
		default:
			panic(fmt.Errorf("combat: unreachable review grapples result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(ReviewGrapplesApplied); ok {
		u.recordEndings(ctx, v.endings)
		u.recordInForce(ctx, op, v.grapples)
	}
	return res, nil
}
