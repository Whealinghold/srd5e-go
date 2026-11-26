package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// seTextOnly is a spell with no structured effects.
func seTextOnly(level int) content.Spell { return seSpell("lore_bolt", level, nil, nil) }

// Requirement: spell-effects.md FR-054, FR-055, FR-056 - a spell without structured effects returns SpellEffectsTextOnly with the spell id and the effective level (the cast level, else the spell's own) and requests no die, even with targets that would be invalid; a partial model reports its coverage.
// Path: unhappy (missing content).
func TestTextOnlyAndPartialCoverage(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewMinimumRoller()
	res, err := seRun(t, seCommand(seTextOnly(2), seReport("lore_bolt", 4, 0), combat.SpellTarget{}), roller)
	text, ok := res.(combat.SpellEffectsTextOnly)
	if err != nil || !ok || text.SpellID() != "lore_bolt" || text.EffectiveLevel() != 4 || len(roller.Requests()) != 0 {
		t.Fatalf("res %#v err %v requests %v, want TextOnly at level 4 and no die [FR-055]", res, err, roller.Requests())
	}
	res, _ = seRun(t, seCommand(seTextOnly(2), seReport("lore_bolt", 0, 0)), roller)
	if got := res.(combat.SpellEffectsTextOnly).EffectiveLevel(); got != 2 {
		t.Errorf("effective level without a cast level = %d, want the spell's own 2", got)
	}
	partial := seSpell("twin_effect", 1, seModelWithZone(), nil)
	r := seResolved(t, seCommand(partial, seReport("twin_effect", 1, 0), seTarget("a", nil)), dicetest.NewFixedRoller(3))
	if r.Coverage() != content.CoveragePartial || r.SpellID() != "twin_effect" || r.EffectiveLevel() != 1 {
		t.Errorf("coverage %v spell %v level %d, want partial [FR-056]", r.Coverage(), r.SpellID(), r.EffectiveLevel())
	}
	complete := seResolved(t, seCommand(seCureWounds(), seReport("cure_wounds", 0, 0), seTarget("a", nil)), dicetest.NewFixedRoller(1, 1))
	if complete.Coverage() != content.CoverageComplete || complete.EffectiveLevel() != 1 {
		t.Errorf("a cast with no level reports the spell's own level: %d", complete.EffectiveLevel())
	}
}

// Requirement: spell-effects.md FR-096 - NewResolveSpellEffects takes a roller, rejects a nil one with an error matching srderr.ErrInvalidInput, and the zero use case refuses to run before any signal.
// Path: unhappy (invalid input).
func TestConstructorAndZeroUseCase(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewResolveSpellEffects(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller err = %v", err)
	}
	rec, opts := recording()
	_ = opts
	var zero combat.ResolveSpellEffects
	if res, err := zero.Run(context.Background(), seCommand(seFireball(), seReport("fireball", 3, 0))); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 {
		t.Error("the zero use case recorded a span")
	}
}

