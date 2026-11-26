package combat

import "context"

// EndActiveEffectsCommand is the input of EndActiveEffects. Construct it with a keyed literal.
type EndActiveEffectsCommand struct {
	// Effects is the registry to end effects in.
	Effects ActiveEffects
	// Cause is what the host reports: a lost Concentration, a dismissal, damage, a death, a removal
	// or the move of an effect to a new creature.
	Cause EndingCause
}

// EndActiveEffectsResult is the closed set of outcomes of EndActiveEffects.Run. Variants:
// EndActiveEffectsEnded, EndActiveEffectsMoved, EndActiveEffectsNothingEnded,
// EndActiveEffectsRejected.
type EndActiveEffectsResult interface{ isEndActiveEffectsResult() }

// EndActiveEffectsEnded: at least one effect ended, in registry order, each with one
// ActiveEffectEnded. The result plans the damage of their onEnd actions (none for a dead subject)
// and the conditions for the host to remove (FR-038, FR-045). Immutable and safe for concurrent use.
type EndActiveEffectsEnded struct{ activeEffectsChange }

// EndActiveEffectsMoved: an effect moved to a new creature with one ActiveEffectSubjectMoved and no
// plan (FR-060). Immutable and safe for concurrent use.
type EndActiveEffectsMoved struct{ activeEffectsChange }

// EndActiveEffectsNothingEnded: the cause matched no effect; the registry is unchanged, no event was
// produced and the class is refused (FR-041).
type EndActiveEffectsNothingEnded struct{ effects ActiveEffects }

// Effects returns the registry, unchanged.
func (r EndActiveEffectsNothingEnded) Effects() ActiveEffects { return r.effects }

// EndActiveEffectsRejected: the cause was refused; the registry is unchanged and nothing ended.
type EndActiveEffectsRejected struct{ rejection ActiveEffectRejection }

// Rejection returns every problem of the first failing group of checks.
func (r EndActiveEffectsRejected) Rejection() ActiveEffectRejection { return r.rejection }

func (EndActiveEffectsEnded) isEndActiveEffectsResult()        {}
func (EndActiveEffectsMoved) isEndActiveEffectsResult()        {}
func (EndActiveEffectsNothingEnded) isEndActiveEffectsResult() {}
func (EndActiveEffectsRejected) isEndActiveEffectsResult()     {}

// EndActiveEffects is the use case that ends effects, or moves one, for a cause the host reports
// (SRD 5.2.1, "Concentration", p. 179; "Duration", p. 106). It rolls nothing and calls no other use
// case. Create it with NewEndActiveEffects; the zero value is usable. Safe for concurrent use if its
// telemetry and clock are.
type EndActiveEffects struct{ use activeEffectsUse }

// NewEndActiveEffects returns the use case. Options: WithTelemetry, WithClock.
func NewEndActiveEffects(opts ...Option) EndActiveEffects {
	return EndActiveEffects{newActiveEffectsUse(opts)}
}

// Run applies the cause of cmd (FR-040 to FR-047). A cause that matches no effect gives
// EndActiveEffectsNothingEnded, a refused one EndActiveEffectsRejected. It records the shared
// signals, the effects ended by reason and the size of the registry, and never logs. It returns an
// error matching srderr.ErrInvalidInput, recorded on the span only, for a nil or unusable cause,
// and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
func (u EndActiveEffects) Run(ctx context.Context, cmd EndActiveEffectsCommand) (EndActiveEffectsResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationEndActiveEffects, spanEndActiveEffects}
	res, err := run(ctx, u.use.in, op, func(context.Context) (EndActiveEffectsResult, error) { return endActiveEffects(cmd) },
		func(r EndActiveEffectsResult) description {
			switch v := r.(type) {
			case EndActiveEffectsEnded:
				return description{Class: resultApplied, Name: valueResultEffectsEnded, Events: eventsOf(v.events)}
			case EndActiveEffectsMoved:
				return description{Class: resultApplied, Name: valueResultEffectMoved, Events: eventsOf(v.events)}
			case EndActiveEffectsNothingEnded:
				return description{Class: resultRefused, Name: valueResultEffectsNothingEnded}
			case EndActiveEffectsRejected:
				return describeRejection(v.rejection)
			default:
				panic(unreachableError("end active effects result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case EndActiveEffectsEnded:
		u.use.recordChange(ctx, op, v.activeEffectsChange)
	case EndActiveEffectsMoved:
		u.use.recordChange(ctx, op, v.activeEffectsChange)
	case EndActiveEffectsNothingEnded:
		u.use.recordActive(ctx, op, v.effects)
	}
	return res, nil
}

func endActiveEffects(cmd EndActiveEffectsCommand) (EndActiveEffectsResult, error) {
	if !validEndingCause(cmd.Cause) {
		return nil, invalidInput("end active effects: the cause is not well formed")
	}
	out := endEffects(cmd.Effects, cmd.Cause)
	change := changeOf(out.effects, out.work)
	switch {
	case len(out.rejection.problems) > 0:
		return EndActiveEffectsRejected{out.rejection}, nil
	case out.moved:
		return EndActiveEffectsMoved{change}, nil
	case len(out.work.ended) == 0:
		return EndActiveEffectsNothingEnded{cmd.Effects}, nil
	}
	return EndActiveEffectsEnded{change}, nil
}
