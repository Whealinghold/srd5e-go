package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

type effectsRow struct {
	name, requirement, path string
	operation, class, name2 string
	reason                  string
	run                     func(opts ...combat.Option) (any, error)
}

func effectsRows(t *testing.T) []effectsRow {
	t.Helper()
	ctx := context.Background()
	hold := aeSeq(t, aeSpec{id: "hp", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"), template: aeHoldPerson(t, 14),
		conditions: []content.ConditionKind{content.ConditionParalyzed}})
	holdSpec := aeSpec{id: "hp", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"), template: aeHoldPerson(t, 14)}
	saves := aeSaves("orc", ability.Wisdom, aeRoll())
	advance := func(cmd combat.AdvanceActiveEffectsCommand, faces ...int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			uc, err := combat.NewAdvanceActiveEffects(dicetest.NewFixedRoller(faces...), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, cmd)
		}
	}
	end := func(cause combat.EndingCause) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return combat.NewEndActiveEffects(opts...).Run(ctx, combat.EndActiveEffectsCommand{Effects: hold, Cause: cause})
		}
	}
	damage := areaPlan(t, "2d6", aeSaveOf(t, ability.Dexterity, 13, combat.SaveSuccessHalfDamage), true)
	return []effectsRow{
		{"a start", "FR-013", "happy", "start_active_effect", "applied", "started", "", func(opts ...combat.Option) (any, error) {
			return combat.NewStartActiveEffect(opts...).Run(ctx, holdSpec.command(t, combat.ActiveEffects{}))
		}},
		{"a duplicate start", "FR-014", "unhappy (state error)", "start_active_effect", "refused", "rejected_refused", "duplicate_id", func(opts ...combat.Option) (any, error) {
			return combat.NewStartActiveEffect(opts...).Run(ctx, holdSpec.command(t, hold))
		}},
		{"a start with a binding that does not match", "FR-012", "unhappy (invalid input)", "start_active_effect", "invalid", "rejected_invalid", "binding_mismatch", func(opts ...combat.Option) (any, error) {
			cmd := holdSpec.command(t, combat.ActiveEffects{})
			cmd.Binding = combat.ConcentrationBinding{}
			return combat.NewStartActiveEffect(opts...).Run(ctx, cmd)
		}},
		{"a start over the cap of a creature", "P10", "unhappy (boundary)", "start_active_effect", "invalid", "limit_exceeded", "limit_exceeded", func(opts ...combat.Option) (any, error) {
			var full combat.ActiveEffects
			for i := range 64 {
				full = aeStart(t, full, bane2(fmt.Sprintf("e%03d", i), "t"))
			}
			return combat.NewStartActiveEffect(opts...).Run(ctx, bane2("new", "t").command(t, full))
		}},
		{"a save that fails", "FR-029", "happy", "advance_active_effects", "applied", "advanced", "", advance(combat.AdvanceActiveEffectsCommand{Effects: hold, Tick: aeEndTick(t, "orc"), Saves: saves}, 3)},
		{"a save that succeeds", "FR-029", "happy", "advance_active_effects", "applied", "advanced", "", advance(combat.AdvanceActiveEffectsCommand{Effects: hold, Tick: aeEndTick(t, "orc"), Saves: saves}, 18)},
		{"a save with no attempt", "FR-031", "unhappy (invalid input)", "advance_active_effects", "invalid", "rejected_invalid", "missing_save_attempt", advance(combat.AdvanceActiveEffectsCommand{Effects: hold, Tick: aeEndTick(t, "orc")})},
		{"an order that is not a permutation", "FR-028", "unhappy (invalid input)", "advance_active_effects", "invalid", "rejected_invalid", "invalid_order", advance(combat.AdvanceActiveEffectsCommand{Effects: hold, Tick: aeEndTick(t, "orc"), Order: []combat.ActiveEffectID{"zzz"}, Saves: saves})},
		{"an end by lost Concentration", "FR-041", "happy", "end_active_effects", "applied", "ended", "", end(aeMust(combat.NewConcentrationLost("wiz", aeBinding(t, "wiz", "hold_person").Subject())))},
		{"an end that matches nothing", "FR-041", "unhappy (empty and minimal)", "end_active_effects", "refused", "nothing_ended", "", end(aeMust(combat.NewSubjectDied("nobody")))},
		{"a dismissal by an incapacitated caster", "FR-042", "unhappy (state error)", "end_active_effects", "refused", "rejected_refused", "caster_incapacitated", end(aeMust(combat.NewDismissed("wiz", true, "hp")))},
		{"an unknown removal", "FR-044", "unhappy (invalid input)", "end_active_effects", "invalid", "rejected_invalid", "unknown_effect", end(combat.NewRemovedByEffect("zzz"))},
		{"a move", "FR-060", "happy", "end_active_effects", "applied", "moved", "", func(opts ...combat.Option) (any, error) {
			reg := aeSeq(t, mark("m", "ranger", "x"))
			return combat.NewEndActiveEffects(opts...).Run(ctx, combat.EndActiveEffectsCommand{Effects: reg, Cause: aeMust(combat.NewMoveEffectSubject("m", aeCreature2("y")))})
		}},
		{"a move over the cap of a creature", "P10, FR-060", "unhappy (boundary)", "end_active_effects", "invalid", "limit_exceeded", "limit_exceeded", func(opts ...combat.Option) (any, error) {
			reg := aeStart(t, heldBane(t, 64), mark("m", "ranger", "x"))
			return combat.NewEndActiveEffects(opts...).Run(ctx, combat.EndActiveEffectsCommand{Effects: reg, Cause: aeMust(combat.NewMoveEffectSubject("m", aeCreature2("t")))})
		}},
		{"damage resolved", "FR-036", "happy", "resolve_effect_damage", "applied", "effect_damage_resolved", "", func(opts ...combat.Option) (any, error) {
			uc, err := combat.NewResolveEffectDamage(dicetest.NewFixedRoller(3, 4, 15), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.ResolveEffectDamageCommand{Plan: damage, Targets: targets(map[string]d20.SaveAttempt{"a": aeRoll()}, "a")})
		}},
		{"damage with no targets", "FR-039", "unhappy (empty and minimal)", "resolve_effect_damage", "refused", "rejected_refused", "no_targets", func(opts ...combat.Option) (any, error) {
			uc, err := combat.NewResolveEffectDamage(dicetest.NewMinimumRoller(), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, combat.ResolveEffectDamageCommand{Plan: damage})
		}},
	}
}