// Requirement: spell-effects.md FR-060, FR-063 - a call changes nothing in its command and a result exposes copies, whose mutation cannot corrupt it; the same command and the same roller sequence give an equal result.
// Path: unhappy (immutability, determinism).
func TestImmutabilityAndDeterminism(t *testing.T) {
	t.Parallel()
	targets := []combat.SpellTarget{seTarget("a", nil), seTarget("b", nil)}
	assignments := []combat.InstanceAssignment{}
	parts := []int{0}
	cmd := seCommand(seFireball(), seReport("fireball", 3, 0), targets...)
	cmd.Assignments, cmd.PartIndexes = assignments, parts
	faces := slices.Concat(seEightD6, []int{5, 19})
	first := seResolved(t, cmd, dicetest.NewFixedRoller(faces...))
	targets[0], parts[0] = seTarget("z", nil), 9
	if got := first.Targets()[0].Target(); got != seRef("a") {
		t.Errorf("the caller's slice changed the result: %v", got)
	}
	cmd2 := seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil), seTarget("b", nil))
	again := seResolved(t, cmd2, dicetest.NewFixedRoller(faces...))
	if !first.Equal(again) {
		t.Error("the same command and rolls gave unequal results [FR-063]")
	}
	different := seResolved(t, cmd2, dicetest.NewFixedRoller(slices.Concat(seEightD6, []int{19, 19})...))
	if first.Equal(different) {
		t.Error("results with other saves are equal")
	}
	ts := first.Targets()
	ts[0] = combat.TargetEffects{}
	dmg := first.Targets()[1].Damage()
	dmg[0] = combat.PlannedDamage{}
	ds := first.Targets()[1].Damage()[0].Damages()
	ds[0] = combat.Damage{}
	ps := first.Parts()
	ps[0] = combat.ResolvedPart{}
	ins := first.Parts()[0].Instances()
	ins[0] = combat.ResolvedInstance{}
	am := first.Parts()[0].Instances()[0].Amounts()
	am[0] = combat.RolledAmount{}
	if len(first.Parts()[0].Instances()[0].Amounts()[0].Faces()) != 8 {
		t.Error("a copy of the amounts changed the evidence")
	}
	faceCopy := first.Parts()[0].Instances()[0].Amounts()[0].Faces()
	faceCopy[0] = 99
	if first.Parts()[0].Instances()[0].Amounts()[0].Faces()[0] != 3 {
		t.Error("a copy of the faces changed the evidence")
	}
	if first.Targets()[0].Target() != seRef("a") || first.Targets()[1].Damage()[0].Damages()[0].Amount() != 28 || first.Parts()[0].Index() != 0 {
		t.Error("mutating the copies changed the result")
	}
	if first.Targets()[0].Equal(first.Targets()[1]) || !first.Targets()[0].Equal(first.Targets()[0]) {
		t.Error("TargetEffects.Equal")
	}
	if first.Equal(combat.SpellEffectsResolved{}) {
		t.Error("equal to the zero result")
	}
}

// Requirement: spell-effects.md FR-061, FR-064 - a roller that fails, panics or answers another expression, at the first die or in the middle of the call, is returned as one *srderr.PortError for dice with the cause kept, with a nil result; a recovered panic never crosses the package.
// Path: unhappy (failure of collaborator).
func TestRollerFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	secondFails := func() combat.Roller {
		n := 0
		inner := dicetest.NewFixedRoller(slices.Concat(seEightD6, []int{5})...)
		return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			if n++; n == 2 {
				return dice.Result{}, cause
			}
			return inner.Roll(e)
		})
	}
	rollers := map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller(), "fails mid-call": secondFails()}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			res, err := seRun(t, seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil)), roller)
			envPortFailure(t, err)
			if res != nil {
				t.Errorf("result = %#v, want nil: no partial result", res)
			}
			if name == "fails" && !errors.Is(err, cause) {
				t.Errorf("err = %v, want the cause kept", err)
			}
		})
	}
	t.Run("a saving throw roll fails", func(t *testing.T) {
		t.Parallel()
		_, err := seRun(t, seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), envFailingRoller(cause))
		envPortFailure(t, err)
	})
	t.Run("an attack roll or its damage fails", func(t *testing.T) {
		t.Parallel()
		_, err := seRun(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), envFailingRoller(cause))
		envPortFailure(t, err)
		n := 0
		inner := dicetest.NewFixedRoller(15)
		damageFails := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			if n++; n == 2 {
				return dice.Result{}, cause
			}
			return inner.Roll(e)
		})
		_, err = seRun(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), damageFails)
		envPortFailure(t, err)
	})
	t.Run("an automatic amount fails", func(t *testing.T) {
		t.Parallel()
		_, err := seRun(t, seCommand(seCureWounds(), seReport("cure_wounds", 1, 0), seTarget("a", nil)), envFailingRoller(cause))
		envPortFailure(t, err)
	})
}

