package telemetry

import (
	"context"
	"time"
)

// Timer measures the time between StartTimer and Stop with an injected clock
// and records the elapsed seconds into a duration histogram.
//
// Domain and use-case code may not read the wall clock, so a use case receives
// a clock through its options and uses Timer. The zero Timer, or one made with
// a nil clock or nil histogram, records nothing. A use case defaults its clock
// to the function value time.Now (referenced, never called directly), which
// callers override with an injected clock; see doc/rules/observability.md. A
// Timer is a value; Stop may be called several times and records each time.
//
// Stop is meant for defer: defer t.Stop() right after StartTimer.
type Timer struct {
	clock func() time.Time
	hist  Histogram
	start time.Time
}

// StartTimer reads clock once and returns a running Timer for hist. A nil clock
// or nil hist yields a Timer that records nothing.
func StartTimer(clock func() time.Time, hist Histogram) Timer {
	if clock == nil || hist == nil {
		return Timer{}
	}
	return Timer{clock: clock, hist: hist, start: clock()}
}

// Stop reads the clock once and records the elapsed time in seconds, with
// attrs, into the histogram. An elapsed time below zero (a clock that went
// backwards) is recorded as zero. It is meant for defer.
func (t Timer) Stop(ctx context.Context, attrs ...Attr) {
	if t.clock == nil || t.hist == nil {
		return
	}
	t.hist.Record(ctx, max(t.clock().Sub(t.start), 0).Seconds(), attrs...)
}
