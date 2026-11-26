package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ResolveFallCommand is the input of ResolveFall.Run: how far the creature falls and
// what it lands on. Build it with a keyed literal; the zero Surface is FallOntoGround.
type ResolveFallCommand struct {
	Distance FallDistance
	Surface  FallSurface
}

// ResolveFallResult is the closed set of outcomes of ResolveFall.Run. Variants:
// FallDamageRolled, FallWithoutDamage.
type ResolveFallResult interface{ isResolveFallResult() }

// FallDamageRolled is the Bludgeoning damage of a fall, full and unadjusted. The host
// passes the damage, with the adjustments, to ApplyDamage (or ApplyDamageToObject).
type FallDamageRolled struct {
	damage      Damage
	adjustments []DamageAdjustment
	roll        dice.Result
	diceCount   int
}

// Damage returns the Bludgeoning damage as rolled.
func (r FallDamageRolled) Damage() Damage { return r.damage }

// Adjustments returns a copy of the adjustments for ApplyDamage: HalfDamage after a
// landing check made in liquid, otherwise none.
func (r FallDamageRolled) Adjustments() []DamageAdjustment { return slices.Clone(r.adjustments) }

// Roll returns the roll of the d6.
func (r FallDamageRolled) Roll() dice.Result { return r.roll }

// DiceCount returns the number of d6 rolled, 1 to 20.
func (r FallDamageRolled) DiceCount() int { return r.diceCount }

// FallWithoutDamage: the fall was under 10 feet, so nothing was rolled and the creature
// does not land Prone. The zero value is the result.
type FallWithoutDamage struct{}

func (FallDamageRolled) isResolveFallResult()  {}
func (FallWithoutDamage) isResolveFallResult() {}

// ResolveFall is the use case that rolls the damage of a fall: 1d6 Bludgeoning per
// full 10 feet, at most 20d6 (SRD 5.2.1, "Falling [Hazard]", p. 182). It uses the dice
// port. Create it with NewResolveFall; the zero value is not usable.
type ResolveFall struct {
	environmentUse
	roller Roller
}

// NewResolveFall returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a
// nil roller.
func NewResolveFall(roller Roller, opts ...Option) (ResolveFall, error) {
	if roller == nil {
		return ResolveFall{}, invalidInput("fall without a dice roller")
	}
	return ResolveFall{newEnvironmentUse(opts), roller}, nil
}

// Run rolls one request of Nd6 for a fall of 10 feet or more and records the shared
// signals plus, for a roll, srd5e.combat.hazard.damage_rolled with the amount as rolled
// (before any HalfDamage).
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case, a Surface that is not a known variant, and ctx.Err()
// unwrapped, before any signal, for a cancelled ctx. It returns a *srderr.PortError
// (error type "dice_roller_error") with a nil result when the roller fails, panics or
// answers another expression; the failure is recorded once and never logged. A fall
// under 10 feet is FallWithoutDamage, whatever the surface.
func (u ResolveFall) Run(ctx context.Context, cmd ResolveFallCommand) (ResolveFallResult, error) {
	switch s := cmd.Surface.(type) {
	case nil, FallOntoGround, FallIntoLiquid:
	default:
		return nil, invalidInput("fall surface is not a known variant: " + typeName(s))
	}
	if u.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationResolveFall, spanResolveFall},
		func(context.Context) (ResolveFallResult, error) {
			n := cmd.Distance.Dice()
			if n == 0 {
				return FallWithoutDamage{}, nil
			}
			roll, err := dice.Roll(u.roller, fallExpression(n))
			if err != nil {
				return nil, err
			}
			return FallDamageRolled{MustDamage(roll.Total(), content.DamageBludgeoning), fallAdjustments(cmd.Surface), roll, n}, nil
		},
		func(r ResolveFallResult) description {
			if _, ok := r.(FallWithoutDamage); ok {
				return description{Class: resultApplied, Name: valueResultNoDamage}
			}
			return description{Class: resultApplied, Name: valueResultDamageRolled}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(FallDamageRolled); ok {
		u.recordDamageRolled(ctx, HazardFalling, v.damage.amount)
	}
	return res, nil
}
