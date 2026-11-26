package combat

import "context"

// EndTemporaryHitPointsCommand is the input of EndTemporaryHitPoints.Run. Build
// it with NewEndTemporaryHitPointsCommand; the zero value is invalid.
type EndTemporaryHitPointsCommand struct {
	subject CombatantID
	status  HitPointStatus
}

// NewEndTemporaryHitPointsCommand returns the command. It returns an error
// matching srderr.ErrInvalidInput for a zero subject or status.
func NewEndTemporaryHitPointsCommand(subject CombatantID, status HitPointStatus) (EndTemporaryHitPointsCommand, error) {
	if err := status.check(subject); err != nil {
		return EndTemporaryHitPointsCommand{}, err
	}
	return EndTemporaryHitPointsCommand{subject, status}, nil
}

// EndTemporaryHitPointsResult is the closed set of outcomes of
// EndTemporaryHitPoints.Run. Variants: TemporaryHitPointsEndApplied,
// TemporaryHitPointsEndRejected.
type EndTemporaryHitPointsResult interface{ isEndTemporaryHitPointsResult() }

// TemporaryHitPointsEndApplied is an end that took effect, unchanged when none
// were held.
type TemporaryHitPointsEndApplied struct{ appliedTransition }

// TemporaryHitPointsEndRejected is an end the rules refuse: the creature is
// Dead. Construct it with a keyed literal.
type TemporaryHitPointsEndRejected struct{ Rejection HitPointRejection }

func (TemporaryHitPointsEndApplied) isEndTemporaryHitPointsResult()  {}
func (TemporaryHitPointsEndRejected) isEndTemporaryHitPointsResult() {}

// EndTemporaryHitPoints is the use case a host calls when a Long Rest is
// finished. Create it with NewEndTemporaryHitPoints; the zero value is silent and
// works.
type EndTemporaryHitPoints struct{ hitPointUse }

// NewEndTemporaryHitPoints returns the use case. Options: WithTelemetry,
// WithClock.
func NewEndTemporaryHitPoints(opts ...Option) EndTemporaryHitPoints {
	return EndTemporaryHitPoints{newHitPointUse(opts)}
}

// Run applies cmd through HitPointStatus.EndTemporary.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is a TemporaryHitPointsEndRejected result, never an error.
func (u EndTemporaryHitPoints) Run(ctx context.Context, cmd EndTemporaryHitPointsCommand) (EndTemporaryHitPointsResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationEndTemporaryHitPoints, spanEndTemporaryHitPoints},
		func() (HitPointTransition, error) { return cmd.status.EndTemporary(cmd.subject) },
		func(a HitPointTransitionApplied) string {
			if len(a.events) == 0 {
				return valueResultNoEffect
			}
			return valueResultEnded
		})
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) EndTemporaryHitPointsResult {
		return TemporaryHitPointsEndApplied{appliedTransition{a.status, a.events}}
	}, func(r HitPointRejection) EndTemporaryHitPointsResult {
		return TemporaryHitPointsEndRejected{Rejection: r}
	}), nil
}
