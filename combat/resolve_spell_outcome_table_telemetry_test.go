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
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

const otSpan = "srd5e.combat.resolve_spell_outcome_table"

// otNoSignals asserts that nothing was recorded for the operation: no span, no operations count, no table roll.
func otNoSignals(t *testing.T, rec *telemetrytest.Recorder) {
	t.Helper()
	if len(rec.Tracer.SpansNamed(otSpan)) != 0 || len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 || len(rec.Meter.MeasurementsNamed("srd5e.combat.spell_table_rolls")) != 0 {
		t.Error("expected no signal")
	}
}

// otIDsNotInAttributes asserts that no attribute of any span or measurement is one of the combatant ids of a test.
func otIDsNotInAttributes(t *testing.T, rec *telemetrytest.Recorder, ids ...string) {
	t.Helper()
	for _, m := range rec.Meter.Measurements() {
		for _, a := range m.Attrs {
			if s, ok := a.Value().(string); ok && slices.Contains(ids, s) {
				t.Errorf("%s attribute %s carries the id %q (FR-103)", m.Name, a.Key(), s)
			}
		}
	}
	for _, s := range rec.Tracer.Spans() {
		for _, a := range s.Attrs {
			if v, ok := a.Value().(string); ok && slices.Contains(ids, v) {
				t.Errorf("span attribute %s carries the id %q (FR-103)", a.Key(), v)
			}
		}
	}
}

