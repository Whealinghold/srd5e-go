package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/conditions.md section 8 scenario 1; FR-016, FR-029, FR-077, FR-090, FR-105 (SRD 5.2.1,
// "Grappled" p. 182, "Unconscious" p. 191, "Paralyzed" p. 186) - a grappled creature is knocked Unconscious, is attacked
// from adjacent (an automatic Critical Hit with Advantage) and from range (Advantage cancelled by the Disadvantage of
// Prone), and wakes Prone, through the public API only.
// Path: happy.
func TestGrappledCreatureKnockedUnconsciousAndWakesProne(t *testing.T) {
	t.Parallel()
	ctx, hero, orc := context.Background(), combat.MustCombatantID("hero"), combat.MustCombatantID("orc")
	rec := telemetrytest.NewRecorder()
	opts := []combat.Option{combat.WithTelemetry(rec.Telemetry())}
	var fx combat.ConditionEffects

	apply := func(s combat.ConditionSet, c combat.Condition) combat.ConditionSet {
		cmd, err := combat.NewApplyConditionToSetCommand(s, hero, c)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewApplyConditionToSet(opts...).Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.Set()
	}
	set := apply(apply(combat.ConditionSet{}, combat.Grappled{Grappler: orc}), combat.Unconscious{})
	if got := set.Effective(); !slices.Equal(got, []content.ConditionKind{content.ConditionGrappled, content.ConditionIncapacitated, content.ConditionProne, content.ConditionUnconscious}) {
		t.Fatalf("effective after Grappled and Unconscious = %v", got)
	}
	if sp, _ := fx.Speed(set, 30); sp.Value != 0 || sp.Increase != combat.SpeedIncreaseForbidden {
		t.Errorf("Speed = %+v, want 0 and forbidden", sp)
	}

	facts := func(feet int) combat.ConditionAttackFacts {
		return combat.ConditionAttackFacts{AttackerID: orc, TargetID: hero, DistanceFeet: feet,
			AttackerSeesTarget: combat.SightRelationCanSee, TargetSeesAttacker: combat.SightRelationCannotSee}
	}
	adjacent, err := fx.AttackRoll(combat.ConditionSet{}, set, facts(5))
	if err != nil || adjacent.Mode() != modifier.WithAdvantage {
		t.Errorf("attack from 5 feet = %v, %v, want Advantage", adjacent.Mode(), err)
	}
	if crit, err := fx.CriticalHit(set, 5); err != nil || crit != combat.AutomaticCriticalAutomatic {
		t.Errorf("hit from 5 feet = %v, %v, want an automatic Critical Hit", crit, err)
	}
	ranged, err := fx.AttackRoll(combat.ConditionSet{}, set, facts(60))
	if err != nil || ranged.Mode() != modifier.Normal {
		t.Errorf("attack from 60 feet = %v, %v, want Advantage and Disadvantage cancelling", ranged.Mode(), err)
	}
	if crit, _ := fx.CriticalHit(set, 60); crit != combat.AutomaticCriticalNotAutomatic {
		t.Errorf("hit from 60 feet = %v, want no automatic Critical Hit", crit)
	}

	cmd, err := combat.NewRemoveConditionFromSetCommand(set, hero, content.ConditionUnconscious)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewRemoveConditionFromSet(opts...).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	cleared, ok := res.(combat.ConditionCleared)
	if !ok || len(cleared.Events()) != 1 {
		t.Fatalf("removal = %#v, want ConditionCleared with one event", res)
	}
	if got := cleared.Set().Effective(); !slices.Equal(got, []content.ConditionKind{content.ConditionGrappled, content.ConditionProne}) {
		t.Errorf("effective after waking = %v, want Grappled and Prone", got)
	}
	if mv, _ := fx.StandUp(cleared.Set(), 30); mv != (combat.CrawlOnly{}) {
		t.Errorf("a grappled Prone creature can %#v, want only to crawl", mv)
	}
	if len(rec.Meter.MeasurementsNamed("srd5e.combat.events")) != 3 {
		t.Errorf("events counted = %d, want 3 (two applied, one removed)", len(rec.Meter.MeasurementsNamed("srd5e.combat.events")))
	}
}

// Requirement: doc/specs/conditions.md section 8 scenario 2; FR-038, FR-039, FR-040, FR-041, FR-042, FR-043 (SRD 5.2.1,
// "Exhaustion", p. 181) - Exhaustion stacks to level 6, where the creature dies and a seventh level is refused, and a Long
// Rest takes a level off, through the public API only.
// Path: unhappy (boundary).
func TestExhaustionStacksToLevelSix(t *testing.T) {
	t.Parallel()
	ctx, hero := context.Background(), combat.MustCombatantID("hero")
	var fx combat.ConditionEffects
	set := combat.ConditionSet{}
	for level := 1; level <= 6; level++ {
		cmd, err := combat.NewApplyConditionToSetCommand(set, hero, combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)})
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewApplyConditionToSet().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := res.(combat.ConditionAdded); !ok {
			t.Fatalf("level %d: %#v, want ConditionAdded", level, res)
		}
		set = res.Set()
		if fx.D20TestReduction(set) != 2*level || fx.IsDeadFromExhaustion(set) != (level == 6) {
			t.Errorf("level %d: reduction %d dead %v", level, fx.D20TestReduction(set), fx.IsDeadFromExhaustion(set))
		}
		if sp, _ := fx.Speed(set, 30); sp.Value != max(30-5*level, 0) {
			t.Errorf("level %d: Speed = %d", level, sp.Value)
		}
	}
	cmd, _ := combat.NewApplyConditionToSetCommand(set, hero, combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)})
	if res, _ := combat.NewApplyConditionToSet().Run(ctx, cmd); !isAtMaximum(res) {
		t.Errorf("a seventh level = %#v, want ExhaustionAtMaximum", res)
	}
	rest, err := combat.NewReduceExhaustionOnSetCommand(set, hero, 1)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewReduceExhaustionOnSet().Run(ctx, rest)
	if err != nil {
		t.Fatal(err)
	}
	if fx.IsDeadFromExhaustion(res.Set()) || fx.D20TestReduction(res.Set()) != 10 {
		t.Errorf("after a Long Rest: dead %v reduction %d, want alive at level 5", fx.IsDeadFromExhaustion(res.Set()), fx.D20TestReduction(res.Set()))
	}
}

func isAtMaximum(r combat.ApplyConditionToSetResult) bool {
	_, ok := r.(combat.ExhaustionAtMaximum)
	return ok
}
