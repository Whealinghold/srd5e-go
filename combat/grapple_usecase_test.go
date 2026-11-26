package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// grappleRow is one variant of one grapple use case.
type grappleRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
	ended                   []string
	inForce                 float64 // -1: no sample
}

func grappleRows(t *testing.T) []grappleRow {
	t.Helper()
	ctx := context.Background()
	ab, cd := held("a", "b", hand(0)), held("c", "d", hand(0))
	one, two := grapplesOf(t, ab), grapplesOf(t, ab, cd)
	full := fullGrapples(t, 256)
	begin := func(s combat.Grapples, grappler, target string, rng int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewBeginGrapple(opts...).Run(ctx, combat.BeginGrappleCommand{Grapples: s, Grappler: combat.CombatantID(grappler),
				Target: combat.CombatantID(target), Part: hand(0), EscapeDC: 13, RangeFeet: rng})
		}
	}
	release := func(s combat.Grapples, grappler, target string) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewReleaseGrapple(opts...).Run(ctx, combat.ReleaseGrappleCommand{Grapples: s, Grappler: combat.CombatantID(grappler), Target: combat.CombatantID(target)})
		}
	}
	escape := func(s combat.Grapples, target string, actions combat.ConditionActionAvailability, attempt d20.CheckAttempt, faces ...int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewEscapeGrappleFromRecordCommand(s, combat.CombatantID(target), combat.GrappleEscapeOptionStrengthAthletics, actions, attempt)
			if err != nil {
				return nil, err
			}
			uc, err := combat.NewEscapeGrappleFromRecord(dicetest.NewFixedRoller(faces...), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, cmd)
		}
	}
	review := func(s combat.Grapples, obs ...combat.GrappleObservation) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewReviewGrapples(opts...).Run(ctx, combat.ReviewGrapplesCommand{Grapples: s, Observations: obs})
		}
	}
	drop := func(s combat.Grapples, target string) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewDropGrappleRecord(opts...).Run(ctx, combat.DropGrappleRecordCommand{Grapples: s, Target: combat.CombatantID(target)})
		}
	}
	obs := func(g, tg string, dist int, cs ...combat.Condition) combat.GrappleObservation {
		return combat.MustGrappleObservation(gid(g), gid(tg), condSet(t, cs), dist)
	}
	avail, none := combat.ConditionActionAvailabilityAvailable, combat.ConditionActionAvailabilityNone
	const table = "grappling.md section 7 closed value sets"
	const a, r, i = "applied", "refused", "invalid"
	return []grappleRow{
		{"begin_grapple applied", table + "; FR-066, FR-070", "happy", "begin_grapple", begin(one, "e", "f", 5), a, "applied", []string{"grapple_started"}, []string{}, 2},
		{"begin_grapple part_in_use", table + "; FR-024", "unhappy (state error)", "begin_grapple", begin(one, "a", "z", 5), r, "part_in_use", []string{}, []string{}, -1},
		{"begin_grapple target_already_grappled", table + "; FR-025", "unhappy (state error)", "begin_grapple", begin(one, "c", "b", 5), r, "target_already_grappled", []string{}, []string{}, -1},
		{"begin_grapple same_grappler_and_target", table + "; FR-007", "unhappy (invalid input)", "begin_grapple", begin(one, "e", "e", 5), i, "same_grappler_and_target", []string{}, []string{}, -1},
		{"begin_grapple negative_range", table + "; FR-008", "unhappy (invalid input)", "begin_grapple", begin(one, "e", "f", -1), i, "negative_range", []string{}, []string{}, -1},
		{"begin_grapple missing_value", table + "; package design", "unhappy (invalid input)", "begin_grapple", begin(one, "", "f", 5), i, "missing_value", []string{}, []string{}, -1},
		{"begin_grapple limit_exceeded", table + "; plan P10", "unhappy (boundary)", "begin_grapple", begin(full, "e", "f", 5), i, "limit_exceeded", []string{}, []string{}, -1},
		{"release_grapple applied", table + "; FR-067, FR-072", "happy", "release_grapple", release(two, "a", "b"), a, "applied", []string{}, []string{"released"}, 1},
		{"release_grapple not_held", table + "; FR-031", "unhappy (state error)", "release_grapple", release(one, "c", "b"), r, "not_held", []string{}, []string{}, -1},
		{"release_grapple same_grappler_and_target", table + "; FR-067", "unhappy (invalid input)", "release_grapple", release(one, "a", "a"), i, "same_grappler_and_target", []string{}, []string{}, -1},
		{"escape_grapple_from_record escaped", table + "; FR-038, FR-072", "happy", "escape_grapple_from_record", escape(two, "b", avail, rollOf(3), 10), a, "escaped", []string{}, []string{"escaped"}, 1},
		{"escape_grapple_from_record held", table + "; FR-039", "unhappy (boundary)", "escape_grapple_from_record", escape(two, "b", avail, rollOf(3), 9), a, "held", []string{}, []string{}, 2},
		{"escape_grapple_from_record held, forced", table + "; FR-045", "unhappy (rule interaction)", "escape_grapple_from_record", escape(one, "b", avail, d20.AutomaticFailure{}), a, "held", []string{}, []string{}, 1},
		{"escape_grapple_from_record not_grappled", table + "; FR-043", "unhappy (state error)", "escape_grapple_from_record", escape(one, "z", avail, rollOf(0), 10), r, "not_grappled", []string{}, []string{}, -1},
		{"escape_grapple_from_record no_action", table + "; FR-042", "unhappy (state error)", "escape_grapple_from_record", escape(one, "b", none, rollOf(0), 10), r, "no_action", []string{}, []string{}, -1},
		{"review_grapples applied, two endings", table + "; FR-068, FR-072", "happy", "review_grapples",
			review(two, obs("a", "b", 60), obs("c", "d", 0, combat.Incapacitated{})), a, "applied", []string{}, []string{"grappler_incapacitated", "out_of_range"}, 0},
		{"review_grapples applied, nothing ended", table + "; FR-056", "unhappy (empty and minimal)", "review_grapples", review(two), a, "applied", []string{}, []string{}, 2},
		{"review_grapples unknown_grapple", table + "; FR-054", "unhappy (invalid input)", "review_grapples", review(one, obs("a", "z", 0)), i, "unknown_grapple", []string{}, []string{}, -1},
		{"review_grapples duplicate_observation", table + "; FR-055", "unhappy (invalid input)", "review_grapples", review(one, obs("a", "b", 0), obs("a", "b", 0)), i, "duplicate_observation", []string{}, []string{}, -1},
		{"review_grapples limit_exceeded", table + "; plan P10", "unhappy (boundary)", "review_grapples", review(one, make([]combat.GrappleObservation, 257)...), i, "limit_exceeded", []string{}, []string{}, -1},
		{"drop_grapple_record applied", table + "; FR-098, FR-072", "happy", "drop_grapple_record", drop(two, "d"), a, "applied", []string{"grapple_record_dropped"}, []string{"condition_ended"}, 1},
		{"drop_grapple_record not_grappled", table + "; FR-061", "unhappy (state error)", "drop_grapple_record", drop(one, "a"), r, "not_grappled", []string{}, []string{}, -1},
	}
}

