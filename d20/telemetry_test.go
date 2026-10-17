package d20_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func abilityName(cmd d20.Command) string {
	switch c := cmd.(type) {
	case d20.AbilityCheckCommand:
		return c.Ability.String()
	case d20.SavingThrowCommand:
		return c.Ability.String()
	case d20.AttackRollCommand:
		return c.Ability.String()
	}
	return ""
}

func expectedAttrs(tc row) map[string]any {
	m := map[string]any{
		d20.AttrKind:          tc.want.kind.String(),
		d20.AttrAbility:       abilityName(tc.cmd),
		telemetry.AttrOutcome: tc.want.outcome.String(),
	}
	if tc.want.reason != 0 {
		m[d20.AttrForcedFailureReason] = tc.want.reason.String()
		return m
	}
	m[d20.AttrRollMode] = tc.want.mode.String()
	if tc.want.kind == d20.AttackRoll {
		m[d20.AttrCriticalHit] = tc.want.critical
	}
	return m
}

func TestSpanAndResolvedCounter(t *testing.T) {
	t.Parallel()
	for _, tc := range caseTable() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := newRecorded(t, tc.rolls...)
			resolve(t, rr.resolver, tc.cmd)

			spans := rr.rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != d20.SpanResolve || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			want := expectedAttrs(tc)
			if got := attrMap(spans[0].Attrs); !mapsEqual(got, want) {
				t.Errorf("span attrs = %v, want %v", got, want)
			}
			resolved := rr.rec.Meter.MeasurementsNamed(d20.MetricResolved)
			if len(resolved) != 1 || resolved[0].Value != 1 || resolved[0].Unit != "{test}" || resolved[0].Kind != telemetrytest.KindCounter {
				t.Fatalf("resolved = %+v", resolved)
			}
			if got := attrMap(resolved[0].Attrs); !mapsEqual(got, want) {
				t.Errorf("resolved attrs = %v, want %v", got, want)
			}
			if v := rr.rec.Meter.MeasurementsNamed(d20.MetricErrors); len(v) != 0 {
				t.Errorf("errors counted for a normal outcome: %+v", v)
			}
			telemetrytest.RequireNoViolations(t, rr.rec)
		})
	}
}

func TestSpanStartsWithKindAndAbility(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	var atRoll map[string]any
	roller := brokenRoller{func(e dice.Expression) (dice.Result, error) {
		atRoll = attrMap(rec.Tracer.Spans()[0].Attrs)
		return dice.NewResult(e, []dice.DieRoll{dice.MustDieRoll(dice.D20(), 10)})
	}}
	r := newResolver(t, roller, d20.WithTelemetry(rec.Telemetry()))
	resolve(t, r, d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0)})
	want := map[string]any{d20.AttrKind: "ability_check", d20.AttrAbility: "strength"}
	if !mapsEqual(atRoll, want) {
		t.Errorf("attrs while rolling = %v, want %v", atRoll, want)
	}
}

func TestDuration(t *testing.T) {
	t.Parallel()
	check := d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0)}
	t.Run("a call records one duration in seconds", func(t *testing.T) {
		t.Parallel()
		rr := newRecorded(t, 10)
		resolve(t, rr.resolver, check)
		d := rr.rec.Meter.MeasurementsNamed(d20.MetricDuration)
		if len(d) != 1 || d[0].Value != 0.25 || d[0].Unit != "s" || d[0].Kind != telemetrytest.KindHistogram {
			t.Fatalf("duration = %+v", d)
		}
		if got, want := attrMap(d[0].Attrs), map[string]any{d20.AttrKind: "ability_check"}; !mapsEqual(got, want) {
			t.Errorf("attrs = %v", got)
		}
	})
	t.Run("a failed call still records its duration", func(t *testing.T) {
		t.Parallel()
		rr := newRecorded(t) // exhausted roller
		if _, err := rr.resolver.Resolve(context.Background(), check); err == nil {
			t.Fatal("expected an error")
		}
		if d := rr.rec.Meter.MeasurementsNamed(d20.MetricDuration); len(d) != 1 || d[0].Value != 0.25 {
			t.Errorf("duration = %+v", d)
		}
	})
	t.Run("a forced failure records a duration", func(t *testing.T) {
		t.Parallel()
		rr := newRecorded(t)
		resolve(t, rr.resolver, d20.SavingThrowCommand{Ability: ability.Wisdom, Target: 10, Attempt: d20.ChooseToFail{}})
		if d := rr.rec.Meter.MeasurementsNamed(d20.MetricDuration); len(d) != 1 {
			t.Errorf("duration = %+v", d)
		}
	})
	t.Run("the span encloses the timed work", func(t *testing.T) {
		t.Parallel()
		rec := telemetrytest.NewRecorder()
		var endedWhileRolling bool
		roller := brokenRoller{func(e dice.Expression) (dice.Result, error) {
			endedWhileRolling = rec.Tracer.Spans()[0].Ended
			return dice.NewResult(e, []dice.DieRoll{dice.MustDieRoll(dice.D20(), 10)})
		}}
		resolve(t, newResolver(t, roller, d20.WithTelemetry(rec.Telemetry())), check)
		if endedWhileRolling || !rec.Tracer.Spans()[0].Ended {
			t.Error("span lifecycle does not enclose the call")
		}
	})
}

