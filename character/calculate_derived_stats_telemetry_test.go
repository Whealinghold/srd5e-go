package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func attrMap(attrs []telemetry.Attr) map[string]any {
	m := make(map[string]any, len(attrs))
	for _, a := range attrs {
		m[a.Key()] = a.Value()
	}
	return m
}

func sameAttrs(t *testing.T, what string, got []telemetry.Attr, want map[string]any) {
	t.Helper()
	g := attrMap(got)
	if len(g) != len(want) {
		t.Errorf("%s attrs = %v, want %v", what, g, want)
		return
	}
	for k, v := range want {
		if g[k] != v {
			t.Errorf("%s attrs = %v, want %v", what, g, want)
			return
		}
	}
}

// recorded builds the use case over a recording bundle and a stepping clock.
func recorded(t *testing.T, source character.ContentSource) (character.CalculateDerivedStats, *telemetrytest.Recorder) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	uc, err := character.NewCalculateDerivedStats(character.WithSource(source), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now))
	if err != nil {
		t.Fatal(err)
	}
	return uc, rec
}

func outcome(v string) map[string]any { return map[string]any{telemetry.AttrOutcome: v} }

// onlyOne returns the measurement of name, failing unless there is exactly one.
func onlyOne(t *testing.T, rec *telemetrytest.Recorder, name string) telemetrytest.Measurement {
	t.Helper()
	ms := rec.Meter.MeasurementsNamed(name)
	if len(ms) != 1 {
		t.Fatalf("%d measurements of %s, want 1: %+v", len(ms), name, ms)
	}
	return ms[0]
}

func noLogsOrViolations(t *testing.T, rec *telemetrytest.Recorder) {
	t.Helper()
	if logs := rec.Logger.Records(); len(logs) != 0 {
		t.Errorf("logged %+v, want nothing", logs)
	}
	telemetrytest.RequireNoViolations(t, rec)
}

// Requirement: doc/specs/derived-stats.md FR-047 - a successful call emits the
// calculate span (outcome success, multiclass), a load_content child span
// (success), the calls counter, the duration and the levels histogram, and no
// log.
// Path: happy.
func TestSuccessSignals(t *testing.T) {
	t.Parallel()
	uc, rec := recorded(t, contenttest.NewFakeSource(wizardClass(t)))
	cmd := wizardCommand(t)
	cmd.ClassLevels = levels(cl("wizard", 3), cl("fighter", 2))
	succeeded(t, run(t, uc, cmd))

	spans := rec.Tracer.Spans()
	if len(spans) != 2 {
		t.Fatalf("spans = %+v", spans)
	}
	calc, load := spans[0], spans[1]
	if calc.Name != character.SpanDerivedStatsCalculate || !calc.Ended || calc.Status != telemetry.StatusUnset || len(calc.Errors) != 0 || calc.ParentID != 0 {
		t.Errorf("calculate span = %+v", calc)
	}
	sameAttrs(t, "calculate span", calc.Attrs, map[string]any{character.AttrMulticlass: true, telemetry.AttrOutcome: "success"})
	if load.Name != character.SpanDerivedStatsLoadContent || !load.Ended || load.ParentID != calc.ID || load.Status != telemetry.StatusUnset {
		t.Errorf("load_content span = %+v", load)
	}
	sameAttrs(t, "load_content span", load.Attrs, outcome("success"))

	calls := onlyOne(t, rec, character.MetricDerivedStatsCalls)
	if calls.Value != 1 || calls.Unit != "{call}" || calls.Kind != telemetrytest.KindCounter {
		t.Errorf("calls = %+v", calls)
	}
	sameAttrs(t, "calls", calls.Attrs, map[string]any{character.AttrMulticlass: true, telemetry.AttrOutcome: "success"})
	d := onlyOne(t, rec, character.MetricDerivedStatsDuration)
	if d.Value != 0.25 || d.Unit != "s" || len(d.Attrs) != 0 {
		t.Errorf("duration = %+v, want 0.25 s and no attributes", d)
	}
	l := onlyOne(t, rec, character.MetricDerivedStatsLevels)
	if l.Value != 5 || l.Unit != "{level}" || len(l.Attrs) != 0 {
		t.Errorf("levels = %+v, want 5 {level} and no attributes", l)
	}
	if n := len(rec.Meter.MeasurementsNamed(character.MetricDerivedStatsErrors)); n != 0 {
		t.Errorf("%d errors counted for a success", n)
	}
	noLogsOrViolations(t, rec)
}

