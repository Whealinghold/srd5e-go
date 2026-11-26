package combat

import "context"

// ApplyConditionToSetCommand is the input of ApplyConditionToSet.Run. Build it
// with NewApplyConditionToSetCommand; the zero value is invalid.
type ApplyConditionToSetCommand struct {
	set       ConditionSet
	target    CombatantID
	condition Condition
}

// NewApplyConditionToSetCommand returns the command to apply condition to the
// creature target, whose conditions are set. It returns an error matching
// srderr.ErrInvalidInput for a zero target or an invalid condition.
func NewApplyConditionToSetCommand(set ConditionSet, target CombatantID, condition Condition) (ApplyConditionToSetCommand, error) {
	if target.IsZero() || !validCondition(condition) {
		return ApplyConditionToSetCommand{}, invalidInput("apply condition needs a target and a valid condition")
	}
	return ApplyConditionToSetCommand{set, target, condition}, nil
}

// ApplyConditionToSet is the use case that applies a condition to a ConditionSet
// through ConditionSet.Apply. Create it with NewApplyConditionToSet; the zero value
// is silent and works. Safe for concurrent use.
type ApplyConditionToSet struct{ conditionUse }

// NewApplyConditionToSet returns the use case. Options: WithTelemetry, WithClock.
func NewApplyConditionToSet(opts ...Option) ApplyConditionToSet {
	return ApplyConditionToSet{newConditionUse(opts)}
}

// Run applies cmd and records the shared signals plus
// srd5e.combat.condition.active (the effective-condition count of the returned
// set, with the kind applied).
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
// A refusal is a ConditionAlreadyHeld, ConditionImmune or ExhaustionAtMaximum
// result, never an error.
func (u ApplyConditionToSet) Run(ctx context.Context, cmd ApplyConditionToSetCommand) (ApplyConditionToSetResult, error) {
	if err := u.begin(ctx, cmd.target); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationApplyConditionToSet, spanApplyConditionToSet}
	res, err := run(ctx, u.in, op, func(context.Context) (ApplyConditionToSetResult, error) {
		return cmd.set.Apply(cmd.target, cmd.condition), nil
	}, describeApplyCondition)
	if err != nil {
		return nil, err
	}
	u.recordActive(ctx, op, cmd.condition.Kind(), res.Set())
	return res, nil
}
