package telemetrytest

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// failures collects failures instead of failing the test.
type failures struct{ msgs []string }

func (*failures) Helper() {}
func (f *failures) Errorf(format string, _ ...any) {
	f.msgs = append(f.msgs, format)
}

// failsCheck reports whether the named check records at least one failure for
// the broken port, recovering a panic as a failure too.
func failsCheck[P any](checks []check[P], name string, port P) (failed bool) {
	for _, c := range checks {
		if c.name != name {
			continue
		}
		f := &failures{}
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		c.run(f, func() P { return port })
		return len(f.msgs) > 0
	}
	panic("unknown check " + name)
}

// brokenLogger runs attrs even when disabled, is not monotonic in level and
// panics on levels outside the defined set when panicky is true.
type brokenLogger struct{ panicky, runAttrs, notMonotonic bool }

func (l brokenLogger) Enabled(_ context.Context, lv telemetry.Level) bool {
	if l.panicky && (lv < telemetry.LevelTrace || lv > telemetry.LevelError) {
		panic("unknown level")
	}
	if l.notMonotonic {
		return lv == telemetry.LevelTrace
	}
	return false
}

func (l brokenLogger) Log(_ context.Context, lv telemetry.Level, _ string, attrs func() []telemetry.Attr) {
	if l.panicky && (lv < telemetry.LevelTrace || lv > telemetry.LevelError) {
		panic("unknown level")
	}
	if l.runAttrs && attrs != nil {
		attrs()
	}
}

type brokenSpan struct{}

func (brokenSpan) SetAttributes(...telemetry.Attr) {}
func (brokenSpan) SetStatus(telemetry.Status)      {}
func (brokenSpan) RecordError(error)               {}
func (brokenSpan) End()                            {}

// brokenTracer returns a nil span or a context that drops the caller's values.
type brokenTracer struct{ nilSpan, dropCtx bool }

func (tr brokenTracer) Start(ctx context.Context, _ string, _ ...telemetry.Attr) (context.Context, telemetry.Span) {
	if tr.dropCtx {
		ctx = context.Background()
	}
	if tr.nilSpan {
		return ctx, nil
	}
	return ctx, brokenSpan{}
}

type wrongDescriptor struct{ d telemetry.Descriptor }

func (w wrongDescriptor) Descriptor() telemetry.Descriptor {
	return telemetry.Descriptor{Name: "other"}
}
func (wrongDescriptor) Add(context.Context, int64, ...telemetry.Attr) {
}
func (wrongDescriptor) Record(context.Context, float64, ...telemetry.Attr) {}

type brokenMeter struct{}

func (brokenMeter) Counter(d telemetry.Descriptor) telemetry.Counter     { return wrongDescriptor{d} }
func (brokenMeter) Histogram(d telemetry.Descriptor) telemetry.Histogram { return wrongDescriptor{d} }
func (brokenMeter) Gauge(d telemetry.Descriptor) telemetry.Gauge         { return wrongDescriptor{d} }

func TestContractsRejectBrokenPorts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func() bool
	}{
		{"logger panics on unknown level", func() bool {
			return failsCheck(loggerChecks(), "Log accepts nil attrs at every level", telemetry.Logger(brokenLogger{panicky: true}))
		}},
		{"logger runs attrs while disabled", func() bool {
			return failsCheck(loggerChecks(), "attrs is called only when enabled, and at most once", telemetry.Logger(brokenLogger{runAttrs: true}))
		}},
		{"logger not monotonic", func() bool {
			return failsCheck(loggerChecks(), "Enabled is monotonic in level", telemetry.Logger(brokenLogger{notMonotonic: true}))
		}},
		{"tracer nil span", func() bool {
			return failsCheck(tracerChecks(), "End is idempotent", telemetry.Tracer(brokenTracer{nilSpan: true}))
		}},
		{"tracer nil span with attrs", func() bool {
			return failsCheck(tracerChecks(), "Start accepts attributes and a cancelled context", telemetry.Tracer(brokenTracer{nilSpan: true}))
		}},
		{"tracer drops context values", func() bool {
			return failsCheck(tracerChecks(), "Start returns a span and keeps the caller's context values", telemetry.Tracer(brokenTracer{dropCtx: true}))
		}},
		{"meter wrong descriptor", func() bool {
			return failsCheck(meterChecks(), "instruments report the descriptor they were created with", telemetry.Meter(brokenMeter{}))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !tt.fn() {
				t.Error("contract did not report the broken port")
			}
		})
	}
}
