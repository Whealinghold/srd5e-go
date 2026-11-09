package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseLifedrinkerCommand is the input of UseLifedrinker.Run. Attack, Turn and DamageType are the facts of the hit; Choice says whether a Hit
// Point Die is expended, and then Pool and Spent are the Warlock's Hit Point Dice and the ones spent, Die the size expended and Constitution
// the Constitution modifier. Construct it with a keyed literal. Run does not alter it.
type UseLifedrinkerCommand struct {
	Character    Character
	Attack       WarlockAttackKind
	Turn         WarlockTurnFacts
	DamageType   content.DamageType
	Choice       HitDieChoice
	Pool         HitDicePool
	Spent        SpentHitDice
	Die          content.HitPointDie
	Constitution ability.Modifier
}

// UseLifedrinkerResult is the closed set of outcomes of UseLifedrinker.Run. Variants: UseLifedrinkerApplied, UseLifedrinkerRejected.
type UseLifedrinkerResult interface{ isUseLifedrinkerResult() }

// UseLifedrinkerApplied is the outcome of a use the rules allowed.
type UseLifedrinkerApplied struct {
	character Character
	damage    dice.Expression
	healed    int
	hasHealed bool
	spent     SpentHitDice
	events    []Event
}

// Character returns the new Character.
func (r UseLifedrinkerApplied) Character() Character { return r.character }

// ExtraDamage returns the 1d6 of extra damage for the host to roll and apply.
func (r UseLifedrinkerApplied) ExtraDamage() dice.Expression { return r.damage }

// HitPointsRegained returns the Hit Points regained and true when a Hit Point Die was expended.
func (r UseLifedrinkerApplied) HitPointsRegained() (int, bool) { return r.healed, r.hasHealed }

// SpentHitDice returns the spent dice after the use; the command's own when no die was expended.
func (r UseLifedrinkerApplied) SpentHitDice() SpentHitDice { return r.spent }

// Events returns a new slice holding the LifedrinkerUsed event.
func (r UseLifedrinkerApplied) Events() []Event { return slices.Clone(r.events) }

// UseLifedrinkerRejected is the outcome of a use a rule refused: the Character is unchanged and the roller was not called.
type UseLifedrinkerRejected struct{ Rejection WarlockRejection }

func (UseLifedrinkerApplied) isUseLifedrinkerResult()  {}
func (UseLifedrinkerRejected) isUseLifedrinkerResult() {}

// UseLifedrinker is the use case that uses Lifedrinker (SRD 5.2.1, "Lifedrinker", p. 73; class-warlock.md FR-077, FR-078, FR-149): it rolls
// the Hit Point Die through the dice port only when a die is expended and the use is allowed. It reads no content. Create it with
// NewUseLifedrinker; the zero value is not usable.
type UseLifedrinker struct {
	warlockUse
	roller Roller
}

// NewUseLifedrinker returns the use case. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a nil
// roller.
func NewUseLifedrinker(roller Roller, opts ...Option) (UseLifedrinker, error) {
	if roller == nil {
		return UseLifedrinker{}, invalidInput("use Lifedrinker without a dice roller")
	}
	u, err := newWarlockUse("use Lifedrinker", nil, false, opts)
	return UseLifedrinker{warlockUse: u, roller: roller}, err
}

// Run uses the feature. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character,
// and, recorded on the span, for an invalid Hit Point Die when one is expended; ctx.Err() unwrapped, before any signal, for a cancelled ctx;
// and a *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract; the
// Character is unchanged and the failure is recorded once and never logged. A refusal is a UseLifedrinkerRejected result and the roller is
// not called.
func (u UseLifedrinker) Run(ctx context.Context, cmd UseLifedrinkerCommand) (UseLifedrinkerResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanUseLifedrinker, ValueFeatureLifedrinker, cmd.Character, false)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return UseLifedrinkerRejected{Rejection: rej}, nil
	}
	apply := func(rolled int) (WarlockTransition, error) {
		return ApplyLifedrinker(cmd.Character.ID(), call.state, cmd.Attack, cmd.Turn, cmd.DamageType, cmd.Choice, cmd.Pool, cmd.Spent, cmd.Die, cmd.Constitution, rolled)
	}
	// A rolled value of 1 stands in for the die while the rules decide, so a refusal never reaches the roller.
	t, err := apply(1)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	if _, refused := t.(WarlockRefused); !refused && cmd.Choice == HitDieExpend {
		rolled, err := rollOne(u.roller, dice.MustDie(cmd.Die.Faces()))
		if err != nil {
			call.run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		if t, err = apply(rolled); err != nil { // unreachable: dice.Roll answers 1 to the faces of the die
			call.run.commandError(err)
			return nil, err
		}
	}
	next, changed, rej, err := call.commit(ctx, cmd.Character, t)
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return UseLifedrinkerRejected{Rejection: rej}, nil
	}
	damage, _ := changed.Damage()
	healed, hasHealed := changed.HitPointsRegained()
	spent, hasSpent := changed.SpentHitDice()
	if !hasSpent {
		spent = cmd.Spent
	}
	return UseLifedrinkerApplied{character: next, damage: damage, healed: healed, hasHealed: hasHealed, spent: spent, events: changed.Events()}, nil
}
