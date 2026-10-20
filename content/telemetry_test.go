package content_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func attrsOf(attrs []telemetry.Attr) map[string]any {
	m := make(map[string]any, len(attrs))
	for _, a := range attrs {
		m[a.Key()] = a.Value()
	}
	return m
}

// recorded is a Loader wired to recording doubles and a stepping clock.
type recorded struct {
	loader content.Loader
	rec    *telemetrytest.Recorder
	fake   *contenttest.FakeDecoder
}

func newRecorded(t *testing.T) recorded {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), 250*time.Millisecond)
	fake := contenttest.NewFakeDecoder()
	return recorded{loader: mustLoader(t, fake, content.WithTelemetry(rec.Telemetry()), content.WithClock(clock.Now)), rec: rec, fake: fake}
}

func (r recorded) load(t *testing.T, doc string) (content.LoadSetResult, error) {
	t.Helper()
	return r.loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(doc)})
}

func (r recorded) spans(t *testing.T) (load, decode telemetrytest.SpanRecord) {
	t.Helper()
	spans := r.rec.Tracer.Spans()
	if len(spans) != 2 {
		t.Fatalf("spans = %+v, want the load span and its decode child", spans)
	}
	load, decode = spans[0], spans[1]
	if load.Name != content.SpanLoad || decode.Name != content.SpanDecode || decode.ParentID != load.ID || load.ParentID != 0 || !load.Ended || !decode.Ended {
		t.Fatalf("spans = %+v, want %s containing %s, both ended", spans, content.SpanLoad, content.SpanDecode)
	}
	return load, decode
}

func outcomeOf(t *testing.T, s telemetrytest.SpanRecord) any {
	t.Helper()
	v, ok := s.AttrValue(telemetry.AttrOutcome)
	if !ok {
		t.Fatalf("span %s has no outcome", s.Name)
	}
	return v
}

type measure struct {
	value float64
	attrs map[string]any
}

func (r recorded) measures(name string) []measure {
	var out []measure
	for _, m := range r.rec.Meter.MeasurementsNamed(name) {
		out = append(out, measure{m.Value, attrsOf(m.Attrs)})
	}
	return out
}

func (r recorded) requireClean(t *testing.T) {
	t.Helper()
	if logs := r.rec.Logger.Records(); len(logs) != 0 {
		t.Errorf("logged %+v; the use case must not log", logs)
	}
	telemetrytest.RequireNoViolations(t, r.rec)
}

