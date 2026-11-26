package combat

import "context"

// GainTemporaryHitPointsCommand is the input of GainTemporaryHitPoints.Run.
// Build it with NewGainTemporaryHitPointsCommand; the zero value is invalid.
type GainTemporaryHitPointsCommand struct {
	subject CombatantID
	status  HitPointStatus
	amount  int
	choice  TemporaryHitPointsChoice
}

// NewGainTemporaryHitPointsCommand returns the command. It returns an error
// matching srderr.ErrInvalidInput for a zero subject or status, an amount below 1
// or an invalid choice.
func NewGainTemporaryHitPointsCommand(subject CombatantID, status HitPointStatus, amount int, choice TemporaryHitPointsChoice) (GainTemporaryHitPointsCommand, error) {
	cmd := GainTemporaryHitPointsCommand{subject, status, amount, choice}
	if _, err := cmd.apply(); err != nil {
		return GainTemporaryHitPointsCommand{}, err
	}
	return cmd, nil
}

func (c GainTemporaryHitPointsCommand) apply() (HitPointTransition, error) {
	return c.status.GainTemporary(c.subject, c.amount, c.choice)
}

// GainTemporaryHitPointsResult is the closed set of outcomes of
// GainTemporaryHitPoints.Run. Variants: TemporaryHitPointsGainApplied,
// TemporaryHitPointsGainRejected.
type GainTemporaryHitPointsResult interface{ isGainTemporaryHitPointsResult() }

// TemporaryHitPointsGainApplied is a gain that took effect, possibly changing
// nothing (current kept, or an equal amount offered).
type TemporaryHitPointsGainApplied struct{ appliedTransition }

// TemporaryHitPointsGainRejected is a gain the rules refuse: the creature is
// Dead. Construct it with a keyed literal.
type TemporaryHitPointsGainRejected struct{ Rejection HitPointRejection }

func (TemporaryHitPointsGainApplied) isGainTemporaryHitPointsResult()  {}
func (TemporaryHitPointsGainRejected) isGainTemporaryHitPointsResult() {}

// GainTemporaryHitPoints is the use case that gives a creature Temporary Hit
// Points. Create it with NewGainTemporaryHitPoints; the zero value is silent and
// works.
type GainTemporaryHitPoints struct{ hitPointUse }

// NewGainTemporaryHitPoints returns the use case. Options: WithTelemetry,
// WithClock.
func NewGainTemporaryHitPoints(opts ...Option) GainTemporaryHitPoints {
	return GainTemporaryHitPoints{newHitPointUse(opts)}
}

// Run applies cmd through HitPointStatus.GainTemporary.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is a TemporaryHitPointsGainRejected result, never an error.
func (u GainTemporaryHitPoints) Run(ctx context.Context, cmd GainTemporaryHitPointsCommand) (GainTemporaryHitPointsResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationGainTemporaryHitPoints, spanGainTemporaryHitPoints}, cmd.apply,
		func(a HitPointTransitionApplied) string {
			if len(a.events) == 0 {
				return valueResultNoEffect
			}
			return valueResultGained
		})
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) GainTemporaryHitPointsResult {
		return TemporaryHitPointsGainApplied{appliedTransition{a.status, a.events}}
	}, func(r HitPointRejection) GainTemporaryHitPointsResult {
		return TemporaryHitPointsGainRejected{Rejection: r}
	}), nil
}
