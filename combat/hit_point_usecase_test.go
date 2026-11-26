package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// recording returns a recorder and the options that send a use case's telemetry to it with a clock that steps 2 s.
func recording() (*telemetrytest.Recorder, []combat.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 2*time.Second)
	return rec, []combat.Option{combat.WithTelemetry(rec.Telemetry()), combat.WithClock(clock.Now)}
}

func attrMap(as []telemetry.Attr) map[string]any {
	m := map[string]any{}
	for _, a := range as {
		m[a.Key()] = a.Value()
	}
	return m
}

// measured returns the attribute maps of every measurement of the named metric.
func measured(rec *telemetrytest.Recorder, name string) []map[string]any {
	var out []map[string]any
	for _, m := range rec.Meter.MeasurementsNamed(name) {
		out = append(out, attrMap(m.Attrs))
	}
	return out
}

func values(maps []map[string]any, key string) []string {
	out := []string{}
	for _, m := range maps {
		out = append(out, m[key].(string))
	}
	slices.Sort(out)
	return out
}

type useCaseRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
	refusal                 string
	deaths                  []string
}

func dyingPC(t *testing.T, s, f int) combat.HitPointStatus {
	t.Helper()
	return dyingStatus(t, 20, s, f)
}

func stablePC(t *testing.T) combat.HitPointStatus {
	t.Helper()
	return status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities())
}

