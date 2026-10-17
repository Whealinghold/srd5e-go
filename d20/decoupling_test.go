package d20_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// These fakes implement the ports from nothing but their interfaces. They prove
// the context builds and runs without any srd5e implementation of a port: no
// dicetest, no telemetrytest, no SeededRoller.

// fakeRoller is a d20.Roller that always rolls a fixed face.
type fakeRoller struct {
	face     int
	requests []dice.Expression
}

func (f *fakeRoller) Roll(e dice.Expression) (dice.Result, error) {
	f.requests = append(f.requests, e)
	rolls := make([]dice.DieRoll, e.Count())
	for i := range rolls {
		rolls[i] = dice.MustDieRoll(e.Die(), f.face)
	}
	return dice.NewResult(e, rolls)
}

var _ d20.Roller = (*fakeRoller)(nil)

type fakeSpan struct {
	telemetry.NoopSpan
	ended bool
	attrs int
}

func (s *fakeSpan) SetAttributes(a ...telemetry.Attr) { s.attrs += len(a) }
func (s *fakeSpan) End()                              { s.ended = true }

type fakeTracer struct{ spans []*fakeSpan }

func (t *fakeTracer) Start(ctx context.Context, _ string, a ...telemetry.Attr) (context.Context, telemetry.Span) {
	s := &fakeSpan{attrs: len(a)}
	t.spans = append(t.spans, s)
	return ctx, s
}

type fakeInstrument struct {
	d telemetry.Descriptor
	n int
}

func (i *fakeInstrument) Descriptor() telemetry.Descriptor              { return i.d }
func (i *fakeInstrument) Add(context.Context, int64, ...telemetry.Attr) { i.n++ }
func (i *fakeInstrument) Record(context.Context, float64, ...telemetry.Attr) {
	i.n++
}

type fakeMeter struct{ made map[string]*fakeInstrument }

func (m *fakeMeter) instrument(d telemetry.Descriptor) *fakeInstrument {
	if m.made == nil {
		m.made = map[string]*fakeInstrument{}
	}
	i := &fakeInstrument{d: d}
	m.made[d.Name] = i
	return i
}
func (m *fakeMeter) Counter(d telemetry.Descriptor) telemetry.Counter     { return m.instrument(d) }
func (m *fakeMeter) Histogram(d telemetry.Descriptor) telemetry.Histogram { return m.instrument(d) }
func (m *fakeMeter) Gauge(d telemetry.Descriptor) telemetry.Gauge         { return m.instrument(d) }

func TestResolverRunsOnFakesOfEveryDependency(t *testing.T) {
	t.Parallel()
	roller := &fakeRoller{face: 14}
	tracer, meter := &fakeTracer{}, &fakeMeter{}
	ticks := 0
	clock := func() time.Time { ticks++; return time.Unix(int64(ticks), 0) }

	r, err := d20.NewResolver(roller,
		d20.WithTelemetry(telemetry.Telemetry{Tracer: tracer, Meter: meter}), // Logger defaults to a no-op
		d20.WithClock(clock))
	if err != nil {
		t.Fatal(err)
	}
	res, err := r.Resolve(context.Background(), d20.AttackRollCommand{Ability: ability.Dexterity, Target: 15, Attempt: rollAttempt(2)})
	if err != nil {
		t.Fatal(err)
	}

	atk, ok := res.(d20.RolledAttack)
	if !ok || atk.Attack().Total() != 16 || !atk.Attack().Hit() {
		t.Fatalf("result = %+v", res)
	}
	if len(roller.requests) != 1 || roller.requests[0] != d20Expr(1) {
		t.Errorf("requests = %v", roller.requests)
	}
	if len(tracer.spans) != 1 || !tracer.spans[0].ended || tracer.spans[0].attrs <= 2 {
		t.Errorf("span = %+v", tracer.spans)
	}
	for _, name := range []string{d20.MetricResolved, d20.MetricDuration, d20.MetricModifiers, d20.MetricNatural} {
		if meter.made[name] == nil || meter.made[name].n != 1 {
			t.Errorf("instrument %s = %+v", name, meter.made[name])
		}
	}
	if meter.made[d20.MetricErrors].n != 0 {
		t.Error("errors counter incremented on success")
	}
	if ticks != 2 {
		t.Errorf("clock read %d times, want start and stop", ticks)
	}
}

func TestResolverNeedsOnlyARollerFunc(t *testing.T) {
	t.Parallel()
	r, err := d20.NewResolver(brokenRoller{func(e dice.Expression) (dice.Result, error) {
		return (&fakeRoller{face: 1}).Roll(e)
	}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := r.Resolve(context.Background(), d20.SavingThrowCommand{Ability: ability.Wisdom, Target: 2, Attempt: rollAttempt(0)})
	if err != nil || res.Outcome() != d20.Failure {
		t.Errorf("res %v err %v", res, err)
	}
}
