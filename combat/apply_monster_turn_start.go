package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ApplyMonsterTurnStartCommand is the input of ApplyMonsterTurnStart.Run: the
// limited-use state of the monster whose turn starts. Construct it with a keyed
// literal. The zero command is invalid.
type ApplyMonsterTurnStartCommand struct{ State LimitedUseState }

// ApplyMonsterTurnStartResult is the closed set of outcomes of
// ApplyMonsterTurnStart.Run. Variants: MonsterTurnStartApplied.
type ApplyMonsterTurnStartResult interface{ isApplyMonsterTurnStartResult() }

// MonsterTurnStartApplied is the state after the start of the monster's turn, the
// events in the order they happened and the Recharge rolls made. Only Run builds one.
type MonsterTurnStartApplied struct {
	transition LimitedUseTransition
	rolls      []MonsterRechargeRolled
}

func (MonsterTurnStartApplied) isApplyMonsterTurnStartResult() {}

// State returns the new state.
func (r MonsterTurnStartApplied) State() LimitedUseState { return r.transition.state }

// Events returns a copy of the events.
func (r MonsterTurnStartApplied) Events() []MonsterLimitedUseEvent { return r.transition.Events() }

// Rolls returns a copy of the Recharge rolls made, in printed order.
func (r MonsterTurnStartApplied) Rolls() []MonsterRechargeRolled { return slices.Clone(r.rolls) }

// ApplyMonsterTurnStart is the use case of the start of a monster's turn (SRD 5.2.1,
// "Limited Usage", "Legendary Actions", p. 257): it rolls the 1d6 of each spent
// Recharge part and returns the new state. It holds no rule; it calls
// LimitedUseState.RechargeRollsNeeded and StartOfOwnTurn. Create it with
// NewApplyMonsterTurnStart; the zero value is not usable.
type ApplyMonsterTurnStart struct {
	in     instrumenter
	tel    telemetry.Telemetry
	roller Roller
}

// NewApplyMonsterTurnStart returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput
// for a nil roller.
func NewApplyMonsterTurnStart(roller Roller, opts ...Option) (ApplyMonsterTurnStart, error) {
	if roller == nil {
		return ApplyMonsterTurnStart{}, invalidInput("monster turn start without a dice roller")
	}
	o := newOptions(opts)
	return ApplyMonsterTurnStart{newInstrumenter(o.tel, o.clock), o.tel, roller}, nil
}

// Run rolls one 1d6 for each part of RechargeRollsNeeded, in printed order, and
// none when there is none, then applies them with StartOfOwnTurn. It records the
// shared signals under the operation apply_monster_turn_start and one
// srd5e.combat.monster_recharge_rolls count per roll; the result is always class
// applied, and srd5e.combat.result is turn_start_recharged, turn_start_not_recharged
// or turn_start_no_rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or a zero state, and ctx.Err() unwrapped, before any signal, for a
// cancelled ctx. It returns a *srderr.PortError (error type "dice_roller_error")
// with a nil result when the roller fails, panics or answers another expression,
// also on the second roll; the failure is recorded once, never logged, and no roll
// is counted.
func (u ApplyMonsterTurnStart) Run(ctx context.Context, cmd ApplyMonsterTurnStartCommand) (ApplyMonsterTurnStartResult, error) {
	if u.roller == nil || cmd.State.IsZero() {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationApplyMonsterTurnStart, spanApplyMonsterTurnStart},
		func(context.Context) (ApplyMonsterTurnStartResult, error) {
			needed := cmd.State.RechargeRollsNeeded()
			faces := make(map[LimitedUsePartRef]dice.DieRoll, len(needed))
			for _, ref := range needed {
				r, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D6(), 0))
				if err != nil {
					return nil, err
				}
				faces[ref] = r.Rolls()[0]
			}
			tr, err := cmd.State.StartOfOwnTurn(faces)
			if err != nil {
				return nil, err
			}
			var rolls []MonsterRechargeRolled
			for _, e := range tr.events {
				if r, ok := e.(MonsterRechargeRolled); ok {
					rolls = append(rolls, r)
				}
			}
			return MonsterTurnStartApplied{tr, rolls}, nil
		},
		func(r ApplyMonsterTurnStartResult) description {
			a := r.(MonsterTurnStartApplied)
			name := valueResultTurnStartNoRolls
			if len(a.rolls) > 0 {
				name = valueResultTurnStartNotRecharged
			}
			if slices.ContainsFunc(a.rolls, func(r MonsterRechargeRolled) bool { return r.result == RechargeRegained }) {
				name = valueResultTurnStartRecharged
			}
			return description{Class: resultApplied, Name: name, Events: widen(a.transition.events)}
		})
	if err != nil {
		return nil, err
	}
	counter := u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricMonsterRechargeRolls, Unit: "{roll}",
		Description: "Recharge rolls made at the start of monster turns, by outcome."})
	for _, r := range res.(MonsterTurnStartApplied).rolls {
		outcome := valueRechargeNotRegained
		if r.result == RechargeRegained {
			outcome = valueRechargeRegained
		}
		counter.Add(ctx, 1, telemetry.String(attrRechargeOutcome, outcome))
	}
	return res, nil
}
