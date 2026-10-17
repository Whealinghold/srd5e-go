package telemetry

// Telemetry bundles the three ports a use case emits through.
//
// A nil field means "no-op": call WithDefaults once in a constructor and the
// use case never checks whether telemetry is enabled. The zero Telemetry is
// therefore valid. Telemetry is a small value type; it is comparable when the
// dynamic types of its fields are. Construct with keyed literals.
type Telemetry struct {
	Logger Logger
	Tracer Tracer
	Meter  Meter
}

// Noop implements Logger, Tracer and Meter, and does nothing for all three. The
// zero value is ready to use.
type Noop struct {
	NoopLogger
	NoopTracer
	NoopMeter
}

var (
	_ Logger = Noop{}
	_ Tracer = Noop{}
	_ Meter  = Noop{}
)

// WithDefaults returns t with every nil port replaced by its no-op.
func (t Telemetry) WithDefaults() Telemetry {
	if t.Logger == nil {
		t.Logger = NoopLogger{}
	}
	if t.Tracer == nil {
		t.Tracer = NoopTracer{}
	}
	if t.Meter == nil {
		t.Meter = NoopMeter{}
	}
	return t
}
