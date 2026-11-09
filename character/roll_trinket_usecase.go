package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollTrinketCommand is the input of RollTrinket.Roll: the host-supplied table to
// roll on. Construct it with a keyed literal. Roll does not alter it.
type RollTrinketCommand struct{ Table TrinketTable }

// RollTrinketResult is the closed set of outcomes of RollTrinket.Roll. Variants:
// RollTrinketRolled.
type RollTrinketResult interface{ isRollTrinketResult() }

// RollTrinketRolled is the outcome of a roll of the 1d100: the roll, 1 to 100 (a
// roll of 100 is the table's 00), and the entry it matches. Only Roll builds one.
type RollTrinketRolled struct {
	roll    int
	trinket Trinket
}

// Roll returns the roll of the 1d100, 1 to 100.
func (r RollTrinketRolled) Roll() int { return r.roll }

// Trinket returns the matching entry.
func (r RollTrinketRolled) Trinket() Trinket { return r.trinket }

func (RollTrinketRolled) isRollTrinketResult() {}

// RollTrinket is the use case that rolls once on the Trinkets table (SRD 5.2.1,
// "Trinkets", pp. 26-27). It needs no Character and keeps no record of earlier
// rolls. It uses the dice port and telemetry only. Create it with NewRollTrinket;
// the zero value is not usable. It is safe for concurrent use only if its roller,
// telemetry and clock are.
type RollTrinket struct {
	roller Roller
	sig    operationSignals
}

// NewRollTrinket returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollTrinket(roller Roller, opts ...Option) (RollTrinket, error) {
	if roller == nil {
		return RollTrinket{}, invalidInput("roll a trinket without a dice roller")
	}
	return RollTrinket{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Roll rolls exactly one 1d100 and returns the matching entry of cmd.Table.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or the zero table, and ctx.Err() unwrapped, before any signal,
// when ctx is already cancelled. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers
// another expression; the failure is recorded once and never logged.
func (u RollTrinket) Roll(ctx context.Context, cmd RollTrinketCommand) (RollTrinketResult, error) {
	if u.roller == nil || cmd.Table.IsZero() {
		return nil, invalidInput("roll a trinket: zero use case or table")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAdvancementTrinket, ValueOperationTrinket)
	defer run.end(ctx)
	res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D100(), 0))
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	trinket, err := cmd.Table.EntryFor(res.Total())
	if err != nil { // unreachable: a roll of the 1d100 is 1 to 100
		run.commandError(err)
		return nil, err
	}
	run.finish(ctx, ValueResultRolled, "")
	return RollTrinketRolled{roll: res.Total(), trinket: trinket}, nil
}