// otEveryCommand returns one command of each variant that rolls a die first, with what a minimum roller and a
// maximum roller need not matter: the roller of these tests fails or lies on a roll.
func otEveryCommand() map[string]combat.ResolveSpellOutcomeTableCommand {
	return map[string]combat.ResolveSpellOutcomeTableCommand{
		"teleport":    combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.VeryFamiliar},
		"confusion":   combat.RollConfusionBehaviorCommand{CreaturesWithinReach: otIDs("a", "b")},
		"spray":       combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: otTargets("a")},
		"wall":        combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Creature: combat.PrismaticSaveTarget{ID: "a", Attempt: otAttempt()}},
		"reincarnate": combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid},
		"blink":       combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnOtherPlane},
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-091, FR-103, errors-and-results FR-031, FR-034, observability.md section 7 - a roller that fails, panics or answers another expression, on the first roll or on a later one, is returned as a *srderr.PortError for port dice_roller with a nil result and no partial result; it is recorded once (error status and one recorded error on the span, one srd5e.combat.errors count with error.type dice_roller_error), with no operations sample, no srd5e.combat.spell_table_rolls sample (also for the rolls made before the failure), no event and no log; errors.Is finds the cause.
// Path: unhappy (failure of collaborator).
func TestResolveSpellOutcomeTablePortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	// laterFailure answers the first roll with a minimum die and fails the second.
	laterFailure := func() combat.Roller {
		n := 0
		return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			if n++; n == 2 {
				return dice.Result{}, cause
			}
			return dicetest.NewMinimumRoller().Roll(e)
		})
	}
	rollers := map[string]func() combat.Roller{
		"fails":                 func() combat.Roller { return envFailingRoller(cause) },
		"panics":                envPanicRoller,
		"wrong expression":      envWrongRoller,
		"fails on a later roll": laterFailure,
	}
	for cmdName, cmd := range otEveryCommand() {
		for rollerName, mk := range rollers {
			if cmdName == "blink" && rollerName == "fails on a later roll" {
				continue // Blink makes one roll, so it has no later roll to fail
			}
			t.Run(cmdName+" "+rollerName, func(t *testing.T) {
				t.Parallel()
				rec, opts := recording()
				res, err := otUse(t, mk(), opts...).Resolve(context.Background(), cmd)
				envPortFailure(t, err)
				if res != nil {
					t.Errorf("result = %#v, want nil", res)
				}
				if (rollerName == "fails" || rollerName == "fails on a later roll") && !errors.Is(err, cause) {
					t.Errorf("errors.Is(err, cause) = false for %v", err)
				}
				if rollerName == "panics" && !errors.Is(err, srderr.ErrPortPanic) {
					t.Errorf("a panic must be recovered as ErrPortPanic, got %v", err)
				}
				spans := rec.Tracer.SpansNamed(otSpan)
				if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
					t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
				}
				if got := values(measured(rec, "srd5e.combat.errors"), "error.type"); !slices.Equal(got, []string{"dice_roller_error"}) {
					t.Errorf("errors counted = %v, want one dice_roller_error", got)
				}
				for _, n := range []string{"srd5e.combat.operations", "srd5e.combat.events", "srd5e.combat.spell_table_rolls"} {
					if m := rec.Meter.MeasurementsNamed(n); len(m) != 0 {
						t.Errorf("%s = %v, want no sample after a port failure (FR-102)", n, m)
					}
				}
				checkTelemetryHygiene(t, rec)
			})
		}
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-090, FR-093, testing.md section 4 - a nil roller and the zero use case are invalid input; a nil command is an invalid input recorded on the span only, with no die and no operations sample; an already cancelled context is returned unwrapped before any signal or die.
// Path: unhappy (invalid input; cancellation).
func TestResolveSpellOutcomeTableInvalidUse(t *testing.T) {
	t.Parallel()
	valid := combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnOtherPlane}
	if _, err := combat.NewResolveSpellOutcomeTable(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewResolveSpellOutcomeTable(nil) = %v, want ErrInvalidInput", err)
	}
	if res, err := (combat.ResolveSpellOutcomeTable{}).Resolve(context.Background(), valid); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(1)
	use := otUse(t, roller, opts...)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Resolve(ctx, valid); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled context = %v, %v, want context.Canceled", res, err)
	}
	otNoSignals(t, rec)
	if res, err := use.Resolve(context.Background(), nil); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("nil command = %v, %v, want ErrInvalidInput", res, err)
	}
	spans := rec.Tracer.SpansNamed(otSpan)
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(roller.Requests()) != 0 || len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 {
		t.Errorf("a malformed command records error status on the span only: spans %+v, %d requests", spans, len(roller.Requests()))
	}
	for name, cmd := range map[string]combat.ResolveSpellOutcomeTableCommand{
		"teleport without a subject":          combat.RollTeleportOutcomeCommand{Familiarity: combat.VeryFamiliar},
		"teleport with an empty creature set": combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingCreatures{}, Familiarity: combat.VeryFamiliar},
		"teleport without a familiarity":      combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}},
		"blink without a location":            combat.RollBlinkTurnEndCommand{},
	} {
		if res, err := use.Resolve(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) || res != nil || len(roller.Requests()) != 0 {
			t.Errorf("%s = %v, %v with %d requests, want ErrInvalidInput and no die", name, res, err, len(roller.Requests()))
		}
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-100, FR-101, FR-102, FR-103, section 7.1 - each call records the shared signals under resolve_spell_outcome_table with the result spell_table_rolled (applied) or spell_table_refused (refused, no error status), the attribute srd5e.combat.spell_table on the span and the operations count and nowhere else, and one srd5e.combat.spell_table_rolls count per roll on a printed table with the closed table_outcome (rerolls included; damage dice, saves and direction or pick dice not counted; a certain Teleport row, a chosen species, a refusal and a wall count nothing); no event, no log, no id in an attribute.
// Path: per case.
func TestResolveSpellOutcomeTableSignals(t *testing.T) {
	t.Parallel()
	twelve := otRepeat(12, 3)
	humanoid := content.CreatureTypeHumanoid
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ResolveSpellOutcomeTableCommand
		faces                   []int
		table, result, class    string
		outcomes                []string
	}{
		{"Teleport Mishap then Similar Area", "FR-102", "happy", combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.VeryFamiliar},
			[]int{1, 1, 1, 1, 6}, "teleport", "spell_table_rolled", "applied", []string{"mishap", "similar_area"}},
		{"Teleport Off Target counts the d100 only", "FR-102", "happy", combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.VeryFamiliar},
			[]int{14, 3, 4, 2}, "teleport", "spell_table_rolled", "applied", []string{"off_target"}},
		{"Teleport On Target", "FR-102", "happy", combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.VeryFamiliar},
			[]int{99}, "teleport", "spell_table_rolled", "applied", []string{"on_target"}},
		{"Teleport to a permanent circle counts nothing", "FR-102", "unhappy (rule interaction)", combat.RollTeleportOutcomeCommand{Subject: combat.TeleportingObject{}, Familiarity: combat.PermanentCircle},
			nil, "teleport", "spell_table_rolled", "applied", nil},
		{"Confusion counts the d10 and not the pick", "FR-102", "happy", combat.RollConfusionBehaviorCommand{CreaturesWithinReach: otIDs("a", "b")},
			[]int{7, 2}, "confusion", "spell_table_rolled", "applied", []string{"attacks_random_creature"}},
		{"Confusion random movement", "FR-102", "happy", combat.RollConfusionBehaviorCommand{}, []int{1, 3}, "confusion", "spell_table_rolled", "applied", []string{"moves_randomly"}},
		{"Confusion does nothing", "FR-102", "happy", combat.RollConfusionBehaviorCommand{}, []int{4}, "confusion", "spell_table_rolled", "applied", []string{"does_nothing"}},
		{"Confusion the target chooses", "FR-102", "happy", combat.RollConfusionBehaviorCommand{}, []int{10}, "confusion", "spell_table_rolled", "applied", []string{"chooses_behavior"}},
		{"Prismatic Spray counts every ray, the special 8 included, and not the damage or saves", "FR-102", "unhappy (rule interaction)",
			combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: otTargets("a")}, slices.Concat([]int{8, 1, 6}, twelve, []int{5}),
			"prismatic_spray", "spell_table_rolled", "applied", []string{"indigo", "red", "special"}},
		{"Prismatic Spray with no target", "FR-067", "unhappy (empty and minimal)", combat.RollPrismaticSprayCommand{SaveDC: 15}, nil,
			"prismatic_spray", "spell_table_rolled", "applied", nil},
		{"Prismatic Wall makes no table roll", "FR-102", "happy", combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Wall: luMust(combat.NewPrismaticWallStateAfter(6)),
			Creature: combat.PrismaticSaveTarget{ID: "a", Attempt: otAttempt()}}, []int{12}, "prismatic_wall", "spell_table_rolled", "applied", nil},
		{"Reincarnate counts every roll, the 1s included", "FR-102", "happy", combat.RollReincarnationSpeciesCommand{CreatureType: humanoid},
			[]int{1, 1, 4}, "reincarnate", "spell_table_rolled", "applied", []string{"elf", "roll_again", "roll_again"}},
		{"Reincarnate with a chosen species counts nothing", "FR-102", "happy", combat.RollReincarnationSpeciesCommand{CreatureType: humanoid, ChosenSpecies: "elf"},
			nil, "reincarnate", "spell_table_rolled", "applied", nil},
		{"Reincarnate refused counts nothing and is no error", "FR-100", "unhappy (rule interaction)", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeBeast},
			nil, "reincarnate", "spell_table_refused", "refused", nil},
		{"Blink stays", "FR-102", "happy", combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnOtherPlane}, []int{3}, "blink", "spell_table_rolled", "applied", []string{"stays"}},
		{"Blink vanishes", "FR-102", "happy", combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnOtherPlane}, []int{4}, "blink", "spell_table_rolled", "applied", []string{"vanishes"}},
		{"Blink already ethereal records vanishes", "FR-102", "unhappy (rule interaction)", combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnEtherealPlane}, []int{6},
			"blink", "spell_table_rolled", "applied", []string{"vanishes"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := otUse(t, dicetest.NewFixedRoller(c.faces...), opts...).Resolve(context.Background(), c.cmd)
			if err != nil {
				t.Fatalf("Resolve = %v", err)
			}
			spans := rec.Tracer.SpansNamed(otSpan)
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError {
				t.Fatalf("spans = %+v, want one ended span, error status only for a failure of the library [%s%s]", spans, otSpec, c.requirement)
			}
			attrs := attrMap(spans[0].Attrs)
			wantOutcome := "success"
			if c.class != "applied" {
				wantOutcome = "failure"
			}
			if attrs["srd5e.combat.operation"] != "resolve_spell_outcome_table" || attrs["srd5e.combat.result"] != c.result || attrs["srd5e.combat.result_class"] != c.class ||
				attrs["srd5e.outcome"] != wantOutcome || attrs["srd5e.combat.spell_table"] != c.table {
				t.Errorf("span attributes = %v", attrs)
			}
			ops := measured(rec, "srd5e.combat.operations")
			if len(ops) != 1 || ops[0]["srd5e.combat.spell_table"] != c.table || ops[0]["srd5e.combat.result"] != c.result {
				t.Errorf("operations = %v, want one sample with spell_table %s", ops, c.table)
			}
			rolls := measured(rec, "srd5e.combat.spell_table_rolls")
			var got []string
			for _, m := range rolls {
				if m["srd5e.combat.spell_table"] != c.table || len(m) != 2 {
					t.Errorf("spell_table_rolls attributes = %v, want only the table and the outcome", m)
				}
				got = append(got, m["srd5e.combat.table_outcome"].(string))
			}
			slices.Sort(got)
			if !slices.Equal(got, c.outcomes) {
				t.Errorf("table outcomes counted = %v, want %v [%s%s]", got, c.outcomes, otSpec, c.requirement)
			}
			for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.spell_table_rolls") {
				if m.Value != 1 || m.Unit != "{roll}" || m.Kind != telemetrytest.KindCounter {
					t.Errorf("a table roll is one count of {roll}: %+v", m)
				}
			}
			if m := rec.Meter.MeasurementsNamed("srd5e.combat.events"); len(m) != 0 {
				t.Errorf("events counted = %v, want none (FR-094)", m)
			}
			if res == nil {
				t.Error("no result")
			}
			otIDsNotInAttributes(t, rec, "a", "b")
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-092, observability.md section 9 - a call returns the same result with the zero-value Noop telemetry and with a recording bundle, and the same twice; the commands are not changed.
// Path: unhappy (determinism).
func TestResolveSpellOutcomeTableSameWithAndWithoutTelemetry(t *testing.T) {
	t.Parallel()
	twelve := otRepeat(12, 3)
	cases := map[string]struct {
		cmd   combat.ResolveSpellOutcomeTableCommand
		faces []int
	}{
		"teleport":    {combat.RollTeleportOutcomeCommand{Subject: luMust(combat.NewTeleportingCreatures(otIDs("a", "b"))), Familiarity: combat.SeenCasually}, []int{1, 1, 1, 1, 2, 2, 2, 60}},
		"confusion":   {combat.RollConfusionBehaviorCommand{CreaturesWithinReach: otIDs("a", "b", "c")}, []int{8, 3}},
		"spray":       {combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: otTargets("a", "b")}, slices.Concat([]int{1, 7}, twelve, []int{5, 15})},
		"wall":        {combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Wall: luMust(combat.NewPrismaticWallStateAfter(5)), Creature: combat.PrismaticSaveTarget{ID: "a", Attempt: otAttempt()}}, []int{5, 5}},
		"reincarnate": {combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid}, []int{1, 9}},
		"blink":       {combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnEtherealPlane}, []int{5}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			before := reflect.ValueOf(c.cmd)
			quiet, err := otUse(t, dicetest.NewFixedRoller(c.faces...)).Resolve(context.Background(), c.cmd)
			_, opts := recording()
			loud, err2 := otUse(t, dicetest.NewFixedRoller(c.faces...), opts...).Resolve(context.Background(), c.cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(quiet, loud) {
				t.Errorf("results differ with telemetry: %v, %v and %#v, %#v", err, err2, quiet, loud)
			}
			if !reflect.DeepEqual(before.Interface(), c.cmd) {
				t.Error("the command was changed")
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-090, FR-094 - the command and result sets are closed: every command variant returns one of the result variants, a Reincarnate the rules refuse is SpellTableRefused with its refusals as a copy, and the use case works through an exhaustive switch over the results.
// Path: happy.
func TestResolveSpellOutcomeTableResultVariants(t *testing.T) {
	t.Parallel()
	faces := map[string][]int{"teleport": {25}, "confusion": {9}, "spray": {6, 5}, "wall": {5, 5, 5, 5, 5, 5, 5}, "reincarnate": {2}, "blink": {1}}
	faces["wall"] = slices.Concat(otRepeat(1, 5), otRepeat(12, 1), otRepeat(1, 5), otRepeat(12, 1), otRepeat(1, 5), otRepeat(12, 1), otRepeat(1, 5), otRepeat(12, 1), otRepeat(1, 5), otRepeat(12, 1), []int{5, 5})
	for name, cmd := range otEveryCommand() {
		res, err := otUse(t, dicetest.NewFixedRoller(faces[name]...)).Resolve(context.Background(), cmd)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		kind := ""
		switch res.(type) {
		case combat.SpellTableTeleportRolled:
			kind = "teleport"
		case combat.SpellTableConfusionRolled:
			kind = "confusion"
		case combat.SpellTablePrismaticSprayRolled:
			kind = "spray"
		case combat.SpellTablePrismaticWallCrossed:
			kind = "wall"
		case combat.SpellTableReincarnationRolled:
			kind = "reincarnate"
		case combat.SpellTableBlinkRolled:
			kind = "blink"
		case combat.SpellTableRefused:
			kind = "refused"
		}
		if kind != name {
			t.Errorf("%s command gave %#v", name, res)
		}
	}
	res, err := otUse(t, dicetest.NewFixedRoller()).Resolve(context.Background(), combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeBeast, DeadFor: 1<<62 - 1})
	refused, ok := res.(combat.SpellTableRefused)
	if err != nil || !ok || !slices.Equal(refused.Refusals(), []combat.SpellTableRefusal{combat.NotHumanoid, combat.DeadTooLong}) {
		t.Fatalf("Resolve = %#v, %v, want a refusal of both reasons", res, err)
	}
	refused.Refusals()[0] = combat.DeadTooLong
	if !slices.Equal(refused.Refusals(), []combat.SpellTableRefusal{combat.NotHumanoid, combat.DeadTooLong}) {
		t.Error("Refusals must return a copy")
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-005, FR-006 - a refusal and a malformed command request no die, and a Prismatic Spray with a ChooseToFail target rolls no d20; the cancelled context is the only other path with no die.
// Path: unhappy (determinism).
func TestRefusedAndMalformedCommandsRollNothing(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller()
	use := otUse(t, roller)
	if _, err := use.Resolve(context.Background(), combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeFey}); err != nil {
		t.Fatal(err)
	}
	if _, err := use.Resolve(context.Background(), combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: []combat.PrismaticSaveTarget{{ID: "a", Attempt: d20.ChooseToFail{}}, {ID: "a", Attempt: d20.ChooseToFail{}}}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("a repeated id = %v", err)
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("requests = %v, want none", roller.Requests())
	}
}
