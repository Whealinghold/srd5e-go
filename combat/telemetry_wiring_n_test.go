package combat_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/rules/testing.md section 4 (cancellation); doc/specs/environment-hazards-and-objects.md FR-096 and doc/specs/mounted-and-underwater-combat.md FR-062 - an already-cancelled context is returned unwrapped by each hazard, object and mounting use case before any span, metric, die or log.
// Path: per case.
func TestEnvironmentAndMountingUseCasesHonourACancelledContext(t *testing.T) {
	t.Parallel()
	med := content.SizeMedium
	water, _ := combat.NewWaterAmount(0)
	food, _ := combat.NewFoodAmount(3)
	mountCmd := mntCommand(t, mntInput(t))
	fallOffCmd, err := combat.NewResolveFallingOffCommand(gid("rider"), mntMounted(t, combat.MountControlControlled),
		combat.FallingOffTriggerRiderKnockedProne, rollWith(0), combat.LandingSpaceUnoccupiedWithin5Feet)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, op string
		run      func(ctx context.Context, roller combat.Roller, opts ...combat.Option) (any, error)
	}{
		{"mount_creature", "mount_creature", func(ctx context.Context, _ combat.Roller, o ...combat.Option) (any, error) {
			return combat.NewMountCreature(o...).Run(ctx, mountCmd)
		}},
		{"resolve_falling_off", "resolve_falling_off", func(ctx context.Context, r combat.Roller, o ...combat.Option) (any, error) {
			uc, err := combat.NewResolveFallingOff(r, o...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, fallOffCmd)
		}},
		{"remove_exhaustion_levels", "remove_exhaustion_levels", func(ctx context.Context, _ combat.Roller, o ...combat.Option) (any, error) {
			return combat.NewRemoveExhaustionLevels(o...).Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: envExhaustion(t, 2), Target: subject(), Levels: 1})
		}},
		{"resolve_dehydration_day_end", "resolve_dehydration_day_end", func(ctx context.Context, _ combat.Roller, o ...combat.Option) (any, error) {
			return combat.NewResolveDehydrationDayEnd(o...).Run(ctx, combat.ResolveDehydrationDayEndCommand{Size: med, Water: water, Conditions: envExhaustion(t, 1), Target: subject(), Record: envRecord(t, 1, 0, 0, 0, 0)})
		}},
		{"resolve_malnutrition_day_end", "resolve_malnutrition_day_end", func(ctx context.Context, _ combat.Roller, o ...combat.Option) (any, error) {
			return combat.NewResolveMalnutritionDayEnd(o...).Run(ctx, combat.ResolveMalnutritionDayEndCommand{Size: med, Food: food, SavingThrow: combat.FoodSaveRolled{Outcome: d20.Failure}, Conditions: combat.ConditionSet{}, Target: subject()})
		}},
		{"resolve_suffocation_turn_end", "resolve_suffocation_turn_end", func(ctx context.Context, _ combat.Roller, o ...combat.Option) (any, error) {
			return combat.NewResolveSuffocationTurnEnd(o...).Run(ctx, combat.ResolveSuffocationTurnEndCommand{State: mustSuffocatingPtr(t), Conditions: combat.ConditionSet{}, Target: subject()})
		}},
		{"restore_breathing", "restore_breathing", func(ctx context.Context, _ combat.Roller, o ...combat.Option) (any, error) {
			return combat.NewRestoreBreathing(o...).Run(ctx, combat.RestoreBreathingCommand{State: mustSuffocatingPtr(t), Conditions: envExhaustion(t, 1), Target: subject(), Record: envRecord(t, 0, 0, 1, 0, 0)})
		}},
		{"resolve_fall", "resolve_fall", func(ctx context.Context, r combat.Roller, o ...combat.Option) (any, error) {
			uc, err := combat.NewResolveFall(r, o...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.ResolveFallCommand{Distance: combat.MustFallDistance(30), Surface: combat.FallIntoLiquid{Landing: combat.LandingCheckMade{Outcome: d20.Success}}})
		}},
		{"roll_burning_damage", "roll_burning_damage", func(ctx context.Context, r combat.Roller, o ...combat.Option) (any, error) {
			uc, err := combat.NewRollBurningDamage(r, o...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.RollBurningDamageCommand{State: combat.BurningStateBurning})
		}},
		{"roll_object_hit_points", "roll_object_hit_points", func(ctx context.Context, r combat.Roller, o ...combat.Option) (any, error) {
			uc, err := combat.NewRollObjectHitPoints(r, o...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeSmall, Durability: combat.ObjectFragile})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			roller := dicetest.NewFixedRoller(3)
			rec, opts := recording()
			res, err := c.run(ctx, roller, opts...)
			if res != nil && !isNilResult(res) || !errors.Is(err, context.Canceled) {
				t.Errorf("a cancelled ctx gives %#v, %v, want no result and context.Canceled", res, err)
			}
			if err != context.Canceled { //nolint:errorlint // the contract is that ctx.Err() is returned unwrapped
				t.Errorf("error = %v, want ctx.Err() unwrapped", err)
			}
			if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()) + len(rec.Logger.Records()); n != 0 {
				t.Errorf("a cancelled call recorded %d signals, want none", n)
			}
			if got := len(roller.Requests()); got != 0 {
				t.Errorf("a cancelled call rolled %d dice, want none", got)
			}
			// Same use case, live context: the signal path works, so the silence above is the cancellation.
			rec2, opts2 := recording()
			if _, err := c.run(context.Background(), dicetest.NewFixedRoller(3, 3, 3), opts2...); err != nil {
				t.Fatalf("live run: %v", err)
			}
			if got := len(rec2.Tracer.SpansNamed("srd5e.combat." + c.op)); got != 1 {
				t.Errorf("live run recorded %d spans named srd5e.combat.%s, want 1", got, c.op)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-083, FR-104 and doc/rules/observability.md section 9 - ResolveStatBlockSave gives the same result with a recording bundle and with Noop, and records one ended span, one operations increment with its operation, result and class, and one duration sample.
// Path: happy.
func TestResolveStatBlockSaveRecordsItsSignals(t *testing.T) {
	t.Parallel()
	cmd, err := combat.NewResolveStatBlockSaveCommand(sbSaveParams(t, sbEncounter(t), nil, 10))
	if err != nil {
		t.Fatal(err)
	}
	run := func(opts ...combat.Option) (combat.ResolveStatBlockSaveResult, error) {
		uc, err := combat.NewResolveStatBlockSave(newFixed(5), opts...)
		if err != nil {
			t.Fatal(err)
		}
		return uc.Run(context.Background(), cmd)
	}
	rec, opts := recording()
	got, err := run(opts...)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if plain, err := run(); err != nil || !reflect.DeepEqual(got, plain) {
		t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v)", got, plain, err)
	}
	const op = "resolve_stat_block_save_effect"
	spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
	if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
		t.Fatalf("spans = %+v, want one ended span without error status", spans)
	}
	ops := measured(rec, "srd5e.combat.operations")
	if len(ops) != 1 || ops[0]["srd5e.combat.operation"] != op || ops[0]["srd5e.combat.result_class"] != "applied" || ops[0]["srd5e.outcome"] != "success" {
		t.Errorf("operations = %v, want one applied success of %s", ops, op)
	}
	if !reflect.DeepEqual(attrMap(spans[0].Attrs), ops[0]) {
		t.Errorf("span attrs = %v, want the operations attrs %v", attrMap(spans[0].Attrs), ops[0])
	}
	if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
		t.Errorf("duration = %+v, want one sample of 2 s", dur)
	}
	checkTelemetryHygiene(t, rec)
}
