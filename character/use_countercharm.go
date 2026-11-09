package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// UseCountercharmCommand is the input of UseCountercharm.Run: the Bard, the saving throw that was failed (by the Bard or by
// a creature within 30 feet of it), the circumstances that applied to that save and the conditions the effect applies.
// Construct it with a keyed literal. Run does not alter it.
type UseCountercharmCommand struct {
	Character     Character
	Failed        d20.CheckResult
	Circumstances []modifier.RollCircumstance
	Conditions    []content.ConditionKind
}

// UseCountercharmResult is the closed set of outcomes of UseCountercharm.Run. Variants: UseCountercharmRerolled,
// UseCountercharmRejected.
type UseCountercharmResult interface{ isUseCountercharmResult() }

// UseCountercharmRerolled is the outcome of a use: the save was rerolled with Advantage. The Bard's pool and spell slots
// are untouched, so there is no new Character. Only Run builds one.
type UseCountercharmRerolled struct{ resolved CountercharmResolved }

// Failed returns the original, failed saving throw.
func (r UseCountercharmRerolled) Failed() d20.CheckResult { return r.resolved.failed }

// Rerolled returns the new saving throw, made with the original modifiers against the original DC.
func (r UseCountercharmRerolled) Rerolled() d20.CheckResult { return r.resolved.rerolled }

// UseCountercharmRejected is the outcome of a use a rule refused: nothing was rolled. Construct it with a keyed literal.
type UseCountercharmRejected struct{ Reason BardReason }

func (UseCountercharmRerolled) isUseCountercharmResult() {}
func (UseCountercharmRejected) isUseCountercharmResult() {}

// UseCountercharm is the use case that uses a Bard's Countercharm (SRD 5.2.1, "Level 7: Countercharm", p. 33): it plans the
// reroll, rolls the d20s through the dice port only for a use that is allowed, resolves the new save and emits the
// srd5e.character.feature_used signals. It calls no other use case and emits no srd5e.d20_test signal. The Reaction cost is
// the host's. It holds no rules logic. Create it with NewUseCountercharm; the zero value is not usable. Safe for concurrent
// use only if its roller, telemetry and clock are.
type UseCountercharm struct{ bardUse }

// NewUseCountercharm returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewUseCountercharm(roller Roller, opts ...Option) (UseCountercharm, error) {
	u, err := newBardUse("use Countercharm", roller, opts)
	return UseCountercharm{u}, err
}

// Run uses Countercharm for cmd.Character on cmd.Failed. The reasons are checked in the order of PlanCountercharm.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or a zero
// saving throw result, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract;
// the failure is recorded once and never logged. A refused use is a result.
func (u UseCountercharm) Run(ctx context.Context, cmd UseCountercharmCommand) (UseCountercharmResult, error) {
	ctx, run, err := u.start(ctx, cmd.Character.IsZero() || cmd.Failed.Roll().IsZero(), SpanUseCountercharm, ValueFeatureCountercharm)
	if err != nil {
		return nil, err
	}
	defer run.end(ctx)
	level, _ := bardLevelOf(cmd.Character)
	plan, reason, err := PlanCountercharm(level, cmd.Failed, cmd.Circumstances, cmd.Conditions)
	if err != nil { // unreachable: the result was checked above
		run.commandError(err)
		return nil, err
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return UseCountercharmRejected{Reason: reason}, nil
	}
	faces, err := rollDiceValues(u.roller, plan.DiceCount(), dice.D20())
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	resolved, err := ResolveCountercharm(plan, faces)
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	run.succeeded(ctx)
	return UseCountercharmRerolled{resolved: resolved}, nil
}
