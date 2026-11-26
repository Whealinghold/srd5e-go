package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

type encUC = func(opts ...combat.Option) (any, error)

func ucStart(est combat.InitiativeOrderEstablished, cs ...combat.Combatant) encUC {
	return func(opts ...combat.Option) (any, error) {
		return combat.NewStartEncounter(opts...).Run(context.Background(), combat.NewStartEncounterCommand(eid(), est, cs))
	}
}

func ucAdvance(e combat.Encounter, a combat.TurnAdvance) encUC {
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewAdvanceEncounterTurnCommand(e, a)
		if err != nil {
			return nil, err
		}
		return combat.NewAdvanceEncounterTurn(opts...).Run(context.Background(), cmd)
	}
}

func ucDamage(e combat.Encounter, id string, n int, delivery combat.DamageDelivery) encUC {
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewApplyDamageToCombatantCommand(e, cid2(id), encSlash(n), nil, delivery, combat.NormalHit)
		if err != nil {
			return nil, err
		}
		return combat.NewApplyDamageToCombatant(opts...).Run(context.Background(), cmd)
	}
}

func encRows(t *testing.T) []encRow {
	t.Helper()
	two := []combat.Combatant{encPC(t, "a", combat.SideA, 20, 20), encPC(t, "b", combat.SideB, 20, 20)}
	est := encEstablished(t, "a", "b")
	healthy := encABC(t)
	dyingE := encABCDying(t)
	stable, _ := encStart(t, encC(t, "a", combat.SideA, status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), combat.Unconscious{}), encPC(t, "b", combat.SideB, 20, 20))
	deadE, _ := encStart(t, encC(t, "a", combat.SideA, deadStatus(t, combat.MonsterDeath)), encPC(t, "b", combat.SideB, 20, 20))
	endedE := endedEncounter(t, healthy)
	grap := grappleEnc(t, nil)
	deadGrap, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "b", combat.SideB, deadStatus(t, combat.MonsterDeath), combat.Grappled{Grappler: cid("a")}))
	immuneSet, err := combat.NewConditionSet(nil, []content.ConditionKind{content.ConditionBlinded})
	if err != nil {
		t.Fatal(err)
	}
	immuneE, _ := encStart(t, mustCombatant(t, "a", combat.SideA, pcStatus(t, 20, 20, 0), immuneSet), encPC(t, "b", combat.SideB, 20, 20))
	blindE, _ := encStart(t, encC(t, "a", combat.SideA, pcStatus(t, 20, 20, 0), combat.Blinded{}), encPC(t, "b", combat.SideB, 20, 20))
	uncE, _ := encStart(t, encC(t, "a", combat.SideA, pcStatus(t, 20, 20, 0), combat.Unconscious{}), encPC(t, "b", combat.SideB, 20, 20))
	exhE, _ := encStart(t, encC(t, "a", combat.SideA, pcStatus(t, 20, 20, 0), exh(2)), encPC(t, "b", combat.SideB, 20, 20))
	madeE := func() combat.Encounter {
		tr, err := dyingE.ApplyDeathSavingThrow(cid("a"), 12, 12)
		e, _ := encOK(t, tr, err)
		return e
	}()
	_ = deadGrap
	ctx := context.Background()
	cond := func(e combat.Encounter, id string, c combat.Condition) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewApplyConditionToCombatantCommand(e, cid2(id), c)
			if err != nil {
				return nil, err
			}
			return combat.NewApplyConditionToCombatant(opts...).Run(ctx, cmd)
		}
	}
	remove := func(e combat.Encounter, id string, k content.ConditionKind) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewRemoveConditionFromCombatantCommand(e, cid2(id), k)
			if err != nil {
				return nil, err
			}
			return combat.NewRemoveConditionFromCombatant(opts...).Run(ctx, cmd)
		}
	}
	reduce := func(e combat.Encounter, id string, n int) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewReduceCombatantExhaustionCommand(e, cid2(id), n)
			if err != nil {
				return nil, err
			}
			return combat.NewReduceCombatantExhaustion(opts...).Run(ctx, cmd)
		}
	}
	heal := func(e combat.Encounter, id string, n int) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewApplyHealingToCombatantCommand(e, cid2(id), n)
			if err != nil {
				return nil, err
			}
			return combat.NewApplyHealingToCombatant(opts...).Run(ctx, cmd)
		}
	}
	save := func(e combat.Encounter, id string, natural int) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewApplyDeathSavingThrowToCombatantCommand(e, cid2(id), natural, natural)
			if err != nil {
				return nil, err
			}
			return combat.NewApplyDeathSavingThrowToCombatant(opts...).Run(ctx, cmd)
		}
	}
	stab := func(e combat.Encounter, id string) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewStabilizeCombatantCommand(e, cid2(id))
			if err != nil {
				return nil, err
			}
			return combat.NewStabilizeCombatant(opts...).Run(ctx, cmd)
		}
	}
	recov := func(e combat.Encounter, id string) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewRecoverStableCombatantCommand(e, cid2(id))
			if err != nil {
				return nil, err
			}
			return combat.NewRecoverStableCombatant(opts...).Run(ctx, cmd)
		}
	}
	temp := func(e combat.Encounter, id string) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewGiveCombatantTemporaryHitPointsCommand(e, cid2(id), 5, combat.AcceptNew)
			if err != nil {
				return nil, err
			}
			return combat.NewGiveCombatantTemporaryHitPoints(opts...).Run(ctx, cmd)
		}
	}
	maxHP := func(e combat.Encounter, id string, n int) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewReduceCombatantHitPointMaximumCommand(e, cid2(id), n)
			if err != nil {
				return nil, err
			}
			return combat.NewReduceCombatantHitPointMaximum(opts...).Run(ctx, cmd)
		}
	}
	end := func(e combat.Encounter) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
			if err != nil {
				return nil, err
			}
			return combat.NewEndEncounter(opts...).Run(ctx, cmd)
		}
	}
	check := func(e combat.Encounter, id string, d, r int) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewCheckCombatantGrappleCommand(e, cid2(id), d, r)
			if err != nil {
				return nil, err
			}
			return combat.NewCheckCombatantGrapple(opts...).Run(ctx, cmd)
		}
	}
	release := func(e combat.Encounter, id string) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewReleaseCombatantGrappleCommand(e, cid2(id))
			if err != nil {
				return nil, err
			}
			return combat.NewReleaseCombatantGrapple(opts...).Run(ctx, cmd)
		}
	}
	escape := func(e combat.Encounter, id string, o d20.Outcome) encUC {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewEscapeCombatantGrappleCommand(e, cid2(id), o)
			if err != nil {
				return nil, err
			}
			return combat.NewEscapeCombatantGrapple(opts...).Run(ctx, cmd)
		}
	}
	const a, r, i = "applied", "refused", "invalid"
	const dead, ended, unk = "combatant_is_dead", "encounter_already_ended", "unknown_combatant"
	const tbl = encSpec + "table 4.10.1, FR-095 "
	damaged := []string{"damage_taken"}
	return []encRow{
		{"start applied", tbl + "FR-057", "happy", "start_encounter", ucStart(est, two...), a, "applied", []string{"encounter_started", "tracker_round_started", "tracker_turn_started"}},
		{"start invalid", tbl + "FR-059", "unhappy (invalid input)", "start_encounter", ucStart(est, two[0], two[0], two[1]), i, "invalid_encounter", nil},
		{"advance applied", tbl, "happy", "advance_encounter_turn", ucAdvance(healthy, combat.TurnAdvance{}), a, "applied", []string{"tracker_turn_ended", "tracker_turn_started"}},
		{"advance ended", tbl, "unhappy (state)", "advance_encounter_turn", ucAdvance(endedE, combat.TurnAdvance{}), r, ended, nil},
		{"advance unknown speeds", tbl, "unhappy (invalid input)", "advance_encounter_turn", ucAdvance(healthy, combat.TurnAdvance{Updates: []combat.SpeedUpdate{{Combatant: "ghost"}}}), i, unk, nil},
		{"damage applied", tbl, "happy", "apply_damage_to_combatant", ucDamage(healthy, "a", 3, combat.OtherSource), a, "applied", damaged},
		{"damage dead", tbl, "unhappy (state)", "apply_damage_to_combatant", ucDamage(deadE, "a", 3, combat.OtherSource), r, dead, nil},
		{"damage ended", tbl, "unhappy (state)", "apply_damage_to_combatant", ucDamage(endedE, "a", 3, combat.OtherSource), r, ended, nil},
		{"damage unknown", tbl, "unhappy (invalid input)", "apply_damage_to_combatant", ucDamage(healthy, "ghost", 3, combat.OtherSource), i, unk, nil},
		{"healing applied", tbl, "happy", "apply_healing_to_combatant", heal(dyingE, "a", 4), a, "applied", []string{"hit_points_restored", "recovered", "condition_removed"}},
		{"healing dead", tbl, "unhappy (state)", "apply_healing_to_combatant", heal(deadE, "a", 4), r, dead, nil},
		{"healing unknown", tbl, "unhappy (invalid input)", "apply_healing_to_combatant", heal(healthy, "ghost", 4), i, unk, nil},
		{"condition applied", tbl, "happy", "apply_condition_to_combatant", cond(healthy, "a", combat.Blinded{}), a, "applied", []string{"condition_applied"}},
		{"condition already held", tbl, "unhappy (rule interaction)", "apply_condition_to_combatant", cond(blindE, "a", combat.Blinded{}), r, "already_has_condition", nil},
		{"condition immune", tbl, "unhappy (rule interaction)", "apply_condition_to_combatant", cond(immuneE, "a", combat.Blinded{}), r, "immune_to_condition", nil},
		{"condition dead", tbl, "unhappy (state)", "apply_condition_to_combatant", cond(deadE, "a", combat.Blinded{}), r, dead, nil},
		{"condition unknown", tbl, "unhappy (invalid input)", "apply_condition_to_combatant", cond(healthy, "ghost", combat.Blinded{}), i, unk, nil},
		{"remove applied", tbl, "happy", "remove_condition_from_combatant", remove(blindE, "a", content.ConditionBlinded), a, "applied", []string{"condition_removed"}},
		{"remove not held", tbl, "unhappy (state)", "remove_condition_from_combatant", remove(healthy, "a", content.ConditionBlinded), r, "does_not_have_condition", nil},
		{"remove implied", tbl, "unhappy (state)", "remove_condition_from_combatant", remove(uncE, "a", content.ConditionProne), r, "condition_implied_by", nil},
		{"remove unknown", tbl, "unhappy (invalid input)", "remove_condition_from_combatant", remove(healthy, "ghost", content.ConditionBlinded), i, unk, nil},
		{"reduce applied", tbl, "happy", "reduce_combatant_exhaustion", reduce(exhE, "a", 1), a, "applied", []string{"condition_level_reduced"}},
		{"reduce none", tbl, "unhappy (state)", "reduce_combatant_exhaustion", reduce(healthy, "a", 1), r, "does_not_have_condition", nil},
		{"reduce dead", tbl, "unhappy (state)", "reduce_combatant_exhaustion", reduce(deadE, "a", 1), r, dead, nil},
		{"reduce unknown", tbl, "unhappy (invalid input)", "reduce_combatant_exhaustion", reduce(healthy, "ghost", 1), i, unk, nil},
		{"save applied", tbl, "happy", "apply_death_saving_throw_to_combatant", save(dyingE, "a", 12), a, "applied", []string{"death_saving_throw_made"}},
		{"save not dying", tbl, "unhappy (state)", "apply_death_saving_throw_to_combatant", save(healthy, "a", 12), r, "not_dying", nil},
		{"save twice", tbl, "unhappy (state)", "apply_death_saving_throw_to_combatant", save(madeE, "a", 12), r, "death_saving_throw_already_made", nil},
		{"save dead", tbl, "unhappy (state)", "apply_death_saving_throw_to_combatant", save(deadE, "a", 12), r, dead, nil},
		{"save unknown", tbl, "unhappy (invalid input)", "apply_death_saving_throw_to_combatant", save(healthy, "ghost", 12), i, unk, nil},
		{"stabilize applied", tbl, "happy", "stabilize_combatant", stab(dyingE, "a"), a, "applied", []string{"became_stable"}},
		{"stabilize not dying", tbl, "unhappy (state)", "stabilize_combatant", stab(healthy, "a"), r, "not_dying", nil},
		{"stabilize unknown", tbl, "unhappy (invalid input)", "stabilize_combatant", stab(healthy, "ghost"), i, unk, nil},
		{"recover applied", tbl, "happy", "recover_stable_combatant", recov(stable, "a"), a, "applied", []string{"hit_points_restored", "recovered", "condition_removed"}},
		{"recover not stable", tbl, "unhappy (state)", "recover_stable_combatant", recov(dyingE, "a"), r, "not_stable", nil},
		{"recover unknown", tbl, "unhappy (invalid input)", "recover_stable_combatant", recov(healthy, "ghost"), i, unk, nil},
		{"temporary applied", tbl, "happy", "give_combatant_temporary_hit_points", temp(healthy, "a"), a, "applied", []string{"temporary_hit_points_gained"}},
		{"temporary dead", tbl, "unhappy (state)", "give_combatant_temporary_hit_points", temp(deadE, "a"), r, dead, nil},
		{"temporary unknown", tbl, "unhappy (invalid input)", "give_combatant_temporary_hit_points", temp(healthy, "ghost"), i, unk, nil},
		{"maximum applied", tbl, "happy", "reduce_combatant_hit_point_maximum", maxHP(healthy, "a", 5), a, "applied", []string{"hit_point_maximum_reduced"}},
		{"maximum dead", tbl, "unhappy (state)", "reduce_combatant_hit_point_maximum", maxHP(deadE, "a", 5), r, dead, nil},
		{"maximum unknown", tbl, "unhappy (invalid input)", "reduce_combatant_hit_point_maximum", maxHP(healthy, "ghost", 5), i, unk, nil},
		{"end applied", tbl, "happy", "end_encounter", end(healthy), a, "applied", []string{"encounter_ended"}},
		{"end ended", tbl, "unhappy (state)", "end_encounter", end(endedE), r, ended, nil},
		{"check applied", tbl, "happy", "check_combatant_grapple", check(grap, "b", 9, 5), a, "applied", []string{"condition_removed", "grapple_ended"}},
		{"check holds", tbl, "unhappy (boundary)", "check_combatant_grapple", check(grap, "b", 5, 5), a, "applied", nil},
		{"check not grappled", tbl, "unhappy (state)", "check_combatant_grapple", check(healthy, "b", 5, 5), r, "does_not_have_condition", nil},
		{"check unknown grappler", tbl, "unhappy (invalid input)", "check_combatant_grapple", check(ghostGrapple(t), "b", 5, 5), i, unk, nil},
		{"check dead", tbl, "unhappy (state)", "check_combatant_grapple", check(deadGrap, "b", 5, 5), r, dead, nil},
		{"release applied", tbl, "happy", "release_combatant_grapple", release(grap, "b"), a, "applied", []string{"condition_removed", "grapple_ended"}},
		{"release not grappled", tbl, "unhappy (state)", "release_combatant_grapple", release(healthy, "b"), r, "does_not_have_condition", nil},
		{"release unknown", tbl, "unhappy (invalid input)", "release_combatant_grapple", release(grap, "ghost"), i, unk, nil},
		{"escape applied", tbl, "happy", "escape_combatant_grapple", escape(grap, "b", d20.Success), a, "applied", []string{"condition_removed", "grapple_ended"}},
		{"escape failed", tbl, "unhappy (rule interaction)", "escape_combatant_grapple", escape(grap, "b", d20.Failure), a, "applied", nil},
		{"escape not grappled", tbl, "unhappy (state)", "escape_combatant_grapple", escape(healthy, "b", d20.Success), r, "does_not_have_condition", nil},
		{"escape unknown", tbl, "unhappy (invalid input)", "escape_combatant_grapple", escape(grap, "ghost", d20.Success), i, unk, nil},
	}
}

