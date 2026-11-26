package combat

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// hitPointUse is what every hit point use case holds: the instrumenter for the
// shared signals and the telemetry for the signals of hit-points-and-damage.md
// section 7. The zero value is silent and usable.
type hitPointUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newHitPointUse(opts []Option) hitPointUse {
	o := newOptions(opts)
	return hitPointUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// begin returns the errors every Run reports before any signal: a zero status
// (a zero command) and an already cancelled ctx.
func (h hitPointUse) begin(ctx context.Context, status HitPointStatus) error {
	if status.IsZero() {
		return invalidInput("zero command")
	}
	return ctx.Err()
}

// appliedTransition is the part every applied variant shares: the new status and
// the events in the order they happened.
type appliedTransition struct {
	status HitPointStatus
	events []HitPointEvent
}

// Status returns the new status.
func (a appliedTransition) Status() HitPointStatus { return a.status }

// Events returns a copy of the events, in the order they happened.
func (a appliedTransition) Events() []HitPointEvent { return slices.Clone(a.events) }

// convert maps a transition to a use case's result variants.
func convert[R any](tr HitPointTransition, ok func(HitPointTransitionApplied) R, refused func(HitPointRejection) R) R {
	switch t := tr.(type) {
	case HitPointTransitionApplied:
		return ok(t)
	case HitPointTransitionRefused:
		return refused(t.Rejection)
	default:
		panic(fmt.Errorf("combat: unreachable hit point transition %T", tr))
	}
}

// asEvents widens hit point events for the instrumenter.
func asEvents(events []HitPointEvent) []Event {
	out := make([]Event, len(events))
	for i, e := range events {
		out[i] = e
	}
	return out
}

// has reports whether events holds an event of type T.
func has[T HitPointEvent](events []HitPointEvent) bool {
	for _, e := range events {
		if _, ok := e.(T); ok {
			return true
		}
	}
	return false
}

// refusalResult is the srd5e.combat.result of a refused operation.
func refusalResult(r HitPointRejection) string {
	if r == CreatureIsDead {
		return valueResultRejectedDead
	}
	return r.String()
}

// run executes work inside the shared signals of op, then records the signals
// that belong to every hit point use case: a refusal by reason and a death by
// cause. name is the srd5e.combat.result of an applied transition. Nothing own
// is recorded for a returned error.
func (h hitPointUse) run(ctx context.Context, op combatOperation, work func() (HitPointTransition, error),
	name func(HitPointTransitionApplied) string) (HitPointTransition, error) {
	tr, err := run(ctx, h.in, op, func(context.Context) (HitPointTransition, error) { return work() },
		func(tr HitPointTransition) description {
			return convert(tr, func(a HitPointTransitionApplied) description {
				return description{Class: resultApplied, Name: name(a), Events: asEvents(a.events)}
			}, func(r HitPointRejection) description {
				return description{Class: resultRefused, Name: refusalResult(r)}
			})
		})
	if err != nil {
		return nil, err
	}
	meter := h.tel.WithDefaults().Meter
	switch t := tr.(type) {
	case HitPointTransitionRefused:
		meter.Counter(telemetry.Descriptor{Name: metricHitPointRefusals, Unit: "{refusal}",
			Description: "Hit point operations the rules refused, by reason."}).
			Add(ctx, 1, telemetry.String(attrOperation, op.Value()), telemetry.String(attrRefusalReason, t.Rejection.String()))
	case HitPointTransitionApplied:
		for _, e := range t.events {
			if d, ok := e.(Died); ok {
				meter.Counter(telemetry.Descriptor{Name: metricDeaths, Unit: "{creature}",
					Description: "Creatures that died, by cause."}).
					Add(ctx, 1, telemetry.String(attrDeathCause, d.cause.String()))
			}
		}
	}
	return tr, nil
}

// damageEffect is the srd5e.combat.damage_effect of one instance of damage
// (hit-points-and-damage.md table 4.10b).
func damageEffect(r DamageResolution) string {
	resisted, vulnerable := r.afterResistance < r.afterAdjustments, r.afterVulnerability > r.afterResistance
	switch {
	case r.incoming == 0:
		return valueDamageEffectUnchanged
	case r.immune:
		return valueDamageEffectImmune
	case r.afterVulnerability == 0:
		return valueDamageEffectReducedToZero
	case resisted && vulnerable:
		return valueDamageEffectResistedAndVulnerable
	case resisted:
		return valueDamageEffectResisted
	case vulnerable:
		return valueDamageEffectVulnerable
	}
	return valueDamageEffectUnchanged
}

// damageType is the lowercase srd5e.combat.damage_type of a resolution.
func damageType(r DamageResolution) string { return strings.ToLower(r.typ.String()) }
