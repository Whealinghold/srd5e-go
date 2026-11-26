package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// activeEffectsChange is what a transition of the registry hands back: the new registry, the events
// in the order they happened and the plans for the host. Its accessors return copies (FR-057).
type activeEffectsChange struct {
	effects  ActiveEffects
	events   []ActiveEffectEvent
	toApply  []EffectConditionPlan
	toRemove []EffectConditionPlan
	damage   []EffectDamagePlan
}

func changeOf(effects ActiveEffects, w *effectWork) activeEffectsChange {
	return activeEffectsChange{effects, slices.Clone(w.events), slices.Clone(w.toApply), slices.Clone(w.toRemove), slices.Clone(w.damage)}
}

// Effects returns the registry after the call.
func (c activeEffectsChange) Effects() ActiveEffects { return c.effects }

// Events returns the events in the order they happened, as a new slice.
func (c activeEffectsChange) Events() []ActiveEffectEvent { return slices.Clone(c.events) }

// ConditionsToApply returns the conditions for the host to apply, as a new slice.
func (c activeEffectsChange) ConditionsToApply() []EffectConditionPlan {
	return slices.Clone(c.toApply)
}

// ConditionsToRemove returns the conditions for the host to remove, as a new slice.
func (c activeEffectsChange) ConditionsToRemove() []EffectConditionPlan {
	return slices.Clone(c.toRemove)
}

// DamagePlans returns the damage plans for ResolveEffectDamage, as a new slice.
func (c activeEffectsChange) DamagePlans() []EffectDamagePlan { return slices.Clone(c.damage) }

// activeEffectsUse is what every active effect use case holds: the instrumenter for the shared
// signals and the telemetry for the signals of active-effects.md section 7. The zero value is silent
// and usable.
type activeEffectsUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newActiveEffectsUse(opts []Option) activeEffectsUse {
	o := newOptions(opts)
	return activeEffectsUse{newInstrumenter(o.tel, o.clock), o.tel}
}

// eventsOf lists events as Event values, for the shared events counter.
func eventsOf(events []ActiveEffectEvent) []Event {
	out := make([]Event, len(events))
	for i, e := range events {
		out[i] = e
	}
	return out
}

// describeRejection classifies a rejection by its problems (FR-063) with the first reason as extra.
func describeRejection(r ActiveEffectRejection) description {
	name := valueResultEffectsRejectedRefused
	if r.class() == resultInvalid {
		name = valueResultEffectsRejectedInvalid
	}
	if r.firstReason() == EffectRejectLimitExceeded {
		name = valueResultLimitExceeded
	}
	return description{Class: r.class(), Name: name,
		Extras: []telemetry.Attr{telemetry.String(attrActiveEffectsRejection, r.firstReason().String())}}
}

// recordSave counts one saving throw.
func (u activeEffectsUse) recordSave(ctx context.Context, kind EffectSaveKind, a ability.Ability, o d20.Outcome) {
	u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricActiveEffectsSaves, Unit: "{save}",
		Description: "Repeated, gate and damage saving throws made by active effects."}).
		Add(ctx, 1, outcomeAttr(o == d20.Success), telemetry.String(d20.AttrAbility, a.String()),
			telemetry.String(attrActiveEffectsSaveKind, kind.String()))
}

// recordChange records what the call of op produced: the effects that ended, the saves made and the
// size of the registry (active-effects.md section 7).
func (u activeEffectsUse) recordChange(ctx context.Context, op combatOperation, c activeEffectsChange) {
	tel := u.tel.WithDefaults()
	ended := tel.Meter.Counter(telemetry.Descriptor{Name: metricActiveEffectsEnded, Unit: "{effect}",
		Description: "Active effects that ended, by reason."})
	for _, e := range c.events {
		switch v := e.(type) {
		case ActiveEffectEnded:
			ended.Add(ctx, 1, telemetry.String(attrActiveEffectsEndReason, v.reason.String()))
		case ActiveEffectRepeatSaveMade:
			u.recordSave(ctx, v.kind, v.ability, v.outcome)
		}
	}
	u.recordActive(ctx, op, c.effects)
}

// recordActive records the effects in the registry after a call.
func (u activeEffectsUse) recordActive(ctx context.Context, op combatOperation, r ActiveEffects) {
	u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricActiveEffectsActive, Unit: "{effect}",
		Description: "Active effects in the registry after a call."}).
		Record(ctx, float64(r.Len()), telemetry.String(attrOperation, op.Value()))
}

// validConditions reports whether every kind is a condition other than Exhaustion.
func validConditions(kinds []content.ConditionKind) bool {
	return !slices.ContainsFunc(kinds, func(k content.ConditionKind) bool { return !k.Valid() || k == content.ConditionExhaustion })
}

// validEffectOrigin reports whether o is a variant its constructor would accept.
func validEffectOrigin(o EffectOrigin) bool {
	switch v := o.(type) {
	case SpellEffectOrigin:
		_, err := NewSpellEffectOrigin(v.spell, v.level)
		return err == nil
	case OtherEffectOrigin:
		_, err := NewOtherEffectOrigin(v.key)
		return err == nil
	}
	return false
}

// validActiveEffectID reports whether id is one NewActiveEffectID would accept.
func validActiveEffectID(id ActiveEffectID) bool {
	_, err := NewActiveEffectID(string(id))
	return err == nil
}