func ghostGrapple(tb testing.TB) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encPC(tb, "a", combat.SideA, 20, 20), encC(tb, "b", combat.SideB, pcStatus(tb, 20, 20, 0), combat.Grappled{Grappler: cid("ghost")}))
	return e
}

type encRow struct {
	name, requirement, path string
	operation               string
	run                     encUC
	class, result           string
	events                  []string
}

// encHygiene asserts no log, no misuse, no id as an attribute value and no d20 signal.
func encHygiene(t *testing.T, rec *telemetrytest.Recorder) {
	t.Helper()
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
	if got := rec.Logger.Records(); len(got) != 0 {
		t.Errorf("logged %v, want nothing [FR-034]", got)
	}
	for _, m := range rec.Meter.Measurements() {
		if strings.HasPrefix(m.Name, "srd5e.d20_test.") || m.Name == "srd5e.combat.errors" {
			t.Errorf("recorded %s: the encounter calls no port and rolls no d20 [FR-061, P12]", m.Name)
		}
		for _, a := range m.Attrs {
			if s, ok := a.Value().(string); ok && slices.Contains([]string{"a", "b", "c", "ghost", "enc-1", "zzz"}, s) {
				t.Errorf("%s attribute %s carries the id %q [FR-057]", m.Name, a.Key(), s)
			}
		}
	}
}