func (r recorded) requireOneDuration(t *testing.T) {
	t.Helper()
	d := r.rec.Meter.MeasurementsNamed(content.MetricLoadDuration)
	if len(d) != 1 || d[0].Value != 0.25 || d[0].Unit != "s" || len(d[0].Attrs) != 0 || d[0].Kind != telemetrytest.KindHistogram {
		t.Errorf("load_duration = %+v, want one measurement of 0.25 s with no attributes", d)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-079 to FR-081, FR-083 - a loaded document gets the
// load span with its decode child, loads +1 success, entries per kind (none
// with zero), one duration, and no log.
// Path: happy.
func TestTelemetryLoaded(t *testing.T) {
	t.Parallel()
	r := newRecorded(t)
	r.fake.Return("d", content.MustSet(plain(content.KindFeat, "c"), plain(content.KindSpell, "a"), plain(content.KindSpell, "b")), nil)
	if _, err := r.load(t, "d"); err != nil {
		t.Fatal(err)
	}
	load, decode := r.spans(t)
	for _, s := range []telemetrytest.SpanRecord{load, decode} {
		if outcomeOf(t, s) != telemetry.OutcomeSuccess || s.Status != telemetry.StatusUnset || len(s.Errors) != 0 {
			t.Errorf("span %s = %+v, want outcome success, no error status", s.Name, s)
		}
	}
	if got := r.measures(content.MetricLoads); len(got) != 1 || got[0].value != 1 || got[0].attrs[telemetry.AttrOutcome] != "success" || len(got[0].attrs) != 1 {
		t.Errorf("loads = %+v", got)
	}
	want := []measure{
		{2, map[string]any{content.AttrKind: "spell"}},
		{1, map[string]any{content.AttrKind: "feat"}},
	}
	got := r.measures(content.MetricEntriesLoaded)
	slices.SortFunc(got, func(a, b measure) int {
		return strings.Compare(a.attrs[content.AttrKind].(string), b.attrs[content.AttrKind].(string))
	})
	slices.SortFunc(want, func(a, b measure) int {
		return strings.Compare(a.attrs[content.AttrKind].(string), b.attrs[content.AttrKind].(string))
	})
	if len(got) != 2 || got[0].value != want[0].value || got[1].value != want[1].value || got[0].attrs[content.AttrKind] != "feat" || got[1].attrs[content.AttrKind] != "spell" {
		t.Errorf("entries_loaded = %+v, want 1 feat and 2 spell", got)
	}
	if got := r.measures(content.MetricDecodeErrors); len(got) != 0 {
		t.Errorf("decode_errors = %+v for a loaded document", got)
	}
	r.requireOneDuration(t)
	r.requireClean(t)
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-081 - an empty set adds nothing to entries_loaded
// (no add of 0), and still counts the load.
// Path: unhappy (empty case).
func TestTelemetryEmptySetAddsNoEntries(t *testing.T) {
	t.Parallel()
	r := newRecorded(t)
	r.fake.Return("d", content.Set{}, nil)
	if _, err := r.load(t, "d"); err != nil {
		t.Fatal(err)
	}
	if got := r.measures(content.MetricEntriesLoaded); len(got) != 0 {
		t.Errorf("entries_loaded = %+v, want none", got)
	}
	if got := r.measures(content.MetricLoads); len(got) != 1 || got[0].attrs[telemetry.AttrOutcome] != "success" {
		t.Errorf("loads = %+v", got)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-082, FR-100, FR-085 and doc/plans/plan-phase-2.md P13, FR-153, FR-154 - a
// rejected document sets both spans to error status, counts loads once as
// failure, adds problems to decode_errors once per kind with the count of that kind, and never logs.
// Path: unhappy (failure).
func TestTelemetryRejected(t *testing.T) {
	t.Parallel()
	r := newRecorded(t)
	secretPath := "/spells/0/very-secret-key"
	r.fake.Return("d", content.Set{}, problemErr(
		content.NewProblem(secretPath, content.ProblemUnknownField),
		content.NewProblem("/spells/1/school", content.ProblemMissingField),
		content.NewProblem("/spells/2/school", content.ProblemMissingField),
		content.NewProblem("/spells/3/school", content.ProblemMissingField),
	))
	res, err := r.load(t, "d")
	if _, ok := res.(content.LoadSetRejected); !ok || err != nil {
		t.Fatalf("LoadSet = (%T, %v), want LoadSetRejected", res, err)
	}
	load, decode := r.spans(t)
	for _, s := range []telemetrytest.SpanRecord{load, decode} {
		if outcomeOf(t, s) != telemetry.OutcomeFailure || s.Status != telemetry.StatusError || len(s.Errors) != 0 {
			t.Errorf("span %s = %+v, want outcome failure and error status", s.Name, s)
		}
	}
	if got := r.measures(content.MetricLoads); len(got) != 1 || got[0].value != 1 || got[0].attrs[telemetry.AttrOutcome] != "failure" || len(got[0].attrs) != 1 {
		t.Errorf("loads = %+v, want one failure with no error.type", got)
	}
	got := r.measures(content.MetricDecodeErrors)
	if len(got) != 2 || got[0].value != 3 || got[0].attrs[content.AttrProblemKind] != "missing_field" || got[1].value != 1 || got[1].attrs[content.AttrProblemKind] != "unknown_field" {
		t.Errorf("decode_errors = %+v, want 3 missing_field then 1 unknown_field (one Add per kind)", got)
	}
	if got := r.measures(content.MetricEntriesLoaded); len(got) != 0 {
		t.Errorf("entries_loaded = %+v for a rejected document", got)
	}
	for _, m := range r.rec.Meter.Measurements() {
		for _, a := range m.Attrs {
			if strings.Contains(a.Key(), "path") || strings.Contains(a.Key(), telemetry.AttrErrorType) || strings.Contains(attrValueText(a), "secret") {
				t.Errorf("measurement %s carries %v", m.Name, a)
			}
		}
	}
	r.requireOneDuration(t)
	r.requireClean(t)
}

func attrValueText(a telemetry.Attr) string {
	s, _ := a.Value().(string)
	return s
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-100 - every problem kind counts under its snake_case
// name, and the values are exactly the ten of the schema.
// Path: happy.
func TestTelemetryProblemKindValues(t *testing.T) {
	t.Parallel()
	r := newRecorded(t)
	var ps []content.Problem
	for k := content.ProblemSyntax; k <= content.ProblemConstraintViolation; k++ {
		ps = append(ps, content.NewProblem("/p", k))
	}
	r.fake.Return("d", content.Set{}, problemErr(ps...))
	if _, err := r.load(t, "d"); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range r.measures(content.MetricDecodeErrors) {
		if m.value != 1 {
			t.Errorf("count %v, want 1 per kind", m.value)
		}
		got = append(got, m.attrs[content.AttrProblemKind].(string))
	}
	want := []string{"syntax", "unsupported_version", "missing_field", "unknown_field", "wrong_type", "out_of_range", "invalid_value", "duplicate", "unresolved_reference", "constraint_violation"}
	if !slices.Equal(got, want) {
		t.Errorf("problem_kind values = %v, want %v", got, want)
	}
	// A kind outside the closed set is not counted rather than invented.
	r2 := newRecorded(t)
	r2.fake.Return("d", content.Set{}, problemErr(content.NewProblem("/p", content.ProblemKind(99))))
	if _, err := r2.load(t, "d"); err != nil || len(r2.measures(content.MetricDecodeErrors)) != 0 {
		t.Errorf("an undefined problem kind was counted: %v", r2.measures(content.MetricDecodeErrors))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-101 - the kind attribute is the snake_case of the
// registered Kind names.
// Path: happy.
func TestTelemetryKindValues(t *testing.T) {
	t.Parallel()
	r := newRecorded(t)
	var es []content.Entry
	for k := content.KindSpecies; k.Valid(); k++ {
		es = append(es, plain(k, "a"))
	}
	r.fake.Return("d", content.MustSet(es...), nil)
	if _, err := r.load(t, "d"); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range r.measures(content.MetricEntriesLoaded) {
		got = append(got, m.attrs[content.AttrKind].(string))
	}
	want := []string{"species", "background", "character_class", "feat", "equipment", "spell", "magic_item", "monster"}
	if !slices.Equal(got, want) {
		t.Errorf("kind values = %v, want %v (equipment before spell)", got, want)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-086, FR-085 and doc/plans/plan-phase-2.md section 5.4 - a
// port failure records outcome and the error type only on both spans, counts
// loads once with error.type, leaves decode_errors and entries_loaded alone,
// measures the duration once, and keeps the panic value out of telemetry.
// Path: per case.
func TestTelemetryPortFailure(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		script func(*contenttest.FakeDecoder)
	}{
		{"a panic", func(f *contenttest.FakeDecoder) { f.Panic("d", "secret document bytes") }},
		{"an error that holds no problem", func(f *contenttest.FakeDecoder) {
			f.Return("d", content.Set{}, errors.New("secret document bytes"))
		}},
		{"a PortError of another port", func(f *contenttest.FakeDecoder) {
			f.Return("d", content.Set{}, srderr.WrapPort("other", "op", errors.New("secret document bytes")))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := newRecorded(t)
			c.script(r.fake)
			res, err := r.load(t, "d")
			if res != nil || srderr.ErrorType(err) != "content_error" {
				t.Fatalf("LoadSet = (%v, %v), want a content PortError", res, err)
			}
			load, decode := r.spans(t)
			for _, s := range []telemetrytest.SpanRecord{load, decode} {
				if outcomeOf(t, s) != telemetry.OutcomeFailure || s.Status != telemetry.StatusError || len(s.Errors) != 1 {
					t.Errorf("span %s = %+v, want failure, error status and one recorded error", s.Name, s)
					continue
				}
				if s.Errors[0].Error() != "content_error" || strings.Contains(s.Errors[0].Error(), "secret") || errors.Is(s.Errors[0], srderr.ErrPortPanic) {
					t.Errorf("span %s recorded %q, want only the error type", s.Name, s.Errors[0])
				}
			}
			loads := r.measures(content.MetricLoads)
			if len(loads) != 1 || loads[0].value != 1 || loads[0].attrs[telemetry.AttrOutcome] != "failure" || loads[0].attrs[telemetry.AttrErrorType] != "content_error" || len(loads[0].attrs) != 2 {
				t.Errorf("loads = %+v, want one failure with error.type content_error", loads)
			}
			if len(r.measures(content.MetricDecodeErrors)) != 0 || len(r.measures(content.MetricEntriesLoaded)) != 0 {
				t.Errorf("decode_errors or entries_loaded changed on a port failure")
			}
			for _, m := range r.rec.Meter.Measurements() {
				for _, a := range m.Attrs {
					if strings.Contains(attrValueText(a), "secret") {
						t.Errorf("measurement %s carries the document text: %v", m.Name, a)
					}
				}
			}
			r.requireOneDuration(t)
			r.requireClean(t)
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-086 - a cancelled ctx counts nothing and starts no span.
// Path: unhappy (failure).
func TestTelemetryCancelledContextEmitsNothing(t *testing.T) {
	t.Parallel()
	r := newRecorded(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.loader.LoadSet(ctx, content.LoadSetCommand{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if spans, ms := r.rec.Tracer.Spans(), r.rec.Meter.Measurements(); len(spans) != 0 || len(ms) != 0 {
		t.Errorf("a cancelled load emitted %d spans and %d measurements", len(spans), len(ms))
	}
	r.requireClean(t)
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-084 - the result is identical with the zero-value
// Telemetry and with a recording one, for every outcome.
// Path: per case.
func TestLoadSetResultIgnoresTelemetry(t *testing.T) {
	t.Parallel()
	script := func(f *contenttest.FakeDecoder) {
		f.Return("ok", content.MustSet(plain(content.KindSpell, "a")), nil)
		f.Return("bad", content.Set{}, problemErr(content.NewProblem("/x", content.ProblemUnknownField)))
		f.Panic("panic", "boom")
	}
	quiet := contenttest.NewFakeDecoder()
	script(quiet)
	loud := newRecorded(t)
	script(loud.fake)
	silent := mustLoader(t, quiet)
	for _, doc := range []string{"ok", "bad", "panic"} {
		a, errA := silent.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(doc)})
		b, errB := loud.load(t, doc)
		if (errA == nil) != (errB == nil) || srderr.ErrorType(errA) != srderr.ErrorType(errB) {
			t.Errorf("%s: errors differ: %v vs %v", doc, errA, errB)
		}
		switch x := a.(type) {
		case content.LoadSetLoaded:
			if y, ok := b.(content.LoadSetLoaded); !ok || !x.Equal(y) {
				t.Errorf("%s: results differ", doc)
			}
		case content.LoadSetRejected:
			if y, ok := b.(content.LoadSetRejected); !ok || !x.Equal(y) {
				t.Errorf("%s: results differ", doc)
			}
		case nil:
			if b != nil {
				t.Errorf("%s: results differ", doc)
			}
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-079, FR-080, FR-100, FR-101 and doc/rules/observability.md
// section 9 - signal names, keys and instrument shapes are a public contract.
// Path: happy.
func TestSignalNamesArePinned(t *testing.T) {
	t.Parallel()
	for got, want := range map[string]string{
		content.SpanLoad: "srd5e.content.load", content.SpanDecode: "srd5e.content.decode",
		content.MetricLoads: "srd5e.content.loads", content.MetricLoadDuration: "srd5e.content.load_duration",
		content.MetricEntriesLoaded: "srd5e.content.entries_loaded", content.MetricDecodeErrors: "srd5e.content.decode_errors",
		content.AttrProblemKind: "srd5e.content.problem_kind", content.AttrKind: "srd5e.content.kind",
	} {
		if got != want {
			t.Errorf("signal name %q, want %q", got, want)
		}
	}
	for _, name := range []string{content.SpanLoad, content.SpanDecode, content.MetricLoads, content.MetricLoadDuration, content.MetricEntriesLoaded, content.MetricDecodeErrors} {
		if !telemetry.ValidName(name) {
			t.Errorf("%q is not a valid signal name", name)
		}
	}
	r := newRecorded(t)
	got := map[string]telemetrytest.InstrumentInfo{}
	for _, i := range r.rec.Meter.Instruments() {
		got[i.Descriptor.Name] = i
	}
	for name, want := range map[string]struct {
		kind telemetrytest.Kind
		unit string
	}{
		content.MetricLoads:         {telemetrytest.KindCounter, "{load}"},
		content.MetricLoadDuration:  {telemetrytest.KindHistogram, "s"},
		content.MetricEntriesLoaded: {telemetrytest.KindCounter, "{entry}"},
		content.MetricDecodeErrors:  {telemetrytest.KindCounter, "{problem}"},
	} {
		if i, ok := got[name]; !ok || i.Kind != want.kind || i.Descriptor.Unit != want.unit || i.Descriptor.Description == "" {
			t.Errorf("instrument %s = %+v, want %v in %s with a description", name, i, want.kind, want.unit)
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-080 - WithClock sets the clock that times the load,
// and a nil clock is ignored, leaving the earlier one in place.
// Path: unhappy (invalid input).
func TestWithClockIgnoresNil(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), 250*time.Millisecond)
	fake := contenttest.NewFakeDecoder().Return("d", content.Set{}, nil)
	l := mustLoader(t, fake, content.WithTelemetry(rec.Telemetry()), content.WithClock(clock.Now), content.WithClock(nil))
	if _, err := l.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("d")}); err != nil {
		t.Fatal(err)
	}
	if d := rec.Meter.MeasurementsNamed(content.MetricLoadDuration); len(d) != 1 || d[0].Value != 0.25 {
		t.Errorf("load_duration = %+v, want one measurement of 0.25 s from the first clock", d)
	}
	// Without any option the default clock times the load: one measurement.
	rec2 := telemetrytest.NewRecorder()
	l2 := mustLoader(t, fake, content.WithTelemetry(rec2.Telemetry()))
	if _, err := l2.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("d")}); err != nil {
		t.Fatal(err)
	}
	if d := rec2.Meter.MeasurementsNamed(content.MetricLoadDuration); len(d) != 1 || d[0].Value < 0 {
		t.Errorf("default clock: load_duration = %+v, want one non-negative measurement", d)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-131 and doc/plans/plan-phase-2.md P16 - ValidateSet
// is pure: a rejected merge records no span, measurement or log even while a recording bundle is alive.
// Path: unhappy (invalid input).
func TestValidateSetRecordsNoSignal(t *testing.T) {
	t.Parallel()
	r := newRecorded(t)
	a, b := content.MustSet(plain(content.KindSpell, "a")), content.MustSet(plain(content.KindSpell, "a"))
	if _, err := content.ValidateSet(a, b); err == nil {
		t.Fatal("ValidateSet accepted a duplicate key")
	}
	if s, m, l := r.rec.Tracer.Spans(), r.rec.Meter.Measurements(), r.rec.Logger.Records(); len(s)+len(m)+len(l) != 0 {
		t.Errorf("ValidateSet recorded spans %v, measurements %v, logs %v; want none", s, m, l)
	}
}
