package combat

import (
	"context"
	"errors"
	"fmt"

	"github.com/Whealinghold/srd5e-go/d20"
)

// BeginGrappleCommand is the input of BeginGrapple.Run: the grapples in force and
// the raw values of the new grapple. Build it with a keyed literal; bad values give
// BeginGrappleInvalid, not an error.
type BeginGrappleCommand struct {
	Grapples  Grapples
	Grappler  CombatantID
	Target    CombatantID
	Part      GraspingPart
	EscapeDC  d20.DifficultyClass
	RangeFeet int
}

// BeginGrappleResult is the closed set of outcomes of BeginGrapple.Run. Variants:
// BeginGrappleApplied, BeginGrappleRejected, BeginGrappleInvalid.
type BeginGrappleResult interface{ isBeginGrappleResult() }

// BeginGrappleApplied: the grapple began. The host applies Grapple.Condition to the
// target.
type BeginGrappleApplied struct {
	grapples Grapples
	grapple  Grapple
	events   []Event
}

// Grapples returns the set with the new grapple.
func (r BeginGrappleApplied) Grapples() Grapples { return r.grapples }

// Grapple returns the grapple that began.
func (r BeginGrappleApplied) Grapple() Grapple { return r.grapple }

// Events returns a copy of the events: one GrappleStarted.
func (r BeginGrappleApplied) Events() []Event { return append([]Event(nil), r.events...) }

// BeginGrappleRejected: the rules refuse the grapple in this state (the part is in
// use, or the target is already grappled). Construct it with a keyed literal.
type BeginGrappleRejected struct{ Rejection GrappleRejection }

// BeginGrappleInvalid: a value of the command is unusable. Construct it with a keyed
// literal.
type BeginGrappleInvalid struct{ Problem GrappleProblem }

func (BeginGrappleApplied) isBeginGrappleResult()  {}
func (BeginGrappleRejected) isBeginGrappleResult() {}
func (BeginGrappleInvalid) isBeginGrappleResult()  {}

// BeginGrapple is the use case that records a grapple that succeeded, from an
// Unarmed Strike or a monster's special attack the host has resolved. It rolls no
// die and applies no condition. Create it with NewBeginGrapple; the zero value is
// silent and works. Safe for concurrent use.
type BeginGrapple struct{ grappleUse }

// NewBeginGrapple returns the use case. Options: WithTelemetry, WithClock.
func NewBeginGrapple(opts ...Option) BeginGrapple { return BeginGrapple{newGrappleUse(opts)} }

// Run builds the Grapple with NewGrapple and adds it with Grapples.Begin. It records
// the shared signals plus srd5e.combat.grapple.in_force for an applied result.
//
// It returns ctx.Err() unwrapped, before any signal, for a cancelled ctx. An unusable
// value is a BeginGrappleInvalid result and a refusal a BeginGrappleRejected one,
// never an error.
func (u BeginGrapple) Run(ctx context.Context, cmd BeginGrappleCommand) (BeginGrappleResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationBeginGrapple, spanBeginGrapple}
	res, err := run(ctx, u.in, op, func(context.Context) (BeginGrappleResult, error) {
		g, err := NewGrapple(cmd.Grappler, cmd.Target, cmd.Part, cmd.EscapeDC, cmd.RangeFeet)
		var pe *GrappleProblemError
		if errors.As(err, &pe) {
			return BeginGrappleInvalid{pe.Problem}, nil
		}
		switch t := cmd.Grapples.Begin(g).(type) {
		case GrappleTransitionApplied:
			return BeginGrappleApplied{t.grapples, g, t.events}, nil
		case GrappleTransitionRejected:
			return BeginGrappleRejected(t), nil
		case GrappleTransitionInvalid:
			return BeginGrappleInvalid(t), nil
		default:
			panic(fmt.Errorf("combat: unreachable grapple transition %T", t))
		}
	}, func(r BeginGrappleResult) description {
		switch v := r.(type) {
		case BeginGrappleApplied:
			return description{Class: resultApplied, Name: valueResultApplied, Events: v.events}
		case BeginGrappleRejected:
			return refusedGrapple(v.Rejection)
		case BeginGrappleInvalid:
			return invalidGrapple(v.Problem)
		default:
			panic(fmt.Errorf("combat: unreachable begin grapple result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(BeginGrappleApplied); ok {
		u.recordInForce(ctx, op, v.grapples)
	}
	return res, nil
}
