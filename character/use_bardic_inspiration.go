package character

import (
	"context"
	"time"
)

// UseBardicInspirationCommand is the input of UseBardicInspiration.Run: the creature holding the die, the die, the time
// elapsed since it was granted (the host's, the domain reads no clock) and the D20 Test the creature failed. Construct it
// with a keyed literal. Run does not alter it.
type UseBardicInspirationCommand struct {
	ID      Identifier
	Grant   BardicInspirationGrant
	Elapsed time.Duration
	Failed  AdjustableD20Result
}

// UseBardicInspirationResult is the closed set of outcomes of UseBardicInspiration.Run. Variants:
// UseBardicInspirationApplied, UseBardicInspirationRejected.
type UseBardicInspirationResult interface{ isUseBardicInspirationResult() }

// UseBardicInspirationApplied is the outcome of a use: the die was rolled and added to the failed test, and it is expended
// whether or not the test now succeeds. Only Run builds one.
type UseBardicInspirationApplied struct{ applied BardicInspirationApplied }

// Applied returns the adjusted test and the event.
func (r UseBardicInspirationApplied) Applied() BardicInspirationApplied { return r.applied }

// Events returns a new slice holding the BardicInspirationExpended event.
func (r UseBardicInspirationApplied) Events() []Event { return []Event{r.applied.event} }

// UseBardicInspirationRejected is the outcome of a use a rule refused: nothing was rolled. For BardExpired the die is
// lost: Discard is the grant to discard and Expired the event; both are zero for BardNotAFailure, where the die stays.
// Construct it with a keyed literal.
type UseBardicInspirationRejected struct {
	Reason  BardReason
	Discard BardicInspirationGrant
	Expired *BardicInspirationExpired
}

func (UseBardicInspirationApplied) isUseBardicInspirationResult()  {}
func (UseBardicInspirationRejected) isUseBardicInspirationResult() {}

// UseBardicInspiration is the use case that uses a Bardic Inspiration die a creature holds (SRD 5.2.1, "Using Bardic
// Inspiration", pp. 31-32): it plans the use, rolls the die through the dice port only for a use that is allowed, applies it
// and emits the srd5e.character.feature_used signals. The recipient need not be a Bard. It holds no rules logic. Create it
// with NewUseBardicInspiration; the zero value is not usable. Safe for concurrent use only if its roller, telemetry and clock are.
type UseBardicInspiration struct{ bardUse }

// NewUseBardicInspiration returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an
// error matching srderr.ErrInvalidInput for a nil roller.
func NewUseBardicInspiration(roller Roller, opts ...Option) (UseBardicInspiration, error) {
	u, err := newBardUse("use Bardic Inspiration", roller, opts)
	return UseBardicInspiration{u}, err
}

// Run uses the die of cmd.Grant on cmd.Failed. The reasons are checked in the order of PlanBardicInspirationUse.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero grant or a nil
// result, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns an error matching
// srderr.ErrInvalidInput for a zero result, and a *srderr.PortError (error type "dice_roller_error") with a nil result when
// the roller fails, panics or breaks its contract; the die remains unexpended and the failure is recorded once and never
// logged. A refused use is a result.
func (u UseBardicInspiration) Run(ctx context.Context, cmd UseBardicInspirationCommand) (UseBardicInspirationResult, error) {
	ctx, run, err := u.start(ctx, cmd.Grant.IsZero() || cmd.Failed == nil, SpanUseBardicInspiration, ValueFeatureBardicInspiration)
	if err != nil {
		return nil, err
	}
	defer run.end(ctx)
	plan, reason, err := PlanBardicInspirationUse(cmd.ID, cmd.Grant, cmd.Elapsed, cmd.Failed)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		rj := UseBardicInspirationRejected{Reason: reason}
		if reason == BardExpired {
			rj.Discard = cmd.Grant
			rj.Expired = &BardicInspirationExpired{id: cmd.ID, die: cmd.Grant.die}
		}
		return rj, nil
	}
	rolled, err := rollOne(u.roller, plan.die.Die())
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	applied, err := ApplyBardicInspirationDie(plan, rolled)
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	run.succeeded(ctx)
	return UseBardicInspirationApplied{applied: applied}, nil
}
