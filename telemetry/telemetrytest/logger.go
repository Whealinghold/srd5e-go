package telemetrytest

import (
	"context"
	"slices"
	"sync"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// LogRecord is one log event stored by a Logger.
type LogRecord struct {
	Level   telemetry.Level
	Message string
	// Attrs is what the attrs closure returned; empty when none was given.
	Attrs []telemetry.Attr
}

// Logger is a telemetry.Logger that stores every enabled event.
//
// Events below the minimum level (LevelTrace by default, so all are recorded)
// are not recorded and Enabled says so. For a recorded event the attrs closure
// is called exactly once; for a disabled event it is never called. It is safe
// for concurrent use; the zero value is ready to use.
type Logger struct {
	mu      sync.Mutex
	min     telemetry.Level
	records []LogRecord
	viol    violations
}

var _ telemetry.Logger = (*Logger)(nil)

// NewLogger returns a Logger that records events at min and above.
func NewLogger(min telemetry.Level) *Logger { return &Logger{min: min} }

// SetMinLevel changes the lowest recorded level, to simulate a configured log
// level.
func (l *Logger) SetMinLevel(min telemetry.Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.min = min
}

// Enabled reports whether level is at or above the minimum level.
func (l *Logger) Enabled(_ context.Context, level telemetry.Level) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return level >= l.min
}

// Log records the event when its level is enabled.
func (l *Logger) Log(ctx context.Context, level telemetry.Level, msg string, attrs func() []telemetry.Attr) {
	if !l.Enabled(ctx, level) {
		return
	}
	var got []telemetry.Attr
	if attrs != nil {
		got = l.viol.validAttrs(attrs())
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, LogRecord{Level: level, Message: msg, Attrs: got})
}

// Records returns the recorded events in call order. The result is a copy.
func (l *Logger) Records() []LogRecord {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := slices.Clone(l.records)
	for i := range out {
		out[i].Attrs = slices.Clone(out[i].Attrs)
	}
	return out
}

// RecordsAt returns the recorded events at exactly level, in call order.
func (l *Logger) RecordsAt(level telemetry.Level) []LogRecord {
	var out []LogRecord
	for _, r := range l.Records() {
		if r.Level == level {
			out = append(out, r)
		}
	}
	return out
}

// Clear forgets the recorded events and violations. The minimum level is kept.
func (l *Logger) Clear() {
	l.mu.Lock()
	l.records = nil
	l.mu.Unlock()
	l.viol.reset()
}

// Violations returns the misuse collected so far.
func (l *Logger) Violations() []error { return l.viol.snapshot() }
