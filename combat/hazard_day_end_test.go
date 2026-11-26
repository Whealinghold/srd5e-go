package combat_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func dehydrationDay(t *testing.T, size content.CreatureSize, drunk int, set combat.ConditionSet, rec combat.HazardRecord) combat.DehydrationDayEnded {
	t.Helper()
	w, err := combat.NewWaterAmount(drunk)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewResolveDehydrationDayEnd().Run(context.Background(), combat.ResolveDehydrationDayEndCommand{Size: size, Water: w, Conditions: set, Target: subject(), Record: rec})
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.DehydrationDayEnded)
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-030 to FR-035, FR-110, FR-111, FR-033 (tables 4.4b, 4.5a) - Dehydration at day's end: less than half the water gives one Exhaustion level, exactly half none, the full amount unlocks the recorded levels without removing any, the record is clamped to the set, and level 6 or Immunity changes nothing.
// Path: per case.
func TestResolveDehydrationDayEnd(t *testing.T) {
	t.Parallel()
	const (
		gain, none, atMax, immune = combat.HazardExhaustionGained, combat.HazardNoExhaustion, combat.HazardExhaustionAtMaximum, combat.HazardImmune
	)
	cases := []struct {
		name, requirement, path string
		size                    content.CreatureSize
		drunk                   int
		set                     combat.ConditionSet
		rec                     combat.HazardRecord
		outcome                 combat.HazardExhaustionOutcome
		levelAfter, recAfter    int
		events                  []string
	}{
		{"nothing drunk", "FR-031", "happy", envMedium(), 0, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), gain, 1, 1, []string{"condition_applied"}},
		{"one below half", "FR-031, FR-030", "unhappy (boundary)", envMedium(), 3, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), gain, 1, 1, []string{"condition_applied"}},
		{"exactly half", "FR-032, FR-030", "unhappy (boundary)", envMedium(), 4, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), none, 0, 0, []string{}},
		{"just under full", "FR-032", "unhappy (boundary)", envMedium(), 7, envExhaustion(t, 2), envRecord(t, 0, 0, 0, 0, 0), none, 2, 0, []string{}},
		{"full amount unlocks", "FR-033", "unhappy (rule interaction)", envMedium(), 8, envExhaustion(t, 2), envRecord(t, 2, 0, 0, 0, 0), none, 2, 0, []string{"dehydration_recovered"}},
		{"more than full", "FR-033", "unhappy (boundary)", envMedium(), 100, envExhaustion(t, 2), envRecord(t, 1, 0, 0, 0, 0), none, 2, 0, []string{"dehydration_recovered"}},
		{"full amount, nothing locked", "FR-033", "unhappy (empty and minimal)", envMedium(), 8, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), none, 0, 0, []string{}},
		{"second day without water", "FR-034", "unhappy (boundary)", envMedium(), 0, envExhaustion(t, 1), envRecord(t, 1, 0, 0, 0, 0), gain, 2, 2, []string{"condition_applied"}},
		{"Tiny exactly half", "FR-032", "unhappy (boundary)", content.SizeTiny, 1, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), none, 0, 0, []string{}},
		{"Tiny nothing", "FR-031", "unhappy (boundary)", content.SizeTiny, 0, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), gain, 1, 1, []string{"condition_applied"}},
		{"Gargantuan one below half", "FR-031", "unhappy (boundary)", content.SizeGargantuan, 255, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), gain, 1, 1, []string{"condition_applied"}},
		{"Gargantuan exactly half", "FR-032", "unhappy (boundary)", content.SizeGargantuan, 256, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), none, 0, 0, []string{}},
		{"level 5 to 6", "FR-031, FR-111", "unhappy (boundary)", envMedium(), 0, envExhaustion(t, 5), envRecord(t, 2, 0, 0, 0, 0), gain, 6, 3, []string{"condition_applied"}},
		{"record above the set", "FR-110", "unhappy (state error)", envMedium(), 0, envExhaustion(t, 2), envRecord(t, 4, 0, 0, 0, 0), gain, 3, 3, []string{"condition_applied"}},
		{"record with no Exhaustion is clamped", "FR-110", "unhappy (state error)", envMedium(), 4, envExhaustion(t, 0), envRecord(t, 3, 0, 0, 0, 0), none, 0, 0, []string{}},
		{"at level 6", "FR-035", "unhappy (rule interaction)", envMedium(), 0, envExhaustion(t, 6), envRecord(t, 3, 0, 0, 0, 0), atMax, 6, 3, []string{}},
		{"immune to Exhaustion", "FR-035", "unhappy (rule interaction)", envMedium(), 0, condSet(t, nil, content.ConditionExhaustion), envRecord(t, 0, 0, 0, 0, 0), immune, 0, 0, []string{}},
		{"maximum int water", "FR-033", "unhappy (boundary)", content.SizeGargantuan, math.MaxInt, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), none, 0, 0, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := dehydrationDay(t, c.size, c.drunk, c.set, c.rec)
			if got.Outcome() != c.outcome || envLevel(got.Conditions()) != c.levelAfter || got.Record().DehydrationLevels() != c.recAfter ||
				!equalStrings(envKinds(got.Events()), c.events) {
				t.Errorf("= %v level %d record %d events %v, want %v %d %d %v [%s]", got.Outcome(), envLevel(got.Conditions()), got.Record().DehydrationLevels(),
					envKinds(got.Events()), c.outcome, c.levelAfter, c.recAfter, c.events, c.requirement)
			}
			if got.Record().MalnutritionLevels() != c.rec.MalnutritionLevels() {
				t.Error("the Malnutrition levels changed")
			}
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-033, FR-110 - Dehydration never removes Exhaustion itself, an event reports the unlocked levels, and a record with other sources is clamped suffocation first.
// Path: unhappy (rule interaction).
func TestResolveDehydrationDayEndKeepsOtherLevels(t *testing.T) {
	t.Parallel()
	got := dehydrationDay(t, envMedium(), 8, envExhaustion(t, 3), envRecord(t, 1, 1, 1, 0, 0))
	if envLevel(got.Conditions()) != 3 || got.Record() != envRecord(t, 0, 1, 1, 0, 0) {
		t.Errorf("= level %d record %+v, want level 3 and only Dehydration cleared", envLevel(got.Conditions()), got.Record())
	}
	ev := got.Events()[0].(combat.DehydrationRecovered)
	if ev.Levels() != 1 || ev.Combatants()[0] != subject() {
		t.Errorf("event = %+v, want 1 level for the target", ev)
	}
	clamped := dehydrationDay(t, envMedium(), 5, envExhaustion(t, 2), envRecord(t, 1, 1, 2, 0, 1))
	if clamped.Record() != envRecord(t, 1, 0, 0, 0, 1) {
		t.Errorf("clamped record = %+v, want Suffocation then Malnutrition cut to fit the level of the set", clamped.Record())
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-027, FR-029 - an invalid size or a zero target is an ErrInvalidInput error, a cancelled context returns ctx.Err() and the input is not changed.
// Path: unhappy (invalid input).
func TestResolveDehydrationDayEndInvalidInput(t *testing.T) {
	t.Parallel()
	uc := combat.NewResolveDehydrationDayEnd()
	good := combat.ResolveDehydrationDayEndCommand{Size: envMedium(), Conditions: envExhaustion(t, 1), Target: subject()}
	for name, cmd := range map[string]combat.ResolveDehydrationDayEndCommand{
		"zero size": {Conditions: good.Conditions, Target: subject()}, "zero target": {Size: good.Size}, "zero command": {}} {
		if _, err := uc.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, good); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
	before := good.Conditions
	if _, err := uc.Run(context.Background(), good); err != nil || good.Conditions != before {
		t.Error("the command changed or failed")
	}
}

func malnutritionDay(t *testing.T, size content.CreatureSize, eaten int, save combat.FoodSavingThrow, set combat.ConditionSet, rec combat.HazardRecord) combat.ResolveMalnutritionDayEndResult {
	t.Helper()
	f, err := combat.NewFoodAmount(eaten)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewResolveMalnutritionDayEnd().Run(context.Background(), combat.ResolveMalnutritionDayEndCommand{Size: size, Food: f, SavingThrow: save, Conditions: set, Target: subject(), Record: rec})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-036 to FR-045, FR-110 (table 4.6a) - Malnutrition at day's end for a Medium creature: half the food or more needs no save and ends the run, less than half asks for a DC 10 Constitution save and gains a level on a failure, nothing eaten gains a level without a save from the fifth day, and level 6 or Immunity changes nothing.
// Path: per case.
func TestResolveMalnutritionDayEnd(t *testing.T) {
	t.Parallel()
	failed, passed := combat.FoodSaveRolled{Outcome: d20.Failure}, combat.FoodSaveRolled{Outcome: d20.Success}
	const (
		gain, none, atMax, immune = combat.HazardExhaustionGained, combat.HazardNoExhaustion, combat.HazardExhaustionAtMaximum, combat.HazardImmune
	)
	cases := []struct {
		name, requirement, path string
		eaten                   int
		save                    combat.FoodSavingThrow
		set                     combat.ConditionSet
		rec                     combat.HazardRecord
		needSave                bool
		outcome                 combat.HazardExhaustionOutcome
		levelAfter, daysAfter   int
		recMalnutrition         int
		events                  []string
	}{
		{"ate enough", "FR-036", "happy", 4, nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 3, 0), false, none, 0, 0, 0, []string{}},
		{"ate the full amount", "FR-036", "happy", 8, nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), false, none, 0, 0, 0, []string{}},
		{"full amount unlocks", "FR-037", "unhappy (rule interaction)", 8, nil, envExhaustion(t, 1), envRecord(t, 0, 1, 0, 0, 0), false, none, 1, 0, 0, []string{"malnutrition_recovered"}},
		{"more than the full amount unlocks", "FR-037", "unhappy (boundary)", 100, nil, envExhaustion(t, 1), envRecord(t, 0, 1, 0, 0, 0), false, none, 1, 0, 0, []string{"malnutrition_recovered"}},
		{"half unlocks nothing", "FR-037", "unhappy (boundary)", 4, nil, envExhaustion(t, 1), envRecord(t, 0, 1, 0, 0, 0), false, none, 1, 0, 1, []string{}},
		{"less than half, no save yet", "FR-038", "happy", 3, combat.FoodSaveNotRolled{}, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 2, 0), true, 0, 0, 2, 0, nil},
		{"less than half, nil save", "FR-038", "unhappy (empty and minimal)", 3, nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), true, 0, 0, 0, 0, nil},
		{"less than half, save fails", "FR-039", "happy", 3, failed, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), false, gain, 1, 0, 1, []string{"condition_applied"}},
		{"less than half, save outcome not set", "FR-039", "unhappy (invalid input)", 3, combat.FoodSaveRolled{}, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), false, gain, 1, 0, 1, []string{"condition_applied"}},
		{"less than half, save succeeds", "FR-040", "unhappy (rule interaction)", 3, passed, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), false, none, 0, 0, 0, []string{}},
		{"one below half, save fails, run ends", "FR-039, FR-044", "unhappy (rule interaction)", 3, failed, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 2, 0), false, gain, 1, 0, 1, []string{"condition_applied"}},
		{"exactly half, save ignored", "FR-041", "unhappy (invalid input)", 4, failed, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), false, none, 0, 0, 0, []string{}},
		{"nothing eaten, save ignored", "FR-041, FR-042", "unhappy (invalid input)", 0, failed, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), false, none, 0, 1, 0, []string{}},
		{"nothing eaten, day 4", "FR-042", "unhappy (boundary)", 0, nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 3, 0), false, none, 0, 4, 0, []string{}},
		{"nothing eaten, day 5", "FR-043", "unhappy (boundary)", 0, nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 4, 0), false, gain, 1, 5, 1, []string{"condition_applied"}},
		{"nothing eaten, day 6", "FR-043", "unhappy (boundary)", 0, nil, envExhaustion(t, 1), envRecord(t, 0, 1, 0, 5, 0), false, gain, 2, 6, 2, []string{"condition_applied"}},
		{"nothing eaten, day 7", "FR-043", "unhappy (boundary)", 0, nil, envExhaustion(t, 2), envRecord(t, 0, 2, 0, 6, 0), false, gain, 3, 7, 3, []string{"condition_applied"}},
		{"a little food resets the run", "FR-044", "unhappy (state error)", 1, passed, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 4, 0), false, none, 0, 0, 0, []string{}},
		{"restart after eating", "FR-044", "unhappy (state error)", 0, nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), false, none, 0, 1, 0, []string{}},
		{"day 5 at level 6", "FR-043, FR-045", "unhappy (rule interaction)", 0, nil, envExhaustion(t, 6), envRecord(t, 0, 0, 0, 4, 0), false, atMax, 6, 5, 0, []string{}},
		{"day 5, immune to Exhaustion", "FR-045", "unhappy (rule interaction)", 0, nil, condSet(t, nil, content.ConditionExhaustion), envRecord(t, 0, 0, 0, 4, 0), false, immune, 0, 5, 0, []string{}},
		{"save fails at level 6", "FR-045", "unhappy (rule interaction)", 3, failed, envExhaustion(t, 6), envRecord(t, 0, 0, 0, 1, 0), false, atMax, 6, 0, 0, []string{}},
		{"huge days without food", "FR-042", "unhappy (boundary)", 0, nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, math.MaxInt-1, 0), false, gain, 1, math.MaxInt, 1, []string{"condition_applied"}},
		{"days saturate", "FR-042", "unhappy (boundary)", 0, nil, envExhaustion(t, 1), envRecord(t, 0, 0, 0, math.MaxInt, 0), false, gain, 2, math.MaxInt, 1, []string{"condition_applied"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := malnutritionDay(t, envMedium(), c.eaten, c.save, c.set, c.rec)
			if c.needSave {
				got, ok := res.(combat.MalnutritionSaveRequired)
				if !ok || got.DC() != 10 || got.Ability() != ability.Constitution {
					t.Fatalf("= %#v, want MalnutritionSaveRequired(DC 10, Constitution) [%s]", res, c.requirement)
				}
				return
			}
			got, ok := res.(combat.MalnutritionDayEnded)
			if !ok {
				t.Fatalf("= %#v, want MalnutritionDayEnded", res)
			}
			if got.Outcome() != c.outcome || envLevel(got.Conditions()) != c.levelAfter || got.Record().DaysWithoutFood() != c.daysAfter ||
				got.Record().MalnutritionLevels() != c.recMalnutrition || !equalStrings(envKinds(got.Events()), c.events) {
				t.Errorf("= %v level %d days %d record %d events %v, want %v %d %d %d %v [%s]", got.Outcome(), envLevel(got.Conditions()), got.Record().DaysWithoutFood(),
					got.Record().MalnutritionLevels(), envKinds(got.Events()), c.outcome, c.levelAfter, c.daysAfter, c.recMalnutrition, c.events, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-038 - a result that asks for the save changes no state: the same call with the save rolled afterwards still sees the unchanged record and set.
// Path: unhappy (state error).
func TestMalnutritionSaveRequiredChangesNothing(t *testing.T) {
	t.Parallel()
	set, rec := envExhaustion(t, 2), envRecord(t, 0, 1, 0, 3, 0)
	if _, ok := malnutritionDay(t, envMedium(), 3, nil, set, rec).(combat.MalnutritionSaveRequired); !ok {
		t.Fatal("the first call did not ask for the save")
	}
	got := malnutritionDay(t, envMedium(), 3, combat.FoodSaveRolled{Outcome: d20.Failure}, set, rec).(combat.MalnutritionDayEnded)
	if got.Outcome() != combat.HazardExhaustionGained || got.Record().MalnutritionLevels() != 2 || got.Record().DaysWithoutFood() != 0 {
		t.Errorf("= %v record %+v", got.Outcome(), got.Record())
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-036 - a size or a saving throw that is not valid and a zero target are ErrInvalidInput errors; a cancelled context returns ctx.Err().
// Path: unhappy (invalid input).
func TestResolveMalnutritionDayEndInvalidInput(t *testing.T) {
	t.Parallel()
	uc := combat.NewResolveMalnutritionDayEnd()
	good := combat.ResolveMalnutritionDayEndCommand{Size: envMedium(), Target: subject()}
	bad := map[string]combat.ResolveMalnutritionDayEndCommand{
		"zero size": {Target: subject()}, "zero target": {Size: envMedium()}, "unknown save": {Size: envMedium(), Target: subject(), SavingThrow: badSave{}}}
	for name, cmd := range bad {
		if _, err := uc.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, good); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}

// badSave is not a FoodSavingThrow variant the package knows.
type badSave struct{ combat.FoodSavingThrow }

func envMedium() content.CreatureSize { return content.SizeMedium }