// Requirement: doc/specs/active-effects.md section 7, FR-058 - every result is counted once under srd5e.combat.operations with its operation, result and result class; the span has the same attributes, error status only for an invalid result; a rejection adds srd5e.combat.active_effects.rejection with its first reason; the duration histogram uses the injected clock; no log is written.
// Path: per case.
func TestActiveEffectsSharedSignals(t *testing.T) {
	t.Parallel()
	for _, r := range effectsRows(t) {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			if _, err := r.run(opts...); err != nil {
				t.Fatalf("Run: %v", err)
			}
			span := "srd5e.combat." + r.operation
			spans := rec.Tracer.SpansNamed(span)
			if len(spans) != 1 || !spans[0].Ended {
				t.Fatalf("spans = %+v, want one ended %s [%s]", spans, span, r.requirement)
			}
			for key, want := range map[string]string{"srd5e.combat.result": r.name2, "srd5e.combat.result_class": r.class, "srd5e.combat.operation": r.operation} {
				if got, _ := spans[0].AttrValue(key); got != want {
					t.Errorf("span %s = %v, want %v [%s]", key, got, want, r.requirement)
				}
			}
			if got, _ := spans[0].AttrValue("srd5e.combat.active_effects.rejection"); r.reason != "" && got != r.reason || r.reason == "" && got != nil {
				t.Errorf("rejection attribute = %v, want %q", got, r.reason)
			}
			if (spans[0].Status == telemetry.StatusError) != (r.class == "invalid") {
				t.Errorf("span status %v for class %s, want error only for invalid", spans[0].Status, r.class)
			}
			ops := measured(rec, "srd5e.combat.operations")
			if len(ops) != 1 || ops[0]["srd5e.combat.result"] != r.name2 || ops[0]["srd5e.outcome"] != map[bool]string{true: "success", false: "failure"}[r.class == "applied"] {
				t.Errorf("operations = %v", ops)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %v, want one of 2 s from the injected clock", dur)
			}
			if len(rec.Logger.Records()) != 0 || len(rec.Violations()) != 0 {
				t.Errorf("logs %v violations %v, want none", rec.Logger.Records(), rec.Violations())
			}
			for _, m := range rec.Meter.Measurements() {
				if strings.HasPrefix(m.Name, "srd5e.d20_test.") || m.Name == "srd5e.combat.errors" {
					t.Errorf("an active effects call recorded %s [FR-056]", m.Name)
				}
			}
		})
	}
}

