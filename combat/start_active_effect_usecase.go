package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// StartActiveEffectCommand is the input of StartActiveEffect. Construct it with a keyed literal.
// The use case changes nothing in it and copies Conditions.
type StartActiveEffectCommand struct {
	// Effects is the registry the effect is added to.
	Effects ActiveEffects
	// ID is the id of the new effect, chosen by the host.
	ID ActiveEffectID
	// Origin is the spell or other source of the effect.
	Origin EffectOrigin
	// Caster is the creature that cast it.
	Caster CombatantID
	// Subject is the creature or place it is on.
	Subject EffectSubject
	// Template is the shape of the effect.
	Template EffectTemplate
	// Binding is the caster's Concentration binding; the zero value is none (FR-012).
	Binding ConcentrationBinding
	// RequiresConcentration is Spell.RequiresConcentration() of a spell origin.
	RequiresConcentration bool
	// Conditions are the conditions the host actually applied for the effect (FR-013).
	Conditions []content.ConditionKind
	// Active is the combatant whose turn it is; the zero CombatantID outside combat.
	Active CombatantID
}

// StartActiveEffectResult is the closed set of outcomes of StartActiveEffect.Run. Variants:
// StartActiveEffectStarted, StartActiveEffectRejected.
type StartActiveEffectResult interface{ isStartActiveEffectResult() }

// StartActiveEffectStarted: the effect was added at the end of the registry, at stage 0, with
// one ActiveEffectStarted. Immutable and safe for concurrent use.
type StartActiveEffectStarted struct{ activeEffectsChange }

// StartActiveEffectRejected: the effect was not added; the registry is unchanged and no event was
// produced.
type StartActiveEffectRejected struct{ rejection ActiveEffectRejection }

// Rejection returns the reason (EffectRejectDuplicateID, EffectRejectBindingMismatch,
// EffectRejectConcentrationConflict or EffectRejectLimitExceeded).
func (r StartActiveEffectRejected) Rejection() ActiveEffectRejection { return r.rejection }

func (StartActiveEffectStarted) isStartActiveEffectResult()  {}
func (StartActiveEffectRejected) isStartActiveEffectResult() {}

// StartActiveEffect is the use case that adds an effect to the registry of active effects (SRD 5.2.1,
// "Duration", p. 106). It rolls nothing, applies no condition and calls no other use case. Create it
// with NewStartActiveEffect; the zero value is usable. Safe for concurrent use if its telemetry and
// clock are.
type StartActiveEffect struct{ use activeEffectsUse }

// NewStartActiveEffect returns the use case. Options: WithTelemetry, WithClock.
func NewStartActiveEffect(opts ...Option) StartActiveEffect {
	return StartActiveEffect{newActiveEffectsUse(opts)}
}

// Run adds the effect of cmd (FR-013) or returns StartActiveEffectRejected for a duplicate id, a
// binding that does not match the spell, a caster bound to another Concentration subject or a creature that already holds 64 effects (plan P10, result limit_exceeded, class invalid)
// (FR-014). It records the shared signals and the size of the registry, and never logs. It returns
// an error matching srderr.ErrInvalidInput, recorded on the span only, for a command that is not
// well formed (a zero id, an unusable origin, subject, template or condition), and ctx.Err()
// unwrapped, before any signal, for a cancelled ctx.
func (u StartActiveEffect) Run(ctx context.Context, cmd StartActiveEffectCommand) (StartActiveEffectResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationStartActiveEffect, spanStartActiveEffect}
	res, err := run(ctx, u.use.in, op, func(context.Context) (StartActiveEffectResult, error) { return startActiveEffect(cmd) },
		func(r StartActiveEffectResult) description {
			switch v := r.(type) {
			case StartActiveEffectStarted:
				return description{Class: resultApplied, Name: valueResultEffectStarted, Events: eventsOf(v.events)}
			case StartActiveEffectRejected:
				return describeRejection(v.rejection)
			default:
				panic(unreachableError("start active effect result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(StartActiveEffectStarted); ok {
		u.use.recordChange(ctx, op, v.activeEffectsChange)
	}
	return res, nil
}

func startActiveEffect(cmd StartActiveEffectCommand) (StartActiveEffectResult, error) {
	if !validActiveEffectID(cmd.ID) || cmd.Caster.IsZero() || cmd.Template.IsZero() || !validEffectOrigin(cmd.Origin) ||
		!validEffectSubject(cmd.Subject) || !validConditions(cmd.Conditions) {
		return nil, invalidInput("start active effect: command is not well formed")
	}
	effects, events, rej := startEffect(effectStart{cmd.Effects, cmd.ID, cmd.Origin, cmd.Caster, cmd.Subject, cmd.Template,
		cmd.Binding, cmd.RequiresConcentration, slices.Clone(cmd.Conditions), cmd.Active})
	if len(rej.problems) > 0 {
		return StartActiveEffectRejected{rej}, nil
	}
	return StartActiveEffectStarted{activeEffectsChange{effects: effects, events: events}}, nil
}