// Requirement: spell-effects.md FR-061, FR-083, errors-and-results.md - a call whose ctx is already cancelled returns ctx.Err() before any signal and any die.
// Path: unhappy (cancellation).
func TestCancelledContext(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	roller := dicetest.NewMinimumRoller()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := seUse(t, roller, opts...).Run(ctx, seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil)))
	if res != nil || !errors.Is(err, context.Canceled) || len(roller.Requests()) != 0 {
		t.Errorf("res %v err %v requests %v, want context.Canceled, no result and no die", res, err, roller.Requests())
	}
	if len(rec.Meter.Measurements()) != 0 || len(rec.Tracer.Spans()) != 0 {
		t.Error("a cancelled call recorded a signal")
	}
}

type seSignalRow struct {
	name, requirement, path string
	cmd                     combat.ResolveSpellEffectsCommand
	faces                   []int
	class, result           string
	coverage, rejection     string
	level                   string           // "" when no resolutions sample is expected
	rolls                   []map[string]any // the rolls samples, in order
	targets                 float64          // the valid targets of a resolved call
}

func seSignalRows() []seSignalRow {
	hold3 := seCommand(seHoldPerson(), seReport("hold_person", 3, 0), seTarget("a", nil), seTarget("b", nil), seTarget("c", nil))
	chooseToFail := seTarget("a", func(p *combat.SpellTargetParams) {
		p.Saves = map[ability.Ability]d20.SaveAttempt{ability.Dexterity: d20.ChooseToFail{}}
	})
	roll := func(kind, outcome, abil string) map[string]any {
		m := map[string]any{"srd5e.combat.spell_effects.roll": kind, "srd5e.outcome": outcome}
		if abil != "" {
			m["srd5e.ability"] = abil
		}
		return m
	}
	nineD6 := slices.Repeat([]int{3}, 9)
	return []seSignalRow{
		{"resolved with a failed and a saved target", spellEffectsSpec + "FR-081 FR-082", "happy", seCommand(seFireball(), seReport("fireball", 4, 0), seTarget("a", nil), seTarget("b", nil)),
			slices.Concat(nineD6, []int{5, 19}), "applied", "resolved", "complete", "", "4", []map[string]any{roll("save", "failure", "dexterity"), roll("save", "success", "dexterity")}, 2},
		{"a forced failure is save_forced", spellEffectsSpec + "FR-082", "unhappy (rule interaction)", seCommand(seFireball(), seReport("fireball", 3, 0), chooseToFail),
			seEightD6, "applied", "resolved", "complete", "", "3", []map[string]any{roll("save_forced", "failure", "dexterity")}, 1},
		{"an attack hit", spellEffectsSpec + "FR-082", "happy", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")),
			[]int{15, 4}, "applied", "resolved", "complete", "", "cantrip", []map[string]any{roll("attack", "success", "")}, 1},
		{"an attack miss", spellEffectsSpec + "FR-082", "unhappy (boundary)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")),
			[]int{2}, "applied", "resolved", "complete", "", "cantrip", []map[string]any{roll("attack", "failure", "")}, 1},
		{"a partial model with no roll", spellEffectsSpec + "FR-081", "happy", seCommand(seSpell("twin_effect", 1, seModelWithZone(), nil), seReport("twin_effect", 1, 0), seTarget("a", nil)),
			[]int{3}, "applied", "resolved", "partial", "", "1", nil, 1},
		{"an empty area counts zero targets", spellEffectsSpec + "FR-081", "unhappy (empty and minimal)", seCommand(seFireball(), seReport("fireball", 3, 0)),
			nil, "applied", "resolved", "complete", "", "3", nil, 0},
		{"text only", spellEffectsSpec + "FR-081", "unhappy (missing content)", seCommand(seTextOnly(2), seReport("lore_bolt", 0, 0)),
			nil, "applied", "text_only", "text_only", "", "2", nil, -1},
		{"text only cantrip", spellEffectsSpec + "FR-081", "unhappy (boundary)", seCommand(seTextOnly(0), seReport("lore_bolt", 0, 0)),
			nil, "applied", "text_only", "text_only", "", "cantrip", nil, -1},
		{"rejected invalid", spellEffectsSpec + "FR-080 FR-081", "unhappy (invalid input)", hold3,
			nil, "invalid", "rejected_invalid", "complete", "wrong_target_count", "", nil, -1},
		{"rejected refused: no clear path", spellEffectsSpec + "FR-080", "unhappy (state error)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1),
			seTarget("a", func(p *combat.SpellTargetParams) {
				p.Path = combat.PathBlockedByTotalCover
				p.AttackFacts = seFacts(nil)
			})),
			nil, "refused", "rejected_refused", "complete", "no_clear_path", "", nil, -1},
		{"rejected refused: caster not allowed", spellEffectsSpec + "FR-080", "unhappy (rule interaction)", seCommand(seWardenBolt(), seReport("warden_bolt", 1, 0), seTarget("caster", nil)),
			nil, "refused", "rejected_refused", "complete", "caster_not_allowed", "", nil, -1},
		{"a mixed rejection is invalid and names the first reason", spellEffectsSpec + "FR-080 FR-081", "unhappy (rule interaction)", seCommand(seHoldPerson(), seReport("hold_person", 2, 0),
			seTarget("a", nil), seTarget("b", func(p *combat.SpellTargetParams) { p.Path = combat.PathBlockedByTotalCover }), seTarget("c", nil)),
			nil, "invalid", "rejected_invalid", "complete", "wrong_target_count", "", nil, -1},
	}
}

