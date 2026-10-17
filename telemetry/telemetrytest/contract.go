package telemetrytest

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

type ctxKey struct{}

// reporter is the part of testing.TB the checks use, so the checks can be
// tested against a recording double.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
}

// check is one named contract rule over a port P; newPort returns a fresh P.
type check[P any] struct {
	name string
	run  func(t reporter, newPort func() P)
}

func runChecks[P any](t *testing.T, factory func(t *testing.T) P, checks []check[P]) {
	t.Helper()
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			c.run(t, func() P { return factory(t) })
		})
	}
}

// allLevels includes levels outside the defined set: a port must not panic for
// any Level.
func allLevels() []telemetry.Level {
	return []telemetry.Level{
		telemetry.Level(-1), telemetry.LevelTrace, telemetry.LevelDebug, telemetry.LevelInfo,
		telemetry.LevelWarn, telemetry.LevelError, telemetry.Level(99),
	}
}

func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func checkViolations(t reporter, port any) {
	t.Helper()
	if v, ok := port.(Violator); ok {
		for _, err := range v.Violations() {
			t.Errorf("telemetry misuse: %v", err)
		}
	}
}

// RunLoggerContract checks that the Logger returned by factory honours the
// telemetry.Logger contract. factory is called once per subtest so subtests
// share no state. Every adapter and double should pass it.
func RunLoggerContract(t *testing.T, factory func(t *testing.T) telemetry.Logger) {
	t.Helper()
	runChecks(t, factory, loggerChecks())
}

func loggerChecks() []check[telemetry.Logger] {
	return []check[telemetry.Logger]{
		{"Enabled is monotonic in level", func(t reporter, newPort func() telemetry.Logger) {
			l := newPort()
			enabled := false
			for _, lv := range []telemetry.Level{telemetry.LevelTrace, telemetry.LevelDebug, telemetry.LevelInfo, telemetry.LevelWarn, telemetry.LevelError} {
				got := l.Enabled(context.Background(), lv)
				if enabled && !got {
					t.Errorf("level %v disabled although a lower level is enabled", lv)
				}
				enabled = enabled || got
			}
		}},
		{"Log accepts nil attrs at every level", func(t reporter, newPort func() telemetry.Logger) {
			l := newPort()
			for _, lv := range allLevels() {
				l.Log(context.Background(), lv, "msg", nil)
			}
			checkViolations(t, l)
		}},
		{"Log tolerates a cancelled context", func(_ reporter, newPort func() telemetry.Logger) {
			l := newPort()
			for _, lv := range allLevels() {
				l.Log(cancelled(), lv, "msg", func() []telemetry.Attr { return nil })
				_ = l.Enabled(cancelled(), lv)
			}
		}},
		{"attrs is called only when enabled, and at most once", func(t reporter, newPort func() telemetry.Logger) {
			l := newPort()
			for _, lv := range allLevels() {
				calls := 0
				enabled := l.Enabled(context.Background(), lv)
				l.Log(context.Background(), lv, "msg", func() []telemetry.Attr {
					calls++
					return []telemetry.Attr{telemetry.String("k", "v"), telemetry.Int("n", 1), telemetry.Float("f", 1.5), telemetry.Bool("b", true)}
				})
				if !enabled && calls != 0 {
					t.Errorf("level %v: attrs called %d times while disabled", lv, calls)
				}
				if calls > 1 {
					t.Errorf("level %v: attrs called %d times", lv, calls)
				}
			}
			checkViolations(t, l)
		}},
		{"Log accepts the Err attribute", func(t reporter, newPort func() telemetry.Logger) {
			l := newPort()
			l.Log(context.Background(), telemetry.LevelError, "msg", func() []telemetry.Attr {
				return []telemetry.Attr{telemetry.Err(context.Canceled)}
			})
			checkViolations(t, l)
		}},
	}
}

// RunTracerContract checks that the Tracer returned by factory honours the
// telemetry.Tracer contract. factory is called once per subtest.
func RunTracerContract(t *testing.T, factory func(t *testing.T) telemetry.Tracer) {
	t.Helper()
	runChecks(t, factory, tracerChecks())
}

const spanName = "srd5e.contract.span"