// Requirement: per case
// Path: per case.
func TestGrappleUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range grappleRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-076]", got, plain, err)
			}
			outcome := map[string]string{"applied": "success", "refused": "failure", "invalid": "failure"}[c.class]
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status only for invalid [%s]", spans, c.requirement)
			}
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v", sa, want)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, c.events) {
				t.Errorf("event kinds counted = %v, want %v [FR-072]", kinds, c.events)
			}
			if ended := values(measured(rec, "srd5e.combat.grapple.ended"), "srd5e.combat.grapple.end_reason"); !slices.Equal(ended, c.ended) {
				t.Errorf("ended reasons = %v, want %v [FR-072]", ended, c.ended)
			}
			for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.grapple.ended") {
				if m.Value != 1 {
					t.Errorf("ended increment = %v, want 1", m.Value)
				}
			}
			force := rec.Meter.MeasurementsNamed("srd5e.combat.grapple.in_force")
			wantAttrs := map[string]any{"srd5e.combat.operation": c.operation}
			switch {
			case c.inForce < 0 && len(force) != 0:
				t.Errorf("in_force = %+v, want none for a result with no set [FR-073]", force)
			case c.inForce >= 0 && (len(force) != 1 || force[0].Value != c.inForce || !reflect.DeepEqual(attrMap(force[0].Attrs), wantAttrs)):
				t.Errorf("in_force = %+v, want %v with %v [FR-073]", force, c.inForce, wantAttrs)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result [FR-074]", n)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/grappling.md FR-077, FR-078 - no attribute of any grapple signal carries a combatant id, a part
// name, a DC or a distance, and no grapple call logs.
// Path: unhappy (invalid input).
func TestGrappleAttributesAreClosedValues(t *testing.T) {
	t.Parallel()
	closed := []string{"begin_grapple", "release_grapple", "escape_grapple_from_record", "review_grapples", "drop_grapple_record",
		"applied", "refused", "invalid", "success", "failure", "escaped", "held", "released", "out_of_range", "grappler_incapacitated", "condition_ended",
		"part_in_use", "target_already_grappled", "not_held", "not_grappled", "no_action", "same_grappler_and_target", "negative_range",
		"unknown_grapple", "duplicate_observation", "missing_value", "limit_exceeded", "grapple_started", "grapple_record_dropped"}
	for _, c := range grappleRows(t) {
		rec, opts := recording()
		if _, err := c.run(opts...); err != nil {
			t.Fatal(err)
		}
		for _, m := range rec.Meter.Measurements() {
			for _, a := range m.Attrs {
				if s, ok := a.Value().(string); !ok || !slices.Contains(closed, s) {
					t.Errorf("%s: %s attribute %s = %v is not a closed value [FR-077]", c.name, m.Name, a.Key(), a.Value())
				}
				if !strings.HasPrefix(a.Key(), "srd5e.") && a.Key() != telemetry.AttrOutcome {
					t.Errorf("%s: attribute key %s is not a srd5e signal key", c.name, a.Key())
				}
			}
		}
		if got := rec.Logger.Records(); len(got) != 0 {
			t.Errorf("%s: logged %v [FR-078]", c.name, got)
		}
	}
}

// Requirement: doc/specs/grappling.md FR-069, FR-066 - a use case called twice with equal commands returns equal results,
// the command (and the zero value use case) stay usable, and every use case is silent and works with no options.
// Path: unhappy (determinism).
func TestGrappleUseCasesAreDeterministicWithoutOptions(t *testing.T) {
	t.Parallel()
	for _, c := range grappleRows(t) {
		first, err1 := c.run()
		second, err2 := c.run()
		if err1 != nil || err2 != nil || !reflect.DeepEqual(first, second) {
			t.Errorf("%s: two calls differ or failed: %v / %v", c.name, err1, err2)
		}
	}
	ctx := context.Background()
	var b combat.BeginGrapple
	if res, err := b.Run(ctx, combat.BeginGrappleCommand{Grappler: gid("a"), Target: gid("b"), Part: hand(0), RangeFeet: 5}); err != nil || reflect.TypeOf(res) != reflect.TypeOf(combat.BeginGrappleApplied{}) {
		t.Errorf("the zero BeginGrapple = %#v, %v, want a working silent use case", res, err)
	}
}

// Requirement: doc/specs/grappling.md FR-066, testing.md section 4 - every grapple use case but the escape returns
// ctx.Err() unwrapped for a cancelled context, before any signal.
// Path: unhappy (cancellation).
func TestGrappleUseCasesRefuseACancelledContext(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec, opts := recording()
	calls := map[string]error{}
	_, calls["begin"] = combat.NewBeginGrapple(opts...).Run(cancelled, combat.BeginGrappleCommand{})
	_, calls["release"] = combat.NewReleaseGrapple(opts...).Run(cancelled, combat.ReleaseGrappleCommand{})
	_, calls["review"] = combat.NewReviewGrapples(opts...).Run(cancelled, combat.ReviewGrapplesCommand{})
	_, calls["drop"] = combat.NewDropGrappleRecord(opts...).Run(cancelled, combat.DropGrappleRecordCommand{})
	for name, err := range calls {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s with a cancelled ctx = %v, want context.Canceled", name, err)
		}
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("signals recorded before the cancelled-context check: %d", n)
	}
}

