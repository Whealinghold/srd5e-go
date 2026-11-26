package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/dice"
)

// RollStableRecoveryDelayCommand is the input of RollStableRecoveryDelay.Run:
// nothing, because the delay belongs to no creature.
type RollStableRecoveryDelayCommand struct{}

// RollStableRecoveryDelayResult is the closed set of outcomes of
// RollStableRecoveryDelay.Run. Variants: StableRecoveryDelayRolled.
type RollStableRecoveryDelayResult interface{ isRollStableRecoveryDelayResult() }

// StableRecoveryDelayRolled is the delay that was rolled. Only Run builds one.
type StableRecoveryDelayRolled struct{ delay StableRecoveryDelay }

// Delay returns the delay, 1 to 4 hours.
func (r StableRecoveryDelayRolled) Delay() StableRecoveryDelay { return r.delay }

func (StableRecoveryDelayRolled) isRollStableRecoveryDelayResult() {}

// RollStableRecoveryDelay is the use case that rolls how long a Stable creature
// waits before it regains 1 Hit Point (SRD 5.2.1, "Stabilizing a Character", p.
// 18). It uses the dice port. Create it with NewRollStableRecoveryDelay; the zero
// value is not usable.
type RollStableRecoveryDelay struct {
	in     instrumenter
	roller Roller
}

// NewRollStableRecoveryDelay returns the use case rolling through roller.
// Options: WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollStableRecoveryDelay(roller Roller, opts ...Option) (RollStableRecoveryDelay, error) {
	if roller == nil {
		return RollStableRecoveryDelay{}, invalidInput("stable recovery delay without a dice roller")
	}
	o := newOptions(opts)
	return RollStableRecoveryDelay{newInstrumenter(o.tel, o.clock), roller}, nil
}

// Run makes one request of 1d4 and returns it as the delay in hours.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or answers another expression; the
// failure is recorded once and never logged.
func (u RollStableRecoveryDelay) Run(ctx context.Context, _ RollStableRecoveryDelayCommand) (RollStableRecoveryDelayResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationRollStableRecoveryDelay, spanRollStableRecoveryDelay},
		func(context.Context) (RollStableRecoveryDelayResult, error) {
			res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D4(), 0))
			if err != nil {
				return nil, err
			}
			delay, err := NewStableRecoveryDelay(res.Total())
			if err != nil {
				return nil, err
			}
			return StableRecoveryDelayRolled{delay}, nil
		},
		func(RollStableRecoveryDelayResult) description {
			return description{Class: resultApplied, Name: valueResultRolled}
		})
}
