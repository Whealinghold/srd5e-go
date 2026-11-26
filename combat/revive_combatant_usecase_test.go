package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

const rcSpec = "doc/specs/returning-to-life.md "

// rcCommand is the baseline of table 4.1 as a use case command: Dead with maximum 20, spirit accepts, Poisoned,
// Blinded and Exhaustion 2, the effects of table 4.2, Poisoned elapsed.
func rcCommand(tb testing.TB) combat.ReviveCombatantCommand {
	tb.Helper()
	c := rtlCommand(tb, 7)
	c.Conditions = meConditions(combat.Poisoned{}, combat.Blinded{}, combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(2)})
	c.ElapsedConditions = meKinds(content.ConditionPoisoned)
	return combat.ReviveCombatantCommand(c)
}

// Requirement: per case
// Path: per case.
func TestReviveCombatant(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		edit                    func(*combat.ReviveCombatantCommand)
		class, result           string
		events                  []string
	}{
		{"a Dead creature returns", rcSpec + "FR-003, FR-009, FR-011, FR-013, section 7", "happy", func(*combat.ReviveCombatantCommand) {},
			"applied", "completed", []string{"condition_removed", "condition_level_reduced", "creature_returned_to_life"}},
		{"an Alive creature", rcSpec + "FR-001, section 7", "unhappy (state error)", func(c *combat.ReviveCombatantCommand) { c.Status = pcStatus(t, 5, 20, 0) },
			"refused", "revive_not_dead", nil},
		{"the spirit refuses", rcSpec + "FR-002, section 7", "unhappy (rule interaction)", func(c *combat.ReviveCombatantCommand) { c.Consent = combat.SpiritRefuses },
			"refused", "revive_refused", nil},
		{"an unset consent", rcSpec + "FR-002, section 7", "unhappy (invalid input)", func(c *combat.ReviveCombatantCommand) { c.Consent = 0 },
			"refused", "revive_refused", nil},
		{"zero Hit Points", rcSpec + "FR-004, section 7", "unhappy (boundary)", func(c *combat.ReviveCombatantCommand) { c.HitPoints = 0 },
			"refused", "revive_hit_points_below_one", nil},
		{"a maximum below 1", rcSpec + "FR-006, section 7", "unhappy (boundary)", func(c *combat.ReviveCombatantCommand) {
			c.Status = status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities())
		}, "refused", "revive_maximum_below_one", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := rcCommand(t)
			c.edit(&cmd)
			rec, opts := recording()
			got, err := combat.NewReviveCombatant(opts...).Run(context.Background(), cmd)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if plain, err := combat.NewReviveCombatant().Run(context.Background(), cmd); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("the recording bundle and Noop disagree [%s]", c.requirement)
			}
			if want := combat.Revive(combat.ReviveCreatureCommand(cmd)); !reflect.DeepEqual(got, want) {
				t.Errorf("result = %#v, want Revive's %#v unchanged [%s]", got, want, c.requirement)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.revive_combatant")
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status [%s]", spans, c.requirement)
			}
			outcome := "success"
			if c.class != "applied" {
				outcome = "failure"
			}
			want := map[string]any{"srd5e.combat.operation": "revive_combatant", "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [%s]", sa, want, c.requirement)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s [%s]", dur, c.requirement)
			}
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, slices.Sorted(slices.Values(c.events))) {
				t.Errorf("event kinds counted = %v, want %v [%s]", kinds, c.events, c.requirement)
			}
			resHygiene(t, rec, false)
		})
	}
}

// Requirement: doc/specs/returning-to-life.md FR-016, section 7 and plan P10 (decided) - each list at its cap runs, and one more is an error
// matching srderr.ErrInvalidInput recorded on the span only, never a counted result or a port error.
// Path: per case.
func TestReviveCombatantCaps(t *testing.T) {
	t.Parallel()
	kinds := func(n int) []content.ConditionKind { return slices.Repeat(meKinds(content.ConditionPoisoned), n) }
	name := func(i int) string { return "e" + string(rune('a'+i/26)) + string(rune('a'+i%26)) }
	ids := func(n int) []combat.OngoingEffectID {
		var out []combat.OngoingEffectID
		for i := range n {
			out = append(out, meEID(name(i)))
		}
		return out
	}
	held := func(n int) combat.OngoingEffects {
		var es []combat.OngoingEffect
		for i := range n {
			es = append(es, meEffect(name(i), combat.OngoingEffectSpell, meSpell("bless")))
		}
		return meEffects(es...)
	}
	cases := []struct {
		name, path string
		edit       func(*combat.ReviveCombatantCommand)
		over       bool
	}{
		{"15 elapsed conditions", "unhappy (boundary)", func(c *combat.ReviveCombatantCommand) { c.ElapsedConditions = kinds(15) }, false},
		{"16 elapsed conditions", "unhappy (boundary)", func(c *combat.ReviveCombatantCommand) { c.ElapsedConditions = kinds(16) }, true},
		{"64 elapsed effects", "unhappy (boundary)", func(c *combat.ReviveCombatantCommand) { c.ElapsedEffects = ids(64) }, false},
		{"65 elapsed effects", "unhappy (boundary)", func(c *combat.ReviveCombatantCommand) { c.ElapsedEffects = ids(65) }, true},
		{"64 held effects (the largest list NewOngoingEffects builds)", "unhappy (boundary)", func(c *combat.ReviveCombatantCommand) { c.Effects = held(64) }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/returning-to-life.md FR-016, plan P10 (the cap of this row's list).
			cmd := rcCommand(t)
			c.edit(&cmd)
			rec, opts := recording()
			got, err := combat.NewReviveCombatant(opts...).Run(context.Background(), cmd)
			spans := rec.Tracer.SpansNamed("srd5e.combat.revive_combatant")
			if !c.over {
				if err != nil || got == nil || len(spans) != 1 || spans[0].Status == telemetry.StatusError {
					t.Errorf("Run = %v, %v, spans %+v, want a result at the cap", got, err, spans)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
				t.Fatalf("Run = %v, %v, want no result and ErrInvalidInput", got, err)
			}
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Errorf("spans = %+v, want one span with error status and the error recorded", spans)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")); n != 0 {
				t.Errorf("%d operations counted, want none for a returned invalid-input error", n)
			}
			assertInvalidInputCounted(t, rec, "revive_combatant")
		})
	}
}

// Requirement: doc/specs/returning-to-life.md FR-016 and doc/rules/testing.md section 4 (cancellation) - an already-cancelled context is returned unwrapped before any signal.
// Path: unhappy (state error).
func TestReviveCombatantCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec, opts := recording()
	if _, err := combat.NewReviveCombatant(opts...).Run(ctx, rcCommand(t)); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if len(rec.Meter.Measurements()) != 0 || len(rec.Tracer.SpansNamed("srd5e.combat.revive_combatant")) != 0 {
		t.Error("a cancelled call recorded a signal")
	}
}

// Requirement: doc/specs/returning-to-life.md FR-014, FR-016 and doc/rules/testing.md section 4 (immutability) - the use case alters none of its inputs.
// Path: unhappy (immutability).
func TestReviveCombatantLeavesItsInputsAlone(t *testing.T) {
	t.Parallel()
	cmd := rcCommand(t)
	before := rcCommand(t)
	cmd.ElapsedEffects = rtlIDs("c1")
	before.ElapsedEffects = rtlIDs("c1")
	if _, err := combat.NewReviveCombatant().Run(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cmd, before) {
		t.Errorf("the command changed: %#v, want %#v", cmd, before)
	}
}
