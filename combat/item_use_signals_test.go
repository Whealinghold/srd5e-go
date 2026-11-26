package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

type itemSignalRow struct {
	name, requirement, path string
	operation, item         string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
}

func itemSignalRows(t *testing.T) []itemSignalRow {
	t.Helper()
	spec := itemUseSpec
	thrown := func(change func(*combat.ResolveThrownItemInput), faces ...int) func(...combat.Option) (any, error) {
		in := thrownIn()
		change(&in)
		return func(opts ...combat.Option) (any, error) {
			return thrownRun(t, in, dicetest.NewFixedRoller(faces...), opts...)
		}
	}
	potion := func(st combat.HitPointStatus, use combat.PotionUse, faces ...int) func(...combat.Option) (any, error) {
		user := "medic"
		if _, drink := use.(combat.DrinkPotion); drink {
			user = "hero"
		}
		return func(opts ...combat.Option) (any, error) {
			return potionRun(t, user, "hero", st, use, dicetest.NewFixedRoller(faces...), opts...)
		}
	}
	fiend := combat.ThrownAtCreature{ID: gid("target"), Size: content.SizeMedium, Type: content.CreatureTypeFiend}
	const a, r = "applied", "refused"
	const th, po = "resolve_thrown_item", "use_potion_of_healing"
	used := []string{"thrown_item_used"}
	return []itemSignalRow{
		{"acid damaged", spec + "FR-076", "happy", th, "acid", thrown(func(*combat.ResolveThrownItemInput) {}, 1, 1, 1), a, "damaged", used},
		{"acid saved", spec + "FR-076", "unhappy (boundary)", th, "acid", thrown(func(*combat.ResolveThrownItemInput) {}, 13), a, "saved", used},
		{"alchemist's fire damaged_and_burning", spec + "FR-076", "happy", th, "alchemists_fire", thrown(func(in *combat.ResolveThrownItemInput) { in.Kind = combat.ThrownAlchemistsFire }, 1, 1), a, "damaged_and_burning", used},
		{"holy water damaged", spec + "FR-076", "happy", th, "holy_water", thrown(func(in *combat.ResolveThrownItemInput) { in.Kind, in.Target = combat.ThrownHolyWater, fiend }, 1, 1, 1), a, "damaged", used},
		{"holy water no_effect", spec + "FR-012 FR-076", "unhappy (rule interaction)", th, "holy_water", thrown(func(in *combat.ResolveThrownItemInput) { in.Kind = combat.ThrownHolyWater }), a, "no_effect", used},
		{"oil covered_in_oil", spec + "FR-076", "happy", th, "oil", thrown(func(in *combat.ResolveThrownItemInput) { in.Kind = combat.ThrownOil }, 1), a, "covered_in_oil", used},
		{"net restrained", spec + "FR-076", "happy", th, "net", thrown(func(in *combat.ResolveThrownItemInput) { in.Kind, in.DistanceFeet = combat.ThrownNet, 15 }, 1), a, "restrained", used},
		{"net no_effect", spec + "FR-014 FR-076", "unhappy (boundary)", th, "net", thrown(func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.DistanceFeet = combat.ThrownNet, 15
			in.Target = combat.ThrownAtCreature{ID: gid("target"), Size: content.SizeHuge, Type: content.CreatureTypeBeast}
		}), a, "no_effect", used},
		{"refused target_not_a_creature", spec + "FR-076", "unhappy (invalid input)", th, "net", thrown(func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.Target = combat.ThrownNet, combat.ThrownAtObject{}
		}), r, "target_not_a_creature", nil},
		{"refused total_cover", spec + "FR-076", "unhappy (state error)", th, "acid", thrown(func(in *combat.ResolveThrownItemInput) { in.Cover = combat.ThrownCoverTotal }), r, "total_cover", nil},
		{"refused beyond_range", spec + "FR-076", "unhappy (boundary)", th, "acid", thrown(func(in *combat.ResolveThrownItemInput) { in.DistanceFeet = 21 }), r, "beyond_range", nil},
		{"refused target_not_seen", spec + "FR-076", "unhappy (rule interaction)", th, "acid", thrown(func(in *combat.ResolveThrownItemInput) { in.Sight = combat.ThrownSightUnseen }), r, "target_not_seen", nil},
		{"potion healed", spec + "FR-076", "happy", po, "potion_of_healing", potion(pcStatus(t, 5, 20, 0), combat.DrinkPotion{}, 1, 1), a, "healed", []string{"hit_points_restored", "potion_of_healing_used"}},
		{"potion healed at the maximum", spec + "FR-044 FR-076", "unhappy (boundary)", po, "potion_of_healing", potion(pcStatus(t, 20, 20, 0), combat.DrinkPotion{}, 1, 1), a, "healed", []string{"potion_of_healing_used"}},
		{"potion refused not_within_five_feet", spec + "FR-076", "unhappy (boundary)", po, "potion_of_healing", potion(pcStatus(t, 5, 20, 0), combat.AdministerPotion{DistanceFeet: 6}), r, "not_within_five_feet", nil},
		{"potion refused patient_is_dead", spec + "FR-076", "unhappy (state error)", po, "potion_of_healing", potion(deadStatus(t, combat.MonsterDeath), combat.AdministerPotion{}), r, "patient_is_dead", nil},
	}
}

