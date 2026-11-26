package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// RollStatBlockDamageCommand is the input of RollStatBlockDamage.Run: the damage instances of one
// part, in printed order, and how each is rolled. Build it with a keyed literal. At least one
// instance is required, and Choices must fit them (DamageNotationChoices.For).
type RollStatBlockDamageCommand struct {
	Damage  []content.StatBlockDamage
	Choices DamageNotationChoices
}

// RollStatBlockDamageResult is the closed set of outcomes of RollStatBlockDamage.Run. Variants:
// StatBlockDamageRolled, StatBlockDamageInvalid.
type RollStatBlockDamageResult interface{ isRollStatBlockDamageResult() }

// StatBlockDamageRolled is the damage of each instance in printed order, full and unadjusted: the
// host passes it to ApplyDamage, where Resistance, Vulnerability, Immunity and the save's
// HalfDamage apply. A Critical Hit adds nothing (monster-stat-block-actions.md FR-029).
type StatBlockDamageRolled struct{ damage []Damage }

// Damage returns the damage of each instance in printed order, as a new slice.
func (r StatBlockDamageRolled) Damage() []Damage { return slices.Clone(r.damage) }

// StatBlockDamageInvalid: a damage instance is the zero value or holds a number outside the range
// combat relies on (plan P13), or the command holds more than 64 instances (plan P10; Err is then a
// LimitExceededError). Nothing was rolled.
type StatBlockDamageInvalid struct{ err error }

// Err returns the LimitExceededError of a command over the cap, and nil for a content number outside its range.
func (r StatBlockDamageInvalid) Err() error { return r.err }

func (StatBlockDamageRolled) isRollStatBlockDamageResult()  {}
func (StatBlockDamageInvalid) isRollStatBlockDamageResult() {}

// RollStatBlockDamage is the use case that rolls the damage of a stat-block part through the dice
// port (SRD 5.2.1, "Damage Notation", p. 257): once for the whole use, so every target that saves
// shares it (SRD 5.2.1, "Saving Throws and Damage", p. 16). It holds no rule beyond the choice of
// number or dice per instance. Create it with NewRollStatBlockDamage; the zero value is not
// usable.
type RollStatBlockDamage struct {
	in     instrumenter
	roller Roller
}

// NewRollStatBlockDamage returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewRollStatBlockDamage(roller Roller, opts ...Option) (RollStatBlockDamage, error) {
	if roller == nil {
		return RollStatBlockDamage{}, invalidInput("stat-block damage without a dice roller")
	}
	o := newOptions(opts)
	return RollStatBlockDamage{newInstrumenter(o.tel, o.clock), roller}, nil
}

// statBlockDamageOK reports whether d is a damage instance combat can roll: not the zero value, a
// printed number from 0 up and, when dice are printed, a count and sides inside the range of
// monsterDiceOK (plan P13).
func statBlockDamageOK(d content.StatBlockDamage) bool {
	if d.IsZero() || d.Number() < 0 || d.Number() > content.MonsterMaxNumber {
		return false
	}
	e, hasDice := d.Dice()
	return !hasDice || monsterDiceOK(e.Count(), e.Die().Sides())
}

// Run rolls each instance chosen as dice once, in order, with the printed flat modifier, and uses
// the printed number for each instance chosen as a number; a total below 0 is 0 (SRD 5.2.1,
// "Damage Rolls", p. 16). It records the shared signals under the operation roll_stat_block_damage
// and, on an applied result, the extra srd5e.combat.damage_notation. srd5e.combat.result is
// stat_block_damage_rolled (at least one die rolled) or stat_block_damage_fixed (numbers only), class
// applied, or content_number_out_of_range or limit_exceeded (more than 64 instances), class invalid (error status,
// no errors count). No event is produced.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, no
// damage instance, choices that do not fit the instances (see DamageNotationChoices.For) and a
// damage type printed as text, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It
// returns a *srderr.PortError (error type "dice_roller_error") with a nil result when the roller
// fails, panics or answers another expression; the failure is recorded once and never logged. A
// damage instance outside the range combat relies on is an Invalid result, found before any roll.
func (u RollStatBlockDamage) Run(ctx context.Context, cmd RollStatBlockDamageCommand) (RollStatBlockDamageResult, error) {
	if u.roller == nil || len(cmd.Damage) == 0 {
		return nil, invalidInput("zero use case or no damage instance")
	}
	picks, err := cmd.Choices.For(cmd.Damage)
	if err != nil {
		return nil, err
	}
	types := make([]content.DamageType, len(cmd.Damage))
	for i, d := range cmd.Damage {
		var readable bool
		if types[i], readable = statBlockDamageType(d); !readable && !d.IsZero() {
			return nil, invalidInput("a damage type printed as text cannot be rolled")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationRollStatBlockDamage, spanRollStatBlockDamage},
		func(context.Context) (RollStatBlockDamageResult, error) {
			if err := overCap("stat block damage instances", len(cmd.Damage), maxStatBlockDamages); err != nil {
				return StatBlockDamageInvalid{err}, nil
			}
			if slices.ContainsFunc(cmd.Damage, func(d content.StatBlockDamage) bool { return !statBlockDamageOK(d) }) {
				return StatBlockDamageInvalid{}, nil
			}
			out := make([]Damage, len(cmd.Damage))
			for i, d := range cmd.Damage {
				amount := d.Number()
				if picks[i] == DamageNotationDice {
					e, _ := d.Dice()
					roll, err := dice.Roll(u.roller, e)
					if err != nil {
						return nil, err
					}
					amount = max(roll.Total(), 0)
				}
				out[i] = MustDamage(amount, types[i])
			}
			return StatBlockDamageRolled{out}, nil
		},
		func(r RollStatBlockDamageResult) description {
			switch v := r.(type) {
			case StatBlockDamageRolled:
				name := valueResultStatBlockDamageFixed
				if slices.Contains(picks, DamageNotationDice) {
					name = valueResultStatBlockDamageRolled
				}
				return description{Class: resultApplied, Name: name, Extras: []telemetry.Attr{telemetry.String(attrDamageNotation, damageNotation(picks))}}
			case StatBlockDamageInvalid:
				name := valueResultContentNumberOutOfRange
				if v.err != nil {
					name = valueResultLimitExceeded
				}
				return description{Class: resultInvalid, Name: name}
			default:
				panic(fmt.Errorf("combat: unreachable stat-block damage result %T", r))
			}
		})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// damageNotation returns the value of srd5e.combat.damage_notation for the choices of a call:
// number, dice or mixed.
func damageNotation(choices []DamageNotationChoice) string {
	switch {
	case !slices.Contains(choices, DamageNotationDice):
		return valueDamageNotationNumber
	case !slices.Contains(choices, DamageNotationNumber):
		return valueDamageNotationDice
	}
	return valueDamageNotationMixed
}
