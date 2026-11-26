package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// ReduceExhaustionOnSetCommand is the input of ReduceExhaustionOnSet.Run. Build it
// with NewReduceExhaustionOnSetCommand; the zero value is invalid.
type ReduceExhaustionOnSetCommand struct {
	set    ConditionSet
	target CombatantID
	levels int
}

// NewReduceExhaustionOnSetCommand returns the command to take levels Exhaustion
// levels off the creature target, whose conditions are set. It returns an error
// matching srderr.ErrInvalidInput for a zero target. levels is not checked here:
// ConditionSet.ReduceExhaustion rejects a value below 1, so the span of Run exists
// and records the error.
func NewReduceExhaustionOnSetCommand(set ConditionSet, target CombatantID, levels int) (ReduceExhaustionOnSetCommand, error) {
	if target.IsZero() {
		return ReduceExhaustionOnSetCommand{}, invalidInput("reduce exhaustion needs a target")
	}
	return ReduceExhaustionOnSetCommand{set, target, levels}, nil
}

// ReduceExhaustionOnSet is the use case that reduces Exhaustion on a ConditionSet
// through ConditionSet.ReduceExhaustion. Create it with NewReduceExhaustionOnSet;
// the zero value is silent and works. Safe for concurrent use.
type ReduceExhaustionOnSet struct{ conditionUse }

// NewReduceExhaustionOnSet returns the use case. Options: WithTelemetry,
// WithClock.
func NewReduceExhaustionOnSet(opts ...Option) ReduceExhaustionOnSet {
	return ReduceExhaustionOnSet{newConditionUse(opts)}
}

// Run applies cmd and records the shared signals plus
// srd5e.combat.condition.active (the effective-condition count of the returned
// set, kind exhaustion).
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
// Levels below 1 give an error matching srderr.ErrInvalidInput that is not a port
// failure: it is returned unchanged, the span records it, and neither
// srd5e.combat.operations, events, errors nor condition.active is recorded; the
// duration is. A creature without Exhaustion is an ExhaustionReductionNotPresent
// result, never an error.
func (u ReduceExhaustionOnSet) Run(ctx context.Context, cmd ReduceExhaustionOnSetCommand) (ReduceExhaustionOnSetResult, error) {
	if err := u.begin(ctx, cmd.target); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationReduceExhaustionOnSet, spanReduceExhaustionOnSet}
	res, err := run(ctx, u.in, op, func(context.Context) (ReduceExhaustionOnSetResult, error) {
		return cmd.set.ReduceExhaustion(cmd.target, cmd.levels)
	}, describeReduceExhaustion)
	if err != nil {
		return nil, err
	}
	u.recordActive(ctx, op, content.ConditionExhaustion, res.Set())
	return res, nil
}
