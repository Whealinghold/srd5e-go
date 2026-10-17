package telemetry_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func TestAttr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		attr  telemetry.Attr
		key   string
		value any
		valid bool
	}{
		{"string", telemetry.String("k", "v"), "k", "v", true},
		{"empty string value is valid", telemetry.String("k", ""), "k", "", true},
		{"int widens to int64", telemetry.Int("k", 3), "k", int64(3), true},
		{"int64", telemetry.Int64("k", -4), "k", int64(-4), true},
		{"float", telemetry.Float("k", 1.5), "k", 1.5, true},
		{"bool", telemetry.Bool("k", false), "k", false, true},
		{"empty key is invalid", telemetry.String("", "v"), "", "v", false},
		{"zero attr is invalid", telemetry.Attr{}, "", nil, false},
		{"nil error is invalid", telemetry.Err(nil), "", nil, false},
		{"error message", telemetry.Err(errors.New("boom")), telemetry.AttrErrorMessage, "boom", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.attr.Key() != tc.key || tc.attr.Value() != tc.value || tc.attr.Valid() != tc.valid {
				t.Fatalf("got (%q, %v, %v), want (%q, %v, %v)", tc.attr.Key(), tc.attr.Value(), tc.attr.Valid(), tc.key, tc.value, tc.valid)
			}
		})
	}
	if telemetry.String("k", "v") != telemetry.String("k", "v") {
		t.Fatal("equal attrs must compare equal")
	}
}

func TestLevel(t *testing.T) {
	t.Parallel()
	want := []struct {
		l    telemetry.Level
		name string
	}{
		{telemetry.LevelTrace, "trace"}, {telemetry.LevelDebug, "debug"}, {telemetry.LevelInfo, "info"},
		{telemetry.LevelWarn, "warn"}, {telemetry.LevelError, "error"},
	}
	for i, w := range want {
		if w.l.String() != w.name || !w.l.Valid() {
			t.Errorf("level %d: String=%q Valid=%v", i, w.l.String(), w.l.Valid())
		}
		if i > 0 && w.l <= want[i-1].l {
			t.Errorf("%v must be more severe than %v", w.l, want[i-1].l)
		}
	}
	if (telemetry.Level(0) != telemetry.LevelTrace) || telemetry.LevelError+1 != telemetry.Level(5) {
		t.Fatal("zero value must be LevelTrace and there is no level above error")
	}
	for _, bad := range []telemetry.Level{-1, 5, 99} {
		if bad.Valid() || bad.String() != "invalid" {
			t.Errorf("level %d: Valid=%v String=%q", bad, bad.Valid(), bad.String())
		}
	}
}

func TestStatus(t *testing.T) {
	t.Parallel()
	for s, want := range map[telemetry.Status]string{
		telemetry.StatusUnset: "unset", telemetry.StatusOK: "ok", telemetry.StatusError: "error", telemetry.Status(9): "invalid",
	} {
		if s.String() != want {
			t.Errorf("Status(%d).String() = %q, want %q", s, s.String(), want)
		}
	}
	if telemetry.Status(0) != telemetry.StatusUnset {
		t.Fatal("zero Status must be unset")
	}
}

func TestSharedKeys(t *testing.T) {
	t.Parallel()
	if telemetry.AttrOutcome != "srd5e.outcome" || telemetry.AttrErrorType != "error.type" {
		t.Fatal("attribute keys are a public contract")
	}
	if telemetry.OutcomeSuccess != "success" || telemetry.OutcomeFailure != "failure" {
		t.Fatal("outcome values are a public contract")
	}
}

