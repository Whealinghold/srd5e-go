package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
)

// AdvanceActiveEffectsCommand is the input of AdvanceActiveEffects. Construct it with a keyed
// literal. The use case changes nothing in it.
type AdvanceActiveEffectsCommand struct {
	// Effects is the registry to age.
	Effects ActiveEffects
	// Tick is one tick of the clock (FR-020 to FR-026); ClockTickFromTurnEvent maps tracker events.
	Tick EffectClockTick
	// AreaEffects lists the effects whose area the combatant of the tick is in (FR-027).
	AreaEffects []ActiveEffectID
	// Order, when not empty, is the order to process the effects that fire: a permutation of their ids
	// (FR-028). Empty means registry order.
	Order []ActiveEffectID
	// Saves are the attempts of the creatures that must make a saving throw at the tick (FR-031).
	Saves map[CombatantID]map[ability.Ability]d20.SaveAttempt
	// Active is the combatant active after the tick, who anchors new stage clocks; zero outside
	// combat (FR-034).
	Active CombatantID
}

// AdvanceActiveEffectsResult is the closed set of outcomes of AdvanceActiveEffects.Run. Variants:
// AdvanceActiveEffectsApplied, AdvanceActiveEffectsRejected.
type AdvanceActiveEffectsResult interface{ isAdvanceActiveEffectsResult() }

// AdvanceActiveEffectsApplied: the tick was applied; the registry equals the input for a tick that
// matches no effect (FR-026). The result carries the events in order, the conditions to apply and
// remove and the damage to resolve. Immutable and safe for concurrent use.
type AdvanceActiveEffectsApplied struct{ activeEffectsChange }

// AdvanceActiveEffectsRejected: the tick was refused before any die: the order is not a permutation
// of the effects that fire, or a needed saving throw has no attempt.
type AdvanceActiveEffectsRejected struct{ rejection ActiveEffectRejection }

// Rejection returns every problem of the first failing group of checks (invalid_order, then
// missing_save_attempt).
func (r AdvanceActiveEffectsRejected) Rejection() ActiveEffectRejection { return r.rejection }

func (AdvanceActiveEffectsApplied) isAdvanceActiveEffectsResult()  {}
func (AdvanceActiveEffectsRejected) isAdvanceActiveEffectsResult() {}

// AdvanceActiveEffects is the use case that ages the registry of active effects at a tick of the
// clock, makes the repeated and gate saving throws and plans the damage and conditions that follow
// (SRD 5.2.1, "Duration", p. 106). It rolls only the saving throws, through the dice port, and
// composes the pure d20 functions; it applies nothing and calls no other use case. Create it with
// NewAdvanceActiveEffects; the zero value is not usable. Safe for concurrent use if its roller,
// telemetry and clock are.
type AdvanceActiveEffects struct {
	use  activeEffectsUse
	flow attackFlow
}

// NewAdvanceActiveEffects returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewAdvanceActiveEffects(roller Roller, opts ...Option) (AdvanceActiveEffects, error) {
	if roller == nil {
		return AdvanceActiveEffects{}, invalidInput("advance active effects without a dice roller")
	}
	return AdvanceActiveEffects{newActiveEffectsUse(opts), attackFlow{roller}}, nil
}

// Run applies the tick of cmd (FR-020 to FR-034). It records the shared signals, the effects ended
// by reason, the saves by kind and the size of the registry, and never logs. It returns an error
// matching srderr.ErrInvalidInput, before any signal, for the zero use case; ctx.Err() unwrapped,
// before any signal, for a cancelled ctx; an error matching srderr.ErrInvalidInput, recorded on the
// span only, for a tick that is nil or unusable; and a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers another
// expression, recorded once.
func (u AdvanceActiveEffects) Run(ctx context.Context, cmd AdvanceActiveEffectsCommand) (AdvanceActiveEffectsResult, error) {
	if u.flow.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationAdvanceActiveEffects, spanAdvanceActiveEffects}
	res, err := run(ctx, u.use.in, op, func(context.Context) (AdvanceActiveEffectsResult, error) { return u.advance(cmd) },
		func(r AdvanceActiveEffectsResult) description {
			switch v := r.(type) {
			case AdvanceActiveEffectsApplied:
				return description{Class: resultApplied, Name: valueResultEffectsAdvanced, Events: eventsOf(v.events)}
			case AdvanceActiveEffectsRejected:
				return describeRejection(v.rejection)
			default:
				panic(unreachableError("advance active effects result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(AdvanceActiveEffectsApplied); ok {
		u.use.recordChange(ctx, op, v.activeEffectsChange)
	}
	return res, nil
}

func (u AdvanceActiveEffects) advance(cmd AdvanceActiveEffectsCommand) (AdvanceActiveEffectsResult, error) {
	if !validEffectClockTick(cmd.Tick) {
		return nil, invalidInput("advance active effects: the tick is not well formed")
	}
	out, err := advanceEffects(effectAdvance{cmd.Effects, cmd.Tick, slices.Clone(cmd.AreaEffects), slices.Clone(cmd.Order), cmd.Saves, cmd.Active}, u.flow)
	switch {
	case err != nil:
		return nil, err
	case len(out.rejection.problems) > 0:
		return AdvanceActiveEffectsRejected{out.rejection}, nil
	}
	return AdvanceActiveEffectsApplied{changeOf(out.effects, out.work)}, nil
}
