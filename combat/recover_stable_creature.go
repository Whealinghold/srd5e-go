package combat

import "context"

// RecoverStableCreatureCommand is the input of RecoverStableCreature.Run. Build
// it with NewRecoverStableCreatureCommand; the zero value is invalid.
type RecoverStableCreatureCommand struct {
	subject CombatantID
	status  HitPointStatus
}

// NewRecoverStableCreatureCommand returns the command. It returns an error
// matching srderr.ErrInvalidInput for a zero subject or status.
func NewRecoverStableCreatureCommand(subject CombatantID, status HitPointStatus) (RecoverStableCreatureCommand, error) {
	if err := status.check(subject); err != nil {
		return RecoverStableCreatureCommand{}, err
	}
	return RecoverStableCreatureCommand{subject, status}, nil
}

// RecoverStableCreatureResult is the closed set of outcomes of
// RecoverStableCreature.Run. Variants: StableRecoveryApplied,
// StableRecoveryRejected.
type RecoverStableCreatureResult interface{ isRecoverStableCreatureResult() }

// StableRecoveryApplied is a Stable creature that regained 1 Hit Point.
type StableRecoveryApplied struct{ appliedTransition }

// StableRecoveryRejected is a recovery the rules refuse: the creature is Dead or
// not Stable. Construct it with a keyed literal.
type StableRecoveryRejected struct{ Rejection HitPointRejection }

func (StableRecoveryApplied) isRecoverStableCreatureResult()  {}
func (StableRecoveryRejected) isRecoverStableCreatureResult() {}

// RecoverStableCreature is the use case a host calls when the delay of a Stable
// creature is over. Create it with NewRecoverStableCreature; the zero value is
// silent and works.
type RecoverStableCreature struct{ hitPointUse }

// NewRecoverStableCreature returns the use case. Options: WithTelemetry,
// WithClock.
func NewRecoverStableCreature(opts ...Option) RecoverStableCreature {
	return RecoverStableCreature{newHitPointUse(opts)}
}

// Run applies cmd through HitPointStatus.RecoverStable.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is a StableRecoveryRejected result, never an error.
func (u RecoverStableCreature) Run(ctx context.Context, cmd RecoverStableCreatureCommand) (RecoverStableCreatureResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationRecoverStableCreature, spanRecoverStableCreature},
		func() (HitPointTransition, error) { return cmd.status.RecoverStable(cmd.subject) },
		func(HitPointTransitionApplied) string { return valueResultRecovered })
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) RecoverStableCreatureResult {
		return StableRecoveryApplied{appliedTransition{a.status, a.events}}
	}, func(r HitPointRejection) RecoverStableCreatureResult { return StableRecoveryRejected{Rejection: r} }), nil
}