// rows builds the table shared by the shared-signal tests: one row per srd5e.combat.result value of table 4.10a.
func rows(t *testing.T) []useCaseRow {
	t.Helper()
	ctx := context.Background()
	d := func(n int) []combat.Damage { return []combat.Damage{fire(n)} }
	apply := func(st combat.HitPointStatus, dmg []combat.Damage, delivery combat.DamageDelivery) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewApplyDamageCommand(subject(), st, dmg, nil, delivery, combat.NormalHit, nil)
			if err != nil {
				return nil, err
			}
			return combat.NewApplyDamage(opts...).Run(ctx, cmd)
		}
	}
	restore := func(st combat.HitPointStatus, n int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewRestoreHitPointsCommand(subject(), st, n, nil)
			if err != nil {
				return nil, err
			}
			return combat.NewRestoreHitPoints(opts...).Run(ctx, cmd)
		}
	}
	gain := func(st combat.HitPointStatus) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewGainTemporaryHitPointsCommand(subject(), st, 5, combat.KeepCurrent)
			if err != nil {
				return nil, err
			}
			return combat.NewGainTemporaryHitPoints(opts...).Run(ctx, cmd)
		}
	}
	end := func(st combat.HitPointStatus) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewEndTemporaryHitPointsCommand(subject(), st)
			if err != nil {
				return nil, err
			}
			return combat.NewEndTemporaryHitPoints(opts...).Run(ctx, cmd)
		}
	}
	save := func(st combat.HitPointStatus, face int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewMakeDeathSavingThrowCommand(subject(), st, modifier.Normal, modifier.New(0))
			if err != nil {
				return nil, err
			}
			uc, err := combat.NewMakeDeathSavingThrow(dicetest.NewFixedRoller(face), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, cmd)
		}
	}
	stabilize := func(st combat.HitPointStatus) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewStabilizeCreatureCommand(subject(), st)
			if err != nil {
				return nil, err
			}
			return combat.NewStabilizeCreature(opts...).Run(ctx, cmd)
		}
	}
	recoverStable := func(st combat.HitPointStatus) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewRecoverStableCreatureCommand(subject(), st)
			if err != nil {
				return nil, err
			}
			return combat.NewRecoverStableCreature(opts...).Run(ctx, cmd)
		}
	}
	reduce := func(st combat.HitPointStatus, n int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewReduceHitPointMaximumCommand(subject(), st, n)
			if err != nil {
				return nil, err
			}
			return combat.NewReduceHitPointMaximum(opts...).Run(ctx, cmd)
		}
	}
	exhaustion := func(st combat.HitPointStatus) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewRecordExhaustionDeathCommand(subject(), st)
			if err != nil {
				return nil, err
			}
			return combat.NewRecordExhaustionDeath(opts...).Run(ctx, cmd)
		}
	}
	pc := pcStatus(t, 20, 20, 0)
	deadSt := deadStatus(t, combat.MassiveDamage)
	const table = "doc/specs/hit-points-and-damage.md table 4.10a, FR-079, FR-080, FR-107, FR-108"
	app, ref := "applied", "refused"
	dmgTaken := "damage_taken"
	return []useCaseRow{
		{"apply_damage damaged", table, "happy", "apply_damage", apply(pc, d(5), combat.OtherSource), app, "damaged", []string{dmgTaken}, "", nil},
		{"apply_damage killed", table, "unhappy (rule interaction)", "apply_damage", apply(pcStatus(t, 6, 12, 0), d(18), combat.OtherSource), app, "killed", []string{dmgTaken, "dropped_to_zero_hit_points", "died"}, "", []string{"massive_damage"}},
		{"apply_damage knocked_out", table, "unhappy (rule interaction)", "apply_damage", apply(pcStatus(t, 5, 20, 0), d(5), combat.MeleeAttackKnockOut), app, "knocked_out", []string{dmgTaken, "knocked_out"}, "", nil},
		{"apply_damage dropped_to_zero", table, "unhappy (rule interaction)", "apply_damage", apply(pcStatus(t, 5, 20, 0), d(5), combat.OtherSource), app, "dropped_to_zero", []string{dmgTaken, "dropped_to_zero_hit_points", "fell_unconscious"}, "", nil},
		{"apply_damage lost_stability", table, "unhappy (rule interaction)", "apply_damage", apply(stablePC(t), d(3), combat.OtherSource), app, "lost_stability", []string{dmgTaken, "lost_stability"}, "", nil},
		{"apply_damage absorbed", table, "unhappy (rule interaction)", "apply_damage", apply(pcStatus(t, 10, 20, 5), d(3), combat.OtherSource), app, "absorbed", []string{dmgTaken}, "", nil},
		{"apply_damage no_effect", table, "unhappy (empty and minimal)", "apply_damage", apply(pc, d(0), combat.OtherSource), app, "no_effect", []string{}, "", nil},
		{"apply_damage rejected_dead", table, "unhappy (state error)", "apply_damage", apply(deadSt, d(3), combat.OtherSource), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"restore_hit_points healed", table, "happy", "restore_hit_points", restore(pcStatus(t, 10, 20, 0), 5), app, "healed", []string{"hit_points_restored"}, "", nil},
		{"restore_hit_points recovered", table, "unhappy (rule interaction)", "restore_hit_points", restore(dyingPC(t, 0, 0), 5), app, "recovered", []string{"hit_points_restored", "recovered"}, "", nil},
		{"restore_hit_points no_effect", table, "unhappy (boundary)", "restore_hit_points", restore(pc, 5), app, "no_effect", []string{}, "", nil},
		{"restore_hit_points rejected_dead", table, "unhappy (state error)", "restore_hit_points", restore(deadSt, 5), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"gain_temporary_hit_points gained", table, "happy", "gain_temporary_hit_points", gain(pc), app, "gained", []string{"temporary_hit_points_gained"}, "", nil},
		{"gain_temporary_hit_points no_effect", table, "unhappy (rule interaction)", "gain_temporary_hit_points", gain(pcStatus(t, 20, 20, 9)), app, "no_effect", []string{}, "", nil},
		{"gain_temporary_hit_points rejected_dead", table, "unhappy (state error)", "gain_temporary_hit_points", gain(deadSt), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"end_temporary_hit_points ended", table, "happy", "end_temporary_hit_points", end(pcStatus(t, 20, 20, 4)), app, "ended", []string{"temporary_hit_points_ended"}, "", nil},
		{"end_temporary_hit_points no_effect", table, "unhappy (boundary)", "end_temporary_hit_points", end(pc), app, "no_effect", []string{}, "", nil},
		{"end_temporary_hit_points rejected_dead", table, "unhappy (state error)", "end_temporary_hit_points", end(deadSt), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"make_death_saving_throw succeeded", table, "happy", "make_death_saving_throw", save(dyingPC(t, 0, 0), 12), app, "succeeded", []string{"death_saving_throw_made"}, "", nil},
		{"make_death_saving_throw failed", table, "unhappy (boundary)", "make_death_saving_throw", save(dyingPC(t, 0, 0), 5), app, "failed", []string{"death_saving_throw_made"}, "", nil},
		{"make_death_saving_throw killed", table, "unhappy (boundary)", "make_death_saving_throw", save(dyingPC(t, 0, 2), 1), app, "killed", []string{"death_saving_throw_made", "died"}, "", []string{"death_saving_throw_failures"}},
		{"make_death_saving_throw recovered", table, "unhappy (boundary)", "make_death_saving_throw", save(dyingPC(t, 0, 0), 20), app, "recovered", []string{"death_saving_throw_made", "hit_points_restored", "recovered"}, "", nil},
		{"make_death_saving_throw stabilized", table, "unhappy (boundary)", "make_death_saving_throw", save(dyingPC(t, 2, 0), 15), app, "stabilized", []string{"death_saving_throw_made", "became_stable"}, "", nil},
		{"make_death_saving_throw not_dying", table, "unhappy (state error)", "make_death_saving_throw", save(pc, 15), ref, "not_dying", []string{}, "not_dying", nil},
		{"make_death_saving_throw rejected_dead", table, "unhappy (state error)", "make_death_saving_throw", save(deadSt, 15), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"stabilize_creature stabilized", table, "happy", "stabilize_creature", stabilize(dyingPC(t, 1, 1)), app, "stabilized", []string{"became_stable"}, "", nil},
		{"stabilize_creature not_dying", table, "unhappy (state error)", "stabilize_creature", stabilize(stablePC(t)), ref, "not_dying", []string{}, "not_dying", nil},
		{"stabilize_creature rejected_dead", table, "unhappy (state error)", "stabilize_creature", stabilize(deadSt), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"recover_stable_creature recovered", table, "happy", "recover_stable_creature", recoverStable(stablePC(t)), app, "recovered", []string{"hit_points_restored", "recovered"}, "", nil},
		{"recover_stable_creature not_stable", table, "unhappy (state error)", "recover_stable_creature", recoverStable(dyingPC(t, 0, 0)), ref, "not_stable", []string{}, "not_stable", nil},
		{"recover_stable_creature rejected_dead", table, "unhappy (state error)", "recover_stable_creature", recoverStable(deadSt), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"reduce_hit_point_maximum reduced", table, "happy", "reduce_hit_point_maximum", reduce(pc, 5), app, "reduced", []string{"hit_point_maximum_reduced"}, "", nil},
		{"reduce_hit_point_maximum no_effect", table, "unhappy (boundary)", "reduce_hit_point_maximum", reduce(pc, 0), app, "no_effect", []string{}, "", nil},
		{"reduce_hit_point_maximum killed", table, "unhappy (boundary)", "reduce_hit_point_maximum", reduce(pc, 20), app, "killed", []string{"hit_point_maximum_reduced", "died"}, "", []string{"hit_point_maximum_zero"}},
		{"reduce_hit_point_maximum rejected_dead", table, "unhappy (state error)", "reduce_hit_point_maximum", reduce(deadSt, 5), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
		{"record_exhaustion_death killed", table, "happy", "record_exhaustion_death", exhaustion(pc), app, "killed", []string{"died"}, "", []string{"exhaustion"}},
		{"record_exhaustion_death rejected_dead", table, "unhappy (state error)", "record_exhaustion_death", exhaustion(deadSt), ref, "rejected_dead", []string{}, "creature_is_dead", nil},
	}
}

