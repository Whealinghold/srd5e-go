package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollFreeCastingCommand is the input of RollFreeCasting.Run. Construct it with a
// keyed literal. Run does not alter it.
type RollFreeCastingCommand struct {
	Character Character
	Feats     TakenFeats
	// SlotLevel is the level of the slot being expended, 1 to 9.
	SlotLevel SlotLevel
}

// RollFreeCastingResult is the closed set of outcomes of RollFreeCasting.Run.
// Variants: RollFreeCastingResolved, RollFreeCastingNotApplicable,
// RollFreeCastingRejected.
type RollFreeCastingResult interface{ isRollFreeCastingResult() }

// RollFreeCastingResolved is the outcome of a roll of 1d4 for a slot of level 1 to
// 4. Only Run builds one.
type RollFreeCastingResolved struct {
	roll    int
	outcome FreeCastingOutcome
}

// Roll returns the roll of 1d4.
func (r RollFreeCastingResolved) Roll() int { return r.roll }

// Outcome returns whether the slot is spared.
func (r RollFreeCastingResolved) Outcome() FreeCastingOutcome { return r.outcome }

// RollFreeCastingNotApplicable is the outcome for a slot above level 4 (slotAboveFour):
// the benefit does not apply and nothing was rolled. Only Run builds one.
type RollFreeCastingNotApplicable struct{ level SlotLevel }

// SlotLevel returns the level of the slot, 5 to 9.
func (r RollFreeCastingNotApplicable) SlotLevel() SlotLevel { return r.level }

// RollFreeCastingRejected is the outcome when the character does not hold Boon of
// Spell Recall (FeatNotHeld); nothing was rolled. Construct it with a keyed literal.
type RollFreeCastingRejected struct{ Rejection FeatRejection }

func (RollFreeCastingResolved) isRollFreeCastingResult()      {}
func (RollFreeCastingNotApplicable) isRollFreeCastingResult() {}
func (RollFreeCastingRejected) isRollFreeCastingResult()      {}

// RollFreeCasting is the use case that rolls the Free Casting of Boon of Spell
// Recall (SRD 5.2.1, p. 88). It uses the dice port and telemetry only. Create it
// with NewRollFreeCasting; the zero value is not usable. It is safe for concurrent
// use only if its roller, telemetry and clock are.
type RollFreeCasting struct {
	roller Roller
	sig    operationSignals
}

// NewRollFreeCasting returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollFreeCasting(roller Roller, opts ...Option) (RollFreeCasting, error) {
	if roller == nil {
		return RollFreeCasting{}, invalidInput("roll free casting without a dice roller")
	}
	return RollFreeCasting{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Run checks in this order: an invalid slot level (an error), FeatNotHeld, a slot
// above level 4, and then makes one request of 1d4.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero Character or the zero SlotLevel, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the
// roller fails, panics or answers another expression (a roll outside 1 to 4 cannot be
// built); the failure is recorded once and never logged.
func (u RollFreeCasting) Run(ctx context.Context, cmd RollFreeCastingCommand) (RollFreeCastingResult, error) {
	if u.roller == nil || cmd.Character.IsZero() || cmd.SlotLevel.IsZero() {
		return nil, invalidInput("roll free casting: zero use case, character or slot level")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollFreeCasting, ValueOperationRollFreeCasting)
	defer run.end(ctx)
	rule := FreeCastingRule{}
	switch {
	case !cmd.Feats.Holds(featBoonOfSpellRecall):
		rj := FeatNotHeld{Feat: featBoonOfSpellRecall}
		run.finish(ctx, ValueResultRejected, rj.String())
		return RollFreeCastingRejected{Rejection: rj}, nil
	case !rule.Applies(cmd.SlotLevel):
		run.finish(ctx, ValueResultNotApplicable, "")
		return RollFreeCastingNotApplicable{level: cmd.SlotLevel}, nil
	}
	res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D4(), 0))
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	run.finish(ctx, ValueResultResolved, "")
	return RollFreeCastingResolved{roll: res.Total(), outcome: freeCastingOutcome(cmd.SlotLevel, res.Total())}, nil
}
