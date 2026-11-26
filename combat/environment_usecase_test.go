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
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// envRow is one result of one hazard or object use case.
type envRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
	kind                    string  // srd5e.combat.hazard.kind of the own signal, "" for none
	amount                  float64 // the sample of damage_rolled, or 1 for exhaustion_gained
}

func envRows(t *testing.T) []envRow {
	t.Helper()
	ctx := context.Background()
	burning, med := combat.BurningStateBurning, content.SizeMedium
	roll := func(r func() combat.Roller, state combat.BurningState) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			uc, err := combat.NewRollBurningDamage(r(), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.RollBurningDamageCommand{State: state})
		}
	}
	extinguish := func(state combat.BurningState, m combat.ExtinguishMethod, set combat.ConditionSet) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewExtinguishBurning(opts...).Run(ctx, combat.ExtinguishBurningCommand{State: state, Method: m, Conditions: set, Target: subject()})
		}
	}
	fall := func(r func() combat.Roller, feet int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			uc, err := combat.NewResolveFall(r(), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.ResolveFallCommand{Distance: combat.MustFallDistance(feet), Surface: combat.FallIntoLiquid{Landing: combat.LandingCheckMade{Outcome: d20.Success}}})
		}
	}
	land := func(damage int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewLandAfterFall(opts...).Run(ctx, combat.LandAfterFallCommand{Target: subject(), DamageTaken: damage})
		}
	}
	dehydrate := func(drunk int, set combat.ConditionSet) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			w, _ := combat.NewWaterAmount(drunk)
			return combat.NewResolveDehydrationDayEnd(opts...).Run(ctx, combat.ResolveDehydrationDayEndCommand{Size: med, Water: w, Conditions: set, Target: subject(), Record: envRecord(t, 1, 0, 0, 0, 0)})
		}
	}
	starve := func(eaten int, save combat.FoodSavingThrow, set combat.ConditionSet) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			f, _ := combat.NewFoodAmount(eaten)
			return combat.NewResolveMalnutritionDayEnd(opts...).Run(ctx, combat.ResolveMalnutritionDayEndCommand{Size: med, Food: f, SavingThrow: save, Conditions: set, Target: subject()})
		}
	}
	breath := func(state combat.BreathState, change combat.BreathChange) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewAdvanceBreath(opts...).Run(ctx, combat.AdvanceBreathCommand{State: state, Change: change, Target: subject()})
		}
	}
	turnEnd := func(state combat.BreathState, set combat.ConditionSet) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewResolveSuffocationTurnEnd(opts...).Run(ctx, combat.ResolveSuffocationTurnEndCommand{State: state, Conditions: set, Target: subject()})
		}
	}
	restore := func(set combat.ConditionSet, rec combat.HazardRecord) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewRestoreBreathing(opts...).Run(ctx, combat.RestoreBreathingCommand{State: mustSuffocatingPtr(t), Conditions: set, Target: subject(), Record: rec})
		}
	}
	remove := func(set combat.ConditionSet, rec combat.HazardRecord) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewRemoveExhaustionLevels(opts...).Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: set, Target: subject(), Record: rec, Levels: 1})
		}
	}
	rollHP := func(size content.CreatureSize) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			uc, err := combat.NewRollObjectHitPoints(dicetest.NewMaximumRoller(), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.RollObjectHitPointsCommand{Size: size, Durability: combat.ObjectFragile})
		}
	}
	hurt := func(s combat.ObjectStatus, harm combat.ObjectHarm) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewApplyDamageToObject(opts...).Run(ctx, combat.ApplyDamageToObjectCommand{Status: s, Harm: harm})
		}
	}
	std := (combat.ObjectAffinities{}).Standard()
	const table = "environment-hazards-and-objects.md section 7 operations and results"
	const a, r = "applied", "refused"
	return []envRow{
		{"roll_burning_damage damage_rolled", table + "; FR-099, FR-119", "happy", "roll_burning_damage", roll(func() combat.Roller { return dicetest.NewFixedRoller(3) }, burning), a, "damage_rolled", nil, "burning", 3},
		{"roll_burning_damage not_burning", table + "; FR-100", "unhappy (state error)", "roll_burning_damage", roll(func() combat.Roller { return dicetest.NewFixedRoller(3) }, combat.BurningStateNotBurning), r, "not_burning", nil, "", 0},
		{"extinguish_burning extinguished", table + "; FR-106", "happy", "extinguish_burning", extinguish(burning, combat.ExtinguishMethodRollOnTheGround, combat.ConditionSet{}), a, "extinguished", []string{"burning_ended", "condition_applied"}, "", 0},
		{"extinguish_burning not_burning", table, "unhappy (state error)", "extinguish_burning", extinguish(combat.BurningStateNotBurning, combat.ExtinguishMethodDoused, combat.ConditionSet{}), r, "not_burning", nil, "", 0},
		{"extinguish_burning blocked_by_immunity", table, "unhappy (rule interaction)", "extinguish_burning", extinguish(burning, combat.ExtinguishMethodRollOnTheGround, condSet(t, nil, content.ConditionProne)), r, "blocked_by_immunity", nil, "", 0},
		{"resolve_fall damage_rolled", table + "; FR-119", "happy", "resolve_fall", fall(func() combat.Roller { return dicetest.NewMaximumRoller() }, 30), a, "damage_rolled", nil, "falling", 18},
		{"resolve_fall no_damage", table + "; FR-100", "unhappy (boundary)", "resolve_fall", fall(func() combat.Roller { return dicetest.NewMaximumRoller() }, 9), a, "no_damage", nil, "", 0},
		{"land_after_fall landed_prone", table, "happy", "land_after_fall", land(4), a, "landed_prone", []string{"condition_applied"}, "", 0},
		{"land_after_fall landed_standing", table, "unhappy (boundary)", "land_after_fall", land(0), a, "landed_standing", nil, "", 0},
		{"resolve_dehydration_day_end exhaustion_gained", table + "; FR-119", "happy", "resolve_dehydration_day_end", dehydrate(0, envExhaustion(t, 1)), a, "exhaustion_gained", []string{"condition_applied"}, "dehydration", 1},
		{"resolve_dehydration_day_end no_exhaustion", table, "unhappy (boundary)", "resolve_dehydration_day_end", dehydrate(8, envExhaustion(t, 1)), a, "no_exhaustion", []string{"dehydration_recovered"}, "", 0},
		{"resolve_dehydration_day_end exhaustion_at_maximum", table, "unhappy (rule interaction)", "resolve_dehydration_day_end", dehydrate(0, envExhaustion(t, 6)), a, "exhaustion_at_maximum", nil, "", 0},
		{"resolve_dehydration_day_end immune", table, "unhappy (rule interaction)", "resolve_dehydration_day_end", dehydrate(0, condSet(t, nil, content.ConditionExhaustion)), a, "immune", nil, "", 0},
		{"resolve_malnutrition_day_end exhaustion_gained", table + "; FR-119", "happy", "resolve_malnutrition_day_end", starve(3, combat.FoodSaveRolled{Outcome: d20.Failure}, combat.ConditionSet{}), a, "exhaustion_gained", []string{"condition_applied"}, "malnutrition", 1},
		{"resolve_malnutrition_day_end save_required", table + "; FR-119, Q-17", "unhappy (state error)", "resolve_malnutrition_day_end", starve(3, nil, combat.ConditionSet{}), a, "save_required", nil, "", 0},
		{"resolve_malnutrition_day_end no_exhaustion", table, "unhappy (rule interaction)", "resolve_malnutrition_day_end", starve(3, combat.FoodSaveRolled{Outcome: d20.Success}, combat.ConditionSet{}), a, "no_exhaustion", nil, "", 0},
		{"resolve_malnutrition_day_end exhaustion_at_maximum", table, "unhappy (rule interaction)", "resolve_malnutrition_day_end", starve(3, combat.FoodSaveRolled{Outcome: d20.Failure}, envExhaustion(t, 6)), a, "exhaustion_at_maximum", nil, "", 0},
		{"resolve_malnutrition_day_end immune", table, "unhappy (rule interaction)", "resolve_malnutrition_day_end", starve(3, combat.FoodSaveRolled{Outcome: d20.Failure}, condSet(t, nil, content.ConditionExhaustion)), a, "immune", nil, "", 0},
		{"advance_breath advanced", table, "happy", "advance_breath", breath(nil, combat.BeginChoking{}), a, "advanced", []string{"began_choking"}, "", 0},
		{"advance_breath not_breathing", table, "unhappy (state error)", "advance_breath", breath(mustSuffocatingPtr(t), combat.BeginHoldingBreath{}), r, "not_breathing", nil, "", 0},
		{"advance_breath not_holding_breath", table, "unhappy (state error)", "advance_breath", breath(nil, combat.HoldBreathFor{}), r, "not_holding_breath", nil, "", 0},
		{"advance_breath already_suffocating", table, "unhappy (state error)", "advance_breath", breath(mustSuffocatingPtr(t), combat.BeginChoking{}), r, "already_suffocating", nil, "", 0},
		{"resolve_suffocation_turn_end exhaustion_gained", table + "; FR-119", "happy", "resolve_suffocation_turn_end", turnEnd(mustSuffocatingPtr(t), combat.ConditionSet{}), a, "exhaustion_gained", []string{"condition_applied"}, "suffocation", 1},
		{"resolve_suffocation_turn_end no_exhaustion", table, "unhappy (state error)", "resolve_suffocation_turn_end", turnEnd(nil, combat.ConditionSet{}), a, "no_exhaustion", nil, "", 0},
		{"restore_breathing restored", table, "happy", "restore_breathing", restore(envExhaustion(t, 1), envRecord(t, 0, 0, 1, 0, 0)), a, "restored", []string{"condition_removed", "breath_restored"}, "", 0},
		{"remove_exhaustion_levels removed", table, "happy", "remove_exhaustion_levels", remove(envExhaustion(t, 2), combat.HazardRecord{}), a, "removed", []string{"condition_level_reduced"}, "", 0},
		{"remove_exhaustion_levels blocked", table + "; FR-062", "unhappy (rule interaction)", "remove_exhaustion_levels", remove(envExhaustion(t, 1), envRecord(t, 1, 0, 0, 0, 0)), r, "blocked", nil, "", 0},
		{"roll_object_hit_points rolled", table, "happy", "roll_object_hit_points", rollHP(content.SizeSmall), a, "rolled", nil, "", 0},
		{"roll_object_hit_points split_required", table + "; FR-071", "unhappy (boundary)", "roll_object_hit_points", rollHP(content.SizeHuge), r, "split_required", nil, "", 0},
		{"apply_damage_to_object damaged", table, "happy", "apply_damage_to_object", hurt(envObject(t, 30, 0, std), instance(t, slash(5))), a, "damaged", []string{"object_damaged"}, "", 0},
		{"apply_damage_to_object deflected", table + "; FR-081", "unhappy (boundary)", "apply_damage_to_object", hurt(envObject(t, 30, 10, std), instance(t, slash(9))), a, "deflected", []string{"object_damage_deflected"}, "", 0},
		{"apply_damage_to_object no_damage", table + "; FR-109", "unhappy (rule interaction)", "apply_damage_to_object", hurt(envObject(t, 30, 0, std), instance(t, combat.MustDamage(5, content.DamagePoison))), a, "no_damage", nil, "", 0},
		{"apply_damage_to_object destroyed", table + "; FR-075", "unhappy (boundary)", "apply_damage_to_object", hurt(envObject(t, 30, 0, std), instance(t, slash(30))), a, "destroyed", []string{"object_damaged", "object_destroyed"}, "", 0},
		{"apply_damage_to_object broken", table + "; FR-086", "unhappy (rule interaction)", "apply_damage_to_object", hurt(envObject(t, 30, 0, std), combat.ObjectBrokenAutomatically{Basis: combat.AutomaticBreakGMAllowed}), a, "destroyed", []string{"object_destroyed"}, "", 0},
		{"apply_damage_to_object already_destroyed", table + "; FR-077", "unhappy (state error)", "apply_damage_to_object", hurt(envObject(t, 0, 0, std), instance(t, slash(5))), r, "already_destroyed", nil, "", 0},
	}
}

