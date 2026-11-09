package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// spellcastingSignals are the instruments the spellcasting use cases share on top
// of the shared operations counter and duration histogram.
type spellcastingSignals struct {
	operationSignals
	changes   telemetry.Counter
	restored  telemetry.Counter
	slotLevel telemetry.Histogram
}

func newSpellcastingSignals(o options) spellcastingSignals {
	m := o.tel.Meter
	return spellcastingSignals{
		operationSignals: newOperationSignals(o),
		changes:          m.Counter(telemetry.Descriptor{Name: MetricPreparedSpellChanges, Unit: "{change}", Description: "Prepared spell changes, by class, result and rejection."}),
		restored:         m.Counter(telemetry.Descriptor{Name: MetricSpellSlotsRestored, Unit: "{rest}", Description: "Rests that restored spell slots, by rest kind."}),
		slotLevel:        m.Histogram(telemetry.Descriptor{Name: MetricSpellSlotLevel, Unit: "{level}", Description: "Level of each spell slot expended."}),
	}
}

// begin is start with the operation attribute added to the span, as the
// spellcasting spans carry it.
func (s spellcastingSignals) begin(ctx context.Context, name, op string, extra ...telemetry.Attr) (context.Context, operationRun) {
	ctx, run := s.start(ctx, name, op, extra...)
	run.span.SetAttributes(telemetry.String(AttrOperation, op))
	return ctx, run
}
