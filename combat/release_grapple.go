package combat

import (
	"context"
	"fmt"
)

// ReleaseGrappleCommand is the input of ReleaseGrapple.Run: the grapples in force
// and the grappler and target of the grapple to release. Build it with a keyed
// literal; equal ids give ReleaseGrappleInvalid. Releasing costs no action, so the
// command carries no action availability.
type ReleaseGrappleCommand struct {
	Grapples Grapples
	Grappler CombatantID
	Target   CombatantID
}

// ReleaseGrappleResult is the closed set of outcomes of ReleaseGrapple.Run.
// Variants: ReleaseGrappleApplied, ReleaseGrappleRejected, ReleaseGrappleInvalid.
type ReleaseGrappleResult interface{ isReleaseGrappleResult() }

// ReleaseGrappleApplied: the grapple was released. The host passes the ending's
// target and cause to GrappleRules.EndOnSet.
type ReleaseGrappleApplied struct {
	grapples Grapples
	ending   GrappleEnding
}

// Grapples returns the set without the released grapple.
func (r ReleaseGrappleApplied) Grapples() Grapples { return r.grapples }

// Ending returns the ending, with GrappleEndCauseReleased.
func (r ReleaseGrappleApplied) Ending() GrappleEnding { return r.ending }

// ReleaseGrappleRejected: the grappler does not hold the target. Construct it with a
// keyed literal.
type ReleaseGrappleRejected struct{ Rejection GrappleRejection }

// ReleaseGrappleInvalid: the grappler and target are the same creature. Construct it
// with a keyed literal.
type ReleaseGrappleInvalid struct{ Problem GrappleProblem }

func (ReleaseGrappleApplied) isReleaseGrappleResult()  {}
func (ReleaseGrappleRejected) isReleaseGrappleResult() {}
func (ReleaseGrappleInvalid) isReleaseGrappleResult()  {}

// ReleaseGrapple is the use case that releases a grapple at any time, at no action
// (SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182), even for an Incapacitated
// grappler. It rolls no die. Create it with NewReleaseGrapple; the zero value is
// silent and works. Safe for concurrent use.
type ReleaseGrapple struct{ grappleUse }

// NewReleaseGrapple returns the use case. Options: WithTelemetry, WithClock.
func NewReleaseGrapple(opts ...Option) ReleaseGrapple { return ReleaseGrapple{newGrappleUse(opts)} }

// Run releases the grapple with Grapples.Release. It records the shared signals plus,
// for an applied result, srd5e.combat.grapple.ended (released) and
// srd5e.combat.grapple.in_force.
//
// It returns ctx.Err() unwrapped, before any signal, for a cancelled ctx. A refusal
// or unusable input is a result, never an error.
func (u ReleaseGrapple) Run(ctx context.Context, cmd ReleaseGrappleCommand) (ReleaseGrappleResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationReleaseGrapple, spanReleaseGrapple}
	res, err := run(ctx, u.in, op, func(context.Context) (ReleaseGrappleResult, error) {
		switch t := cmd.Grapples.Release(cmd.Grappler, cmd.Target).(type) {
		case GrappleTransitionApplied:
			return ReleaseGrappleApplied{t.grapples, t.endings[0]}, nil
		case GrappleTransitionRejected:
			return ReleaseGrappleRejected(t), nil
		case GrappleTransitionInvalid:
			return ReleaseGrappleInvalid(t), nil
		default:
			panic(fmt.Errorf("combat: unreachable grapple transition %T", t))
		}
	}, func(r ReleaseGrappleResult) description {
		switch v := r.(type) {
		case ReleaseGrappleApplied:
			return description{Class: resultApplied, Name: valueResultApplied}
		case ReleaseGrappleRejected:
			return refusedGrapple(v.Rejection)
		case ReleaseGrappleInvalid:
			return invalidGrapple(v.Problem)
		default:
			panic(fmt.Errorf("combat: unreachable release grapple result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(ReleaseGrappleApplied); ok {
		u.recordEnded(ctx, v.ending.cause.String())
		u.recordInForce(ctx, op, v.grapples)
	}
	return res, nil
}