// Requirement: per case
// Path: per case.
func TestItemUseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range itemSignalRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if plain, err := c.run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-076]", got, plain, err)
			}
			outcome := map[string]string{"applied": "success", "refused": "failure"}[c.class]
			base := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			want := map[string]any{"srd5e.combat.item_use.item": c.item}
			for k, v := range base {
				want[k] = v
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status, even for a refusal [%s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [section 7]", sa, want)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [section 7]", ops, want)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 || len(dur[0].Attrs) != 1 {
				t.Errorf("duration = %+v, want one sample of 2 s with the operation only", dur)
			}
			wantKinds := slices.Clone(c.events)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(counted, sortedCopy(wantKinds)) {
				t.Errorf("event kinds counted = %v, want %v", counted, wantKinds)
			}
			for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.events") {
				if len(m.Attrs) != 1 {
					t.Errorf("an event sample carries the item attribute: %v", m.Attrs)
				}
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result", n)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-074, errors-and-results.md FR-031, FR-034, FR-051 - a roller that fails, panics or answers another expression is recorded once by each rolling use case: error status and one recorded error on the span, one srd5e.combat.errors sample with error.type dice_roller_error, and no operations or events sample; nothing is logged.
// Path: unhappy (failure of collaborator, telemetry on failure).
func TestItemUsePortFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	uses := map[string]func(combat.Roller, ...combat.Option) error{
		"resolve_thrown_item": func(rl combat.Roller, o ...combat.Option) error {
			_, err := thrownRun(t, thrownIn(), rl, o...)
			return err
		},
		"use_potion_of_healing": func(rl combat.Roller, o ...combat.Option) error {
			_, err := potionRun(t, "hero", "hero", pcStatus(t, 5, 20, 0), combat.DrinkPotion{}, rl, o...)
			return err
		},
	}
	for op, run := range uses {
		for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("roller down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
			rec, opts := recording()
			err := run(roller, opts...)
			envPortFailure(t, err)
			spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("%s %s: spans = %+v, want one ended span with one recorded error", op, name, spans)
			}
			errs := measured(rec, "srd5e.combat.errors")
			if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["error.type"] != "dice_roller_error" || errs[0]["srd5e.combat.operation"] != op {
				t.Errorf("%s %s: errors = %v, want one dice_roller_error", op, name, errs)
			}
			for _, n := range []string{"operations", "events"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
					t.Errorf("%s %s: %s recorded %d on an error", op, name, n, len(m))
				}
			}
			checkTelemetryHygiene(t, rec)
		}
	}
}

// Requirement: doc/specs/item-use-effects.md FR-076 - a call whose ctx is already cancelled, or whose command is the zero command, is answered before any signal: nothing is recorded.
// Path: unhappy (cancellation, invalid input).
func TestItemUseEarlyErrorsRecordNothing(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	thrown, _ := combat.NewResolveThrownItem(dicetest.NewFixedRoller(1), opts...)
	potion, _ := combat.NewUsePotionOfHealing(dicetest.NewFixedRoller(1, 1), opts...)
	if _, err := thrown.Run(ctx, combat.MustResolveThrownItemCommand(thrownIn())); !errors.Is(err, context.Canceled) {
		t.Errorf("thrown: %v", err)
	}
	if _, err := potion.Run(ctx, combat.MustUsePotionOfHealingCommand("a", "a", pcStatus(t, 1, 5, 0), combat.DrinkPotion{})); !errors.Is(err, context.Canceled) {
		t.Errorf("potion: %v", err)
	}
	if _, err := thrown.Run(context.Background(), combat.ResolveThrownItemCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("thrown zero command: %v", err)
	}
	if _, err := potion.Run(context.Background(), combat.UsePotionOfHealingCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("potion zero command: %v", err)
	}
	if len(rec.Meter.Measurements()) != 0 || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("recorded %d measurements and %d spans, want none", len(rec.Meter.Measurements()), len(rec.Tracer.Spans()))
	}
}

// Requirement: doc/specs/item-use-effects.md section 7 - the item attribute is the content id of the item, one of the six catalogued values, and never an id of a combatant; a Net's save that is forced fails without a d20 and is still counted as one operation.
// Path: unhappy (telemetry).
func TestItemUseItemAttributeValues(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, c := range itemSignalRows(t) {
		seen[c.item] = true
	}
	want := []string{"acid", "alchemists_fire", "holy_water", "oil", "net", "potion_of_healing"}
	for _, w := range want {
		if !seen[w] {
			t.Errorf("no signal row exercises the item %s", w)
		}
	}
	if len(seen) != len(want) {
		t.Errorf("rows exercise %v, want exactly %v", seen, want)
	}
	rec, opts := recording()
	in := thrownIn()
	in.Attempt = d20.AutomaticFailure{}
	if _, err := thrownRun(t, in, dicetest.NewFixedRoller(1, 1), opts...); err != nil {
		t.Fatal(err)
	}
	if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || ops[0]["srd5e.combat.item_use.item"] != "acid" {
		t.Errorf("operations = %v", ops)
	}
}
