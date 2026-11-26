package combat

import (
	"context"
	"testing"
)

// Requirement: observability.md section 5 - every returned result records one srd5e.combat.events_per_operation sample
// holding the number of non-nil events, by operation, and nothing else as an attribute.
// Path: per case.
func TestRunRecordsEventsPerOperation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		events     []Event
		want       float64
	}{
		{"no events", "unhappy (empty)", nil, 0},
		{"two events", "happy", []Event{fakeEvent{"a"}, fakeEvent{"b"}}, 2},
		{"a nil event is not counted", "unhappy (nil element)", []Event{fakeEvent{"a"}, nil}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, in := newRun(t)
			if _, err := runOutcome(in, context.Background(), description{Class: resultApplied, Name: "x", Events: c.events}); err != nil {
				t.Fatal(err)
			}
			got := rec.Meter.MeasurementsNamed(metricEventsPerOperation)
			if len(got) != 1 || got[0].Value != c.want || len(got[0].Attrs) != 1 || got[0].Attrs[0].Key() != attrOperation {
				t.Errorf("events_per_operation = %+v, want one sample of %v with only the operation attribute [%s]", got, c.want, c.path)
			}
			assertNoViolations(t, rec)
		})
	}
}

// Requirement: observability.md section 5 - a returned ShapeShiftEnded or ZoneEnded event increments its end counter once
// with its cause or reason as the only attribute; other events and a call that returned an error record neither.
// Path: per case.
func TestRunCountsShapeShiftAndZoneEndings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path           string
		events               []Event
		shapeCauses, reasons []string
	}{
		{"every cause", "happy", []Event{
			ShapeShiftEnded{cause: CauseDurationElapsed}, ShapeShiftEnded{cause: CauseDispelled}, ZoneEnded{reason: ZoneEndBudgetSpent},
		}, []string{"duration_elapsed", "dispelled"}, []string{"budget_spent"}},
		{"other events", "unhappy (no ending)", []Event{fakeEvent{"a"}, ShapeShiftBegan{}}, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, in := newRun(t)
			if _, err := runOutcome(in, context.Background(), description{Class: resultApplied, Name: "x", Events: c.events}); err != nil {
				t.Fatal(err)
			}
			check := func(metric, key string, want []string) {
				var got []string
				for _, m := range rec.Meter.MeasurementsNamed(metric) {
					if len(m.Attrs) != 1 || m.Attrs[0].Key() != key || m.Value != 1 {
						t.Errorf("%s sample = %+v, want 1 with only %s", metric, m, key)
						continue
					}
					got = append(got, m.Attrs[0].Value().(string))
				}
				if len(got) != len(want) {
					t.Fatalf("%s causes = %v, want %v [%s]", metric, got, want, c.path)
				}
				for i := range want {
					if got[i] != want[i] {
						t.Errorf("%s causes = %v, want %v", metric, got, want)
					}
				}
			}
			check(metricShapeShiftEnded, attrShapeShiftEndCause, c.shapeCauses)
			check(metricZoneEnded, attrZoneEndReason, c.reasons)
			assertNoViolations(t, rec)
		})
	}
}

// Requirement: observability.md section 6 - a call that returns an error records no end counter and no events sample.
// Path: unhappy (failure).
func TestRunRecordsNoEndingsOnError(t *testing.T) {
	t.Parallel()
	rec, in := newRun(t)
	_, _ = run(context.Background(), in, testOp(),
		func(context.Context) (outcome, error) { return outcome{}, invalidInput("x") },
		func(outcome) description { return description{Events: []Event{ZoneEnded{reason: ZoneEndDispelled}}} })
	for _, n := range []string{metricZoneEnded, metricEventsPerOperation} {
		if got := rec.Meter.MeasurementsNamed(n); len(got) != 0 {
			t.Errorf("%s = %+v, want none after an error", n, got)
		}
	}
}