// Requirement: per case
// Path: per case.
func TestHitPointUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range rows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-081]", got, plain, err)
			}
			outcome := "success"
			if c.class == "refused" {
				outcome = "failure"
			}
			span := "srd5e.combat." + c.operation
			spans := rec.Tracer.SpansNamed(span)
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status [%s]", spans, c.requirement)
			}
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v", ops, want)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v", sa, want)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			if gotKinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(gotKinds, sorted(c.events)) {
				t.Errorf("event kinds counted = %v, want %v", gotKinds, sorted(c.events))
			}
			refusals := measured(rec, "srd5e.combat.hit_point_refusals")
			if c.refusal == "" && len(refusals) != 0 || c.refusal != "" && (len(refusals) != 1 || refusals[0]["srd5e.combat.refusal_reason"] != c.refusal || refusals[0]["srd5e.combat.operation"] != c.operation) {
				t.Errorf("refusals = %v, want reason %q", refusals, c.refusal)
			}
			if deaths := values(measured(rec, "srd5e.combat.deaths"), "srd5e.combat.death_cause"); !slices.Equal(deaths, sorted(c.deaths)) {
				t.Errorf("deaths = %v, want %v", deaths, c.deaths)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result", n)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

func sorted(s []string) []string {
	out := slices.Clone(s)
	if out == nil {
		out = []string{}
	}
	slices.Sort(out)
	return out
}

// checkTelemetryHygiene asserts the rules of FR-082, FR-080 and plan P12: no misuse, no log, no attribute that is
// the CombatantID, no srd5e.d20_test signal.
func checkTelemetryHygiene(t *testing.T, rec *telemetrytest.Recorder) {
	t.Helper()
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
	if got := rec.Logger.Records(); len(got) != 0 {
		t.Errorf("logged %v, want nothing (FR-080)", got)
	}
	for _, m := range rec.Meter.Measurements() {
		if strings.HasPrefix(m.Name, "srd5e.d20_test.") {
			t.Errorf("a combat call recorded the d20 signal %s (P12)", m.Name)
		}
		for _, a := range m.Attrs {
			if a.Value() == string(subject()) {
				t.Errorf("%s attribute %s carries the CombatantID (FR-082)", m.Name, a.Key())
			}
		}
	}
	for _, s := range rec.Tracer.Spans() {
		if strings.HasPrefix(s.Name, "srd5e.d20_test.") {
			t.Errorf("a combat call started the d20 span %s (P12)", s.Name)
		}
		for _, a := range s.Attrs {
			if a.Value() == string(subject()) {
				t.Errorf("span attribute %s carries the CombatantID", a.Key())
			}
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-110, shared-types FR-044 - a use case takes its inputs only
// through its command; the zero command is invalid input before any signal, and an already cancelled context is
// returned unwrapped before any signal (go.md section 4).
// Path: unhappy (invalid input).
func TestHitPointUseCasesZeroCommandAndCancelledContext(t *testing.T) {
	t.Parallel()
	pc := pcStatus(t, 10, 20, 0)
	roller := func() combat.Roller { return dicetest.NewFixedRoller(12, 3) }
	save, err := combat.NewMakeDeathSavingThrow(roller())
	if err != nil {
		t.Fatal(err)
	}
	delay, err := combat.NewRollStableRecoveryDelay(roller())
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	live := context.Background()
	mk := func(opts ...combat.Option) map[string][2]func(ctx context.Context) error {
		dmg, _ := combat.NewApplyDamageCommand(subject(), pc, []combat.Damage{fire(1)}, nil, combat.OtherSource, combat.NormalHit, nil)
		heal, _ := combat.NewRestoreHitPointsCommand(subject(), pc, 1, nil)
		gain, _ := combat.NewGainTemporaryHitPointsCommand(subject(), pc, 1, combat.AcceptNew)
		end, _ := combat.NewEndTemporaryHitPointsCommand(subject(), pc)
		sv, _ := combat.NewMakeDeathSavingThrowCommand(subject(), dyingStatus(t, 20, 0, 0), modifier.Normal, modifier.New(0))
		stab, _ := combat.NewStabilizeCreatureCommand(subject(), pc)
		rec, _ := combat.NewRecoverStableCreatureCommand(subject(), pc)
		red, _ := combat.NewReduceHitPointMaximumCommand(subject(), pc, 1)
		ex, _ := combat.NewRecordExhaustionDeathCommand(subject(), pc)
		save, _ := combat.NewMakeDeathSavingThrow(roller(), opts...)
		delay, _ := combat.NewRollStableRecoveryDelay(roller(), opts...)
		e := func(_ any, err error) error { return err }
		return map[string][2]func(context.Context) error{
			"apply_damage": {func(c context.Context) error {
				return e(combat.NewApplyDamage(opts...).Run(c, dmg))
			}, func(c context.Context) error {
				return e(combat.NewApplyDamage(opts...).Run(c, combat.ApplyDamageCommand{}))
			}},
			"restore_hit_points": {func(c context.Context) error { return e(combat.NewRestoreHitPoints(opts...).Run(c, heal)) },
				func(c context.Context) error {
					return e(combat.NewRestoreHitPoints(opts...).Run(c, combat.RestoreHitPointsCommand{}))
				}},
			"gain_temporary_hit_points": {func(c context.Context) error { return e(combat.NewGainTemporaryHitPoints(opts...).Run(c, gain)) },
				func(c context.Context) error {
					return e(combat.NewGainTemporaryHitPoints(opts...).Run(c, combat.GainTemporaryHitPointsCommand{}))
				}},
			"end_temporary_hit_points": {func(c context.Context) error { return e(combat.NewEndTemporaryHitPoints(opts...).Run(c, end)) },
				func(c context.Context) error {
					return e(combat.NewEndTemporaryHitPoints(opts...).Run(c, combat.EndTemporaryHitPointsCommand{}))
				}},
			"make_death_saving_throw": {func(c context.Context) error { return e(save.Run(c, sv)) },
				func(c context.Context) error { return e(save.Run(c, combat.MakeDeathSavingThrowCommand{})) }},
			"stabilize_creature": {func(c context.Context) error { return e(combat.NewStabilizeCreature(opts...).Run(c, stab)) },
				func(c context.Context) error {
					return e(combat.NewStabilizeCreature(opts...).Run(c, combat.StabilizeCreatureCommand{}))
				}},
			"roll_stable_recovery_delay": {func(c context.Context) error { return e(delay.Run(c, combat.RollStableRecoveryDelayCommand{})) },
				func(c context.Context) error {
					return e(combat.RollStableRecoveryDelay{}.Run(c, combat.RollStableRecoveryDelayCommand{}))
				}},
			"recover_stable_creature": {func(c context.Context) error { return e(combat.NewRecoverStableCreature(opts...).Run(c, rec)) },
				func(c context.Context) error {
					return e(combat.NewRecoverStableCreature(opts...).Run(c, combat.RecoverStableCreatureCommand{}))
				}},
			"reduce_hit_point_maximum": {func(c context.Context) error { return e(combat.NewReduceHitPointMaximum(opts...).Run(c, red)) },
				func(c context.Context) error {
					return e(combat.NewReduceHitPointMaximum(opts...).Run(c, combat.ReduceHitPointMaximumCommand{}))
				}},
			"record_exhaustion_death": {func(c context.Context) error { return e(combat.NewRecordExhaustionDeath(opts...).Run(c, ex)) },
				func(c context.Context) error {
					return e(combat.NewRecordExhaustionDeath(opts...).Run(c, combat.RecordExhaustionDeathCommand{}))
				}},
		}
	}
	_, _ = save, delay
	rec, opts := recording()
	for name, calls := range mk(opts...) {
		if err := calls[0](cancelled); !errors.Is(err, context.Canceled) {
			t.Errorf("%s with a cancelled ctx = %v, want context.Canceled", name, err)
		}
		if err := calls[1](live); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s with the zero command = %v, want ErrInvalidInput", name, err)
		}
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("signals recorded before validation: %d", n)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-110, FR-109 - a use case returns the operation's status and
// events unchanged, and hands out copies of its events.
// Path: unhappy (immutability).
func TestAppliedResultsHandOutCopies(t *testing.T) {
	t.Parallel()
	cmd, _ := combat.NewApplyDamageCommand(subject(), pcStatus(t, 20, 20, 0), []combat.Damage{fire(5)}, nil, combat.OtherSource, combat.NormalHit, nil)
	res, err := combat.NewApplyDamage().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	a := res.(combat.DamageApplied)
	tr, _ := pcStatus(t, 20, 20, 0).ApplyDamage(subject(), []combat.Damage{fire(5)}, nil, combat.OtherSource, combat.NormalHit, nil)
	want := tr.(combat.HitPointTransitionApplied)
	if a.Status() != want.Status() || !reflect.DeepEqual(a.Events(), want.Events()) || !reflect.DeepEqual(a.Resolutions(), want.Resolutions()) {
		t.Errorf("result differs from the operation's")
	}
	ev := a.Events()
	ev[0] = nil
	if a.Events()[0] == nil {
		t.Errorf("Events aliases the stored slice")
	}
	if a.Resolutions()[0].Dealt() != 5 {
		t.Errorf("resolutions lost")
	}
}
