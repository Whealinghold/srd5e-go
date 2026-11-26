package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// RollBurningDamageCommand is the input of RollBurningDamage.Run: whether the
// creature is on fire. Build it with a keyed literal; the zero State is not burning.
type RollBurningDamageCommand struct{ State BurningState }

// RollBurningDamageResult is the closed set of outcomes of RollBurningDamage.Run.
// Variants: BurningDamageRolled, BurningDamageRejected.
type RollBurningDamageResult interface{ isRollBurningDamageResult() }

// BurningDamageRolled is the Fire damage of one turn of burning, full and unadjusted:
// the host passes it to ApplyDamage (or ApplyDamageToObject), where Immunity applies.
// The state stays burning.
type BurningDamageRolled struct {
	damage Damage
	roll   dice.Result
	state  BurningState
}

// Damage returns the Fire damage as rolled.
func (r BurningDamageRolled) Damage() Damage { return r.damage }

// Roll returns the 1d4 roll.
func (r BurningDamageRolled) Roll() dice.Result { return r.roll }

// State returns the state after the roll, BurningStateBurning: the fire does not go
// out by itself.
func (r BurningDamageRolled) State() BurningState { return r.state }

// BurningDamageRejected: the creature is not burning, so nothing was rolled.
// Construct it with a keyed literal.
type BurningDamageRejected struct{ Rejection BurningRejection }

func (BurningDamageRolled) isRollBurningDamageResult()   {}
func (BurningDamageRejected) isRollBurningDamageResult() {}

// RollBurningDamage is the use case that rolls the Fire damage a burning creature
// takes at the start of each of its turns (SRD 5.2.1, "Burning [Hazard]", p. 178). It
// uses the dice port. Create it with NewRollBurningDamage; the zero value is not
// usable.
type RollBurningDamage struct {
	environmentUse
	roller Roller
}

// NewRollBurningDamage returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a
// nil roller.
func NewRollBurningDamage(roller Roller, opts ...Option) (RollBurningDamage, error) {
	if roller == nil {
		return RollBurningDamage{}, invalidInput("burning damage without a dice roller")
	}
	return RollBurningDamage{newEnvironmentUse(opts), roller}, nil
}

// Run makes one request of 1d4 for a burning creature and records the shared signals
// plus, for a roll, srd5e.combat.hazard.damage_rolled with the amount as rolled.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case or a State that is not valid, and ctx.Err() unwrapped, before any signal,
// for a cancelled ctx. It returns a *srderr.PortError (error type "dice_roller_error")
// with a nil result when the roller fails, panics or answers another expression; the
// failure is recorded once and never logged. A creature that is not burning is a
// BurningDamageRejected result, with no roll.
func (u RollBurningDamage) Run(ctx context.Context, cmd RollBurningDamageCommand) (RollBurningDamageResult, error) {
	if u.roller == nil || !cmd.State.Valid() {
		return nil, invalidInput("zero use case or invalid burning state")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationRollBurningDamage, spanRollBurningDamage},
		func(context.Context) (RollBurningDamageResult, error) {
			if cmd.State != BurningStateBurning {
				return BurningDamageRejected{BurningRejectionNotBurning}, nil
			}
			roll, err := dice.Roll(u.roller, burningDamageDie())
			if err != nil {
				return nil, err
			}
			return BurningDamageRolled{MustDamage(roll.Total(), content.DamageFire), roll, BurningStateBurning}, nil
		},
		func(r RollBurningDamageResult) description {
			if v, ok := r.(BurningDamageRejected); ok {
				return description{Class: resultRefused, Name: v.Rejection.String()}
			}
			return description{Class: resultApplied, Name: valueResultDamageRolled}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(BurningDamageRolled); ok {
		u.recordDamageRolled(ctx, HazardBurning, v.damage.amount)
	}
	return res, nil
}
