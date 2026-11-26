package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// recordEndCauses counts each shape-shift and zone ending among events by its cause, so a consumer can tell why
// transformations and zones end. It reads the events a use case already produced, adds no port call and counts nothing
// else.
func recordEndCauses(ctx context.Context, meter telemetry.Meter, events []Event) {
	for _, e := range events {
		switch v := e.(type) {
		case ShapeShiftEnded:
			meter.Counter(telemetry.Descriptor{Name: metricShapeShiftEnded, Unit: "{transformation}",
				Description: "Shape-shifting transformations that ended, by cause."}).
				Add(ctx, 1, telemetry.String(attrShapeShiftEndCause, v.Cause().String()))
		case ZoneEnded:
			meter.Counter(telemetry.Descriptor{Name: metricZoneEnded, Unit: "{zone}",
				Description: "Persistent zones that ended, by reason."}).
				Add(ctx, 1, telemetry.String(attrZoneEndReason, v.Reason().String()))
		}
	}
}
