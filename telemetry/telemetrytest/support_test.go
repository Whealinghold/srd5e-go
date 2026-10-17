package telemetrytest_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func TestContracts(t *testing.T) {
	t.Parallel()
	t.Run("logger", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunLoggerContract(t, func(*testing.T) telemetry.Logger { return new(telemetrytest.Logger) })
		telemetrytest.RunLoggerContract(t, func(*testing.T) telemetry.Logger { return telemetrytest.NewLogger(telemetry.LevelError) })
	})
	t.Run("tracer", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunTracerContract(t, func(*testing.T) telemetry.Tracer { return new(telemetrytest.Tracer) })
	})
	t.Run("meter", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunMeterContract(t, func(*testing.T) telemetry.Meter { return new(telemetrytest.Meter) })
	})
	t.Run("recorder", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunLoggerContract(t, func(*testing.T) telemetry.Logger { return telemetrytest.NewRecorder().Telemetry().Logger })
		telemetrytest.RunTracerContract(t, func(*testing.T) telemetry.Tracer { return telemetrytest.NewRecorder().Telemetry().Tracer })
		telemetrytest.RunMeterContract(t, func(*testing.T) telemetry.Meter { return telemetrytest.NewRecorder().Telemetry().Meter })
	})
}

func TestRecorder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec := telemetrytest.NewRecorder()
	tel := rec.Telemetry()
	if tel.Logger != telemetry.Logger(rec.Logger) || tel.Tracer != telemetry.Tracer(rec.Tracer) || tel.Meter != telemetry.Meter(rec.Meter) {
		t.Fatal("Telemetry must wrap exactly the recorder's doubles")
	}
	tel.Logger.Log(ctx, telemetry.LevelInfo, "m", nil)
	_, s := tel.Tracer.Start(ctx, "bad")
	s.End()
	tel.Meter.Counter(telemetry.Descriptor{Name: "bad"})
	tel.Logger.Log(ctx, telemetry.LevelInfo, "m", func() []telemetry.Attr { return []telemetry.Attr{{}} })
	if len(rec.Violations()) != 3 {
		t.Fatalf("violations = %v, want logger+tracer+meter", rec.Violations())
	}
	rec.Clear()
	if len(rec.Logger.Records()) != 0 || len(rec.Violations()) != 0 {
		t.Fatal("Clear must reset all three")
	}
}

type fakeTB struct {
	testing.TB
	errs int
}

func (f *fakeTB) Helper()               {}
func (f *fakeTB) Errorf(string, ...any) { f.errs++ }

func TestRequireNoViolations(t *testing.T) {
	t.Parallel()
	clean, dirty := telemetrytest.NewRecorder(), telemetrytest.NewRecorder()
	dirty.Tracer.Start(context.Background(), "bad")
	f := &fakeTB{TB: t}
	telemetrytest.RequireNoViolations(f, clean)
	if f.errs != 0 {
		t.Fatalf("clean recorder reported %d errors", f.errs)
	}
	telemetrytest.RequireNoViolations(f, clean, dirty)
	if f.errs != 1 {
		t.Fatalf("dirty recorder reported %d errors, want 1", f.errs)
	}
}

func TestClock(t *testing.T) {
	t.Parallel()
	start := time.Unix(10, 0)
	c := telemetrytest.NewClock(start, time.Second)
	if c.Now() != start || c.Now() != start.Add(time.Second) {
		t.Fatal("Now must return the current time then advance by the step")
	}
	c.Advance(time.Minute)
	if c.Now() != start.Add(time.Second*2+time.Minute) {
		t.Fatal("Advance not applied")
	}
	var zero telemetrytest.Clock
	if !zero.Now().IsZero() {
		t.Fatal("zero Clock must stand still at the zero time")
	}
}
