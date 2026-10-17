package telemetrytest_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// A test asserts what the code under test emitted.
func ExampleRecorder() {
	rec := telemetrytest.NewRecorder()
	tel := rec.Telemetry()

	ctx, span := tel.Tracer.Start(context.Background(), "srd5e.example.run")
	tel.Meter.Counter(telemetry.Descriptor{Name: "srd5e.example.runs", Unit: "{run}"}).Add(ctx, 1)
	span.End()

	fmt.Println(rec.Tracer.Spans()[0].Name, rec.Meter.Measurements()[0].Value, len(rec.Violations()))
	// Output: srd5e.example.run 1 0
}
