package telemetrytest

import (
	"context"
	"fmt"
	"sync"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Kind is the kind of an instrument.
type Kind int

// The instrument kinds.
const (
	KindCounter Kind = iota
	KindHistogram
	KindGauge
)

// String returns the lowercase kind name, or "invalid".
func (k Kind) String() string {
	switch k {
	case KindCounter:
		return "counter"
	case KindHistogram:
		return "histogram"
	case KindGauge:
		return "gauge"
	default:
		return "invalid"
	}
}

// Measurement is one value stored by a Meter.
type Measurement struct {
	Name string
	Kind Kind
	Unit string
	// Value is the amount added to a counter, or the value recorded. A
	// duration is in seconds.
	Value float64
	Attrs []telemetry.Attr
}

// InstrumentInfo describes an instrument created through a Meter.
type InstrumentInfo struct {
	Kind       Kind
	Descriptor telemetry.Descriptor
}

// Meter is a telemetry.Meter that stores one Measurement per recorded value.
//
// Creating an instrument again with an identical kind and descriptor returns an
// equivalent instrument. Misuse is collected as a violation: ErrInvalidName for
// a name that is not srd5e.<context>.<noun>, ErrInstrumentConflict for a name
// redefined with a different kind, unit or description (the first definition is
// kept), ErrNegativeCount for a negative counter increment, ErrInvalidAttr for a
// bad attribute. The instrument returned after a name or conflict violation
// records nothing. It is safe for concurrent use; the zero value is ready to
// use.
type Meter struct {
	mu           sync.Mutex
	instruments  []InstrumentInfo
	byName       map[string]InstrumentInfo
	measurements []Measurement
	viol         violations
}

var _ telemetry.Meter = (*Meter)(nil)

// instrument implements Counter, Histogram and Gauge.
type instrument struct {
	m    *Meter
	kind Kind
	d    telemetry.Descriptor
}

func (i *instrument) Descriptor() telemetry.Descriptor { return i.d }

func (i *instrument) Add(_ context.Context, n int64, attrs ...telemetry.Attr) {
	if n < 0 {
		i.m.viol.add(fmt.Errorf("%w: %s by %d", ErrNegativeCount, i.d.Name, n))
		return
	}
	i.m.measure(i, float64(n), attrs)
}

func (i *instrument) Record(_ context.Context, value float64, attrs ...telemetry.Attr) {
	i.m.measure(i, value, attrs)
}

func (m *Meter) measure(i *instrument, value float64, attrs []telemetry.Attr) {
	valid := m.viol.validAttrs(attrs)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.measurements = append(m.measurements, Measurement{Name: i.d.Name, Kind: i.kind, Unit: i.d.Unit, Value: value, Attrs: valid})
}

// obtain returns the recording instrument for d, or nil after a violation.
func (m *Meter) obtain(kind Kind, d telemetry.Descriptor) *instrument {
	if !telemetry.ValidName(d.Name) {
		m.viol.add(fmt.Errorf("%w: instrument %q", ErrInvalidName, d.Name))
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	want := InstrumentInfo{Kind: kind, Descriptor: d}
	if have, ok := m.byName[d.Name]; ok {
		if have != want {
			m.viol.add(fmt.Errorf("%w: %s exists as a %s with unit %q", ErrInstrumentConflict, d.Name, have.Kind, have.Descriptor.Unit))
			return nil
		}
		return &instrument{m: m, kind: kind, d: d}
	}
	if m.byName == nil {
		m.byName = map[string]InstrumentInfo{}
	}
	m.byName[d.Name] = want
	m.instruments = append(m.instruments, want)
	return &instrument{m: m, kind: kind, d: d}
}

// Counter creates or returns the counter described by d.
func (m *Meter) Counter(d telemetry.Descriptor) telemetry.Counter {
	if i := m.obtain(KindCounter, d); i != nil {
		return i
	}
	return telemetry.NoopMeter{}.Counter(d)
}

// Histogram creates or returns the histogram described by d.
func (m *Meter) Histogram(d telemetry.Descriptor) telemetry.Histogram {
	if i := m.obtain(KindHistogram, d); i != nil {
		return i
	}
	return telemetry.NoopMeter{}.Histogram(d)
}

// Gauge creates or returns the gauge described by d.
func (m *Meter) Gauge(d telemetry.Descriptor) telemetry.Gauge {
	if i := m.obtain(KindGauge, d); i != nil {
		return i
	}
	return telemetry.NoopMeter{}.Gauge(d)
}

// Measurements returns the recorded values in call order across all
// instruments. The result is a copy.
func (m *Meter) Measurements() []Measurement {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Measurement, len(m.measurements))
	for i, x := range m.measurements {
		out[i] = x
		out[i].Attrs = append([]telemetry.Attr(nil), x.Attrs...)
	}
	return out
}

// MeasurementsNamed returns the values recorded into the instrument called
// name, in call order.
func (m *Meter) MeasurementsNamed(name string) []Measurement {
	var out []Measurement
	for _, x := range m.Measurements() {
		if x.Name == name {
			out = append(out, x)
		}
	}
	return out
}

// Instruments returns the instruments created so far, in creation order.
func (m *Meter) Instruments() []InstrumentInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]InstrumentInfo(nil), m.instruments...)
}

// Clear forgets the recorded values and violations. Instruments and earlier
// instrument handles are kept and keep recording.
func (m *Meter) Clear() {
	m.mu.Lock()
	m.measurements = nil
	m.mu.Unlock()
	m.viol.reset()
}

// Violations returns the misuse collected so far.
func (m *Meter) Violations() []error { return m.viol.snapshot() }