// Requirement: doc/specs/grappling.md FR-066, FR-067, FR-068, FR-098 - the use cases return what the domain operations
// return: the new set, the grapple, the events and the endings, as copies.
// Path: happy.
func TestGrappleUseCasesMatchTheDomainOperations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ab, cd := held("a", "b", hand(0)), held("c", "d", hand(0))
	s := grapplesOf(t, ab)
	b, _ := combat.NewBeginGrapple().Run(ctx, combat.BeginGrappleCommand{Grapples: s, Grappler: gid("c"), Target: gid("d"), Part: hand(0), EscapeDC: 13, RangeFeet: 5})
	ba := b.(combat.BeginGrappleApplied)
	if ba.Grapple() != cd || !ba.Grapples().Equal(grapplesOf(t, ab, cd)) || len(ba.Events()) != 1 {
		t.Errorf("BeginGrapple = %#v", b)
	}
	ba.Events()[0] = nil
	if len(ba.Events()) != 1 || ba.Events()[0] == nil {
		t.Error("BeginGrapple events are not a copy")
	}
	r, _ := combat.NewReleaseGrapple().Run(ctx, combat.ReleaseGrappleCommand{Grapples: s, Grappler: gid("a"), Target: gid("b")})
	if ra := r.(combat.ReleaseGrappleApplied); ra.Grapples().Len() != 0 || ra.Ending().Grapple() != ab || ra.Ending().Cause() != combat.GrappleEndCauseReleased {
		t.Errorf("ReleaseGrapple = %#v", r)
	}
	v, _ := combat.NewReviewGrapples().Run(ctx, combat.ReviewGrapplesCommand{Grapples: s, Observations: []combat.GrappleObservation{combat.MustGrappleObservation(gid("a"), gid("b"), combat.ConditionSet{}, 6)}})
	va := v.(combat.ReviewGrapplesApplied)
	if e := va.Endings(); len(e) != 1 || e[0].Cause() != combat.GrappleEndCauseOutOfRange {
		t.Errorf("ReviewGrapples = %#v", v)
	}
	va.Endings()[0] = combat.GrappleEnding{}
	if va.Endings()[0].Cause() != combat.GrappleEndCauseOutOfRange {
		t.Error("ReviewGrapples endings are not a copy")
	}
	if nothing, _ := combat.NewReviewGrapples().Run(ctx, combat.ReviewGrapplesCommand{Grapples: s}); nothing.(combat.ReviewGrapplesApplied).Endings() == nil {
		t.Error("ReviewGrapples endings are nil, want an empty list [FR-068]")
	}
	d, _ := combat.NewDropGrappleRecord().Run(ctx, combat.DropGrappleRecordCommand{Grapples: s, Target: gid("b")})
	da := d.(combat.DropGrappleRecordApplied)
	if da.Grapples().Len() != 0 || len(da.Events()) != 1 || da.Events()[0].Kind() != "grapple_record_dropped" {
		t.Errorf("DropGrappleRecord = %#v", d)
	}
	da.Events()[0] = nil
	if da.Events()[0] == nil {
		t.Error("DropGrappleRecord events are not a copy")
	}
}

