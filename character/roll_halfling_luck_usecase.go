package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollHalflingLuckCommand is the input of RollHalflingLuck.Run: the roll, the index of
// the die to reroll and the indexes already rerolled by an effect. Build it with
// NewRollHalflingLuckCommand; the zero value is invalid. Run does not alter it.
type RollHalflingLuckCommand struct {
	character Character
	traits    SpeciesTraits
	roll      d20.Roll
	dieIndex  int
	rerolled  []int
}

// NewRollHalflingLuckCommand returns the command. rerolled is copied.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or roll, a
// die index outside the roll, or an index in rerolled outside the roll. It never panics.
func NewRollHalflingLuckCommand(c Character, t SpeciesTraits, roll d20.Roll, dieIndex int, rerolled []int) (RollHalflingLuckCommand, error) {
	n := len(roll.Dice())
	if c.IsZero() || roll.IsZero() || dieIndex < 0 || dieIndex >= n || slices.ContainsFunc(rerolled, func(i int) bool { return i < 0 || i >= n }) {
		return RollHalflingLuckCommand{}, invalidInput("roll halfling luck: zero character or roll, or a die index outside the roll")
	}
	return RollHalflingLuckCommand{character: c, traits: t, roll: roll, dieIndex: dieIndex, rerolled: slices.Clone(rerolled)}, nil
}

// RollHalflingLuckResult is the closed set of outcomes of RollHalflingLuck.Run.
// Variants: RollHalflingLuckRerolled, RollHalflingLuckNotApplicable,
// RollHalflingLuckRejected.
type RollHalflingLuckResult interface{ isRollHalflingLuckResult() }

// RollHalflingLuckRerolled is the outcome of a reroll: the new face, the new Roll
// (which must be used, even when lower), the indexes rerolled with the die added, and
// the event. Only Run builds one.
type RollHalflingLuckRerolled struct {
	newValue int
	roll     d20.Roll
	rerolled []int
	event    HalflingLuckUsed
}

// NewValue returns the face of the new d20.
func (r RollHalflingLuckRerolled) NewValue() int { return r.newValue }

// Roll returns the roll with the die replaced.
func (r RollHalflingLuckRerolled) Roll() d20.Roll { return r.roll }

// RerolledIndexes returns a new slice of the indexes rerolled; pass it back for a
// later reroll of the same roll so that no die is rerolled twice.
func (r RollHalflingLuckRerolled) RerolledIndexes() []int { return slices.Clone(r.rerolled) }

// Event returns the HalflingLuckUsed event.
func (r RollHalflingLuckRerolled) Event() HalflingLuckUsed { return r.event }

// RollHalflingLuckNotApplicable is the outcome for a die that does not show a 1:
// nothing was rolled (a normal outcome, not a rejection). Only Run builds one.
type RollHalflingLuckNotApplicable struct{ face int }

// Face returns the face the die shows.
func (r RollHalflingLuckNotApplicable) Face() int { return r.face }

// RollHalflingLuckRejected is the outcome of a reroll a rule refused. Construct it
// with a keyed literal.
type RollHalflingLuckRejected struct{ Rejection SpeciesTraitRejection }

func (RollHalflingLuckRerolled) isRollHalflingLuckResult()      {}
func (RollHalflingLuckNotApplicable) isRollHalflingLuckResult() {}
func (RollHalflingLuckRejected) isRollHalflingLuckResult()      {}

// RollHalflingLuck is the use case of the Halfling trait Luck (SRD 5.2.1, p. 86). It
// uses the dice port and telemetry only. Create it with NewRollHalflingLuck; the zero
// value is not usable. It is safe for concurrent use only if its roller, telemetry and
// clock are.
type RollHalflingLuck struct {
	roller Roller
	sig    operationSignals
}

// NewRollHalflingLuck returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollHalflingLuck(roller Roller, opts ...Option) (RollHalflingLuck, error) {
	if roller == nil {
		return RollHalflingLuck{}, invalidInput("roll halfling luck without a dice roller")
	}
	return RollHalflingLuck{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Run checks, in this order: the zero SpeciesTraits (SpeciesBehaviourUnknown), a
// species that is not a Halfling (SpeciesUseNotOffered), a die already rerolled
// (DieAlreadyRerolled) and a die that does not show a 1 (RollHalflingLuckNotApplicable,
// no roll); then it makes one request of 1d20. Luck has no use limit and no state.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case or a zero command, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. It returns a *srderr.PortError (error type "dice_roller_error")
// with a nil result when the roller fails, panics or answers another expression; the
// failure is recorded once and never logged.
func (u RollHalflingLuck) Run(ctx context.Context, cmd RollHalflingLuckCommand) (RollHalflingLuckResult, error) {
	if u.roller == nil || cmd.character.IsZero() {
		return nil, invalidInput("roll halfling luck: zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollHalflingLuck, ValueOperationRollHalflingLuck)
	defer run.end(ctx)
	reject := func(r SpeciesTraitRejection) (RollHalflingLuckResult, error) {
		run.finish(ctx, ValueResultRejected, r.String())
		return RollHalflingLuckRejected{Rejection: r}, nil
	}
	face := cmd.roll.Dice()[cmd.dieIndex]
	switch {
	case cmd.traits.IsZero():
		return reject(SpeciesBehaviourUnknown{})
	case cmd.traits.species != speciesHalfling:
		return reject(SpeciesUseNotOffered{})
	case slices.Contains(cmd.rerolled, cmd.dieIndex):
		return reject(DieAlreadyRerolled{Index: cmd.dieIndex})
	case !slices.Contains(HalflingLuck{}.EligibleDice(cmd.roll, cmd.rerolled), cmd.dieIndex):
		run.finish(ctx, ValueResultNotApplicable, "")
		return RollHalflingLuckNotApplicable{face: face}, nil
	}
	res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D20(), 0))
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	rolled, err := cmd.roll.ApplyReroll(d20.RerollDie{Index: cmd.dieIndex, Face: res.Total()})
	if err != nil { // unreachable: dice.Roll vouches for a face of 1 to 20
		err = portViolation(dice.Port, "roll", "a d20 face outside 1 to 20")
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	run.finish(ctx, ValueResultResolved, "")
	return RollHalflingLuckRerolled{newValue: res.Total(), roll: rolled, rerolled: append(slices.Clone(cmd.rerolled), cmd.dieIndex),
		event: HalflingLuckUsed{id: cmd.character.ID(), index: cmd.dieIndex, face: res.Total()}}, nil
}