// Requirement: doc/specs/active-effects.md section 7 - effects ended are counted by reason, saving throws by outcome, ability and kind, and the registry size after a call is recorded with the operation; a rejected call records none of them.
// Path: per case.
func TestActiveEffectsOwnSignals(t *testing.T) {
	t.Parallel()
	rows := effectsRows(t)
	byName := map[string]effectsRow{}
	for _, r := range rows {
		byName[r.name] = r
	}
	run := func(name string) *telemetrytest.Recorder {
		rec, opts := recording()
		if _, err := byName[name].run(opts...); err != nil {
			t.Fatal(err)
		}
		return rec
	}
	pass := run("a save that succeeds")
	if got := measured(pass, "srd5e.combat.active_effects.saves"); len(got) != 1 || got[0]["srd5e.outcome"] != "success" || got[0]["srd5e.ability"] != "wisdom" || got[0]["srd5e.combat.active_effects.save_kind"] != "repeat" {
		t.Errorf("saves = %v", got)
	}
	if got := measured(pass, "srd5e.combat.active_effects.ended"); len(got) != 1 || got[0]["srd5e.combat.active_effects.end_reason"] != "repeat_save_succeeded" {
		t.Errorf("ended = %v", got)
	}
	if got := pass.Meter.MeasurementsNamed("srd5e.combat.active_effects.active"); len(got) != 1 || got[0].Value != 0 || got[0].Unit != "{effect}" || attrMap(got[0].Attrs)["srd5e.combat.operation"] != "advance_active_effects" {
		t.Errorf("active = %v", got)
	}
	fail := run("a save that fails")
	if got := measured(fail, "srd5e.combat.active_effects.saves"); len(got) != 1 || got[0]["srd5e.outcome"] != "failure" {
		t.Errorf("saves on a failure = %v", got)
	}
	if got := fail.Meter.MeasurementsNamed("srd5e.combat.active_effects.active"); got[0].Value != 1 || len(fail.Meter.MeasurementsNamed("srd5e.combat.active_effects.ended")) != 0 {
		t.Errorf("active = %v", got)
	}
	events := measured(pass, "srd5e.combat.events")
	if !slices.Equal(values(events, "srd5e.combat.event"), []string{"active_effect_ended", "active_effect_repeat_save_made"}) {
		t.Errorf("events = %v", events)
	}
	if got := measured(run("an end by lost Concentration"), "srd5e.combat.active_effects.ended"); len(got) != 1 || got[0]["srd5e.combat.active_effects.end_reason"] != "concentration_lost" {
		t.Errorf("ended on lost Concentration = %v", got)
	}
	if got := run("damage resolved").Meter.MeasurementsNamed("srd5e.combat.active_effects.saves"); len(got) != 1 || attrMap(got[0].Attrs)["srd5e.combat.active_effects.save_kind"] != "damage" || attrMap(got[0].Attrs)["srd5e.ability"] != "dexterity" {
		t.Errorf("damage saves = %v", got)
	}
	if got := run("an end that matches nothing").Meter.MeasurementsNamed("srd5e.combat.active_effects.active"); len(got) != 1 || got[0].Value != 1 {
		t.Errorf("active after nothing ended = %v", got)
	}
	for _, name := range []string{"a duplicate start", "a save with no attempt", "damage with no targets", "an unknown removal"} {
		rec := run(name)
		for _, own := range []string{"saves", "ended", "active"} {
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.active_effects." + own)); n != 0 {
				t.Errorf("%s recorded %d %s", name, n, own)
			}
		}
	}
	if got := run("a start").Meter.MeasurementsNamed("srd5e.combat.active_effects.active"); got[0].Value != 1 {
		t.Errorf("active after a start = %v", got)
	}
}