// Requirement: doc/specs/grappling.md FR-070 - the five operation values, spans and result values are the closed names of
// section 7 (pinned against accidental renames).
// Path: happy.
func TestGrappleSignalNamesArePinned(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	for _, c := range grappleRows(t) {
		if _, err := c.run(opts...); err != nil {
			t.Fatal(err)
		}
	}
	var spans []string
	for _, s := range rec.Tracer.Spans() {
		spans = append(spans, s.Name)
	}
	slices.Sort(spans)
	spans = slices.Compact(spans)
	if want := []string{"srd5e.combat.begin_grapple", "srd5e.combat.drop_grapple_record", "srd5e.combat.escape_grapple_from_record",
		"srd5e.combat.release_grapple", "srd5e.combat.review_grapples"}; !slices.Equal(spans, want) {
		t.Errorf("spans = %v, want %v", spans, want)
	}
	for _, name := range []string{"srd5e.combat.grapple.ended", "srd5e.combat.grapple.in_force"} {
		if len(rec.Meter.MeasurementsNamed(name)) == 0 {
			t.Errorf("metric %s was never recorded", name)
		}
	}
	for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.grapple.ended") {
		if _, ok := attrMap(m.Attrs)["srd5e.combat.grapple.end_reason"]; !ok || len(m.Attrs) != 1 {
			t.Errorf("ended attributes = %v, want only end_reason", m.Attrs)
		}
	}
}
