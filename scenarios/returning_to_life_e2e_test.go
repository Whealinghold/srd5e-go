package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/returning-to-life.md FR-003, FR-005, FR-008, FR-009, FR-010, FR-011, FR-013 (SRD 5.2.1, "Dead", p. 180) - a creature killed with Exhaustion level 2, a Poisoned condition that
// lapsed, a Stunned condition still ongoing and a curse still ongoing is returned to life by ReviveCombatant: its Hit Points are capped at the maximum, Exhaustion drops to level 1, Poisoned is dropped,
// the curse is kept, and the events come in the order the spec gives.
// Path: happy.
func TestGrpBReviveCombatantReturnsTheCreatureWithItsLeftovers(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("brenna")
	rec := telemetrytest.NewRecorder()
	source, err := combat.LabeledMagicalSource(content.MustFeatureID("hag_curse"))
	if err != nil {
		t.Fatal(err)
	}
	curse, err := combat.NewOngoingEffect("curse-1", combat.OngoingEffectCurse, source)
	if err != nil {
		t.Fatal(err)
	}
	effects, err := combat.NewOngoingEffects([]combat.OngoingEffect{curse})
	if err != nil {
		t.Fatal(err)
	}
	conditions := combat.MustConditionSet([]combat.Condition{
		combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(2)}, combat.Poisoned{}, combat.Stunned{}}, nil)
	cmd := combat.ReviveCombatantCommand{Subject: id, Status: grpBDead(t, combat.MassiveDamage), HitPoints: 99, Consent: combat.SpiritAccepts,
		Conditions: conditions, Effects: effects, ElapsedConditions: []content.ConditionKind{content.ConditionPoisoned}}

	res, err := combat.NewReviveCombatant(combat.WithTelemetry(rec.Telemetry())).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	applied, ok := res.(combat.ReviveCreatureApplied)
	if !ok {
		t.Fatalf("ReviveCombatant = %T, want ReviveCreatureApplied", res)
	}
	if hp := applied.Status().HitPoints(); hp.Current() != 20 || hp.Maximum() != 20 || applied.Status().LifeState() != (combat.Alive{}) {
		t.Errorf("revived = %+v %v, want 20 of 20 (capped) and Alive", hp, applied.Status().LifeState())
	}
	kindsLeft := []content.ConditionKind{}
	for _, c := range applied.Conditions().Explicit() {
		kindsLeft = append(kindsLeft, c.Kind())
	}
	slices.Sort(kindsLeft)
	want := []content.ConditionKind{content.ConditionExhaustion, content.ConditionStunned}
	slices.Sort(want)
	if !slices.Equal(kindsLeft, want) {
		t.Errorf("conditions left = %v, want Exhaustion and Stunned", kindsLeft)
	}
	if applied.Effects().Len() != 1 {
		t.Errorf("effects left = %d, want the curse kept", applied.Effects().Len())
	}
	got := kinds(applied.Events())
	if len(got) == 0 || got[0] != "creature_returned_to_life" || !slices.Contains(got, "condition_removed") || !slices.Contains(got, "condition_level_reduced") {
		t.Errorf("events = %v, want creature_returned_to_life first, then condition_removed and condition_level_reduced", got)
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

// Requirement: doc/specs/returning-to-life.md FR-001, FR-002, FR-004, FR-006, FR-007, FR-016 - ReviveCombatant refuses a creature that is not Dead, a spirit that refuses, revival Hit Points below 1 and a
// creature whose maximum is 0, each leaving the status as given with no event; revival works after every cause of death other than a maximum of 0.
// Path: unhappy (state error, boundary), happy (every cause).
func TestGrpBReviveCombatantRefusals(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("brenna")
	dead := grpBDead(t, combat.MonsterDeath)
	run := func(c combat.ReviveCombatantCommand) combat.ReviveCombatantResult {
		t.Helper()
		res, err := combat.NewReviveCombatant().Run(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	base := combat.ReviveCombatantCommand{Subject: id, Status: dead, HitPoints: 5, Consent: combat.SpiritAccepts}

	alive := base
	alive.Status = combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
	if _, isNotDead := run(alive).(combat.ReviveCreatureNotDead); !isNotDead {
		t.Errorf("an Alive creature was not refused as not dead")
	}
	refused := base
	refused.Consent = combat.SpiritRefuses
	if r, isRefused := run(refused).(combat.ReviveCreatureRefused); !isRefused || r.Status() != dead {
		t.Errorf("a refusing spirit = %#v, want ReviveCreatureRefused with the status unchanged", r)
	}
	low := base
	low.HitPoints = 0
	if _, isLow := run(low).(combat.ReviveCreatureHitPointsBelowOne); !isLow {
		t.Errorf("revival with 0 Hit Points was not refused")
	}
	gone, err := combat.NewDead(combat.HitPointMaximumZero)
	if err != nil {
		t.Fatal(err)
	}
	zeroMax, err := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 0, 0), gone, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	noMax := base
	noMax.Status = zeroMax
	if _, isNoMax := run(noMax).(combat.ReviveCreatureMaximumBelowOne); !isNoMax {
		t.Errorf("a creature with a maximum of 0 was not refused")
	}

	for _, cause := range []combat.DeathCause{combat.MonsterDeath, combat.MassiveDamage, combat.DeathSavingThrowFailures, combat.Exhaustion} {
		c := base
		c.Status = grpBDead(t, cause)
		if _, isApplied := run(c).(combat.ReviveCreatureApplied); !isApplied {
			t.Errorf("a creature dead by %v was not revived", cause)
		}
	}
}

// Requirement: doc/specs/returning-to-life.md FR-016 and combat-action-resolution.md FR-169 (plan P10 caps) - a revival reporting more than 15 elapsed conditions is an ErrInvalidInput error and a cancelled
// ctx is the ctx error, both before a result.
// Path: unhappy (invalid input, cancellation).
func TestGrpBReviveCombatantInvalidInputAndCancellation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cmd := combat.ReviveCombatantCommand{Subject: combat.MustCombatantID("brenna"), Status: grpBDead(t, combat.MonsterDeath), HitPoints: 5, Consent: combat.SpiritAccepts,
		ElapsedConditions: make([]content.ConditionKind, 16)}
	if res, err := combat.NewReviveCombatant().Run(ctx, cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("16 elapsed conditions = %v, %v, want no result and ErrInvalidInput", res, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	cmd.ElapsedConditions = nil
	if _, err := combat.NewReviveCombatant().Run(cancelled, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled ctx err = %v, want context.Canceled", err)
	}
}
