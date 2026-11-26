package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// RemoveConditionFromSetCommand is the input of RemoveConditionFromSet.Run. Build
// it with NewRemoveConditionFromSetCommand; the zero value is invalid.
type RemoveConditionFromSetCommand struct {
	set    ConditionSet
	target CombatantID
	kind   content.ConditionKind
}

// NewRemoveConditionFromSetCommand returns the command to remove the condition of
// kind from the creature target, whose conditions are set. It returns an error
// matching srderr.ErrInvalidInput for a zero target or an invalid kind.
func NewRemoveConditionFromSetCommand(set ConditionSet, target CombatantID, kind content.ConditionKind) (RemoveConditionFromSetCommand, error) {
	if target.IsZero() || !kind.Valid() {
		return RemoveConditionFromSetCommand{}, invalidInput("remove condition needs a target and a valid kind")
	}
	return RemoveConditionFromSetCommand{set, target, kind}, nil
}

// RemoveConditionFromSet is the use case that removes a condition from a
// ConditionSet through ConditionSet.Remove. Create it with
// NewRemoveConditionFromSet; the zero value is silent and works. Safe for
// concurrent use.
type RemoveConditionFromSet struct{ conditionUse }

// NewRemoveConditionFromSet returns the use case. Options: WithTelemetry,
// WithClock.
func NewRemoveConditionFromSet(opts ...Option) RemoveConditionFromSet {
	return RemoveConditionFromSet{newConditionUse(opts)}
}

// Run applies cmd and records the shared signals plus
// srd5e.combat.condition.active (the effective-condition count of the returned
// set, with the kind removed).
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
// A refusal is a ConditionRemovalNotPresent or ConditionRemovalImpliedBy result,
// never an error.
func (u RemoveConditionFromSet) Run(ctx context.Context, cmd RemoveConditionFromSetCommand) (RemoveConditionFromSetResult, error) {
	if err := u.begin(ctx, cmd.target); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationRemoveConditionFromSet, spanRemoveConditionFromSet}
	res, err := run(ctx, u.in, op, func(context.Context) (RemoveConditionFromSetResult, error) {
		return cmd.set.Remove(cmd.target, cmd.kind), nil
	}, describeRemoveCondition)
	if err != nil {
		return nil, err
	}
	u.recordActive(ctx, op, cmd.kind, res.Set())
	return res, nil
}
