package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// grappleUse is what every grapple use case holds: the instrumenter for the shared
// signals and the telemetry for the signals of grappling.md section 7. The zero
// value is silent and usable.
type grappleUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newGrappleUse(opts []Option) grappleUse {
	o := newOptions(opts)
	return grappleUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// recordGrappleEnded counts one grapple that ended for the reason.
func recordGrappleEnded(ctx context.Context, tel telemetry.Telemetry, reason string) {
	tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricGrappleEnded, Unit: "{grapple}",
		Description: "Grapples that ended, by reason."}).Add(ctx, 1, telemetry.String(attrGrappleEndReason, reason))
}

// recordEnded counts one grapple that ended for the reason.
func (u grappleUse) recordEnded(ctx context.Context, reason string) {
	recordGrappleEnded(ctx, u.tel, reason)
}

// recordEndings counts each ending by its cause.
func (u grappleUse) recordEndings(ctx context.Context, endings []GrappleEnding) {
	for _, e := range endings {
		u.recordEnded(ctx, e.cause.String())
	}
}

// recordGrappleInForce records the size of the set a result carries.
func recordGrappleInForce(ctx context.Context, tel telemetry.Telemetry, op combatOperation, set Grapples) {
	tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricGrappleInForce, Unit: "{grapple}",
		Description: "Grapples in force in the set a result carries."}).
		Record(ctx, float64(set.Len()), telemetry.String(attrOperation, op.Value()))
}

// refusedGrapple describes a refusal by its GrappleRejection.
func refusedGrapple(r GrappleRejection) description {
	return description{Class: resultRefused, Name: r.String()}
}

// invalidGrapple describes unusable input by its GrappleProblem.
func invalidGrapple(p GrappleProblem) description {
	return description{Class: resultInvalid, Name: p.String()}
}

// recordInForce records the size of the set a result carries.
func (u grappleUse) recordInForce(ctx context.Context, op combatOperation, set Grapples) {
	recordGrappleInForce(ctx, u.tel, op, set)
}