// Requirement: per case
// Path: per case.
func TestEnvironmentUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range envRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-102]", got, plain, err)
			}
			outcome := map[string]string{"applied": "success", "refused": "failure"}[c.class]
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status [%s]", spans, c.requirement)
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
			counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event")
			wantKinds := slices.Clone(c.events)
			slices.Sort(wantKinds)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if !slices.Equal(counted, wantKinds) {
				t.Errorf("event kinds counted = %v, want %v [FR-099]", counted, wantKinds)
			}
			damage := rec.Meter.MeasurementsNamed("srd5e.combat.hazard.damage_rolled")
			gained := rec.Meter.MeasurementsNamed("srd5e.combat.hazard.exhaustion_gained")
			wantDamage, wantGained := c.kind == "burning" || c.kind == "falling", c.kind == "dehydration" || c.kind == "malnutrition" || c.kind == "suffocation"
			if wantDamage != (len(damage) == 1) || wantGained != (len(gained) == 1) || len(damage)+len(gained) > 1 {
				t.Fatalf("hazard signals = %v / %v for kind %q [FR-119]", damage, gained, c.kind)
			}
			for _, m := range append(damage, gained...) {
				if m.Value != c.amount || !reflect.DeepEqual(attrMap(m.Attrs), map[string]any{"srd5e.combat.hazard.kind": c.kind}) {
					t.Errorf("%s = %v %v, want %v for kind %s [FR-119]", m.Name, m.Value, attrMap(m.Attrs), c.amount, c.kind)
				}
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result", n)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-103 - no attribute of any hazard signal carries an id, an amount, a distance or free text: every value is in a closed set.
// Path: unhappy (invalid input).
func TestEnvironmentAttributesAreClosedValues(t *testing.T) {
	t.Parallel()
	closed := []string{"applied", "refused", "success", "failure", "burning", "falling", "dehydration", "malnutrition", "suffocation",
		"roll_burning_damage", "extinguish_burning", "resolve_fall", "land_after_fall", "resolve_dehydration_day_end", "resolve_malnutrition_day_end",
		"advance_breath", "resolve_suffocation_turn_end", "restore_breathing", "remove_exhaustion_levels", "roll_object_hit_points", "apply_damage_to_object",
		"damage_rolled", "not_burning", "extinguished", "blocked_by_immunity", "no_damage", "landed_prone", "landed_standing", "exhaustion_gained", "no_exhaustion",
		"exhaustion_at_maximum", "immune", "save_required", "advanced", "not_breathing", "not_holding_breath", "already_suffocating", "restored", "removed", "blocked",
		"rolled", "split_required", "damaged", "deflected", "destroyed", "already_destroyed",
		"burning_ended", "condition_applied", "dehydration_recovered", "began_choking", "condition_removed", "breath_restored", "condition_level_reduced",
		"object_damaged", "object_destroyed", "object_damage_deflected"}
	for _, c := range envRows(t) {
		rec, opts := recording()
		if _, err := c.run(opts...); err != nil {
			t.Fatal(err)
		}
		for _, m := range rec.Meter.Measurements() {
			for _, a := range m.Attrs {
				if s, ok := a.Value().(string); !ok || !slices.Contains(closed, s) {
					t.Errorf("%s: %s attribute %s = %v is not a closed value [FR-103]", c.name, m.Name, a.Key(), a.Value())
				}
				if !strings.HasPrefix(a.Key(), "srd5e.") && a.Key() != telemetry.AttrOutcome {
					t.Errorf("%s: attribute key %s is not a srd5e signal key", c.name, a.Key())
				}
			}
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-097, FR-102 - a use case called twice with equal commands returns equal results, and every use case works silently with no options.
// Path: unhappy (determinism).
func TestEnvironmentUseCasesAreDeterministic(t *testing.T) {
	t.Parallel()
	for _, c := range envRows(t) {
		first, err1 := c.run()
		second, err2 := c.run()
		if err1 != nil || err2 != nil || !reflect.DeepEqual(first, second) {
			t.Errorf("%s: two calls differ or failed: %v / %v", c.name, err1, err2)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-101 - a roller failure of each dice use case is recorded once: the span has error status and one recorded error, srd5e.combat.errors counts it with the error type, and operations, events and the hazard signals do not.
// Path: unhappy (telemetry on failure).
func TestEnvironmentPortFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	ctx := context.Background()
	calls := map[string]func(combat.Roller, ...combat.Option) error{
		"roll_burning_damage": func(r combat.Roller, o ...combat.Option) error {
			uc, _ := combat.NewRollBurningDamage(r, o...)
			_, err := uc.Run(ctx, combat.RollBurningDamageCommand{State: combat.BurningStateBurning})
			return err
		},
		"resolve_fall": func(r combat.Roller, o ...combat.Option) error {
			uc, _ := combat.NewResolveFall(r, o...)
			_, err := uc.Run(ctx, combat.ResolveFallCommand{Distance: combat.MustFallDistance(40)})
			return err
		},
		"roll_object_hit_points": func(r combat.Roller, o ...combat.Option) error {
			uc, _ := combat.NewRollObjectHitPoints(r, o...)
			_, err := uc.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeSmall, Durability: combat.ObjectResilient})
			return err
		},
	}
	for op, call := range calls {
		for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
			rec, opts := recording()
			err := call(roller, opts...)
			envPortFailure(t, err)
			spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("%s/%s: spans = %+v, want one ended span with one recorded error", op, name, spans)
			}
			errs := measured(rec, "srd5e.combat.errors")
			if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["srd5e.combat.operation"] != op {
				t.Errorf("%s/%s: errors = %v, want one with error.type", op, name, errs)
			}
			for _, n := range []string{"operations", "events", "hazard.damage_rolled", "hazard.exhaustion_gained"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
					t.Errorf("%s/%s: %s recorded %d on an error", op, name, n, len(m))
				}
			}
			checkTelemetryHygiene(t, rec)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-100 - an ErrInvalidInput error from inside a call (a negative landing damage, negative elapsed time, levels below 1, a nil harm) is recorded on the span but is not a port failure: no operations, events or errors sample, and it is not logged.
// Path: unhappy (telemetry on failure).
func TestEnvironmentPreconditionErrorsAreSpanOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	calls := map[string]func(...combat.Option) error{
		"land_after_fall": func(o ...combat.Option) error {
			_, err := combat.NewLandAfterFall(o...).Run(ctx, combat.LandAfterFallCommand{Target: subject(), DamageTaken: -1})
			return err
		},
		"advance_breath": func(o ...combat.Option) error {
			_, err := combat.NewAdvanceBreath(o...).Run(ctx, combat.AdvanceBreathCommand{State: holding(t, 0, sec(60)), Change: combat.HoldBreathFor{Elapsed: -1}, Target: subject()})
			return err
		},
		"remove_exhaustion_levels": func(o ...combat.Option) error {
			_, err := combat.NewRemoveExhaustionLevels(o...).Run(ctx, combat.RemoveExhaustionLevelsCommand{Target: subject()})
			return err
		},
		"apply_damage_to_object": func(o ...combat.Option) error {
			_, err := combat.NewApplyDamageToObject(o...).Run(ctx, combat.ApplyDamageToObjectCommand{Status: envObject(t, 5, 0, combat.NoAffinities())})
			return err
		},
		"roll_object_hit_points": func(o ...combat.Option) error {
			uc, _ := combat.NewRollObjectHitPoints(dicetest.NewMinimumRoller(), o...)
			_, err := uc.Run(ctx, combat.RollObjectHitPointsCommand{})
			return err
		},
	}
	for op, call := range calls {
		rec, opts := recording()
		if err := call(opts...); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("%s = %v, want ErrInvalidInput", op, err)
		}
		spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
			t.Errorf("%s: spans = %+v, want one span with error status", op, spans)
		}
		for _, n := range []string{"operations", "events"} {
			if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
				t.Errorf("%s: %s recorded %d for a precondition error", op, n, len(m))
			}
		}
		assertInvalidInputCounted(t, rec, op)
		checkTelemetryHygiene(t, rec)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-096, FR-098 - a use case does not alter its command, and the Exhaustion hazards give the same set and events as ConditionSet.Apply and ReduceExhaustion called directly.
// Path: unhappy (immutability).
func TestHazardsMatchTheConditionOperations(t *testing.T) {
	t.Parallel()
	set := envExhaustion(t, 2)
	direct := set.Apply(subject(), condLevel(1))
	day := dehydrationDay(t, envMedium(), 0, set, combat.HazardRecord{})
	if day.Conditions() != direct.Set() || len(day.Events()) != 1 || day.Events()[0].Kind() != direct.(combat.ConditionAdded).Events()[0].Kind() {
		t.Errorf("hazard = %v, direct Apply = %v", day.Conditions(), direct.Set())
	}
	reduced, _ := set.ReduceExhaustion(subject(), 1)
	got, _ := removeLevels(t, set, combat.HazardRecord{}, 1)
	if got.(combat.ExhaustionLevelsRemoved).Conditions() != reduced.Set() || !reflect.DeepEqual(got.(combat.ExhaustionLevelsRemoved).Reduction(), reduced) {
		t.Error("RemoveExhaustionLevels differs from ReduceExhaustion")
	}
	if envLevel(set) != 2 {
		t.Error("the input set changed")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-099 - every hazard and object use case works as its zero value where it has no port, silently, and the types with a port refuse the zero value with an error.
// Path: unhappy (empty and minimal).
func TestEnvironmentZeroUseCases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var ex combat.ExtinguishBurning
	if res, err := ex.Run(ctx, combat.ExtinguishBurningCommand{State: combat.BurningStateBurning, Method: combat.ExtinguishMethodDoused, Target: subject()}); err != nil || res == nil {
		t.Errorf("zero ExtinguishBurning = %v, %v", res, err)
	}
	var ab combat.AdvanceBreath
	if res, err := ab.Run(ctx, combat.AdvanceBreathCommand{Change: combat.BeginChoking{}, Target: subject()}); err != nil || res == nil {
		t.Errorf("zero AdvanceBreath = %v, %v", res, err)
	}
}

// mustSuffocatingPtr is a choking creature as a BreathState.
func mustSuffocatingPtr(t *testing.T) combat.BreathState {
	t.Helper()
	return mustSuffocating(t, combat.SuffocationChoking)
}
