package telemetrytest

import "github.com/Whealinghold/srd5e-go/telemetry"

// Recorder bundles one Logger, one Tracer and one Meter. Pass Telemetry to the
// code under test, then read the fields to assert what it emitted.
type Recorder struct {
	Logger *Logger
	Tracer *Tracer
	Meter  *Meter
}

var _ Violator = (*Recorder)(nil)

// NewRecorder returns a Recorder with the three recording doubles.
func NewRecorder() *Recorder {
	return &Recorder{Logger: new(Logger), Tracer: new(Tracer), Meter: new(Meter)}
}

// Telemetry returns a bundle wrapping exactly the recorder's three doubles.
func (r *Recorder) Telemetry() telemetry.Telemetry {
	return telemetry.Telemetry{Logger: r.Logger, Tracer: r.Tracer, Meter: r.Meter}
}

// Clear removes the log, span and measurement records and all violations.
func (r *Recorder) Clear() {
	r.Logger.Clear()
	r.Tracer.Clear()
	r.Meter.Clear()
}

// Violations returns the misuse collected by all three doubles.
func (r *Recorder) Violations() []error {
	out := r.Logger.Violations()
	out = append(out, r.Tracer.Violations()...)
	return append(out, r.Meter.Violations()...)
}