var nameRE = regexp.MustCompile(`^srd5e\.[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

func TestValidName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"three parts", "srd5e.d20_test.resolve", true},
		{"four parts", "srd5e.d20_test.resolve.step", true},
		{"digits and underscores", "srd5e.a1.b_2", true},
		{"two parts", "srd5e.resolve", false},
		{"empty", "", false},
		{"wrong prefix", "other.d20_test.resolve", false},
		{"uppercase", "srd5e.D20.resolve", false},
		{"segment starts with digit", "srd5e.1a.resolve", false},
		{"segment starts with underscore", "srd5e._a.b", false},
		{"empty segment", "srd5e..resolve", false},
		{"trailing dot", "srd5e.a.b.", false},
		{"hyphen", "srd5e.a-b.c", false},
		{"space", "srd5e.a b.c", false},
		{"unicode", "srd5e.é.c", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := telemetry.ValidName(tc.in); got != tc.want {
				t.Fatalf("ValidName(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func FuzzValidName(f *testing.F) {
	for _, s := range []string{"srd5e.a.b", "srd5e..", "", "srd5e.A.b", "srd5e.a.b\n", "x.y.z", "srd5e.a.b.c_9"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := telemetry.ValidName(s), nameRE.MatchString(s); got != want {
			t.Fatalf("ValidName(%q) = %v, regexp says %v", s, got, want)
		}
	})
}

func TestNoopContracts(t *testing.T) {
	t.Parallel()
	t.Run("logger", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunLoggerContract(t, func(*testing.T) telemetry.Logger { return telemetry.NoopLogger{} })
	})
	t.Run("tracer", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunTracerContract(t, func(*testing.T) telemetry.Tracer { return telemetry.NoopTracer{} })
	})
	t.Run("meter", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunMeterContract(t, func(*testing.T) telemetry.Meter { return telemetry.NoopMeter{} })
	})
	t.Run("Noop implements all three", func(t *testing.T) {
		t.Parallel()
		telemetrytest.RunLoggerContract(t, func(*testing.T) telemetry.Logger { return telemetry.Noop{} })
		telemetrytest.RunTracerContract(t, func(*testing.T) telemetry.Tracer { return telemetry.Noop{} })
		telemetrytest.RunMeterContract(t, func(*testing.T) telemetry.Meter { return telemetry.Noop{} })
	})
}

type ctxKey struct{}

func TestNoopBehaviour(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxKey{}, 1)
	var l telemetry.NoopLogger
	called := false
	l.Log(ctx, telemetry.LevelError, "m", func() []telemetry.Attr { called = true; return nil })
	if called || l.Enabled(ctx, telemetry.LevelError) {
		t.Fatal("noop logger must enable nothing and never call attrs")
	}
	got, span := telemetry.NoopTracer{}.Start(ctx, "not even a valid name")
	if got != ctx || span != (telemetry.NoopSpan{}) {
		t.Fatal("noop tracer must return ctx unchanged and a NoopSpan, validating nothing")
	}
	c := telemetry.NoopMeter{}.Counter(telemetry.Descriptor{Name: "x"})
	c.Add(ctx, -5) // not validated
	if c.Descriptor().Name != "x" {
		t.Fatal("descriptor must be kept")
	}
}

func TestTelemetryWithDefaults(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	tests := []struct {
		name string
		in   telemetry.Telemetry
		keep func(telemetry.Telemetry) bool
	}{
		{"zero", telemetry.Telemetry{}, func(telemetry.Telemetry) bool { return true }},
		{"logger kept", telemetry.Telemetry{Logger: rec.Logger}, func(t telemetry.Telemetry) bool { return t.Logger == rec.Logger }},
		{"tracer kept", telemetry.Telemetry{Tracer: rec.Tracer}, func(t telemetry.Telemetry) bool { return t.Tracer == rec.Tracer }},
		{"meter kept", telemetry.Telemetry{Meter: rec.Meter}, func(t telemetry.Telemetry) bool { return t.Meter == rec.Meter }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.in.WithDefaults()
			if got.Logger == nil || got.Tracer == nil || got.Meter == nil {
				t.Fatalf("nil port after WithDefaults: %+v", got)
			}
			if !tc.keep(got) {
				t.Fatal("supplied port was replaced")
			}
		})
	}
	if (telemetry.Telemetry{}).WithDefaults() != (telemetry.Telemetry{}).WithDefaults() {
		t.Fatal("the default bundle must equal itself")
	}
}

func TestTimer(t *testing.T) {
	t.Parallel()
	start := time.Unix(100, 0)
	hist := func() (*telemetrytest.Meter, telemetry.Histogram) {
		m := new(telemetrytest.Meter)
		return m, m.Histogram(telemetry.Descriptor{Name: "srd5e.test.duration", Unit: "s"})
	}
	tests := []struct {
		name string
		step time.Duration
		want []float64
	}{
		{"records elapsed seconds", 1500 * time.Millisecond, []float64{1.5}},
		{"zero elapsed", 0, []float64{0}},
		{"backwards clock is clamped to zero", -time.Second, []float64{0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, h := hist()
			timer := telemetry.StartTimer(telemetrytest.NewClock(start, tc.step).Now, h)
			timer.Stop(context.Background(), telemetry.String("k", "v"))
			got := m.MeasurementsNamed("srd5e.test.duration")
			if len(got) != len(tc.want) || got[0].Value != tc.want[0] || len(got[0].Attrs) != 1 {
				t.Fatalf("measurements = %+v, want %v", got, tc.want)
			}
		})
	}
	t.Run("nil clock, nil histogram and zero timer record nothing", func(t *testing.T) {
		t.Parallel()
		m, h := hist()
		telemetry.StartTimer(nil, h).Stop(context.Background())
		telemetry.StartTimer(telemetrytest.NewClock(start, time.Second).Now, nil).Stop(context.Background())
		telemetry.Timer{}.Stop(context.Background())
		if len(m.Measurements()) != 0 {
			t.Fatal("recorded without clock or histogram")
		}
	})
	t.Run("clock is read once at start and once at stop", func(t *testing.T) {
		t.Parallel()
		m, h := hist()
		clock := telemetrytest.NewClock(start, time.Second)
		timer := telemetry.StartTimer(clock.Now, h)
		timer.Stop(context.Background())
		timer.Stop(context.Background())
		got := m.Measurements()
		if len(got) != 2 || got[0].Value != 1 || got[1].Value != 2 {
			t.Fatalf("got %+v", got)
		}
	})
}

// TestPortDocComments pins the FR-056 and FR-057 sentences by reading the
// declarations with go/parser.
func TestPortDocComments(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs["telemetry"]
	if pkg == nil {
		t.Fatal("package telemetry not found")
	}
	docs := map[string]string{"package": ""}
	for _, f := range pkg.Files {
		if f.Doc != nil {
			docs["package"] += f.Doc.Text()
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range gd.Specs {
				ts, ok := s.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if gd.Doc != nil {
					docs[ts.Name.Name] = gd.Doc.Text()
				}
				if it, ok := ts.Type.(*ast.InterfaceType); ok && ts.Name.Name == "Tracer" {
					for _, m := range it.Methods.List {
						if m.Doc != nil && len(m.Names) > 0 {
							docs["Tracer."+m.Names[0].Name] = m.Doc.Text()
						}
					}
				}
			}
		}
	}
	tests := []struct{ name, want string }{
		{"package", "srd5e does not recover"},
		{"Logger", "srd5e does not recover"},
		{"Tracer", "srd5e does not recover"},
		{"Meter", "srd5e does not recover"},
		{"Tracer.Start", "defers Span.End"},
		{"Tracer.Start", "passes the returned context"},
		{"Timer", "defer"},
	}
	for _, tt := range tests {
		if !strings.Contains(strings.Join(strings.Fields(docs[tt.name]), " "), tt.want) {
			t.Errorf("doc comment of %s lacks %q", tt.name, tt.want)
		}
	}
}
