package combat

import "context"

// StabilizeCreatureCommand is the input of StabilizeCreature.Run. Build it with
// NewStabilizeCreatureCommand; the zero value is invalid.
type StabilizeCreatureCommand struct {
	subject CombatantID
	status  HitPointStatus
}

// NewStabilizeCreatureCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a zero subject or status.
func NewStabilizeCreatureCommand(subject CombatantID, status HitPointStatus) (StabilizeCreatureCommand, error) {
	if err := status.check(subject); err != nil {
		return StabilizeCreatureCommand{}, err
	}
	return StabilizeCreatureCommand{subject, status}, nil
}

// StabilizeCreatureResult is the closed set of outcomes of
// StabilizeCreature.Run. Variants: StabilizationApplied, StabilizationRejected.
type StabilizeCreatureResult interface{ isStabilizeCreatureResult() }

// StabilizationApplied is a creature made Stable.
type StabilizationApplied struct{ appliedTransition }

// StabilizationRejected is a stabilization the rules refuse: the creature is
// Dead or not Dying. Construct it with a keyed literal.
type StabilizationRejected struct{ Rejection HitPointRejection }

func (StabilizationApplied) isStabilizeCreatureResult()  {}
func (StabilizationRejected) isStabilizeCreatureResult() {}

// StabilizeCreature is the use case that stabilizes a Dying creature; the
// success of the Help action check is the host's input. Create it with
// NewStabilizeCreature; the zero value is silent and works.
type StabilizeCreature struct{ hitPointUse }

// NewStabilizeCreature returns the use case. Options: WithTelemetry, WithClock.
func NewStabilizeCreature(opts ...Option) StabilizeCreature {
	return StabilizeCreature{newHitPointUse(opts)}
}

// Run applies cmd through HitPointStatus.Stabilize.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. A refusal is a StabilizationRejected result, never an error.
func (u StabilizeCreature) Run(ctx context.Context, cmd StabilizeCreatureCommand) (StabilizeCreatureResult, error) {
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationStabilizeCreature, spanStabilizeCreature},
		func() (HitPointTransition, error) { return cmd.status.Stabilize(cmd.subject) },
		func(HitPointTransitionApplied) string { return valueResultStabilized })
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) StabilizeCreatureResult {
		return StabilizationApplied{appliedTransition{a.status, a.events}}
	}, func(r HitPointRejection) StabilizeCreatureResult { return StabilizationRejected{Rejection: r} }), nil
}
