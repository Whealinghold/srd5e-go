package combat

import "context"

// RecordExhaustionDeathCommand is the input of RecordExhaustionDeath.Run. Build
// it with NewRecordExhaustionDeathCommand; the zero value is invalid.
type RecordExhaustionDeathCommand struct {
	subject CombatantID
	status  HitPointStatus
}

// NewRecordExhaustionDeathCommand returns the command. It returns an error
// matching srderr.ErrInvalidInput for a zero subject or status.
func NewRecordExhaustionDeathCommand(subject CombatantID, status HitPointStatus) (RecordExhaustionDeathCommand, error) {
	if err := status.check(subject); err != nil {
		return RecordExhaustionDeathCommand{}, err
	}
	return RecordExhaustionDeathCommand{subject, status}, nil
}

// RecordExhaustionDeathResult is the closed set of outcomes of
// RecordExhaustionDeath.Run. Variants: ExhaustionDeathApplied,
// ExhaustionDeathRejected.
type RecordExhaustionDeathResult interface{ isRecordExhaustionDeathResult() }

// ExhaustionDeathApplied is a creature that died of Exhaustion.
type ExhaustionDeathApplied struct{ appliedTransition }

// ExhaustionDeathRejected is a death the rules refuse to record: the creature is
// already Dead. Construct it with a keyed literal.
type ExhaustionDeathRejected struct{ Rejection HitPointRejection }

func (ExhaustionDeathApplied) isRecordExhaustionDeathResult()  {}
func (ExhaustionDeathRejected) isRecordExhaustionDeathResult() {}

// RecordExhaustionDeath is the use case a host calls when Exhaustion reached
// level 6. Create it with NewRecordExhaustionDeath; the zero value is silent and
// works.
type RecordExhaustionDeath struct{ hitPointUse }

// NewRecordExhaustionDeath returns the use case. Options: WithTelemetry,
// WithClock.
func NewRecordExhaustionDeath(opts ...Option) RecordExhaustionDeath {
	return RecordExhaustionDeath{newHitPointUse(opts)}
}

// Run applies cmd through HitPointStatus.RecordExhaustionDeath.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is an ExhaustionDeathRejected result, never an error.
func (u RecordExhaustionDeath) Run(ctx context.Context, cmd RecordExhaustionDeathCommand) (RecordExhaustionDeathResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationRecordExhaustionDeath, spanRecordExhaustionDeath},
		func() (HitPointTransition, error) { return cmd.status.RecordExhaustionDeath(cmd.subject) },
		func(HitPointTransitionApplied) string { return valueResultKilled })
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) RecordExhaustionDeathResult {
		return ExhaustionDeathApplied{appliedTransition{a.status, a.events}}
	}, func(r HitPointRejection) RecordExhaustionDeathResult { return ExhaustionDeathRejected{Rejection: r} }), nil
}