func tracerChecks() []check[telemetry.Tracer] {
	return []check[telemetry.Tracer]{
		{"Start returns a span and keeps the caller's context values", func(t reporter, newPort func() telemetry.Tracer) {
			tr := newPort()
			parent := context.WithValue(context.Background(), ctxKey{}, "kept")
			ctx, span := tr.Start(parent, spanName)
			if ctx == nil || span == nil {
				t.Errorf("Start returned ctx=%v span=%v", ctx, span)
				return
			}
			if ctx.Value(ctxKey{}) != "kept" {
				t.Errorf("returned context lost the caller's values")
			}
			span.End()
			checkViolations(t, tr)
		}},
		{"Start accepts attributes and a cancelled context", func(t reporter, newPort func() telemetry.Tracer) {
			tr := newPort()
			_, span := tr.Start(cancelled(), spanName, telemetry.String("k", "v"), telemetry.Int("n", 1), telemetry.Bool("b", false))
			if span == nil {
				t.Errorf("Start returned a nil span for a cancelled context")
				return
			}
			span.End()
			_, span = tr.Start(context.Background(), spanName)
			if span == nil {
				t.Errorf("Start returned a nil span")
				return
			}
			span.End()
			checkViolations(t, tr)
		}},
		{"Span methods accept empty and nil arguments", func(t reporter, newPort func() telemetry.Tracer) {
			tr := newPort()
			_, span := tr.Start(context.Background(), spanName)
			if span == nil {
				t.Errorf("Start returned a nil span")
				return
			}
			span.SetAttributes()
			span.SetAttributes(telemetry.String("k", "v"), telemetry.String("k", "w"))
			span.RecordError(nil)
			span.RecordError(context.Canceled)
			for _, s := range []telemetry.Status{telemetry.StatusUnset, telemetry.StatusOK, telemetry.StatusError, telemetry.Status(99)} {
				span.SetStatus(s)
			}
			span.End()
			checkViolations(t, tr)
		}},
		{"End is idempotent", func(t reporter, newPort func() telemetry.Tracer) {
			tr := newPort()
			_, span := tr.Start(context.Background(), spanName)
			if span == nil {
				t.Errorf("Start returned a nil span")
				return
			}
			span.End()
			span.End()
			checkViolations(t, tr)
		}},
		{"a child span can be started from the returned context", func(t reporter, newPort func() telemetry.Tracer) {
			tr := newPort()
			ctx, parent := tr.Start(context.Background(), spanName)
			if ctx == nil || parent == nil {
				t.Errorf("Start returned ctx=%v span=%v", ctx, parent)
				return
			}
			_, child := tr.Start(ctx, spanName+"_child")
			if child == nil {
				t.Errorf("Start returned a nil child span")
				return
			}
			child.End()
			parent.End()
			checkViolations(t, tr)
		}},
	}
}

// RunMeterContract checks that the Meter returned by factory honours the
// telemetry.Meter contract. factory is called once per subtest.
func RunMeterContract(t *testing.T, factory func(t *testing.T) telemetry.Meter) {
	t.Helper()
	runChecks(t, factory, meterChecks())
}

func contractDescriptor() telemetry.Descriptor {
	return telemetry.Descriptor{Name: "srd5e.contract.things", Unit: "{thing}", Description: "things"}
}

func meterChecks() []check[telemetry.Meter] {
	return []check[telemetry.Meter]{
		{"instruments report the descriptor they were created with", func(t reporter, newPort func() telemetry.Meter) {
			m := newPort()
			d := contractDescriptor()
			if got := m.Counter(d).Descriptor(); got != d {
				t.Errorf("counter descriptor = %+v, want %+v", got, d)
			}
			h := telemetry.Descriptor{Name: "srd5e.contract.sizes", Unit: "{size}"}
			if got := m.Histogram(h).Descriptor(); got != h {
				t.Errorf("histogram descriptor = %+v, want %+v", got, h)
			}
			g := telemetry.Descriptor{Name: "srd5e.contract.level"}
			if got := m.Gauge(g).Descriptor(); got != g {
				t.Errorf("gauge descriptor = %+v, want %+v", got, g)
			}
			checkViolations(t, m)
		}},
		{"an identical instrument can be created twice", func(t reporter, newPort func() telemetry.Meter) {
			m := newPort()
			a, b := m.Counter(contractDescriptor()), m.Counter(contractDescriptor())
			a.Add(context.Background(), 1)
			b.Add(context.Background(), 1)
			checkViolations(t, m)
		}},
		{"instruments accept values with and without attributes", func(t reporter, newPort func() telemetry.Meter) {
			m := newPort()
			ctx := context.Background()
			c := m.Counter(contractDescriptor())
			c.Add(ctx, 0)
			c.Add(ctx, 3, telemetry.String("k", "v"), telemetry.Int("n", 2))
			h := m.Histogram(telemetry.Descriptor{Name: "srd5e.contract.duration", Unit: "s"})
			h.Record(ctx, 0)
			h.Record(ctx, 1.5, telemetry.Bool("b", true))
			g := m.Gauge(telemetry.Descriptor{Name: "srd5e.contract.level"})
			g.Record(ctx, -2.5)
			checkViolations(t, m)
		}},
		{"instruments tolerate a cancelled context", func(t reporter, newPort func() telemetry.Meter) {
			m := newPort()
			m.Counter(contractDescriptor()).Add(cancelled(), 1)
			m.Histogram(telemetry.Descriptor{Name: "srd5e.contract.sizes"}).Record(cancelled(), 1)
			m.Gauge(telemetry.Descriptor{Name: "srd5e.contract.level"}).Record(cancelled(), 1)
			checkViolations(t, m)
		}},
		{"a Timer records into a histogram", func(t reporter, newPort func() telemetry.Meter) {
			m := newPort()
			clock := NewClock(time.Unix(0, 0), time.Second)
			h := m.Histogram(telemetry.Descriptor{Name: "srd5e.contract.duration", Unit: "s"})
			timer := telemetry.StartTimer(clock.Now, h)
			timer.Stop(context.Background(), telemetry.String("k", "v"))
			checkViolations(t, m)
		}},
	}
}