// Requirement: spell-effects.md FR-080, FR-081, FR-082, FR-084, FR-085 - the call is one operation resolve_spell_effects with the result, class, outcome, coverage and rejection attributes on the span and the operations counter; resolutions (resolved and text_only only, with coverage and level), rolls (with srd5e.ability for saves) and targets (resolved only) are recorded beside them; no event, no log, no resolve_attack or srd5e.d20_test signal; the result is equal with the no-op telemetry.
// Path: per case.
func TestResolveSpellEffectsSignals(t *testing.T) {
	t.Parallel()
	for _, c := range seSignalRows() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := seRun(t, c.cmd, dicetest.NewFixedRoller(c.faces...), opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			plain, err := seRun(t, c.cmd, dicetest.NewFixedRoller(c.faces...))
			if err != nil || !reflect.DeepEqual(res, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-085]", res, plain, err)
			}
			outcome := map[string]string{"applied": "success", "refused": "failure", "invalid": "failure"}[c.class]
			want := map[string]any{"srd5e.combat.operation": "resolve_spell_effects", "srd5e.outcome": outcome, "srd5e.combat.result": c.result,
				"srd5e.combat.result_class": c.class, "srd5e.combat.spell_effects.coverage": c.coverage}
			if c.rejection != "" {
				want["srd5e.combat.spell_effects.rejection"] = c.rejection
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.resolve_spell_effects")
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, in error status only for invalid [%s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [%s]", sa, want, c.requirement)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 || len(dur[0].Attrs) != 1 {
				t.Errorf("duration = %+v, want one sample of 2 s with the operation only", dur)
			}
			res1 := measured(rec, "srd5e.combat.spell_effects.resolutions")
			if c.level == "" {
				if len(res1) != 0 {
					t.Errorf("resolutions = %v, want none for a rejection [FR-081]", res1)
				}
			} else if wantRes := map[string]any{"srd5e.combat.spell_effects.coverage": c.coverage, "srd5e.combat.spell_effects.level": c.level}; len(res1) != 1 || !reflect.DeepEqual(res1[0], wantRes) {
				t.Errorf("resolutions = %v, want %v [FR-081]", res1, wantRes)
			}
			rolls := measured(rec, "srd5e.combat.spell_effects.rolls")
			if len(rolls) != len(c.rolls) || (len(rolls) > 0 && !reflect.DeepEqual(rolls, c.rolls)) {
				t.Errorf("rolls = %v, want %v [FR-082]", rolls, c.rolls)
			}
			tg := rec.Meter.MeasurementsNamed("srd5e.combat.spell_effects.targets")
			if c.targets < 0 && len(tg) != 0 || c.targets >= 0 && c.result == "resolved" && (len(tg) != 1 || tg[0].Value != c.targets || len(tg[0].Attrs) != 0) {
				t.Errorf("targets = %+v, want %v for a resolved call only [FR-081]", tg, c.targets)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.events")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("events or errors recorded: %d [FR-084]", n)
			}
			for _, m := range rec.Meter.Measurements() {
				if m.Name == "srd5e.combat.attack.modifiers" {
					t.Errorf("the attack signal %s was recorded: no use case is called [FR-084]", m.Name)
				}
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: spell-effects.md FR-083, FR-084, observability.md §7 - a roller failure is recorded once by the use case (error status, one recorded error, one srd5e.combat.errors sample with error.type dice_roller_error) and not logged, with no operations, resolutions, rolls or targets sample; no span of resolve_attack is started.
// Path: unhappy (collaborator failure, telemetry on failure).
func TestRollerFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		rec, opts := recording()
		_, err := seRun(t, seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil)), roller, opts...)
		envPortFailure(t, err)
		spans := rec.Tracer.SpansNamed("srd5e.combat.resolve_spell_effects")
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended || len(rec.Tracer.Spans()) != 1 {
			t.Errorf("%s: spans = %+v, want one ended span with one recorded error", name, rec.Tracer.Spans())
		}
		errs := measured(rec, "srd5e.combat.errors")
		if len(errs) != 1 || errs[0]["error.type"] != "dice_roller_error" || errs[0]["srd5e.combat.operation"] != "resolve_spell_effects" {
			t.Errorf("%s: errors = %v, want one dice_roller_error", name, errs)
		}
		for _, n := range []string{"operations", "events", "spell_effects.resolutions", "spell_effects.rolls", "spell_effects.targets"} {
			if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
				t.Errorf("%s: %s recorded %d on an error", name, n, len(m))
			}
		}
		checkTelemetryHygiene(t, rec)
	}
}

