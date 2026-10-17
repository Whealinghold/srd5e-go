package telemetry_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// A use case defaults its ports once, then emits without checking whether
// telemetry is enabled.
func ExampleTelemetry_WithDefaults() {
	rec := telemetrytest.NewRecorder()
	// Only the tracer is supplied; logger and meter become no-ops.
	tel := telemetry.Telemetry{Tracer: rec.Tracer}.WithDefaults()

	ctx, span := tel.Tracer.Start(context.Background(), "srd5e.example.run",
		telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeSuccess))
	defer span.End()
	tel.Logger.Log(ctx, telemetry.LevelInfo, "ignored", nil)

	fmt.Println(len(rec.Tracer.Spans()), len(rec.Logger.Records()))
	// Output: 1 0
}

func ExampleStartTimer() {
	rec := telemetrytest.NewRecorder()
	hist := rec.Meter.Histogram(telemetry.Descriptor{Name: "srd5e.example.duration", Unit: "s"})
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)

	timer := telemetry.StartTimer(clock.Now, hist)
	timer.Stop(context.Background())

	fmt.Println(rec.Meter.Measurements()[0].Value)
	// Output: 0.25
}

func ExampleValidName() {
	fmt.Println(telemetry.ValidName("srd5e.d20_test.resolve"), telemetry.ValidName("resolve"))
	// Output: true false
}
