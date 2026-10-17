package telemetrytest_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

var (
	ctr  = telemetry.Descriptor{Name: "srd5e.test.things", Unit: "{thing}", Description: "things"}
	hist = telemetry.Descriptor{Name: "srd5e.test.sizes", Unit: "{size}"}
)

func TestMeterRecords(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m := new(telemetrytest.Meter)
	c := m.Counter(ctr)
	h := m.Histogram(hist)
	g := m.Gauge(telemetry.Descriptor{Name: "srd5e.test.level"})
	c.Add(ctx, 2, telemetry.String("k", "v"))
	h.Record(ctx, 1.5)
	g.Record(ctx, -3)
	c.Add(ctx, 0)

	got := m.Measurements()
	if len(got) != 4 {
		t.Fatalf("measurements = %+v", got)
	}
	want := []struct {
		name string
		kind telemetrytest.Kind
		unit string
		val  float64
	}{
		{ctr.Name, telemetrytest.KindCounter, "{thing}", 2},
		{hist.Name, telemetrytest.KindHistogram, "{size}", 1.5},
		{"srd5e.test.level", telemetrytest.KindGauge, "", -3},
		{ctr.Name, telemetrytest.KindCounter, "{thing}", 0},
	}
	for i, w := range want {
		if got[i].Name != w.name || got[i].Kind != w.kind || got[i].Unit != w.unit || got[i].Value != w.val {
			t.Errorf("measurement %d = %+v, want %+v", i, got[i], w)
		}
	}
	if len(got[0].Attrs) != 1 || len(got[1].Attrs) != 0 {
		t.Errorf("attrs wrong: %+v", got)
	}
	if n := m.MeasurementsNamed(ctr.Name); len(n) != 2 {
		t.Errorf("MeasurementsNamed = %d, want 2", len(n))
	}
	if len(m.MeasurementsNamed("srd5e.test.none")) != 0 {
		t.Error("unknown name must give no measurements")
	}
	telemetrytest.RequireNoViolations(t, m)
}

func TestMeterInstruments(t *testing.T) {
	t.Parallel()
	m := new(telemetrytest.Meter)
	m.Counter(ctr)
	m.Counter(ctr) // identical: no duplicate
	m.Histogram(hist)
	m.Gauge(telemetry.Descriptor{Name: "srd5e.test.level"})
	got := m.Instruments()
	if len(got) != 3 || got[0].Kind != telemetrytest.KindCounter || got[0].Descriptor != ctr || got[1].Kind != telemetrytest.KindHistogram || got[2].Kind != telemetrytest.KindGauge {
		t.Fatalf("instruments = %+v", got)
	}
	telemetrytest.RequireNoViolations(t, m)
}

func TestMeterViolations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name string
		do   func(*telemetrytest.Meter)
		want error
	}{
		{"invalid counter name", func(m *telemetrytest.Meter) { m.Counter(telemetry.Descriptor{Name: "bad"}).Add(ctx, 1) }, telemetrytest.ErrInvalidName},
		{"invalid histogram name", func(m *telemetrytest.Meter) { m.Histogram(telemetry.Descriptor{Name: "srd5e.x"}).Record(ctx, 1) }, telemetrytest.ErrInvalidName},
		{"invalid gauge name", func(m *telemetrytest.Meter) { m.Gauge(telemetry.Descriptor{}).Record(ctx, 1) }, telemetrytest.ErrInvalidName},
		{"negative counter", func(m *telemetrytest.Meter) { m.Counter(ctr).Add(ctx, -1) }, telemetrytest.ErrNegativeCount},
		{"invalid attr", func(m *telemetrytest.Meter) { m.Counter(ctr).Add(ctx, 1, telemetry.Attr{}) }, telemetrytest.ErrInvalidAttr},
		{"different kind", func(m *telemetrytest.Meter) { m.Counter(ctr); m.Gauge(ctr) }, telemetrytest.ErrInstrumentConflict},
		{"different unit", func(m *telemetrytest.Meter) {
			m.Histogram(hist)
			m.Histogram(telemetry.Descriptor{Name: hist.Name, Unit: "s"})
		}, telemetrytest.ErrInstrumentConflict},
		{"different description", func(m *telemetrytest.Meter) {
			m.Counter(ctr)
			d := ctr
			d.Description = "other"
			m.Counter(d)
		}, telemetrytest.ErrInstrumentConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := new(telemetrytest.Meter)
			tc.do(m)
			if v := m.Violations(); len(v) != 1 || !errors.Is(v[0], tc.want) {
				t.Fatalf("violations = %v, want one %v", v, tc.want)
			}
		})
	}
}

func TestMeterBadInstrumentsRecordNothing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m := new(telemetrytest.Meter)
	m.Counter(ctr)
	conflicting := m.Gauge(ctr) // first definition is kept
	conflicting.Record(ctx, 5)
	m.Counter(telemetry.Descriptor{Name: "bad"}).Add(ctx, 1)
	m.Counter(ctr).Add(ctx, -1)
	if len(m.Measurements()) != 0 {
		t.Fatalf("measurements = %+v", m.Measurements())
	}
	if got := m.Instruments(); len(got) != 1 || got[0].Kind != telemetrytest.KindCounter {
		t.Fatalf("instruments = %+v", got)
	}
	if conflicting.Descriptor() != ctr {
		t.Fatal("detached instrument must still report its descriptor")
	}
}

func TestMeterClearKeepsInstruments(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m := new(telemetrytest.Meter)
	held := m.Counter(ctr)
	held.Add(ctx, 1)
	held.Add(ctx, -1)
	m.Clear()
	if len(m.Measurements()) != 0 || len(m.Violations()) != 0 || len(m.Instruments()) != 1 {
		t.Fatal("Clear must drop measurements and violations only")
	}
	held.Add(ctx, 4)
	if got := m.Measurements(); len(got) != 1 || got[0].Value != 4 {
		t.Fatalf("earlier instrument must keep recording: %+v", got)
	}
}

func TestMeterMeasurementsAreCopies(t *testing.T) {
	t.Parallel()
	m := new(telemetrytest.Meter)
	m.Counter(ctr).Add(context.Background(), 1, telemetry.String("k", "v"))
	m.Measurements()[0].Attrs[0] = telemetry.String("changed", "x")
	if m.Measurements()[0].Attrs[0].Key() != "k" {
		t.Fatal("mutation leaked into the meter")
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()
	for k, want := range map[telemetrytest.Kind]string{
		telemetrytest.KindCounter: "counter", telemetrytest.KindHistogram: "histogram",
		telemetrytest.KindGauge: "gauge", telemetrytest.Kind(7): "invalid",
	} {
		if k.String() != want {
			t.Errorf("Kind(%d) = %q, want %q", k, k.String(), want)
		}
	}
}

func TestMeterConcurrent(t *testing.T) {
	t.Parallel()
	m := new(telemetrytest.Meter)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Counter(ctr).Add(context.Background(), 1)
		}()
	}
	wg.Wait()
	if len(m.Measurements()) != 20 || len(m.Instruments()) != 1 {
		t.Fatalf("measurements=%d instruments=%d", len(m.Measurements()), len(m.Instruments()))
	}
}
