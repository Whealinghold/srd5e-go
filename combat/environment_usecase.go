package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// environmentUse is what every hazard and object use case holds: the instrumenter
// for the shared signals and the telemetry for the signals of
// environment-hazards-and-objects.md section 7. The zero value is silent and usable.
type environmentUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newEnvironmentUse(opts []Option) environmentUse {
	o := newOptions(opts)
	return environmentUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// begin returns the errors every Run reports before any signal: a zero target (a
// zero command) and an already cancelled ctx.
func (u environmentUse) begin(ctx context.Context, target CombatantID) error {
	if target.IsZero() {
		return invalidInput("zero command")
	}
	return ctx.Err()
}

// recordDamageRolled records the hazard damage handed to the host, as rolled.
func (u environmentUse) recordDamageRolled(ctx context.Context, kind HazardKind, amount int) {
	u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricHazardDamageRolled, Unit: "{hit_point}",
		Description: "Hazard damage handed to the host before the damage pipeline."}).
		Record(ctx, float64(amount), telemetry.String(attrHazardKind, kind.String()))
}

// recordExhaustionGained counts the one Exhaustion level a hazard applied.
func (u environmentUse) recordExhaustionGained(ctx context.Context, kind HazardKind, outcome HazardExhaustionOutcome) {
	if outcome != HazardExhaustionGained {
		return
	}
	u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricHazardExhaustionGained, Unit: "{level}",
		Description: "Exhaustion levels applied by hazards."}).Add(ctx, 1, telemetry.String(attrHazardKind, kind.String()))
}

// widen widens events of one family for the instrumenter and for a result.
func widen[E Event](events []E) []Event {
	out := make([]Event, len(events))
	for i, e := range events {
		out[i] = e
	}
	return out
}

// exhaustionDescription classifies the outcome of a day-end or turn-end result.
func exhaustionDescription(o HazardExhaustionOutcome, events []Event) description {
	return description{Class: resultApplied, Name: o.String(), Events: events}
}

// copyEvents returns a copy of events, never nil.
func copyEvents(events []Event) []Event { return append([]Event{}, events...) }
