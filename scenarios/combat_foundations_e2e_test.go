package scenarios_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// The signal names are a public contract (observability.md section 5), so the scenario names them as literals.
const (
	grpCOperation    = "srd5e.combat.operation"
	grpCResultAttr   = "srd5e.combat.result"
	grpCClassAttr    = "srd5e.combat.result_class"
	grpCEventAttr    = "srd5e.combat.event"
	grpCOutcomeAttr  = "srd5e.outcome"
	grpCErrorType    = "error.type"
	grpCOperations   = "srd5e.combat.operations"
	grpCDuration     = "srd5e.combat.operation_duration"
	grpCErrorsMetric = "srd5e.combat.errors"
	grpCEventsMetric = "srd5e.combat.events"
)

// grpCRecording returns a recorder and the options that wire it and a deterministic clock into a use case.
func grpCRecording() (*telemetrytest.Recorder, []combat.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 2*time.Second)
	return rec, []combat.Option{combat.WithTelemetry(rec.Telemetry()), combat.WithClock(clock.Now)}
}

func grpCAttrs(as []telemetry.Attr) map[string]any {
	m := map[string]any{}
	for _, a := range as {
		m[a.Key()] = a.Value()
	}
	return m
}

// grpCMeasured returns the attribute maps of every measurement of the named metric.
func grpCMeasured(rec *telemetrytest.Recorder, name string) []map[string]any {
	var out []map[string]any
	for _, m := range rec.Meter.MeasurementsNamed(name) {
		out = append(out, grpCAttrs(m.Attrs))
	}
	return out
}

// grpCRound is what a host sees of one round: the event kinds of each step and the state it ends with. Two runs that
// differ only in their telemetry must give equal rounds (observability.md section 9).
type grpCRound struct {
	steps      [][]string
	hp         map[string]int
	round      int
	active     combat.CombatantID
	conditions []content.ConditionKind
}