// Requirement: doc/specs/derived-stats.md FR-048 - each returned failure result
// is recorded once: calculate error status and outcome failure, one calls
// increment with the failure attribute and no error.type, load_content failure
// for content_missing and success for the others, duration recorded, no levels,
// no log.
// Path: per case.
func TestFailureResultSignals(t *testing.T) {
	t.Parallel()
	plate, shield := plateArmor(t), shieldOf(t)
	cases := []struct {
		name        string
		entries     []content.Entry
		mutate      func(*character.CalculateDerivedStatsCommand)
		failure     string
		loadOutcome string
		loadStatus  telemetry.Status
	}{
		{"content missing", nil, func(*character.CalculateDerivedStatsCommand) {}, character.ValueDerivedStatsContentMissing, "failure", telemetry.StatusError},
		{"slot mismatch", []content.Entry{wizardClass(t), shield}, func(c *character.CalculateDerivedStatsCommand) { c.ArmorID = shield.ID() }, character.ValueDerivedStatsEquipmentSlotMismatch, "success", telemetry.StatusUnset},
		{"basis not applicable", []content.Entry{wizardClass(t), plate}, func(c *character.CalculateDerivedStatsCommand) {
			c.ArmorID = plate.ID()
			c.ChosenBasis = character.ChooseStandard{}
		}, character.ValueDerivedStatsBasisNotApplicable, "success", telemetry.StatusUnset},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, rec := recorded(t, contenttest.NewFakeSource(c.entries...))
			cmd := wizardCommand(t)
			c.mutate(&cmd)
			if _, ok := run(t, uc, cmd).(character.CalculateDerivedStatsSucceeded); ok {
				t.Fatal("the call succeeded")
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 0 || !spans[0].Ended {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "calculate span", spans[0].Attrs, map[string]any{character.AttrMulticlass: false, telemetry.AttrOutcome: "failure"})
			if spans[1].Status != c.loadStatus || len(spans[1].Errors) != 0 {
				t.Errorf("load_content = %+v, want status %v and no error", spans[1], c.loadStatus)
			}
			sameAttrs(t, "load_content span", spans[1].Attrs, outcome(c.loadOutcome))
			calls := onlyOne(t, rec, character.MetricDerivedStatsCalls)
			sameAttrs(t, "calls", calls.Attrs, map[string]any{character.AttrMulticlass: false, telemetry.AttrOutcome: "failure", character.AttrDerivedStatsFailure: c.failure})
			onlyOne(t, rec, character.MetricDerivedStatsDuration)
			if n := len(rec.Meter.MeasurementsNamed(character.MetricDerivedStatsErrors)) + len(rec.Meter.MeasurementsNamed(character.MetricDerivedStatsLevels)); n != 0 {
				t.Errorf("%d errors or levels measurements for a failure result", n)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-048, FR-074 - a port failure puts
// the error and error status on both spans, increments calls once with no failure
// attribute and errors once with error.type content_error, records the duration,
// and logs nothing.
// Path: per case.
func TestPortFailureSignals(t *testing.T) {
	t.Parallel()
	cause := errors.New("backend down")
	cases := []struct {
		name   string
		source character.ContentSource
		is     error
	}{
		{"source error", contenttest.NewFakeSource(wizardClass(t)).FailLookup(cause), cause},
		{"source panic", panicSource{}, srderr.ErrPortPanic},
		{"contract breach", nilSource{}, srderr.ErrContractViolation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, rec := recorded(t, c.source)
			_, err := uc.Run(context.Background(), wizardCommand(t))
			if !errors.Is(err, c.is) {
				t.Fatalf("error = %v, want %v", err, c.is)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 {
				t.Fatalf("spans = %+v", spans)
			}
			for _, s := range spans {
				if s.Status != telemetry.StatusError || len(s.Errors) != 1 || !errors.Is(s.Errors[0], c.is) || !s.Ended {
					t.Errorf("span %s = %+v, want error status and the error once", s.Name, s)
				}
				if v, _ := s.AttrValue(telemetry.AttrOutcome); v != "failure" {
					t.Errorf("span %s outcome = %v", s.Name, v)
				}
			}
			calls := onlyOne(t, rec, character.MetricDerivedStatsCalls)
			sameAttrs(t, "calls", calls.Attrs, map[string]any{character.AttrMulticlass: false, telemetry.AttrOutcome: "failure"})
			errs := onlyOne(t, rec, character.MetricDerivedStatsErrors)
			sameAttrs(t, "errors", errs.Attrs, map[string]any{telemetry.AttrErrorType: "content_error"})
			if errs.Value != 1 || errs.Unit != "{error}" {
				t.Errorf("errors = %+v", errs)
			}
			onlyOne(t, rec, character.MetricDerivedStatsDuration)
			if n := len(rec.Meter.MeasurementsNamed(character.MetricDerivedStatsLevels)); n != 0 {
				t.Errorf("levels recorded for a failure")
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-046 - the no-op and a recording
// telemetry bundle give equal results; the zero bundle is accepted.
// Path: happy.
func TestNoopAndRecordingGiveEqualResults(t *testing.T) {
	t.Parallel()
	src := contenttest.NewFakeSource(wizardClass(t))
	noop, err := character.NewCalculateDerivedStats(character.WithSource(src), character.WithTelemetry(telemetry.Telemetry{}))
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := recorded(t, src)
	a, b := succeeded(t, run(t, noop, wizardCommand(t))), succeeded(t, run(t, rec, wizardCommand(t)))
	if !a.ArmorClassBreakdown().Equal(b.ArmorClassBreakdown()) || a.HitPointMaximum() != b.HitPointMaximum() ||
		a.PassivePerception().Value() != b.PassivePerception().Value() || !a.Skill(content.SkillArcana).Breakdown().Equal(b.Skill(content.SkillArcana).Breakdown()) {
		t.Error("results differ between no-op and recording telemetry")
	}
}

// Requirement: doc/specs/derived-stats.md FR-048 - a cancelled ctx is returned
// before any signal; a nil clock option is ignored.
// Path: unhappy (cancellation).
func TestCancelledContextEmitsNothing(t *testing.T) {
	t.Parallel()
	src := contenttest.NewFakeSource(wizardClass(t))
	rec := telemetrytest.NewRecorder()
	uc, err := character.NewCalculateDerivedStats(character.WithSource(src), character.WithTelemetry(rec.Telemetry()), character.WithClock(nil))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, wizardCommand(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("signals emitted for a cancelled call")
	}
}

// Requirement: doc/rules/observability.md section 9 - metric
// names and attribute keys are part of the public contract, so each constant is
// pinned to its catalogue value.
// Path: happy.
func TestSignalNamesArePinned(t *testing.T) {
	t.Parallel()
	for got, want := range map[string]string{
		character.SpanDerivedStatsCalculate:              "srd5e.character.derived_stats.calculate",
		character.SpanDerivedStatsLoadContent:            "srd5e.character.derived_stats.load_content",
		character.MetricDerivedStatsCalls:                "srd5e.character.derived_stats.calls",
		character.MetricDerivedStatsErrors:               "srd5e.character.derived_stats.errors",
		character.MetricDerivedStatsDuration:             "srd5e.character.derived_stats.duration",
		character.MetricDerivedStatsLevels:               "srd5e.character.derived_stats.levels",
		character.AttrMulticlass:                         "srd5e.character.multiclass",
		character.AttrDerivedStatsFailure:                "srd5e.character.derived_stats.failure",
		character.ValueDerivedStatsContentMissing:        "content_missing",
		character.ValueDerivedStatsBasisNotApplicable:    "armor_class_basis_not_applicable",
		character.ValueDerivedStatsEquipmentSlotMismatch: "equipment_slot_mismatch",
	} {
		if got != want {
			t.Errorf("constant = %q, want %q", got, want)
		}
	}
}

// Requirement: doc/rules/observability.md section 4 (Timing) - without WithClock
// the duration is still recorded, from the default clock.
// Path: happy.
func TestDefaultClockRecordsDuration(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	uc, err := character.NewCalculateDerivedStats(character.WithSource(contenttest.NewFakeSource(wizardClass(t))), character.WithTelemetry(rec.Telemetry()))
	if err != nil {
		t.Fatal(err)
	}
	succeeded(t, run(t, uc, wizardCommand(t)))
	if d := onlyOne(t, rec, character.MetricDerivedStatsDuration); d.Value < 0 {
		t.Errorf("duration = %v", d.Value)
	}
}
