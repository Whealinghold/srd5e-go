package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// grpBExhausted returns a condition set holding Exhaustion at level.
func grpBExhausted(level int) combat.ConditionSet {
	return combat.MustConditionSet([]combat.Condition{combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(level)}}, nil)
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-061, FR-063, FR-065 and conditions.md FR-036 (SRD 5.2.1, "Exhaustion", "Dehydration [Hazard]", p. 181) - a traveller at Exhaustion level 4 of which 2
// levels are held by Dehydration asks a Long Rest to remove 3: only the 2 free levels come off, 1 is blocked, the Suffocation record is cut to the Exhaustion that remains, and once the water is drunk (the host sets
// the Dehydration levels to 0) the same request removes the rest.
// Path: happy and unhappy (boundary, rule interaction).
func TestGrpBLongRestCannotRemoveLockedExhaustion(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("traveller")
	rec := telemetrytest.NewRecorder()
	uc := combat.NewRemoveExhaustionLevels(combat.WithTelemetry(rec.Telemetry()))

	cmd := combat.RemoveExhaustionLevelsCommand{Conditions: grpBExhausted(4), Target: id, Record: combat.MustHazardRecord(2, 0, 3, 0), Levels: 3}
	res, err := uc.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	removed, ok := res.(combat.ExhaustionLevelsRemoved)
	if !ok {
		t.Fatalf("RemoveExhaustionLevels = %#v, want ExhaustionLevelsRemoved", res)
	}
	if removed.Removed() != 2 || removed.Blocked() != 1 {
		t.Errorf("removed %d blocked %d, want 2 and 1", removed.Removed(), removed.Blocked())
	}
	if got := removed.Conditions().Explicit(); len(got) != 1 || got[0] != (combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(2)}) {
		t.Errorf("conditions after = %v, want Exhaustion level 2 (the two locked levels)", got)
	}
	if got := removed.Record().SuffocationLevels(); got != 0 || removed.Record().DehydrationLevels() != 2 {
		t.Errorf("record after: suffocation %d dehydration %d, want 0 (cut to the free Exhaustion) and 2", got, removed.Record().DehydrationLevels())
	}
	if got := kinds(removed.Events()); len(got) != 1 || got[0] != "condition_level_reduced" {
		t.Errorf("events = %v, want one condition_level_reduced", got)
	}

	watered := combat.RemoveExhaustionLevelsCommand{Conditions: removed.Conditions(), Target: id, Record: combat.MustHazardRecord(0, 0, 0, 0), Levels: 2}
	res, err = uc.Run(ctx, watered)
	if err != nil {
		t.Fatal(err)
	}
	if after := res.(combat.ExhaustionLevelsRemoved); after.Removed() != 2 || len(after.Conditions().Explicit()) != 0 {
		t.Errorf("after drinking: removed %d, conditions %v, want 2 and none", after.Removed(), after.Conditions().Explicit())
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-062, FR-064 and conditions.md FR-043 - a creature whose every level is locked, and one with no Exhaustion, get a RemoveExhaustionLevelsRejected with
// the blocked count and the cause; levels below 1 and a zero target are invalid input and a cancelled ctx is the ctx error, each before any result.
// Path: unhappy (state error, boundary, invalid input, cancellation).
func TestGrpBRemoveExhaustionLevelsRefusalsAndInvalidInput(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("traveller")
	uc := combat.NewRemoveExhaustionLevels()

	locked, err := uc.Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: grpBExhausted(2), Target: id, Record: combat.MustHazardRecord(0, 2, 0, 3), Levels: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r, isRej := locked.(combat.RemoveExhaustionLevelsRejected); !isRej || r.Rejection.Cause() != combat.ExhaustionRemovalLocked || r.Rejection.Blocked() != 1 {
		t.Errorf("every level locked by Malnutrition = %#v, want rejected as locked with 1 blocked", locked)
	}
	none, err := uc.Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: combat.ConditionSet{}, Target: id, Record: combat.EmptyHazardRecord(), Levels: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r, isRej := none.(combat.RemoveExhaustionLevelsRejected); !isRej || r.Rejection.Cause() != combat.ExhaustionRemovalNoExhaustion {
		t.Errorf("no Exhaustion = %#v, want rejected as no_exhaustion", none)
	}

	for _, levels := range []int{0, -2} {
		res, err := uc.Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: grpBExhausted(2), Target: id, Record: combat.EmptyHazardRecord(), Levels: levels})
		if res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%d levels = %v, %v, want no result and ErrInvalidInput", levels, res, err)
		}
	}
	if res, err := uc.Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: grpBExhausted(2), Record: combat.EmptyHazardRecord(), Levels: 1}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero target = %v, %v, want no result and ErrInvalidInput", res, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, combat.RemoveExhaustionLevelsCommand{Conditions: grpBExhausted(2), Target: id, Record: combat.EmptyHazardRecord(), Levels: 1}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled ctx err = %v, want context.Canceled", err)
	}
}
