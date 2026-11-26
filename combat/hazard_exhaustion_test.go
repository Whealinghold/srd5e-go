package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func removeLevels(t *testing.T, set combat.ConditionSet, rec combat.HazardRecord, levels int) (combat.RemoveExhaustionLevelsResult, error) {
	t.Helper()
	return combat.NewRemoveExhaustionLevels().Run(context.Background(), combat.RemoveExhaustionLevelsCommand{Conditions: set, Target: subject(), Record: rec, Levels: levels})
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-060 to FR-063, FR-115, FR-120 (table 4.8a) - removing Exhaustion takes off min(asked, removable) levels, blocks the levels Dehydration, Malnutrition and contagion lock, cuts the Suffocation record to what remains, and rejects with the blocked count and cause when nothing is removable.
// Path: per case.
func TestRemoveExhaustionLevels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		level                   int
		rec                     [4]int // dehydration, malnutrition, suffocation, contagion
		asked                   int
		removed, blocked        int // when removed is 0 the call is rejected
		levelAfter, suffAfter   int
		cause                   combat.ExhaustionRemovalCause
	}{
		{"nothing locked", "FR-061", "happy", 3, [4]int{}, 1, 1, 0, 2, 0, 0},
		{"partly locked", "FR-061", "unhappy (rule interaction)", 3, [4]int{2, 0, 0, 0}, 1, 1, 0, 2, 0, 0},
		{"all locked", "FR-062", "unhappy (rule interaction)", 3, [4]int{3, 0, 0, 0}, 1, 0, 1, 3, 0, combat.ExhaustionRemovalLocked},
		{"locked by both", "FR-062", "unhappy (rule interaction)", 3, [4]int{2, 1, 0, 0}, 1, 0, 1, 3, 0, combat.ExhaustionRemovalLocked},
		{"ask for more than removable", "FR-061", "unhappy (boundary)", 4, [4]int{2, 0, 0, 0}, 3, 2, 1, 2, 0, 0},
		{"ask for exactly removable", "FR-061", "unhappy (boundary)", 4, [4]int{2, 0, 0, 0}, 2, 2, 0, 2, 0, 0},
		{"suffocation is not locked, record cut", "FR-061, FR-063", "unhappy (rule interaction)", 2, [4]int{0, 0, 2, 0}, 1, 1, 0, 1, 1, 0},
		{"record cut to nothing", "FR-063", "unhappy (rule interaction)", 2, [4]int{0, 0, 2, 0}, 2, 2, 0, 0, 0, 0},
		{"no Exhaustion", "FR-062", "unhappy (empty and minimal)", 0, [4]int{}, 1, 0, 1, 0, 0, combat.ExhaustionRemovalNoExhaustion},
		{"from level 6", "FR-061", "unhappy (boundary)", 6, [4]int{1, 1, 2, 0}, 1, 1, 0, 5, 2, 0},
		{"contagion level locked", "FR-115, FR-062", "unhappy (rule interaction)", 1, [4]int{0, 0, 0, 1}, 1, 0, 1, 1, 0, combat.ExhaustionRemovalLocked},
		{"one level free", "FR-115, FR-061", "unhappy (rule interaction)", 3, [4]int{0, 0, 0, 1}, 1, 1, 0, 2, 0, 0},
		{"contagion and dehydration", "FR-115, FR-061", "unhappy (rule interaction)", 3, [4]int{1, 0, 0, 1}, 2, 1, 1, 2, 0, 0},
		{"record larger than the set is read clamped", "FR-110", "unhappy (state error)", 1, [4]int{0, 0, 3, 0}, 1, 1, 0, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := envRecord(t, c.rec[0], c.rec[1], c.rec[2], 0, c.rec[3])
			set := envExhaustion(t, c.level)
			res, err := removeLevels(t, set, rec, c.asked)
			if err != nil {
				t.Fatal(err)
			}
			if c.removed == 0 {
				rej, ok := res.(combat.RemoveExhaustionLevelsRejected)
				if !ok || rej.Rejection.Blocked() != c.blocked || rej.Rejection.Cause() != c.cause {
					t.Errorf("= %#v, want a rejection with %d blocked (%v) [%s]", res, c.blocked, c.cause, c.requirement)
				}
				return
			}
			got := res.(combat.ExhaustionLevelsRemoved)
			if got.Removed() != c.removed || got.Blocked() != c.blocked || envLevel(got.Conditions()) != c.levelAfter || got.Record().SuffocationLevels() != c.suffAfter {
				t.Errorf("= removed %d blocked %d level %d suffocation %d, want %d %d %d %d [%s]", got.Removed(), got.Blocked(), envLevel(got.Conditions()),
					got.Record().SuffocationLevels(), c.removed, c.blocked, c.levelAfter, c.suffAfter, c.requirement)
			}
			if got.Record().DehydrationLevels() != rec.DehydrationLevels() || got.Record().ContagionLevels() != rec.ContagionLevels() || got.Reduction() == nil {
				t.Error("a lock changed or the reduction is missing")
			}
			if len(got.Events()) != 1 {
				t.Errorf("events = %v, want the one condition event", envKinds(got.Events()))
			}
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-062 - a rejection leaves the set and the record alone, with no events; the cause tells locked from no Exhaustion.
// Path: unhappy (rule interaction).
func TestRemoveExhaustionLevelsRejectionChangesNothing(t *testing.T) {
	t.Parallel()
	set, rec := envExhaustion(t, 2), envRecord(t, 2, 0, 0, 0, 0)
	res, _ := removeLevels(t, set, rec, 5)
	rej := res.(combat.RemoveExhaustionLevelsRejected).Rejection
	if rej.Blocked() != 5 || rej.Cause() != combat.ExhaustionRemovalLocked || rej.Cause().String() != "locked" || combat.ExhaustionRemovalNoExhaustion.String() != "no_exhaustion" || combat.ExhaustionRemovalCause(0).String() != "unknown" {
		t.Errorf("rejection = %+v", rej)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-064 - levels below 1 are an ErrInvalidInput error before any other check; a zero target and a cancelled context are errors before any signal.
// Path: unhappy (invalid input).
func TestRemoveExhaustionLevelsInvalidInput(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1} {
		if _, err := removeLevels(t, envExhaustion(t, 3), combat.HazardRecord{}, n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("levels %d = %v, want ErrInvalidInput", n, err)
		}
	}
	uc := combat.NewRemoveExhaustionLevels()
	if _, err := uc.Run(context.Background(), combat.RemoveExhaustionLevelsCommand{Levels: 1}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero target = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, combat.RemoveExhaustionLevelsCommand{Target: subject(), Levels: 1}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-065, FR-033, FR-115 - the lock lifts with the cause: a day without water leaves the level blocked, a full day's water unlocks it, and then it can be removed; a contagion that ends does the same.
// Path: unhappy (rule interaction).
func TestExhaustionLockLiftsWhenTheCauseEnds(t *testing.T) {
	t.Parallel()
	set, rec := envExhaustion(t, 0), combat.HazardRecord{}
	day := dehydrationDay(t, envMedium(), 0, set, rec)
	set, rec = day.Conditions(), day.Record()
	if res, _ := removeLevels(t, set, rec, 1); res.(combat.RemoveExhaustionLevelsRejected).Rejection.Blocked() != 1 {
		t.Fatal("the level was not blocked after a day without water")
	}
	day = dehydrationDay(t, envMedium(), 8, set, rec)
	set, rec = day.Conditions(), day.Record()
	res, err := removeLevels(t, set, rec, 1)
	got, ok := res.(combat.ExhaustionLevelsRemoved)
	if err != nil || !ok || envLevel(got.Conditions()) != 0 {
		t.Fatalf("after a full day's water = %#v, %v, want the level removed", res, err)
	}
	locked, _ := envRecord(t, 0, 0, 0, 0, 0).WithContagionLevels(1)
	if res, _ := removeLevels(t, envExhaustion(t, 1), locked, 1); res.(combat.RemoveExhaustionLevelsRejected).Rejection.Cause() != combat.ExhaustionRemovalLocked {
		t.Error("a contagion level was removable")
	}
	if res, _ := removeLevels(t, envExhaustion(t, 1), combat.HazardRecord{}, 1); res.(combat.ExhaustionLevelsRemoved).Removed() != 1 {
		t.Error("the level was blocked after the contagion ended")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-065, FR-038 - the malnutrition lock lifts the same way, with a Constitution save in between.
// Path: unhappy (rule interaction).
func TestMalnutritionLockLiftsAfterEating(t *testing.T) {
	t.Parallel()
	set, rec := envExhaustion(t, 0), combat.HazardRecord{}
	d := malnutritionDay(t, envMedium(), 2, combat.FoodSaveRolled{Outcome: d20.Failure}, set, rec).(combat.MalnutritionDayEnded)
	set, rec = d.Conditions(), d.Record()
	if res, _ := removeLevels(t, set, rec, 1); res.(combat.RemoveExhaustionLevelsRejected).Rejection.Blocked() != 1 {
		t.Fatal("not blocked")
	}
	d = malnutritionDay(t, envMedium(), 8, nil, set, rec).(combat.MalnutritionDayEnded)
	if res, _ := removeLevels(t, d.Conditions(), d.Record(), 1); res.(combat.ExhaustionLevelsRemoved).Removed() != 1 {
		t.Error("not removable after a full meal")
	}
}
