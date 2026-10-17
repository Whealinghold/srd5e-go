package telemetry

import "context"

// Descriptor names an instrument and says what it measures.
//
// Name follows srd5e.<context>.<noun> (see ValidName). Unit follows UCUM and is
// stated here, not in the name: durations are "s", counts of things use an
// annotation such as "{modifier}"; it may be empty. Description is
// documentation for dashboard authors and may be empty. Descriptor is
// comparable. Construct with keyed literals.
type Descriptor struct {
	Name        string
	Unit        string
	Description string
}

// Counter counts things: a value that only goes up. Attributes are per call.
type Counter interface {
	// Descriptor returns the descriptor exactly as given at creation.
	Descriptor() Descriptor
	// Add adds n to the count. A recording double reports a negative n.
	Add(ctx context.Context, n int64, attrs ...Attr)
}

// Histogram records the distribution of measurements such as durations and
// sizes. Attributes are per call.
type Histogram interface {
	// Descriptor returns the descriptor exactly as given at creation.
	Descriptor() Descriptor
	// Record records one measurement.
	Record(ctx context.Context, value float64, attrs ...Attr)
}

// Gauge records the current value of something that goes up and down.
// Attributes are per call.
type Gauge interface {
	// Descriptor returns the descriptor exactly as given at creation.
	Descriptor() Descriptor
	// Record records the current value.
	Record(ctx context.Context, value float64, attrs ...Attr)
}

// Meter is the port through which the module creates instruments.
//
// A use case creates its instruments once, in its constructor. Creating an
// instrument again with an identical descriptor returns an equivalent one. The
// methods return no error; a recording double reports misuse (an invalid name, a
// conflicting redefinition) as a violation. Durations are measured with Timer,
// not by the Meter.
//
// The methods return no error: a backend handles its own failures, and a backend
// that panics is the host's bug; srd5e does not recover.
//
// Implementations, and the instruments they return, MUST be safe for concurrent
// use: a d20.Resolver creates its instruments once and then shares them across
// every call, so Add and Record may run on many goroutines at once.
type Meter interface {
	// Counter creates or returns the counter described by d.
	Counter(d Descriptor) Counter
	// Histogram creates or returns the histogram described by d. Create a
	// duration histogram with Unit "s".
	Histogram(d Descriptor) Histogram
	// Gauge creates or returns the gauge described by d.
	Gauge(d Descriptor) Gauge
}

// NoopMeter is a Meter whose instruments record nothing. It validates nothing.
// The zero value is ready to use.
type NoopMeter struct{}

var _ Meter = NoopMeter{}

// Counter returns a counter that counts nothing.
func (NoopMeter) Counter(d Descriptor) Counter { return noopInstrument{d} }

// Histogram returns a histogram that records nothing.
func (NoopMeter) Histogram(d Descriptor) Histogram { return noopInstrument{d} }

// Gauge returns a gauge that records nothing.
func (NoopMeter) Gauge(d Descriptor) Gauge { return noopInstrument{d} }

// noopInstrument implements Counter, Histogram and Gauge.
type noopInstrument struct{ d Descriptor }

func (i noopInstrument) Descriptor() Descriptor                 { return i.d }
func (noopInstrument) Add(context.Context, int64, ...Attr)      {}
func (noopInstrument) Record(context.Context, float64, ...Attr) {}
