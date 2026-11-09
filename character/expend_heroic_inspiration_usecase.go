package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ExpendHeroicInspirationCommand is the input of ExpendHeroicInspiration.Expend:
// the character and the one die to reroll. It carries no original value.
// Construct it with a keyed literal. Expend does not alter it.
type ExpendHeroicInspirationCommand struct {
	Character Character
	Target    RerollTarget
}

// ExpendHeroicInspirationResult is the closed set of outcomes of
// ExpendHeroicInspiration.Expend. Variants: ExpendHeroicInspirationRerolled,
// ExpendHeroicInspirationRejected.
type ExpendHeroicInspirationResult interface{ isExpendHeroicInspirationResult() }

// ExpendHeroicInspirationRerolled: Heroic Inspiration was spent and the die
// rolled again.
type ExpendHeroicInspirationRerolled struct {
	character Character
	rerolled  RerolledDie
	events    []Event
}

// Character returns the character after the expenditure, no longer Inspired.
func (r ExpendHeroicInspirationRerolled) Character() Character { return r.character }

// Rerolled returns the new roll, which must be used.
func (r ExpendHeroicInspirationRerolled) Rerolled() RerolledDie { return r.rerolled }

// Events returns a new slice holding the one HeroicInspirationExpended event.
func (r ExpendHeroicInspirationRerolled) Events() []Event { return slices.Clone(r.events) }

// ExpendHeroicInspirationRejected: the character was not Inspired; the roller
// was not called and nothing changed.
type ExpendHeroicInspirationRejected struct{ reason HeroicInspirationRejection }

// Reason returns the aggregate's rejection, HeroicInspirationNotAvailable.
func (r ExpendHeroicInspirationRejected) Reason() HeroicInspirationRejection { return r.reason }

func (ExpendHeroicInspirationRerolled) isExpendHeroicInspirationResult() {}
func (ExpendHeroicInspirationRejected) isExpendHeroicInspirationResult() {}

// ExpendHeroicInspiration is the use case that spends Heroic Inspiration to
// reroll one die: it delegates the state change to
// Character.ExpendHeroicInspiration, rolls one die of the target's size through
// the dice port, and emits the srd5e.character.expend_heroic_inspiration span
// and the shared character signals. It holds no rules logic. Create it with
// NewExpendHeroicInspiration; the zero value is not usable. It is safe for
// concurrent use only if its roller, telemetry and clock are.
type ExpendHeroicInspiration struct {
	roller Roller
	sig    operationSignals
}

// NewExpendHeroicInspiration returns the use case rolling through roller.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an
// error matching srderr.ErrInvalidInput for a nil roller.
func NewExpendHeroicInspiration(roller Roller, opts ...Option) (ExpendHeroicInspiration, error) {
	if roller == nil {
		return ExpendHeroicInspiration{}, invalidInput("expend Heroic Inspiration without a dice roller")
	}
	return ExpendHeroicInspiration{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Expend spends the character's Heroic Inspiration and rolls the target die
// once. A character who is not Inspired gets an ExpendHeroicInspirationRejected
// result and the roller is not called.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case, a zero Character or a zero or unbuilt Target, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It
// returns a *srderr.PortError (error type "dice_roller_error") with a nil result
// when the roller fails, panics or breaks its contract; nothing is spent, and
// the failure is recorded once and never logged.
func (u ExpendHeroicInspiration) Expend(ctx context.Context, cmd ExpendHeroicInspirationCommand) (ExpendHeroicInspirationResult, error) {
	if u.roller == nil || cmd.Character.IsZero() || cmd.Target.IsZero() {
		return nil, invalidInput("expend Heroic Inspiration: zero use case, character or target")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanExpendHeroicInspiration, ValueOperationExpend)
	defer run.end(ctx)
	t, ok := cmd.Character.ExpendHeroicInspiration().(TransitionChanged[HeroicInspirationRejection])
	if !ok {
		run.finish(ctx, ValueResultRejected, ValueRejectionNotInspired)
		return ExpendHeroicInspirationRejected{reason: HeroicInspirationNotAvailable{}}, nil
	}
	face, err := u.roll(cmd.Target.Die())
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	run.finish(ctx, ValueResultExpended, "")
	return ExpendHeroicInspirationRerolled{
		character: t.Character(), events: t.events,
		rerolled: RerolledDie{die: cmd.Target.Die(), position: cmd.Target.Position(), newValue: face},
	}, nil
}

// roll rolls one die once and returns the face. A roller failure, panic or
// broken contract comes back as a *srderr.PortError.
func (u ExpendHeroicInspiration) roll(die dice.Die) (int, error) {
	expr, err := dice.NewExpression(1, die, 0)
	if err != nil { // unreachable: the command holds a valid die
		return 0, err
	}
	res, err := dice.Roll(u.roller, expr)
	if err != nil {
		return 0, err
	}
	return res.DiceTotal(), nil
}
