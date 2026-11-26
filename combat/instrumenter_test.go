package combat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

type fakeEvent struct{ kind string }

func (fakeEvent) event()                    {}
func (e fakeEvent) Kind() string            { return e.kind }
func (fakeEvent) Combatants() []CombatantID { return nil }

// fakeEnvelope wraps another event for transport and reports its kind.
type fakeEnvelope struct{ Event }

func (fakeEnvelope) transportEnvelope() {}

type outcome struct{ class combatResultClass }

var testOp = func() combatOperation {
	op, err := newCombatOperation("apply_damage", "srd5e.combat.apply_damage")
	if err != nil {
		panic(err)
	}
	return op
}

func attrs(as []telemetry.Attr) map[string]any {
	m := map[string]any{}
	for _, a := range as {
		m[a.Key()] = a.Value()
	}
	return m
}

func newRun(t *testing.T) (*telemetrytest.Recorder, instrumenter) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 2*time.Second)
	return rec, newInstrumenter(rec.Telemetry(), clock.Now)
}

func assertNoViolations(t *testing.T, rec *telemetrytest.Recorder) {
	t.Helper()
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

func runOutcome(in instrumenter, ctx context.Context, d description) (outcome, error) {
	return run(ctx, in, testOp(),
		func(context.Context) (outcome, error) { return outcome{d.Class}, nil },
		func(outcome) description { return d })
}

// Requirement: doc/specs/combat-foundations.md FR-029, FR-030, FR-031 - one
// span, one operations increment, one duration sample and one events increment
// per event; refusals set no error status, invalid sets it.
// Path: per case.
func TestRunRecordsEachResultClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path, outcome, class string
		d                          description
		spanError                  bool
	}{
		{"applied", "happy", "success", "applied", description{Class: resultApplied, Name: "damaged"}, false},
		{"refused", "unhappy (rule interaction)", "failure", "refused", description{Class: resultRefused, Name: "rejected_dead"}, false},
		{"invalid", "unhappy (rule interaction)", "failure", "invalid", description{Class: resultInvalid, Name: "unknown_combatant"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, in := newRun(t)
			got, err := runOutcome(in, context.Background(), c.d)
			if err != nil || got.class != c.d.Class {
				t.Fatalf("run = %v, %v", got, err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.apply_damage")
			if len(spans) != 1 || !spans[0].Ended {
				t.Fatalf("spans = %+v, want one ended span", spans)
			}
			want := map[string]any{
				"srd5e.combat.operation": "apply_damage", "srd5e.outcome": c.outcome,
				"srd5e.combat.result": c.d.Name, "srd5e.combat.result_class": c.class,
			}
			sa := attrs(spans[0].Attrs)
			ops := rec.Meter.MeasurementsNamed("srd5e.combat.operations")
			if len(ops) != 1 || ops[0].Value != 1 {
				t.Fatalf("operations = %+v, want one increment", ops)
			}
			if oa := attrs(ops[0].Attrs); fmt.Sprint(oa) != fmt.Sprint(want) {
				t.Errorf("operations attrs = %v, want %v", oa, want)
			}
			delete(want, "srd5e.combat.operation") // set when the span starts, same value
			for k, v := range want {
				if sa[k] != v {
					t.Errorf("span attr %s = %v, want %v", k, sa[k], v)
				}
			}
			if (spans[0].Status == telemetry.StatusError) != c.spanError {
				t.Errorf("span status = %v, want error=%v", spans[0].Status, c.spanError)
			}
			dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration")
			if len(dur) != 1 || dur[0].Value != 2 || fmt.Sprint(attrs(dur[0].Attrs)) != "map[srd5e.combat.operation:apply_damage]" {
				t.Errorf("duration = %+v, want one sample of 2 s with the operation only", dur)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.events")); n != 0 {
				t.Errorf("errors/events measurements = %d, want 0", n)
			}
			assertNoViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-029, FR-038 - one events
// increment per event under its kind; an envelope counts as the wrapped event.
// Path: happy.
func TestRunCountsEvents(t *testing.T) {
	t.Parallel()
	rec, in := newRun(t)
	d := description{Class: resultApplied, Name: "damaged", Events: []Event{
		fakeEvent{"damage_taken"}, fakeEnvelope{fakeEvent{"tracker_round_started"}}, nil, fakeEvent{"damage_taken"}}}
	if _, err := runOutcome(in, context.Background(), d); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.events") {
		if m.Value != 1 || len(m.Attrs) != 1 {
			t.Errorf("events measurement = %+v, want +1 with one attribute", m)
		}
		got = append(got, attrs(m.Attrs)["srd5e.combat.event"].(string))
	}
	if want := []string{"damage_taken", "tracker_round_started", "damage_taken"}; !slices.Equal(got, want) {
		t.Errorf("event kinds counted = %v, want %v", got, want)
	}
	var _ transportEnvelope = fakeEnvelope{}
	assertNoViolations(t, rec)
}

// Requirement: doc/specs/combat-foundations.md FR-041 - extras go on the span
// and the operations increment only; a shared key or a key combat does not own
// is ErrInvalidInput recorded on the span, with no operations increment.
// Path: per case.
func TestRunExtras(t *testing.T) {
	t.Parallel()
	good := []telemetry.Attr{telemetry.String(attrRosterReason, "fled"), telemetry.String(attrAttackCover, "half"), telemetry.String(d20.AttrRollMode, "advantage"), telemetry.String(d20.AttrAbility, "strength")}
	t.Run("closed extras", func(t *testing.T) {
		t.Parallel()
		rec, in := newRun(t)
		if _, err := runOutcome(in, context.Background(), description{Class: resultApplied, Name: "n", Extras: good, Events: []Event{fakeEvent{"k"}}}); err != nil {
			t.Fatal(err)
		}
		ops := attrs(rec.Meter.MeasurementsNamed("srd5e.combat.operations")[0].Attrs)
		sp := attrs(rec.Tracer.Spans()[0].Attrs)
		for _, k := range []string{attrRosterReason, attrAttackCover, d20.AttrRollMode, d20.AttrAbility} {
			if ops[k] == nil || sp[k] == nil {
				t.Errorf("extra %s missing: operations %v span %v", k, ops, sp)
			}
		}
		for _, name := range []string{"srd5e.combat.operation_duration", "srd5e.combat.events"} {
			for _, m := range rec.Meter.MeasurementsNamed(name) {
				if _, ok := attrs(m.Attrs)[attrRosterReason]; ok {
					t.Errorf("extra leaked onto %s", name)
				}
			}
		}
	})
	for _, key := range []string{attrOperation, attrResult, attrResultClass, attrEvent, telemetry.AttrOutcome, telemetry.AttrErrorType, "other.key", "srd5e.character.result"} {
		t.Run("rejects "+key, func(t *testing.T) {
			t.Parallel()
			rec, in := newRun(t)
			got, err := runOutcome(in, context.Background(), description{Class: resultApplied, Name: "n", Extras: []telemetry.Attr{telemetry.String(key, "v")}})
			if !errors.Is(err, srderr.ErrInvalidInput) || got != (outcome{}) {
				t.Fatalf("run = %v, %v, want zero and ErrInvalidInput", got, err)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")); n != 0 {
				t.Errorf("operations increments = %d, want 0", n)
			}
			sp := rec.Tracer.Spans()[0]
			if sp.Status != telemetry.StatusError || len(sp.Errors) != 1 || !sp.Ended {
				t.Errorf("span = %+v, want one recorded error, error status, ended", sp)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-032, FR-035 - a returned
// *srderr.PortError increments errors once with error.type, records the error
// on the span once, counts no operation or event, and is returned unchanged;
// an error matching srderr.ErrInvalidInput does the same with error.type
// invalid_input; any other error is on the span only.
// Path: unhappy (failure).
func TestRunErrors(t *testing.T) {
	t.Parallel()
	portErr := srderr.WrapPort(dice.Port, "roll", errors.New("down"))
	cases := []struct {
		name string
		err  error
		typ  string
	}{
		{"port failure", portErr, srderr.ErrorType(portErr)},
		{"invalid input", fmt.Errorf("bad command: %w", srderr.ErrInvalidInput), "invalid_input"},
		{"other error", errors.New("other"), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, in := newRun(t)
			_, err := run(context.Background(), in, testOp(),
				func(context.Context) (outcome, error) { return outcome{}, c.err },
				func(outcome) description { t.Error("describe called on error"); return description{} })
			if !errors.Is(err, c.err) {
				t.Fatalf("error = %v, want the same value %v", err, c.err)
			}
			sp := rec.Tracer.Spans()[0]
			if sp.Status != telemetry.StatusError || len(sp.Errors) != 1 || !errors.Is(sp.Errors[0], c.err) || !sp.Ended {
				t.Errorf("span = %+v, want the error recorded once, error status, ended", sp)
			}
			errs := rec.Meter.MeasurementsNamed("srd5e.combat.errors")
			if c.typ == "" {
				if len(errs) != 0 {
					t.Errorf("errors = %+v, want none for a non-port error", errs)
				}
			} else if len(errs) != 1 || fmt.Sprint(attrs(errs[0].Attrs)) != fmt.Sprintf("map[error.type:%s srd5e.combat.operation:apply_damage]", c.typ) {
				t.Errorf("errors = %+v, want one with error.type %s", errs, c.typ)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.events")); n != 0 {
				t.Errorf("operations/events = %d, want 0", n)
			}
			if d := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(d) != 1 {
				t.Errorf("duration samples = %d, want 1", len(d))
			}
			if len(rec.Logger.Records()) != 0 {
				t.Errorf("logged %v, want nothing", rec.Logger.Records())
			}
			assertNoViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-035 - a panic from domain
// code propagates unchanged with the span ended and no errors increment.
// Path: unhappy (failure).
func TestRunPanicPropagates(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(in instrumenter){
		"work": func(in instrumenter) {
			_, _ = run(context.Background(), in, testOp(),
				func(context.Context) (outcome, error) { panic(errors.New("boom")) },
				func(outcome) description { return description{} })
		},
		"describe": func(in instrumenter) {
			_, _ = run(context.Background(), in, testOp(),
				func(context.Context) (outcome, error) { return outcome{}, nil },
				func(outcome) description { panic(errors.New("boom")) })
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, in := newRun(t)
			func() {
				defer func() {
					if r := recover(); r == nil || r.(error).Error() != "boom" {
						t.Errorf("recovered %v, want the original panic", r)
					}
				}()
				f(in)
			}()
			if sp := rec.Tracer.Spans(); len(sp) != 1 || !sp[0].Ended {
				t.Errorf("spans = %+v, want one ended span", sp)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors increments = %d, want 0", n)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-034, FR-036 - no log at Info
// or above, and the same result with Noop, the zero instrumenter and a
// recording bundle; the work receives the span's context.
// Path: happy.
func TestRunSameResultAndSilent(t *testing.T) {
	t.Parallel()
	d := description{Class: resultRefused, Name: "rejected_dead", Events: []Event{fakeEvent{"k"}}}
	rec, in := newRun(t)
	withRec, err1 := runOutcome(in, context.Background(), d)
	withNoop, err2 := runOutcome(newInstrumenter(telemetry.Telemetry{}, nil), context.Background(), d)
	zero, err3 := runOutcome(instrumenter{}, context.Background(), d)
	if err1 != nil || err2 != nil || err3 != nil || withRec != withNoop || withRec != zero {
		t.Errorf("results differ: %v %v %v (%v %v %v)", withRec, withNoop, zero, err1, err2, err3)
	}
	if got := rec.Logger.Records(); len(got) != 0 {
		t.Errorf("logged %v, want nothing", got)
	}
	var seen context.Context
	type key struct{}
	_, _ = run(context.WithValue(context.Background(), key{}, 1), in, testOp(),
		func(ctx context.Context) (outcome, error) { seen = ctx; return outcome{resultApplied}, nil },
		func(outcome) description { return description{Class: resultApplied} })
	if seen.Value(key{}) != 1 {
		t.Error("the work's context lost the caller's values")
	}
}

// Requirement: doc/specs/combat-foundations.md FR-033 - no attribute value is a
// supplied CombatantID.
// Path: unhappy (telemetry).
func TestRunAttributesCarryNoCombatantID(t *testing.T) {
	t.Parallel()
	rec, in := newRun(t)
	id := MustCombatantID("goblin_1")
	_, _ = runOutcome(in, context.Background(), description{Class: resultApplied, Name: "damaged", Events: []Event{fakeEvent{"damage_taken"}}})
	for _, s := range rec.Tracer.Spans() {
		for _, a := range s.Attrs {
			if a.Value() == string(id) {
				t.Errorf("span attribute %s carries a combatant id", a.Key())
			}
		}
	}
	for _, m := range rec.Meter.Measurements() {
		for _, a := range m.Attrs {
			if a.Value() == string(id) {
				t.Errorf("%s attribute %s carries a combatant id", m.Name, a.Key())
			}
		}
	}
}

// Requirement: doc/specs/combat-foundations.md FR-035 - a nil clock defaults to
// time.Now, so the duration is still recorded once (never negative).
// Path: unhappy (boundary).
func TestNewInstrumenterNilClockStillTimes(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	for name, in := range map[string]instrumenter{"constructed": newInstrumenter(rec.Telemetry(), nil), "literal": {tel: rec.Telemetry()}} {
		rec.Clear()
		if _, err := runOutcome(in, context.Background(), description{Class: resultApplied, Name: "n"}); err != nil {
			t.Fatal(err)
		}
		d := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration")
		if len(d) != 1 || d[0].Value < 0 {
			t.Errorf("%s: duration samples = %+v, want one non-negative sample", name, d)
		}
	}
}