// grpCPlayRound runs an encounter round through the use cases built with opts: start, damage, a condition, the
// same condition again (refused), healing of a combatant that is not in the encounter (invalid) and turn advances.
// It returns the round and the encounter.
func grpCPlayRound(t *testing.T, opts ...combat.Option) (grpCRound, combat.Encounter) {
	t.Helper()
	ctx := context.Background()
	hero, ogre := grpCID("hero"), grpCID("ogre")
	speed, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: hero, Count: 20}, {Combatant: ogre, Count: 10}},
		[]combat.CombatantProfile{{ID: hero, Size: content.SizeMedium, Speeds: speed}, {ID: ogre, Size: content.SizeLarge, Speeds: speed}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	est, err := combat.NewEstablishInitiativeOrder(opts...).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	var r grpCRound
	established := []string{}
	for _, ev := range est.(combat.InitiativeOrderEstablished).Events() {
		established = append(established, ev.Kind())
	}
	r.steps = append(r.steps, established)
	start := combat.NewStartEncounterCommand(combat.MustEncounterID("grpc"), est.(combat.InitiativeOrderEstablished),
		[]combat.Combatant{grpCPlayer(t, "hero", combat.SideA, 20), grpCPlayer(t, "ogre", combat.SideB, 59)})
	started, err := combat.NewStartEncounter(opts...).Run(ctx, start)
	if err != nil {
		t.Fatal(err)
	}
	e := started.(combat.StartEncounterApplied).Encounter()
	r.steps = append(r.steps, grpCKinds(started.(combat.StartEncounterApplied).Events()))

	dmg, err := combat.NewApplyDamageToCombatantCommand(e, hero, []combat.Damage{combat.MustDamage(7, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	dr, err := combat.NewApplyDamageToCombatant(opts...).Run(ctx, dmg)
	if err != nil {
		t.Fatal(err)
	}
	e = dr.(combat.ApplyDamageToCombatantApplied).Encounter()
	r.steps = append(r.steps, grpCKinds(dr.(combat.ApplyDamageToCombatantApplied).Events()))

	for range 2 { // the second application is refused: the creature already has the condition
		cond, err := combat.NewApplyConditionToCombatantCommand(e, hero, combat.Paralyzed{})
		if err != nil {
			t.Fatal(err)
		}
		cr, err := combat.NewApplyConditionToCombatant(opts...).Run(ctx, cond)
		if err != nil {
			t.Fatal(err)
		}
		switch v := cr.(type) {
		case combat.ApplyConditionToCombatantApplied:
			e = v.Encounter()
			r.steps = append(r.steps, grpCKinds(v.Events()))
		case combat.ApplyConditionToCombatantRejected:
			r.steps = append(r.steps, []string{"rejected"})
		}
	}

	ghost, err := combat.NewApplyHealingToCombatantCommand(e, grpCID("ghost"), 5)
	if err != nil {
		t.Fatal(err)
	}
	gr, err := combat.NewApplyHealingToCombatant(opts...).Run(ctx, ghost)
	if err != nil {
		t.Fatal(err)
	}
	if _, rejected := gr.(combat.ApplyHealingToCombatantRejected); !rejected {
		t.Fatalf("healing an unknown combatant = %#v, want the rejection that carries the unknown-combatant problem", gr)
	}
	r.steps = append(r.steps, []string{"rejected"})

	adv, err := combat.NewAdvanceEncounterTurnCommand(e, combat.TurnAdvance{})
	if err != nil {
		t.Fatal(err)
	}
	ar, err := combat.NewAdvanceEncounterTurn(opts...).Run(ctx, adv)
	if err != nil {
		t.Fatal(err)
	}
	e = ar.(combat.AdvanceEncounterTurnApplied).Encounter()
	r.steps = append(r.steps, grpCKinds(ar.(combat.AdvanceEncounterTurnApplied).Events()))

	r.hp = map[string]int{}
	for _, id := range []string{"hero", "ogre"} {
		m, _ := e.Find(grpCID(id))
		r.hp[id] = m.HitPoints().HitPoints().Current()
	}
	r.round, r.active = e.Tracker().Round(), e.Tracker().Active().Combatant
	hm, _ := e.Find(hero)
	r.conditions = slices.Clone(hm.Conditions().Effective())
	return r, e
}

// Requirement: doc/specs/combat-foundations.md FR-029, FR-030, FR-031, FR-033, FR-034, FR-036 and doc/rules/observability.md section 9 - one encounter round through the use cases gives the same results with the zero Noop bundle and with a recording bundle; each call is one span named srd5e.combat.<operation>, counted once under operations with the operation, the outcome, the result and its class, each event is counted under its kind, the duration is recorded once per call, nothing is logged, nothing is an error, and no attribute holds an id.
// Path: happy.
func TestGrpCCombatRoundEmitsTheSharedSignalsAndChangesNothing(t *testing.T) {
	t.Parallel()
	silent, silentEnc := grpCPlayRound(t)
	rec, opts := grpCRecording()
	loud, loudEnc := grpCPlayRound(t, opts...)

	if fmt.Sprintf("%v|%v|%d|%v|%v", silent.steps, silent.hp, silent.round, silent.active, silent.conditions) !=
		fmt.Sprintf("%v|%v|%d|%v|%v", loud.steps, loud.hp, loud.round, loud.active, loud.conditions) {
		t.Errorf("the round differs with telemetry: silent %+v, recorded %+v", silent, loud)
	}
	if !silentEnc.Equal(loudEnc) {
		t.Errorf("the final encounters differ with telemetry")
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
	if recs := rec.Logger.Records(); len(recs) != 0 {
		t.Errorf("logged %v, want nothing for rule outcomes (FR-034)", recs)
	}

	// One span per call, named for the operation, ended, in error status only for the invalid result.
	want := []struct {
		name  string
		error bool
	}{
		{"srd5e.combat.establish_initiative_order", false},
		{"srd5e.combat.start_encounter", false},
		{"srd5e.combat.apply_damage_to_combatant", false},
		{"srd5e.combat.apply_condition_to_combatant", false},
		{"srd5e.combat.apply_condition_to_combatant", false}, // refused: a normal outcome, no error status
		{"srd5e.combat.apply_healing_to_combatant", true},    // unknown combatant: invalid, error status
		{"srd5e.combat.advance_encounter_turn", false},
	}
	spans := rec.Tracer.Spans()
	if len(spans) != len(want) {
		t.Fatalf("%d spans, want %d: %v", len(spans), len(want), spans)
	}
	for i, w := range want {
		s := spans[i]
		if s.Name != w.name || !s.Ended || (s.Status == telemetry.StatusError) != w.error || len(s.Errors) != 0 {
			t.Errorf("span %d = %s ended %v status %v errors %v, want %s ended, error status %v, no recorded error", i, s.Name, s.Ended, s.Status, s.Errors, w.name, w.error)
		}
	}

	// operations: one sample per call that returned a result, by operation, outcome, result and class.
	ops := grpCMeasured(rec, grpCOperations)
	type key struct{ op, result, class, outcome string }
	var got []key
	for _, o := range ops {
		got = append(got, key{o[grpCOperation].(string), o[grpCResultAttr].(string), o[grpCClassAttr].(string), o[grpCOutcomeAttr].(string)})
	}
	for _, k := range []key{
		{"start_encounter", "applied", "applied", "success"},
		{"apply_damage_to_combatant", "applied", "applied", "success"},
		{"apply_condition_to_combatant", "applied", "applied", "success"},
		{"apply_condition_to_combatant", "already_has_condition", "refused", "failure"},
		{"apply_healing_to_combatant", "unknown_combatant", "invalid", "failure"},
		{"advance_encounter_turn", "applied", "applied", "success"},
	} {
		if !slices.Contains(got, k) {
			t.Errorf("operations samples %v lack %+v", got, k)
		}
	}
	// One duration per call, in seconds, with the operation attribute.
	durations := rec.Meter.MeasurementsNamed(grpCDuration)
	if len(durations) != len(want) {
		t.Errorf("%d duration samples, want one per call (%d)", len(durations), len(want))
	}
	for _, d := range durations {
		if d.Unit != "s" || d.Value <= 0 {
			t.Errorf("duration %+v, want a positive value in seconds", d)
		}
	}
	// The events are counted under their kind, once each.
	counted := map[string]int{}
	for _, e := range grpCMeasured(rec, grpCEventsMetric) {
		counted[e[grpCEventAttr].(string)]++
	}
	produced := map[string]int{}
	for _, step := range loud.steps {
		for _, k := range step {
			if k != "rejected" {
				produced[k]++
			}
		}
	}
	for k, n := range produced {
		if counted[k] != n {
			t.Errorf("event %q counted %d times, want %d (the host saw it %d times)", k, counted[k], n, n)
		}
	}
	// A rule outcome is no error and an id is no attribute value (FR-032, FR-033).
	if errs := grpCMeasured(rec, grpCErrorsMetric); len(errs) != 0 {
		t.Errorf("errors counted %v for a round with no error", errs)
	}
	forbidden := []string{"hero", "ogre", "ghost", "grpc"}
	check := func(where string, as []telemetry.Attr) {
		for _, a := range as {
			for _, f := range forbidden {
				if v, ok := a.Value().(string); ok && v == f {
					t.Errorf("%s holds the id %q under %s", where, f, a.Key())
				}
			}
		}
	}
	for _, s := range rec.Tracer.Spans() {
		check("span "+s.Name, s.Attrs)
	}
	for _, m := range rec.Meter.Measurements() {
		check("metric "+m.Name, m.Attrs)
	}
}

// Requirement: doc/specs/combat-foundations.md FR-032, FR-029, FR-035 and doc/specs/monster-combatant-setup.md FR-040 - an error that matches srderr.ErrInvalidInput is counted once under errors with error.type invalid_input and the operation, is recorded once on the span with error status, produces no operations or events sample and still records the duration; a roller failure is counted once with its port's error type and is not logged.
// Path: unhappy (failure).
func TestGrpCCombatErrorsAreCountedOnceWithTheirType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Invalid input: the zero command of a use case.
	rec, opts := grpCRecording()
	_, err := combat.NewTeleportCombatant(opts...).Run(ctx, combat.TeleportCombatantCommand{})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("zero command error = %v, want ErrInvalidInput", err)
	}
	errs := grpCMeasured(rec, grpCErrorsMetric)
	if len(errs) != 1 || errs[0][grpCErrorType] != "invalid_input" || errs[0][grpCOperation] != "teleport_combatant" {
		t.Errorf("errors = %v, want one with error.type invalid_input for teleport_combatant", errs)
	}
	grpCCheckErrorPath(t, rec, "srd5e.combat.teleport_combatant", err)

	// A roller failure: the dice port's error type, from a use case that rolls.
	rec2, opts2 := grpCRecording()
	cause := errors.New("dice service down")
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })
	uc, err := combat.NewSetUpMonsterCombatant(failing, opts2...)
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.Run(ctx, combat.SetUpMonsterCombatantCommand{Monster: grpCSetUpMonster(), HitPoints: combat.RollHitPointDice{}})
	var port *srderr.PortError
	if !errors.As(err, &port) || port.Port != dice.Port || !errors.Is(err, cause) {
		t.Fatalf("roller failure = %v, want a dice PortError wrapping the cause", err)
	}
	errs = grpCMeasured(rec2, grpCErrorsMetric)
	if len(errs) != 1 || errs[0][grpCErrorType] != "dice_roller_error" || errs[0][grpCOperation] != "set_up_monster_combatant" {
		t.Errorf("errors = %v, want one with error.type dice_roller_error for set_up_monster_combatant", errs)
	}
	grpCCheckErrorPath(t, rec2, "srd5e.combat.set_up_monster_combatant", err)
}

