package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// RollSpellBonusDiceCommand is the input of RollSpellBonusDice.Run: the dice of one roll made with a d20, in the
// order they are rolled. The host takes them from SpellRollContributions.BonusDice and appends the signed dice of
// its other sources. Build it with a keyed literal; it may be empty. Run does not alter it.
type RollSpellBonusDiceCommand struct{ Dice []BonusDie }

// RollSpellBonusDiceResult is the closed set of outcomes of RollSpellBonusDice.Run. Variants: SpellBonusDiceRolled.
type RollSpellBonusDiceResult interface{ isRollSpellBonusDiceResult() }

// SpellBonusDiceRolled is the dice rolled: one signed modifier per die (the rolled total, negated for a penalty
// die) and the rolls as evidence, both in the order of the command. The host appends the modifiers to the roll with
// d20.RollModifiers.WithOthers. Both are empty when the command held no die. Not comparable with ==.
type SpellBonusDiceRolled struct {
	modifiers []modifier.Modifier
	rolls     []dice.Result
}

// Modifiers returns the signed modifiers, as a new slice.
func (r SpellBonusDiceRolled) Modifiers() []modifier.Modifier { return slices.Clone(r.modifiers) }

// Rolls returns the results of the dice, as a new slice.
func (r SpellBonusDiceRolled) Rolls() []dice.Result { return slices.Clone(r.rolls) }

func (SpellBonusDiceRolled) isRollSpellBonusDiceResult() {}

// RollSpellBonusDice is the use case that rolls the bonus dice of one roll through the dice port (SRD 5.2.1,
// "Bless", p. 113; "Guidance", p. 138). It holds no rule beyond the sign of a die. Create it with
// NewRollSpellBonusDice; the zero value is not usable.
type RollSpellBonusDice struct {
	in     instrumenter
	roller Roller
}

// NewRollSpellBonusDice returns the use case rolling through roller. Options: WithTelemetry, WithClock. It returns
// an error matching srderr.ErrInvalidInput for a nil roller.
func NewRollSpellBonusDice(roller Roller, opts ...Option) (RollSpellBonusDice, error) {
	if roller == nil {
		return RollSpellBonusDice{}, invalidInput("spell bonus dice without a dice roller")
	}
	o := newOptions(opts)
	return RollSpellBonusDice{newInstrumenter(o.tel, o.clock), roller}, nil
}

// Run rolls each die of cmd once, in list order, and returns SpellBonusDiceRolled. A natural 20 or 1 stays judged on
// the d20, since the dice are modifiers of the total (FR-033). The host calls it only for a roll made with a d20
// (FR-034). It records the shared signals under the operation roll_spell_bonus_dice with the result rolled (at least
// one die) or nothing_to_roll (none), class applied. No event is produced.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case and a zero die in
// cmd, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers another expression; the failure is
// recorded once, never logged, and no partial result is returned.
func (u RollSpellBonusDice) Run(ctx context.Context, cmd RollSpellBonusDiceCommand) (RollSpellBonusDiceResult, error) {
	if u.roller == nil || slices.ContainsFunc(cmd.Dice, BonusDie.IsZero) {
		return nil, invalidInput("zero use case or zero bonus die")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationRollSpellBonusDice, spanRollSpellBonusDice},
		func(context.Context) (RollSpellBonusDiceResult, error) {
			out := SpellBonusDiceRolled{}
			for _, d := range cmd.Dice {
				roll, err := dice.Roll(u.roller, d.expr)
				if err != nil {
					return nil, err
				}
				total := roll.Total()
				if d.negative {
					total = -total
				}
				out.modifiers = append(out.modifiers, modifier.New(total))
				out.rolls = append(out.rolls, roll)
			}
			return out, nil
		},
		func(r RollSpellBonusDiceResult) description {
			switch v := r.(type) {
			case SpellBonusDiceRolled:
				name := valueResultSpellBonusDiceNothingToRoll
				if len(v.rolls) > 0 {
					name = valueResultSpellBonusDiceRolled
				}
				return description{Class: resultApplied, Name: name}
			default:
				panic(fmt.Errorf("combat: unreachable spell bonus dice result %T", r))
			}
		})
	if err != nil {
		return nil, err
	}
	return res, nil
}