// Requirement: per case
// Path: per case.
func TestEncounterUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range encRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("the recording bundle and Noop disagree [FR-056]")
			}
			outcome := "success"
			if c.class != "applied" {
				outcome = "failure"
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			wantErr := c.class == "invalid"
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != wantErr || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status %v, no recorded error [%s]", spans, wantErr, c.requirement)
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
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, encSorted(c.events)) {
				t.Errorf("event kinds counted = %v, want %v [FR-060]", kinds, encSorted(c.events))
			}
			encHygiene(t, rec)
		})
	}
}

func encSorted(s []string) []string {
	out := slices.Clone(s)
	if out == nil {
		out = []string{}
	}
	slices.Sort(out)
	return out
}

// Requirement: doc/specs/encounter-aggregate.md FR-095, table 4.10.1 - the operation names of the sixteen use cases are the table's and the result sets are closed: every operation of the table is exercised by a row above, and no operation repeats one of the hit point, conditions or initiative specs.
// Path: unhappy (rule interaction).
func TestEncounterOperationNamesAreTheTables(t *testing.T) {
	t.Parallel()
	want := strings.Fields("start_encounter advance_encounter_turn apply_damage_to_combatant apply_healing_to_combatant apply_condition_to_combatant remove_condition_from_combatant reduce_combatant_exhaustion apply_death_saving_throw_to_combatant stabilize_combatant recover_stable_combatant give_combatant_temporary_hit_points reduce_combatant_hit_point_maximum end_encounter check_combatant_grapple release_combatant_grapple escape_combatant_grapple")
	seen := map[string]bool{}
	for _, r := range encRows(t) {
		seen[r.operation] = true
	}
	for _, op := range want {
		if !seen[op] {
			t.Errorf("no row for the operation %s", op)
		}
	}
	if len(seen) != len(want) {
		t.Errorf("rows name %d operations, the table has %d", len(seen), len(want))
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-062 - StartEncounter records srd5e.combat.encounter_size on a valid start and not when invalid.
// Path: unhappy (boundary).
func TestStartEncounterRecordsTheSizeOnlyWhenValid(t *testing.T) {
	t.Parallel()
	two := []combat.Combatant{encPC(t, "a", combat.SideA, 20, 20), encPC(t, "b", combat.SideB, 20, 20)}
	est := encEstablished(t, "a", "b")
	rec, opts := recording()
	if _, err := ucStart(est, two...)(opts...); err != nil {
		t.Fatal(err)
	}
	if m := rec.Meter.MeasurementsNamed("srd5e.combat.encounter_size"); len(m) != 1 || m[0].Value != 2 || m[0].Unit != "{combatant}" || len(m[0].Attrs) != 0 {
		t.Errorf("encounter_size = %+v, want one sample of 2 with no attributes", m)
	}
	rec2, opts2 := recording()
	if _, err := ucStart(est, two[0], two[0], two[1])(opts2...); err != nil {
		t.Fatal(err)
	}
	if n := len(rec2.Meter.MeasurementsNamed("srd5e.combat.encounter_size")); n != 0 {
		t.Errorf("an invalid start recorded the size %d times [FR-062]", n)
	}
	res, _ := ucStart(est, two[0], two[0], two[1])()
	inv := res.(combat.StartEncounterInvalid)
	if !errors.Is(inv.Err(), srderr.ErrInvalidInput) || !slices.Equal(combat.ProblemsOf(inv.Err()), inv.Problems()) || len(inv.Problems()) == 0 {
		t.Errorf("StartEncounterInvalid carries %v", inv.Err())
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-081 - each wrapped Died increments srd5e.combat.deaths once with its cause, for each of the five causes, and a rejection counts none.
// Path: happy.
func TestDeathsAreCountedOncePerWrappedDied(t *testing.T) {
	t.Parallel()
	monsterOne, _ := encStart(t, encC(t, "a", combat.SideA, status(t, 5, 5, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())), encPC(t, "b", combat.SideB, 20, 20))
	massive, _ := encStart(t, encPC(t, "a", combat.SideA, 6, 12), encPC(t, "b", combat.SideB, 20, 20))
	threeFails, _ := encStart(t, encC(t, "a", combat.SideA, dyingStatus(t, 20, 0, 2), combat.Unconscious{}), encPC(t, "b", combat.SideB, 20, 20))
	exhaust, _ := encStart(t, encC(t, "a", combat.SideA, pcStatus(t, 9, 9, 0), exh(5)), encPC(t, "b", combat.SideB, 20, 20))
	maxZero, _ := encStart(t, encPC(t, "a", combat.SideA, 5, 5), encPC(t, "b", combat.SideB, 20, 20))
	rows := []struct {
		cause string
		run   encUC
	}{
		{"monster_death", ucDamage(monsterOne, "a", 5, combat.OtherSource)},
		{"massive_damage", ucDamage(massive, "a", 18, combat.OtherSource)},
		{"death_saving_throw_failures", func(o ...combat.Option) (any, error) {
			cmd, _ := combat.NewApplyDeathSavingThrowToCombatantCommand(threeFails, cid("a"), 1, 1)
			return combat.NewApplyDeathSavingThrowToCombatant(o...).Run(context.Background(), cmd)
		}},
		{"exhaustion", func(o ...combat.Option) (any, error) {
			cmd, _ := combat.NewApplyConditionToCombatantCommand(exhaust, cid("a"), exh(1))
			return combat.NewApplyConditionToCombatant(o...).Run(context.Background(), cmd)
		}},
		{"hit_point_maximum_zero", func(o ...combat.Option) (any, error) {
			cmd, _ := combat.NewReduceCombatantHitPointMaximumCommand(maxZero, cid("a"), 5)
			return combat.NewReduceCombatantHitPointMaximum(o...).Run(context.Background(), cmd)
		}},
	}
	for _, r := range rows {
		rec, opts := recording()
		if _, err := r.run(opts...); err != nil {
			t.Fatalf("%s: %v", r.cause, err)
		}
		got := measured(rec, "srd5e.combat.deaths")
		if len(got) != 1 || got[0]["srd5e.combat.death_cause"] != r.cause {
			t.Errorf("deaths = %v, want one with cause %s [FR-081]", got, r.cause)
		}
		if ms := rec.Meter.MeasurementsNamed("srd5e.combat.deaths"); ms[0].Value != 1 {
			t.Errorf("death counter incremented by %v, want 1", ms[0].Value)
		}
	}
	rec, opts := recording()
	if _, err := ucDamage(monsterOne, "zzz", 5, combat.OtherSource)(opts...); err != nil {
		t.Fatal(err)
	}
	if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.deaths")); n != 0 {
		t.Errorf("a rejection counted %d deaths", n)
	}
}

// panicMeter is a Meter whose counters panic, for a backend that is the host's bug.
type panicMeter struct{ telemetry.NoopMeter }

func (panicMeter) Counter(telemetry.Descriptor) telemetry.Counter { panic(errors.New("backend bug")) }

// Requirement: doc/specs/encounter-aggregate.md FR-061 - a panic raised inside a telemetry backend propagates unchanged out of the use case, the span is ended, and srd5e.combat.errors is not incremented.
// Path: unhappy (failure).
func TestATelemetryPanicPropagatesAndTheSpanEnds(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	opts := []combat.Option{combat.WithTelemetry(telemetry.Telemetry{Tracer: rec.Tracer, Meter: panicMeter{}})}
	run := ucAdvance(encABC(t), combat.TurnAdvance{})
	func() {
		defer func() {
			if r := recover(); r == nil || r.(error).Error() != "backend bug" {
				t.Errorf("recovered %v, want the original panic", r)
			}
		}()
		_, _ = run(opts...)
	}()
	spans := rec.Tracer.SpansNamed("srd5e.combat.advance_encounter_turn")
	if len(spans) != 1 || !spans[0].Ended {
		t.Errorf("spans = %+v, want one ended span", spans)
	}
}