// Requirement: doc/specs/active-effects.md FR-055, observability.md §9 - a use case returns an identical result with the zero no-op telemetry and with a recording bundle.
// Path: per case.
func TestActiveEffectsResultsDoNotDependOnTelemetry(t *testing.T) {
	t.Parallel()
	for _, r := range effectsRows(t) {
		quiet, err1 := r.run()
		_, opts := recording()
		loud, err2 := r.run(opts...)
		if err1 != nil || err2 != nil {
			t.Fatalf("%s: %v %v", r.name, err1, err2)
		}
		if kindOfResult(quiet) != kindOfResult(loud) {
			t.Errorf("%s: %s with no telemetry, %s with a recorder", r.name, kindOfResult(quiet), kindOfResult(loud))
		}
	}
}

func kindOfResult(r any) string {
	switch v := r.(type) {
	case combat.AdvanceActiveEffectsApplied:
		return "advanced:" + strings.Join(aeKinds(v.Events()), ",")
	case combat.EndActiveEffectsEnded:
		return "ended:" + strings.Join(aeKinds(v.Events()), ",")
	case combat.StartActiveEffectStarted:
		return "started:" + strings.Join(aeKinds(v.Events()), ",")
	}
	return sprint(r)
}

// Requirement: doc/specs/active-effects.md FR-056, errors-and-results.md FR-031, FR-034, FR-051 - a roller failure is recorded once on the span with error status and counted once under srd5e.combat.errors with error.type dice_roller_error, with no operations counter; a panicking telemetry backend propagates and the span still ends.
// Path: unhappy (collaborator failure).
func TestActiveEffectsPortFailureAndTelemetryPanic(t *testing.T) {
	t.Parallel()
	hold := aeSeq(t, aeSpec{id: "hp", spell: "hold_person", caster: "wiz", subject: aeCreature2("orc"), template: aeHoldPerson(t, 14)})
	cmd := combat.AdvanceActiveEffectsCommand{Effects: hold, Tick: aeEndTick(t, "orc"), Saves: aeSaves("orc", ability.Wisdom, aeRoll())}
	rec, opts := recording()
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, errors.New("down") })
	uc, _ := combat.NewAdvanceActiveEffects(failing, opts...)
	_, err := uc.Run(context.Background(), cmd)
	if et := srderr.ErrorType(err); et != "dice_roller_error" {
		t.Fatalf("error type = %q", et)
	}
	spans := rec.Tracer.SpansNamed("srd5e.combat.advance_active_effects")
	if len(spans) != 1 || len(spans[0].Errors) != 1 || spans[0].Status != telemetry.StatusError {
		t.Errorf("spans = %+v, want one with the error recorded once", spans)
	}
	if got := measured(rec, "srd5e.combat.errors"); len(got) != 1 || got[0]["error.type"] != "dice_roller_error" {
		t.Errorf("errors = %v", got)
	}
	if len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 || len(rec.Logger.Records()) != 0 {
		t.Error("a port failure counted an operation or was logged")
	}
	rec2 := telemetrytest.NewRecorder()
	loud, _ := combat.NewAdvanceActiveEffects(dicetest.NewMinimumRoller(), combat.WithTelemetry(telemetry.Telemetry{Tracer: rec2.Tracer, Meter: panicMeter{}}))
	func() {
		defer func() {
			if r := recover(); r == nil || r.(error).Error() != "backend bug" {
				t.Errorf("recovered %v, want the original panic", r)
			}
		}()
		_, _ = loud.Run(context.Background(), cmd)
	}()
	if spans := rec2.Tracer.SpansNamed("srd5e.combat.advance_active_effects"); len(spans) != 1 || !spans[0].Ended {
		t.Errorf("spans = %+v, want one ended span", spans)
	}
}

// Requirement: doc/specs/active-effects.md FR-014, FR-057 - a refusal changes nothing in the input and the same input gives the same result twice.
// Path: unhappy (immutability, determinism).
func TestActiveEffectsRefusalsLeaveTheInputAlone(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, holdSpec("hp", "orc"))
	snapshot := reg.Effects()
	cmd := holdSpec("hp", "goblin").command(t, reg)
	for range 2 {
		res, err := combat.NewStartActiveEffect().Run(context.Background(), cmd)
		if _, ok := res.(combat.StartActiveEffectRejected); !ok || err != nil {
			t.Fatalf("= %#v, %v", res, err)
		}
	}
	if !slices.EqualFunc(snapshot, reg.Effects(), combat.ActiveEffect.Equal) {
		t.Error("a refused start changed the registry")
	}
}