// Requirement: spell-effects.md FR-062, FR-083 - an invalid-command error is recorded on the span only: error status and one recorded error, no srd5e.combat.errors sample, no operations sample and no log.
// Path: unhappy (invalid input, telemetry on failure).
func TestInvalidCommandIsRecordedOnTheSpanOnly(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	_, err := seRun(t, seCommand(seFireball(), seReport("fireball", 2, 0), seTarget("a", nil)), dicetest.NewMinimumRoller(), opts...)
	if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
		t.Fatalf("err = %v", err)
	}
	spans := rec.Tracer.SpansNamed("srd5e.combat.resolve_spell_effects")
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
		t.Errorf("spans = %+v, want one span in error status", spans)
	}
	for _, n := range []string{"operations", "events"} {
		if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
			t.Errorf("%s recorded %d for an invalid command", n, len(m))
		}
	}
	assertInvalidInputCounted(t, rec, "resolve_spell_effects")
	checkTelemetryHygiene(t, rec)
}

// Requirement: spell-effects.md FR-082 - the ability attribute of a spell save is the d20 key and one of six lowercase ability names, and no sample carries a spell id or a combatant id.
// Path: unhappy (telemetry).
func TestSpellEffectsAttributeValues(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		abil string
		cmd  combat.ResolveSpellEffectsCommand
		face int
	}{
		{"wisdom", seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), 5},
		{"constitution", seCommand(seHarm(), seReport("harm", 6, 0), seTarget("a", nil)), 5},
	} {
		rec, opts := recording()
		faces := []int{c.face}
		if c.abil == "constitution" {
			faces = slices.Concat(slices.Repeat([]int{3}, 14), faces)
		}
		seResolved(t, c.cmd, dicetest.NewFixedRoller(faces...), opts...)
		rolls := measured(rec, "srd5e.combat.spell_effects.rolls")
		if len(rolls) != 1 || rolls[0][d20.AttrAbility] != c.abil {
			t.Errorf("rolls = %v, want the ability %s", rolls, c.abil)
		}
		for _, m := range rec.Meter.Measurements() {
			for _, a := range m.Attrs {
				if v, ok := a.Value().(string); ok && (v == string(seCaster) || v == "hold_person" || v == "harm") {
					t.Errorf("%s carries the id %q", m.Name, v)
				}
			}
		}
	}
}
