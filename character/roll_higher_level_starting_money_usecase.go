package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollHigherLevelStartingMoneyCommand is the input of
// RollHigherLevelStartingMoney.Roll. Construct it with a keyed literal.
type RollHigherLevelStartingMoneyCommand struct{ StartingLevel Level }

// RollHigherLevelStartingMoneyResult is the closed set of outcomes of
// RollHigherLevelStartingMoney.Roll. Variants: RollHigherLevelStartingMoneyRolled,
// RollHigherLevelStartingMoneyNotApplicable.
type RollHigherLevelStartingMoneyResult interface{ isRollHigherLevelStartingMoneyResult() }

// RollHigherLevelStartingMoneyRolled is the starting money of a level 2 or more:
// the allowance, the roll of the 1d10 (none for levels 2 to 4, which roll
// nothing) and the total gold as a Cost in GP. Only Roll builds one.
type RollHigherLevelStartingMoneyRolled struct {
	allowance HigherLevelEquipmentAllowance
	roll      int
	rolled    bool
	total     content.Cost
}

// Allowance returns the allowance of the starting level.
func (r RollHigherLevelStartingMoneyRolled) Allowance() HigherLevelEquipmentAllowance {
	return r.allowance
}

// Roll returns the roll of the 1d10 and true, or false when the level has no
// rolled gold and nothing was rolled.
func (r RollHigherLevelStartingMoneyRolled) Roll() (int, bool) { return r.roll, r.rolled }

// Total returns the fixed gold plus the roll times the multiplier, in GP.
func (r RollHigherLevelStartingMoneyRolled) Total() content.Cost { return r.total }

// RollHigherLevelStartingMoneyNotApplicable is the outcome for starting level 1,
// which has no higher-level equipment; nothing was rolled.
type RollHigherLevelStartingMoneyNotApplicable struct{}

func (RollHigherLevelStartingMoneyRolled) isRollHigherLevelStartingMoneyResult()        {}
func (RollHigherLevelStartingMoneyNotApplicable) isRollHigherLevelStartingMoneyResult() {}

// RollHigherLevelStartingMoney is the use case that rolls the extra gold of the
// Starting Equipment at Higher Levels table (SRD 5.2.1, p. 24). It uses the dice
// port and telemetry only. Create it with NewRollHigherLevelStartingMoney; the
// zero value is not usable. It is safe for concurrent use only if its roller,
// telemetry and clock are.
type RollHigherLevelStartingMoney struct {
	roller Roller
	sig    operationSignals
}

// NewRollHigherLevelStartingMoney returns the use case rolling through roller.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewRollHigherLevelStartingMoney(roller Roller, opts ...Option) (RollHigherLevelStartingMoney, error) {
	if roller == nil {
		return RollHigherLevelStartingMoney{}, invalidInput("roll starting money without a dice roller")
	}
	return RollHigherLevelStartingMoney{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Roll rolls the money of cmd.StartingLevel: nothing for level 1, no die for
// levels 2 to 4 (0 GP), and exactly one 1d10 for levels 5 to 20, whose total comes
// from HigherLevelEquipmentAllowance.TotalGold.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or the zero Level, and ctx.Err() unwrapped, before any signal,
// when ctx is already cancelled. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers
// another expression; the failure is recorded once and never logged.
func (u RollHigherLevelStartingMoney) Roll(ctx context.Context, cmd RollHigherLevelStartingMoneyCommand) (RollHigherLevelStartingMoneyResult, error) {
	if u.roller == nil || cmd.StartingLevel.IsZero() {
		return nil, invalidInput("roll starting money: zero use case or starting level")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAdvancementStartingMoney, ValueOperationStartingMoney)
	defer run.end(ctx)
	allowance, ok := HigherLevelEquipmentRules{}.ForStartingLevel(cmd.StartingLevel).(HigherLevelEquipmentAllowance)
	if !ok {
		run.finish(ctx, ValueResultNotApplicable, "")
		return RollHigherLevelStartingMoneyNotApplicable{}, nil
	}
	var roll *int
	if gold := allowance.RolledGold(); len(gold) > 0 {
		res, err := dice.Roll(u.roller, gold[0].Expression())
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		n := res.Total()
		roll = &n
	}
	total, err := allowance.TotalGold(roll)
	if err != nil { // unreachable: the roll is a face of the die
		run.commandError(err)
		return nil, err
	}
	cost, err := content.NewCost(total, content.CoinGP, 0)
	if err != nil { // unreachable: the largest total is 22,500 GP
		run.commandError(err)
		return nil, err
	}
	run.finish(ctx, ValueResultRolled, "")
	out := RollHigherLevelStartingMoneyRolled{allowance: allowance, total: cost}
	if roll != nil {
		out.roll, out.rolled = *roll, true
	}
	return out, nil
}