func TestModifiersAndNaturalHistograms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mods      d20.RollModifiers
		cmd       func(d20.RollAttempt) d20.Command
		face      int
		wantCount float64
		wantKind  string
	}{
		{"ability only", abilityMods(3), checkCmd, 10, 1, "ability_check"},
		{"with proficiency", abilityMods(3).WithProficiencyBonus(modifier.New(2)), checkCmd, 7, 2, "ability_check"},
		{"proficiency and two others", abilityMods(3).WithProficiencyBonus(modifier.New(2)).WithOthers(modifier.New(1), modifier.New(-2)), saveCmd, 20, 4, "saving_throw"},
		{"attack roll", abilityMods(3).WithOthers(modifier.New(1)), attackCmd, 1, 2, "attack_roll"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := newRecorded(t, tc.face)
			resolve(t, rr.resolver, tc.cmd(d20.RollAttempt{Modifiers: tc.mods}))
			wantAttrs := map[string]any{d20.AttrKind: tc.wantKind}

			m := rr.rec.Meter.MeasurementsNamed(d20.MetricModifiers)
			if len(m) != 1 || m[0].Value != tc.wantCount || m[0].Unit != "{modifier}" || !mapsEqual(attrMap(m[0].Attrs), wantAttrs) {
				t.Errorf("modifiers = %+v", m)
			}
			n := rr.rec.Meter.MeasurementsNamed(d20.MetricNatural)
			if len(n) != 1 || n[0].Value != float64(tc.face) || n[0].Unit != "{face}" || !mapsEqual(attrMap(n[0].Attrs), wantAttrs) {
				t.Errorf("natural = %+v", n)
			}
		})
	}
}

func checkCmd(a d20.RollAttempt) d20.Command {
	return d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: a}
}
func saveCmd(a d20.RollAttempt) d20.Command {
	return d20.SavingThrowCommand{Ability: ability.Strength, Target: 10, Attempt: a}
}
func attackCmd(a d20.RollAttempt) d20.Command {
	return d20.AttackRollCommand{Ability: ability.Strength, Target: 10, Attempt: a}
}

func TestNaturalIsTheKeptDie(t *testing.T) {
	t.Parallel()
	rr := newRecorded(t, 18, 3)
	resolve(t, rr.resolver, checkCmd(rollAttempt(0, modifier.Disadvantage{})))
	if n := rr.rec.Meter.MeasurementsNamed(d20.MetricNatural); len(n) != 1 || n[0].Value != 3 {
		t.Errorf("natural = %+v, want the kept die 3", n)
	}
}

func TestForcedFailureRecordsNoRollSignals(t *testing.T) {
	t.Parallel()
	rr := newRecorded(t)
	resolve(t, rr.resolver, d20.AbilityCheckCommand{Ability: ability.Wisdom, Target: 10, Attempt: d20.AutomaticFailure{}})
	for _, name := range []string{d20.MetricModifiers, d20.MetricNatural, d20.MetricErrors} {
		if m := rr.rec.Meter.MeasurementsNamed(name); len(m) != 0 {
			t.Errorf("%s recorded for a forced failure: %+v", name, m)
		}
	}
	if _, ok := rr.rec.Tracer.Spans()[0].AttrValue(d20.AttrRollMode); ok {
		t.Error("forced failure span has a roll mode")
	}
}

func TestNothingIsLogged(t *testing.T) {
	t.Parallel()
	rr := newRecorded(t, 10)
	resolve(t, rr.resolver, checkCmd(rollAttempt(0)))
	resolve(t, rr.resolver, d20.SavingThrowCommand{Ability: ability.Wisdom, Target: 10, Attempt: d20.ChooseToFail{}})
	_, _ = rr.resolver.Resolve(context.Background(), checkCmd(rollAttempt(0))) // exhausted: fails
	if n := len(rr.rec.Logger.Records()); n != 0 {
		t.Errorf("%d log records, want none", n)
	}
}