// grpCCheckErrorPath checks what every returned error leaves behind: one span with error status that holds the error
// once, a duration, no operations or events sample, nothing logged and no misuse of the ports.
func grpCCheckErrorPath(t *testing.T, rec *telemetrytest.Recorder, span string, err error) {
	t.Helper()
	spans := rec.Tracer.SpansNamed(span)
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !errors.Is(spans[0].Errors[0], err) || !spans[0].Ended {
		t.Errorf("spans = %+v, want one ended span in error status holding the error once", spans)
	}
	if n := len(rec.Meter.MeasurementsNamed(grpCOperations)) + len(rec.Meter.MeasurementsNamed(grpCEventsMetric)); n != 0 {
		t.Errorf("%d operations or events samples on the error path, want none", n)
	}
	if d := rec.Meter.MeasurementsNamed(grpCDuration); len(d) != 1 {
		t.Errorf("%d duration samples, want 1", len(d))
	}
	if recs := rec.Logger.Records(); len(recs) != 0 {
		t.Errorf("logged %v, want nothing (an error that is returned is not also logged)", recs)
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

// Requirement: doc/specs/combat-foundations.md FR-029, FR-031, FR-036 and doc/specs/monster-combatant-setup.md FR-044, FR-048, FR-049, FR-050 - the monster setup use case gives an equal result with the Noop bundle and with the recorder, records its result and the hit-points method (and only that extra) on the span and the operations sample, and counts the clamp of a rolled total raised to 1 once, with no attribute that holds an id or an amount.
// Path: happy.
func TestGrpCMonsterSetupSignalsAreTheSpecifiedOnes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m := grpCMonster(func(f *content.MonsterFields) {
		f.HitPoints = grpCMust(content.NewHitPointsEntry(1, dice.MustExpression(1, dice.D4(), -3)))
	})
	cmd := combat.SetUpMonsterCombatantCommand{Monster: m, HitPoints: combat.RollHitPointDice{}}

	plain, err := grpCMust(combat.NewSetUpMonsterCombatant(dicetest.NewFixedRoller(1))).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	rec, opts := grpCRecording()
	recorded, err := grpCMust(combat.NewSetUpMonsterCombatant(dicetest.NewFixedRoller(1), opts...)).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !plain.(combat.SetUpMonsterCombatantApplied).Equal(recorded.(combat.SetUpMonsterCombatantApplied)) {
		t.Errorf("the result differs with telemetry")
	}
	ops := grpCMeasured(rec, grpCOperations)
	if len(ops) != 1 || ops[0][grpCOperation] != "set_up_monster_combatant" || ops[0][grpCResultAttr] != "set_up_rolled" ||
		ops[0][grpCClassAttr] != "applied" || ops[0]["srd5e.combat.monster_hit_points_method"] != "rolled" {
		t.Errorf("operations = %v, want one applied set_up_rolled with the rolled method", ops)
	}
	span := rec.Tracer.SpansNamed("srd5e.combat.set_up_monster_combatant")
	if len(span) != 1 || grpCAttrs(span[0].Attrs)["srd5e.combat.monster_hit_points_method"] != "rolled" {
		t.Errorf("span = %+v, want the method on the span too", span)
	}
	if clamped := rec.Meter.MeasurementsNamed("srd5e.combat.monster_hit_points_clamped"); len(clamped) != 1 || len(clamped[0].Attrs) != 0 {
		t.Errorf("clamp samples = %v, want one with no attributes", clamped)
	}
	if got := rec.Meter.MeasurementsNamed(grpCDuration); len(got) != 1 {
		t.Errorf("%d duration samples, want 1", len(got))
	}
	// The average path counts no clamp.
	rec3, opts3 := grpCRecording()
	if _, err := grpCMust(combat.NewSetUpMonsterCombatant(dicetest.NewMinimumRoller(), opts3...)).Run(ctx,
		combat.SetUpMonsterCombatantCommand{Monster: m, HitPoints: combat.UseAverageHitPoints{}}); err != nil {
		t.Fatal(err)
	}
	if n := len(rec3.Meter.MeasurementsNamed("srd5e.combat.monster_hit_points_clamped")); n != 0 {
		t.Errorf("the average path counted %d clamps, want none", n)
	}
	if got := grpCMeasured(rec3, grpCOperations); len(got) != 1 || got[0]["srd5e.combat.monster_hit_points_method"] != "average" || got[0][grpCResultAttr] != "set_up_average" {
		t.Errorf("average operations = %v, want set_up_average with the average method", got)
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-038, FR-039 and doc/specs/monster-stat-block-actions.md FR-047 - the turn start and the stat-block damage use cases record their operations under the shared signals, count each Recharge roll by its outcome with no id attribute, and give equal results with the Noop bundle and the recorder.
// Path: happy.
func TestGrpCMonsterUseCasesRecordTheirOperations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec, opts := grpCRecording()

	s := grpCSpent(t, grpCStart(t, combat.OutsideLair), grpCSpray)
	plain, err := grpCMust(combat.NewApplyMonsterTurnStart(dicetest.NewFixedRoller(5))).Run(ctx, combat.ApplyMonsterTurnStartCommand{State: s})
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := grpCMust(combat.NewApplyMonsterTurnStart(dicetest.NewFixedRoller(5), opts...)).Run(ctx, combat.ApplyMonsterTurnStartCommand{State: s})
	if err != nil {
		t.Fatal(err)
	}
	if !plain.(combat.MonsterTurnStartApplied).State().Equal(recorded.(combat.MonsterTurnStartApplied).State()) {
		t.Errorf("the turn start result differs with telemetry")
	}
	if ops := grpCMeasured(rec, grpCOperations); len(ops) != 1 || ops[0][grpCOperation] != "apply_monster_turn_start" || ops[0][grpCResultAttr] != "turn_start_recharged" {
		t.Errorf("operations = %v, want one apply_monster_turn_start with turn_start_recharged", ops)
	}
	rolls := grpCMeasured(rec, "srd5e.combat.monster_recharge_rolls")
	if len(rolls) != 1 || rolls[0]["srd5e.combat.recharge_outcome"] != "regained" {
		t.Errorf("recharge roll samples = %v, want one regained", rolls)
	}

	instances := []content.StatBlockDamage{grpCDamage(5, dice.MustExpression(1, dice.D8(), 1), content.DamageFire)}
	cmd := combat.RollStatBlockDamageCommand{Damage: instances, Choices: grpCMust(combat.NewDamageNotationChoices(combat.DamageNotationNumber))}
	a, err := grpCMust(combat.NewRollStatBlockDamage(dicetest.NewMinimumRoller())).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	b, err := grpCMust(combat.NewRollStatBlockDamage(dicetest.NewMinimumRoller(), opts...)).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.(combat.StatBlockDamageRolled).Damage(), b.(combat.StatBlockDamageRolled).Damage()) {
		t.Errorf("the damage differs with telemetry")
	}
	var dmg map[string]any
	for _, o := range grpCMeasured(rec, grpCOperations) {
		if o[grpCOperation] == "roll_stat_block_damage" {
			dmg = o
		}
	}
	if dmg == nil || dmg[grpCResultAttr] != "stat_block_damage_fixed" || dmg["srd5e.combat.damage_notation"] != "number" {
		t.Errorf("roll_stat_block_damage sample = %v, want stat_block_damage_fixed with the number notation", dmg)
	}
}
