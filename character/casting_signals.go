package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// castingSignals are the instruments the casting use cases share on top of the
// shared operations counter and duration histogram.
type castingSignals struct {
	operationSignals
	cast      telemetry.Counter
	slotLevel telemetry.Histogram
}

func newCastingSignals(o options) castingSignals {
	m := o.tel.Meter
	return castingSignals{
		operationSignals: newOperationSignals(o),
		cast:             m.Counter(telemetry.Descriptor{Name: MetricSpellsCast, Unit: "{cast}", Description: "Casts and completions, by source, class, result and rejection."}),
		slotLevel:        m.Histogram(telemetry.Descriptor{Name: MetricSpellSlotLevel, Unit: "{level}", Description: "Level of each spell slot expended."}),
	}
}

// begin is start with the operation attribute added to the span.
func (s castingSignals) begin(ctx context.Context, name, op string, extra ...telemetry.Attr) (context.Context, operationRun) {
	ctx, run := s.start(ctx, name, op, extra...)
	run.span.SetAttributes(telemetry.String(AttrOperation, op))
	return ctx, run
}
