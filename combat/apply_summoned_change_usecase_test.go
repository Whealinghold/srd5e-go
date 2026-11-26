package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// sumChangeRun builds one call of ApplySummonedChange.
func sumChangeRun(tb testing.TB, roster combat.SummonedRoster, change combat.SummonedChange, active string) func(...combat.Option) (any, error) {
	tb.Helper()
	cmd, err := combat.NewApplySummonedChangeCommand(roster, change, combat.CombatantID(active))
	if err != nil {
		tb.Fatalf("NewApplySummonedChangeCommand(%T): %v", change, err)
	}
	return func(opts ...combat.Option) (any, error) {
		res, err := combat.NewApplySummonedChange(opts...).Run(context.Background(), cmd)
		if res == nil {
			return nil, err
		}
		return res, err
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-072, FR-073, FR-074, section 7 - ApplySummonedChange records the shared signals under apply_summoned_change with srd5e.combat.result commanded, departed, lapsed, elapsed, dismissed, recalled, repaired, touch_delivered, senses_shared, life_bond, turn_reported, task_completed, action_allowed, unchanged (class applied) or rejected_<reason> (class refused), counts one srd5e.combat.summon_endings for each creature that left or whose control lapsed, carries no id or number as an attribute, logs nothing and behaves the same with the Noop and the recording telemetry.
// Path: per case.
func TestApplySummonedChangeSignals(t *testing.T) {
	t.Parallel()
	const op = "apply_summoned_change"
	world := sumWorld(t)
	pocket := sumApplied(t, sumChange(t, world, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	wild := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", combat.FamiliarOriginWildCompanion)
	linked := sumApplied(t, sumChange(t, world, combat.ShareFamiliarSenses{Summoner: cid("pc")}, "")).Roster()
	ordered := sumApplied(t, sumChange(t, world, sumCommand(combat.SummonAnimatedObject, 0, "o1"), "")).Roster()
	standing := sumApplied(t, sumChange(t, world, sumCommand(combat.SummonAnimatedUndead, 0, "u1"), "")).Roster()
	const a, r = "applied", "refused"
	type end = string
	rows := []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		change                  combat.SummonedChange
		class, result           string
		events                  []string
		endings                 []end
	}{
		{"an order", "FR-072", "happy", world, sumCommand(combat.SummonAnimatedObject, 0, "o1", "o2"), a, "commanded", []string{"summoned_creature_commanded"}, nil},
		{"an insect at 0 Hit Points", "FR-072", "happy", world, combat.ReportZeroHitPoints{Creature: cid("i1")}, a, "departed", []string{"summoned_creature_departed"}, []end{"giant_insect:dropped_to_zero_hit_points"}},
		{"a spell that ends", "FR-072", "happy", world, combat.ReportSpellEnded{Summoner: cid("pc"), Spell: "summon_dragon", Cause: combat.SummonSpellEndConcentrationEnded}, a, "departed", []string{"summoned_creature_departed", "shared_resistance_ended"}, []end{"draconic_spirit:spell_ended"}},
		{"the summoner dying", "FR-072", "happy", world, combat.ReportSummonerDied{Summoner: cid("pc")}, a, "departed", []string{"summoned_creature_departed"}, []end{"otherworldly_steed:summoner_died"}},
		{"a Long Rest", "FR-072", "happy", wild, combat.ReportLongRestFinished{Summoner: cid("pc")}, a, "departed", []string{"summoned_creature_departed"}, []end{"familiar:long_rest_finished"}},
		{"time that lapses control", "FR-072", "unhappy (boundary)", world, combat.ElapseControlTime{Elapsed: 24 * time.Hour}, a, "lapsed", []string{"summon_control_lapsed", "summon_control_lapsed"}, []end{"animated_undead:control_lapsed", "animated_undead:control_lapsed"}},
		{"time that does not", "FR-072", "unhappy (boundary)", world, combat.ElapseControlTime{Elapsed: time.Hour}, a, "elapsed", nil, nil},
		{"a temporary dismissal", "FR-072", "happy", world, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, a, "dismissed", []string{"familiar_dismissed_temporarily"}, nil},
		{"a dismissal for good", "FR-072", "happy", world, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalForever}, a, "dismissed", []string{"summoned_creature_departed"}, []end{"familiar:dismissed_forever"}},
		{"a recall", "FR-072", "happy", pocket, combat.RecallFamiliar{Summoner: cid("pc"), SpaceAvailable: true}, a, "recalled", []string{"familiar_reappeared"}, nil},
		{"a sight link", "FR-072", "happy", world, combat.ShareFamiliarSenses{Summoner: cid("pc")}, a, "senses_shared", []string{"familiar_sight_link_started"}, nil},
		{"a touch", "FR-072", "happy", world, combat.DeliverTouch{Summoner: cid("pc"), SpellIsTouch: true, Familiar: combat.ConditionActionAvailabilityAvailable}, a, "touch_delivered", nil, nil},
		{"a repair", "FR-072", "happy", world, combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Missing: 2, Restore: 2, LongRestTaken: true}, a, "repaired", []string{"simulacrum_repaired"}, nil},
		{"a Life Bond", "FR-072", "happy", world, combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), Regained: 4, SpellLevel: 1}, a, "life_bond", nil, nil},
		{"the start of a turn", "FR-072", "happy", linked, combat.ReportTurnStarted{Combatant: cid("pc")}, a, "turn_reported", []string{"familiar_sight_link_ended"}, nil},
		{"the end of a turn", "FR-013", "happy", ordered, combat.ReportTurnEnded{Creature: cid("o1")}, a, "turn_reported", nil, nil},
		{"a task completed", "FR-013", "happy", standing, combat.ReportTaskComplete{Creature: cid("u1")}, a, "task_completed", nil, nil},
		{"an action", "FR-048", "happy", world, combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionDodge}, a, "action_allowed", nil, nil},
		{"a creature that is not there", "FR-026", "unhappy (state error)", world, combat.ReportZeroHitPoints{Creature: cid("ghost")}, a, "unchanged", nil, nil},
		{"an order out of range", "FR-009", "unhappy (boundary)", world, sumCommand(combat.SummonAnimatedObject, 501, "o1"), r, "rejected_command_out_of_range", nil, nil},
		{"an order to a stranger", "FR-014", "unhappy (invalid input)", world, sumCommand(combat.SummonAnimatedObject, 0, "ghost"), r, "rejected_command_not_your_creature", nil, nil},
		{"an attack by a familiar", "FR-048", "unhappy (rule interaction)", world, combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionAttack}, r, "rejected_familiar_cannot_attack", nil, nil},
		{"a recall of a familiar that is here", "FR-051", "unhappy (state error)", world, combat.RecallFamiliar{Summoner: cid("pc"), SpaceAvailable: true}, r, "rejected_familiar_not_dismissed", nil, nil},
		{"a touch that cannot be delivered", "FR-047", "unhappy (state error)", world, combat.DeliverTouch{Summoner: cid("pc"), Familiar: combat.ConditionActionAvailabilityAvailable}, r, "rejected_touch_not_deliverable", nil, nil},
		{"a Life Bond out of reach", "FR-057", "unhappy (boundary)", world, combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), Regained: 4, SpellLevel: 1, DistanceFeet: 6}, r, "rejected_life_bond_out_of_reach", nil, nil},
	}
	for _, c := range rows {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			run := sumChangeRun(t, c.roster, c.change, "")
			got, err := run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if plain, err := run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-074]", got, plain, err)
			}
			outcome := "failure"
			if c.class == a {
				outcome = "success"
			}
			want := map[string]any{"srd5e.combat.operation": op, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status [FR-074; %s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [FR-074]", sa, want)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [FR-074]", ops, want)
			}
			wantKinds := slices.Clone(c.events)
			slices.Sort(wantKinds)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(counted, wantKinds) {
				t.Errorf("event kinds counted = %v, want %v [FR-069]", counted, wantKinds)
			}
			var endings []string
			for _, m := range measured(rec, "srd5e.combat.summon_endings") {
				endings = append(endings, m["srd5e.combat.summon_kind"].(string)+":"+m["srd5e.combat.summon_end_cause"].(string))
			}
			if !slices.Equal(endings, c.endings) {
				t.Errorf("endings counted = %v, want %v [section 7]", endings, c.endings)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.summons")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("a change counted %d summons or errors, want none", n)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			checkTelemetryHygiene(t, rec)
			for _, m := range rec.Meter.Measurements() {
				for _, at := range m.Attrs {
					for _, id := range []string{"pc", "o1", "o2", "i1", "u1", "fam", "st", "sim", "ghost"} {
						if at.Value() == id {
							t.Errorf("%s attribute %s carries the id %q [observability.md section 5]", m.Name, at.Key(), id)
						}
					}
				}
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-072, FR-073, FR-077 - each result of ApplySummonedChange holds what it names: Applied the new roster, the events, the instructions for the encounter's roster, the familiar's Reaction of a touch and the Hit Points of a Life Bond; Rejected the SummonRejection and the roster that came in; Unchanged the roster that came in; and the active combatant of the command decides whether a departure is pending.
// Path: per case.
func TestApplySummonedChangeResults(t *testing.T) {
	t.Parallel()
	world := sumWorld(t)
	prior := sumWorld(t)
	ctx := context.Background()
	run := func(c combat.SummonedChange, active string) combat.ApplySummonedChangeResult {
		cmd, err := combat.NewApplySummonedChangeCommand(world, c, combat.CombatantID(active))
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewApplySummonedChange().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	a, ok := run(combat.ReportZeroHitPoints{Creature: cid("i1")}, "i1").(combat.ApplySummonedChangeApplied)
	if !ok || a.Roster().Len() != world.Len()-1 || len(a.Events()) != 1 || !a.Events()[0].(combat.SummonedCreatureDeparted).PendingRemoval() ||
		!reflect.DeepEqual(a.RosterActions(), []combat.SummonedRosterAction{combat.SummonedRosterLeave{Creature: cid("i1"), Cause: combat.RemovalCauseDisappeared, AfterTurn: true}}) {
		t.Errorf("result = %#v, want the departure of the active insect, pending [FR-077]", a)
	}
	a2 := run(combat.ReportZeroHitPoints{Creature: cid("i1")}, "pc").(combat.ApplySummonedChangeApplied)
	if a2.Events()[0].(combat.SummonedCreatureDeparted).PendingRemoval() {
		t.Errorf("a departure of a creature that is not active is pending")
	}
	events, actions := a.Events(), a.RosterActions()
	events[0], actions[0] = nil, nil
	if a.Events()[0] == nil || a.RosterActions()[0] == nil {
		t.Errorf("a returned slice changed the result [FR-065]")
	}
	touch := run(combat.DeliverTouch{Summoner: cid("pc"), SpellIsTouch: true, Familiar: combat.ConditionActionAvailabilityAvailable}, "").(combat.ApplySummonedChangeApplied)
	if r, has := touch.Reaction(); !has || r.Feature != "find_familiar" || touch.LifeBondHitPoints() != 0 {
		t.Errorf("touch = %v, %v, want the familiar's Reaction [FR-047]", r, has)
	}
	bond := run(combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), Regained: 9, SpellLevel: 2}, "").(combat.ApplySummonedChangeApplied)
	if _, has := bond.Reaction(); has || bond.LifeBondHitPoints() != 9 {
		t.Errorf("Life Bond = %d, want 9 Hit Points [FR-057]", bond.LifeBondHitPoints())
	}
	rej, ok := run(sumCommand(combat.SummonAnimatedObject, 600, "o1"), "").(combat.ApplySummonedChangeRejected)
	if !ok || rej.Rejection != (combat.CommandOutOfRangeRejection{Creature: cid("o1"), RangeFeet: 500}) || !rej.Roster().Equal(prior) {
		t.Errorf("result = %#v, want ApplySummonedChangeRejected with the roster unchanged [FR-073]", rej)
	}
	un, ok := run(combat.ReportSummonerDied{Summoner: cid("nobody")}, "").(combat.ApplySummonedChangeUnchanged)
	if !ok || !un.Roster().Equal(prior) {
		t.Errorf("result = %#v, want ApplySummonedChangeUnchanged [FR-026]", un)
	}
	if !world.Equal(prior) {
		t.Errorf("a use case altered the roster of its command [FR-065]")
	}
	pending := sumApplied(t, sumChange(t, world, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "fam"))
	if !pending.Events()[0].(combat.FamiliarDismissedTemporarily).PendingRemoval() {
		t.Errorf("a dismissal of the active familiar is not pending [FR-077]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-067, errors-and-results.md FR-051 - the zero command is invalid input before any signal, an already cancelled ctx returns ctx.Err() unwrapped before any signal, the zero use case is silent and works, and a panic in a telemetry backend propagates with the span ended and no error counted.
// Path: unhappy (failure).
func TestApplySummonedChangeGuards(t *testing.T) {
	t.Parallel()
	world := sumWorld(t)
	cmd, _ := combat.NewApplySummonedChangeCommand(world, combat.ReportZeroHitPoints{Creature: cid("i1")}, "")
	rec, opts := recording()
	uc := combat.NewApplySummonedChange(opts...)
	if res, err := uc.Run(context.Background(), combat.ApplySummonedChangeCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero command = %v, %v, want srderr.ErrInvalidInput", res, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled ctx = %v, %v, want a nil result and context.Canceled itself", res, err)
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("a refused call recorded %d things, want none", n)
	}
	var zero combat.ApplySummonedChange
	if res, err := zero.Run(context.Background(), cmd); err != nil || res == nil {
		t.Errorf("the zero use case = %v, %v, want it to work silently", res, err)
	}
	loud := telemetrytest.NewRecorder()
	bad := combat.NewApplySummonedChange(combat.WithTelemetry(telemetry.Telemetry{Tracer: loud.Tracer, Meter: panicMeter{}}))
	func() {
		defer func() {
			if r := recover(); r == nil || r.(error).Error() != "backend bug" {
				t.Errorf("recovered %v, want the original panic", r)
			}
		}()
		_, _ = bad.Run(context.Background(), cmd)
	}()
	if spans := loud.Tracer.SpansNamed("srd5e.combat.apply_summoned_change"); len(spans) != 1 || !spans[0].Ended {
		t.Errorf("spans = %+v, want one ended span after the panic", spans)
	}
	if n := len(loud.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
		t.Errorf("a telemetry panic counted %d errors", n)
	}
}
