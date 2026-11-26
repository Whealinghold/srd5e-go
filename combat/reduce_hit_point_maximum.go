package combat

import "context"

// ReduceHitPointMaximumCommand is the input of ReduceHitPointMaximum.Run. Build
// it with NewReduceHitPointMaximumCommand; the zero value is invalid.
type ReduceHitPointMaximumCommand struct {
	subject CombatantID
	status  HitPointStatus
	amount  int
}

// NewReduceHitPointMaximumCommand returns the command. It returns an error
// matching srderr.ErrInvalidInput for a zero subject or status or a negative
// amount.
func NewReduceHitPointMaximumCommand(subject CombatantID, status HitPointStatus, amount int) (ReduceHitPointMaximumCommand, error) {
	if _, err := status.ReduceMaximum(subject, amount); err != nil {
		return ReduceHitPointMaximumCommand{}, err
	}
	return ReduceHitPointMaximumCommand{subject, status, amount}, nil
}

// ReduceHitPointMaximumResult is the closed set of outcomes of
// ReduceHitPointMaximum.Run. Variants: HitPointMaximumReductionApplied,
// HitPointMaximumReductionRejected.
type ReduceHitPointMaximumResult interface{ isReduceHitPointMaximumResult() }

// HitPointMaximumReductionApplied is a reduction that took effect, unchanged
// for an amount of 0.
type HitPointMaximumReductionApplied struct{ appliedTransition }

// HitPointMaximumReductionRejected is a reduction the rules refuse: the creature
// is Dead. Construct it with a keyed literal.
type HitPointMaximumReductionRejected struct{ Rejection HitPointRejection }

func (HitPointMaximumReductionApplied) isReduceHitPointMaximumResult()  {}
func (HitPointMaximumReductionRejected) isReduceHitPointMaximumResult() {}

// ReduceHitPointMaximum is the use case that lowers a creature's Hit Point
// maximum. Create it with NewReduceHitPointMaximum; the zero value is silent and
// works.
type ReduceHitPointMaximum struct{ hitPointUse }

// NewReduceHitPointMaximum returns the use case. Options: WithTelemetry,
// WithClock.
func NewReduceHitPointMaximum(opts ...Option) ReduceHitPointMaximum {
	return ReduceHitPointMaximum{newHitPointUse(opts)}
}

// Run applies cmd through HitPointStatus.ReduceMaximum.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is a HitPointMaximumReductionRejected result, never an error.
func (u ReduceHitPointMaximum) Run(ctx context.Context, cmd ReduceHitPointMaximumCommand) (ReduceHitPointMaximumResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationReduceHitPointMaximum, spanReduceHitPointMaximum},
		func() (HitPointTransition, error) { return cmd.status.ReduceMaximum(cmd.subject, cmd.amount) },
		func(a HitPointTransitionApplied) string {
			switch {
			case has[Died](a.events):
				return valueResultKilled
			case len(a.events) == 0:
				return valueResultNoEffect
			}
			return valueResultReduced
		})
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) ReduceHitPointMaximumResult {
		return HitPointMaximumReductionApplied{appliedTransition{a.status, a.events}}
	}, func(r HitPointRejection) ReduceHitPointMaximumResult {
		return HitPointMaximumReductionRejected{Rejection: r}
	}), nil
}