func TestAttributeValuesComeFromClosedSets(t *testing.T) {
	t.Parallel()
	closed := map[string][]any{
		d20.AttrKind:                {"ability_check", "saving_throw", "attack_roll"},
		d20.AttrAbility:             {"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma"},
		d20.AttrRollMode:            {"normal", "advantage", "disadvantage"},
		telemetry.AttrOutcome:       {"success", "failure"},
		d20.AttrCriticalHit:         {true, false},
		d20.AttrForcedFailureReason: {"chosen_to_fail", "automatic"},
		d20.AttrRollerFailure:       {"unavailable", "invalid_result"},
		telemetry.AttrErrorType:     {"dice_roller_error"},
	}
	check := func(t *testing.T, attrs []telemetry.Attr) {
		t.Helper()
		for _, a := range attrs {
			allowed, ok := closed[a.Key()]
			if !ok {
				t.Errorf("unexpected attribute %q", a.Key())
				continue
			}
			found := false
			for _, v := range allowed {
				found = found || v == a.Value()
			}
			if !found {
				t.Errorf("%s = %v (%T) outside its closed set", a.Key(), a.Value(), a.Value())
			}
		}
	}
	for _, tc := range caseTable() {
		rr := newRecorded(t, tc.rolls...)
		resolve(t, rr.resolver, tc.cmd)
		check(t, rr.rec.Tracer.Spans()[0].Attrs)
		for _, m := range rr.rec.Meter.Measurements() {
			check(t, m.Attrs)
		}
	}
	for _, a := range ability.All() {
		rr := newRecorded(t, 10)
		resolve(t, rr.resolver, d20.AbilityCheckCommand{Ability: a, Target: 10, Attempt: rollAttempt(0)})
		if v, _ := rr.rec.Tracer.Spans()[0].AttrValue(d20.AttrAbility); v != a.String() {
			t.Errorf("ability attribute = %v, want %v", v, a)
		}
	}
	rr := newRecorded(t)
	_, _ = rr.resolver.Resolve(context.Background(), checkCmd(rollAttempt(0)))
	check(t, rr.rec.Tracer.Spans()[0].Attrs)
	for _, m := range rr.rec.Meter.Measurements() {
		check(t, m.Attrs)
	}
}

func TestSignalNamesAreValid(t *testing.T) {
	t.Parallel()
	for _, n := range []string{d20.SpanResolve, d20.MetricResolved, d20.MetricDuration, d20.MetricModifiers, d20.MetricNatural, d20.MetricErrors} {
		if !telemetry.ValidName(n) {
			t.Errorf("%q is not a valid signal name", n)
		}
	}
}

func TestSilentAndRecordingResolversAgree(t *testing.T) {
	t.Parallel()
	for _, tc := range caseTable() {
		silent := resolve(t, newResolver(t, newFixed(tc.rolls)), tc.cmd)
		recorded := resolve(t, newRecorded(t, tc.rolls...).resolver, tc.cmd)
		if silent.Kind() != recorded.Kind() || silent.Outcome() != recorded.Outcome() {
			t.Errorf("%s: results differ between silent and recording resolvers", tc.name)
		}
	}
	failing := brokenRoller{func(dice.Expression) (dice.Result, error) { return dice.Result{}, errors.New("down") }}
	rec := telemetrytest.NewRecorder()
	_, e1 := newResolver(t, failing).Resolve(context.Background(), checkCmd(rollAttempt(0)))
	_, e2 := newResolver(t, failing, d20.WithTelemetry(rec.Telemetry())).Resolve(context.Background(), checkCmd(rollAttempt(0)))
	if e1 == nil || e2 == nil || e1.Error() != e2.Error() {
		t.Errorf("errors differ: %v vs %v", e1, e2)
	}
}

func TestPartialTelemetryDefaultsToNoop(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	r := newResolver(t, newFixed([]int{10}), d20.WithTelemetry(telemetry.Telemetry{Tracer: rec.Tracer}), d20.WithClock(nil))
	resolve(t, r, checkCmd(rollAttempt(0)))
	if len(rec.Tracer.Spans()) != 1 || len(rec.Meter.Measurements()) != 0 {
		t.Error("only the supplied tracer should have recorded")
	}
}

func TestDefaultClockIsUsed(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	r := newResolver(t, newFixed([]int{10}), d20.WithTelemetry(rec.Telemetry()))
	resolve(t, r, checkCmd(rollAttempt(0)))
	d := rec.Meter.MeasurementsNamed(d20.MetricDuration)
	if len(d) != 1 || d[0].Value < 0 || d[0].Value > float64(time.Minute/time.Second) {
		t.Errorf("duration = %+v", d)
	}
}
