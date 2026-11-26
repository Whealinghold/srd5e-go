package scenarios_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpEExhaustion returns the Exhaustion level a condition set holds, read through its public explicit conditions.
func grpEExhaustion(set combat.ConditionSet) int {
	for _, c := range set.Explicit() {
		if e, ok := c.(combat.ExhaustionCondition); ok {
			return e.Level.Value()
		}
	}
	return 0
}

// grpEKinds returns the Kind of every event, in order.
func grpEKinds(events []combat.Event) []string {
	kinds := make([]string, 0, len(events))
	for _, e := range events {
		kinds = append(kinds, e.Kind())
	}
	return kinds
}

func grpEKindsEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// Requirement: doc/specs/environment-hazards-and-objects.md section 8 scenario, FR-046 to FR-050, FR-053, FR-056, FR-057, FR-107 (SRD 5.2.1,
// "Suffocation [Hazard]", p. 189) - a diver with Constitution 14 holds their breath for 3 minutes (1 + 2), runs out of breath, gains one
// Exhaustion level at the end of each of three turns, and when the air returns loses exactly those three levels while keeping the level that
// another hazard gave.
// Path: happy.
func TestDiverHoldsBreathSuffocatesAndRecovers(t *testing.T) {
	t.Parallel()
	ctx, diver := context.Background(), combat.MustCombatantID("diver")
	conMod := ability.MustScore(14).Modifier()
	if got := (combat.BreathCapacity{}).ForConstitution(conMod); got != 3*time.Minute {
		t.Fatalf("capacity for Constitution 14 = %v, want 3m0s [FR-046]", got)
	}

	advance := combat.NewAdvanceBreath()
	step := func(state combat.BreathState, change combat.BreathChange) combat.BreathAdvanced {
		t.Helper()
		res, err := advance.Run(ctx, combat.AdvanceBreathCommand{State: state, Change: change, Target: diver})
		if err != nil {
			t.Fatal(err)
		}
		adv, ok := res.(combat.BreathAdvanced)
		if !ok {
			t.Fatalf("result = %#v, want BreathAdvanced", res)
		}
		return adv
	}
	held := step(combat.BreathNormal{}, combat.BeginHoldingBreath{ConstitutionModifier: conMod})
	if h, ok := held.State().(combat.BreathHolding); !ok || h.Elapsed() != 0 || h.Capacity() != 3*time.Minute ||
		!grpEKindsEqual(grpEKinds(held.Events()), []string{"breath_held_started"}) {
		t.Fatalf("after beginning = %#v %v, want holding 0 of 3m with breath_held_started [FR-047]", held.State(), grpEKinds(held.Events()))
	}
	held = step(held.State(), combat.HoldBreathFor{Elapsed: 2 * time.Minute})
	if h, ok := held.State().(combat.BreathHolding); !ok || h.Elapsed() != 2*time.Minute || len(held.Events()) != 0 {
		t.Fatalf("after 2 minutes = %#v events %v, want still holding with no event [FR-048]", held.State(), grpEKinds(held.Events()))
	}
	gasping := step(held.State(), combat.HoldBreathFor{Elapsed: time.Minute}) // exactly the capacity
	suffocating, ok := gasping.State().(combat.BreathSuffocating)
	if !ok || suffocating.Cause() != combat.SuffocationRanOutOfBreath || !grpEKindsEqual(grpEKinds(gasping.Events()), []string{"ran_out_of_breath"}) {
		t.Fatalf("after 3 minutes = %#v %v, want suffocating (ran out of breath) [FR-049]", gasping.State(), grpEKinds(gasping.Events()))
	}

	// One Exhaustion level from another source (Dehydration) is already held; suffocation adds three more.
	one := combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)}
	set, ok := (combat.ConditionSet{}).Apply(diver, one).(combat.ConditionAdded)
	if !ok {
		t.Fatal("could not give the diver their first Exhaustion level")
	}
	conds, record := set.Set(), combat.MustHazardRecord(1, 0, 0, 0)
	turnEnd := combat.NewResolveSuffocationTurnEnd()
	for i := 1; i <= 3; i++ {
		res, err := turnEnd.Run(ctx, combat.ResolveSuffocationTurnEndCommand{State: suffocating, Conditions: conds, Target: diver, Record: record})
		if err != nil {
			t.Fatal(err)
		}
		ended := res.(combat.SuffocationTurnEnded)
		conds, record = ended.Conditions(), ended.Record()
		if ended.Outcome() != combat.HazardExhaustionGained || grpEExhaustion(conds) != 1+i || record.SuffocationLevels() != i {
			t.Fatalf("turn %d: outcome %v level %d suffocation levels %d, want a gained level (%d) and %d recorded [FR-053]",
				i, ended.Outcome(), grpEExhaustion(conds), record.SuffocationLevels(), 1+i, i)
		}
	}
	if (combat.ConditionEffects{}).IsDeadFromExhaustion(conds) {
		t.Fatal("level 4 is not death")
	}

	back, err := combat.NewRestoreBreathing().Run(ctx, combat.RestoreBreathingCommand{State: suffocating, Conditions: conds, Target: diver, Record: record})
	if err != nil {
		t.Fatal(err)
	}
	restored := back.(combat.BreathingRestored)
	if restored.State() != (combat.BreathNormal{}) || restored.LevelsRemoved() != 3 || grpEExhaustion(restored.Conditions()) != 1 ||
		restored.Record().SuffocationLevels() != 0 || restored.Record().DehydrationLevels() != 1 {
		t.Fatalf("restored = state %#v removed %d level %d record %+v, want normal, 3 removed, level 1 kept, the Dehydration level untouched [FR-056, FR-057]",
			restored.State(), restored.LevelsRemoved(), grpEExhaustion(restored.Conditions()), restored.Record())
	}
	kinds := grpEKinds(restored.Events())
	if len(kinds) == 0 || kinds[len(kinds)-1] != "breath_restored" {
		t.Errorf("events = %v, want breath_restored last [FR-107]", kinds)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-046, FR-050, FR-054, FR-049 (SRD 5.2.1, "Suffocation [Hazard]", p. 189) - a
// creature with Constitution 8 holds its breath for the 30-second minimum, a choking creature suffocates at once, and six choking turns end
// at Exhaustion 6 where the host sees death and a seventh turn reports the maximum.
// Path: unhappy (boundary).
func TestChokingSuffocatesAtOnceAndSixTurnsKill(t *testing.T) {
	t.Parallel()
	ctx, victim := context.Background(), combat.MustCombatantID("victim")
	if got := (combat.BreathCapacity{}).ForConstitution(ability.MustScore(8).Modifier()); got != 30*time.Second {
		t.Errorf("capacity for Constitution 8 = %v, want the 30s minimum [FR-046]", got)
	}
	holding, err := combat.NewBreathHolding(10*time.Second, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewAdvanceBreath().Run(ctx, combat.AdvanceBreathCommand{State: holding, Change: combat.BeginChoking{}, Target: victim})
	if err != nil {
		t.Fatal(err)
	}
	choking := res.(combat.BreathAdvanced)
	if s, ok := choking.State().(combat.BreathSuffocating); !ok || s.Cause() != combat.SuffocationChoking ||
		!grpEKindsEqual(grpEKinds(choking.Events()), []string{"began_choking"}) {
		t.Fatalf("choking = %#v %v, want suffocating (choking) at once [FR-050]", choking.State(), grpEKinds(choking.Events()))
	}

	// A creature that is only holding its breath gains nothing at turn end (FR-054).
	calm, err := combat.NewResolveSuffocationTurnEnd().Run(ctx, combat.ResolveSuffocationTurnEndCommand{State: holding, Target: victim})
	if err != nil {
		t.Fatal(err)
	}
	if c := calm.(combat.SuffocationTurnEnded); c.Outcome() != combat.HazardNoExhaustion || grpEExhaustion(c.Conditions()) != 0 || len(c.Events()) != 0 {
		t.Errorf("holding at turn end = %v level %d, want no Exhaustion [FR-054]", c.Outcome(), grpEExhaustion(c.Conditions()))
	}

	var conds combat.ConditionSet
	record := combat.EmptyHazardRecord()
	turnEnd := combat.NewResolveSuffocationTurnEnd()
	for i := 1; i <= 6; i++ {
		out, err := turnEnd.Run(ctx, combat.ResolveSuffocationTurnEndCommand{State: choking.State(), Conditions: conds, Target: victim, Record: record})
		if err != nil {
			t.Fatal(err)
		}
		ended := out.(combat.SuffocationTurnEnded)
		conds, record = ended.Conditions(), ended.Record()
	}
	if grpEExhaustion(conds) != 6 || !(combat.ConditionEffects{}).IsDeadFromExhaustion(conds) {
		t.Fatalf("after six turns level = %d dead = %v, want 6 and the host sees death [FR-053, FR-111]", grpEExhaustion(conds),
			(combat.ConditionEffects{}).IsDeadFromExhaustion(conds))
	}
	out, err := turnEnd.Run(ctx, combat.ResolveSuffocationTurnEndCommand{State: choking.State(), Conditions: conds, Target: victim, Record: record})
	if err != nil {
		t.Fatal(err)
	}
	if o := out.(combat.SuffocationTurnEnded); o.Outcome() != combat.HazardExhaustionAtMaximum || grpEExhaustion(o.Conditions()) != 6 {
		t.Errorf("a seventh turn = %v level %d, want exhaustion_at_maximum and level 6 [FR-055]", o.Outcome(), grpEExhaustion(o.Conditions()))
	}

	// An immune creature is not harmed (FR-055).
	immune := combat.MustConditionSet(nil, []content.ConditionKind{content.ConditionExhaustion})
	safe, err := turnEnd.Run(ctx, combat.ResolveSuffocationTurnEndCommand{State: choking.State(), Conditions: immune, Target: victim})
	if err != nil {
		t.Fatal(err)
	}
	if s := safe.(combat.SuffocationTurnEnded); s.Outcome() != combat.HazardImmune || grpEExhaustion(s.Conditions()) != 0 || s.Record().SuffocationLevels() != 0 {
		t.Errorf("immune creature = %v level %d, want immune with nothing recorded [FR-055]", s.Outcome(), grpEExhaustion(s.Conditions()))
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-051, FR-052, section 7 error rows (SRD 5.2.1, "Suffocation [Hazard]", p. 189) -
// a breath change that does not fit the state is a refusal result, a negative elapsed time or a zero target is an invalid-input error, and a
// cancelled context is returned as is.
// Path: unhappy (invalid input, state error).
func TestBreathChangesThatDoNotFitAreRefusedOrInvalid(t *testing.T) {
	t.Parallel()
	ctx, who := context.Background(), combat.MustCombatantID("swimmer")
	holding, err := combat.NewBreathHolding(time.Second, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	choking, err := combat.NewBreathSuffocating(combat.SuffocationChoking)
	if err != nil {
		t.Fatal(err)
	}
	advance := combat.NewAdvanceBreath()
	for name, c := range map[string]struct {
		state  combat.BreathState
		change combat.BreathChange
		want   combat.BreathRejection
	}{
		"begin while already holding": {holding, combat.BeginHoldingBreath{}, combat.BreathRejectionNotBreathing},
		"begin while suffocating":     {choking, combat.BeginHoldingBreath{}, combat.BreathRejectionNotBreathing},
		"hold while breathing freely": {combat.BreathNormal{}, combat.HoldBreathFor{Elapsed: time.Second}, combat.BreathRejectionNotHoldingBreath},
		"hold while suffocating":      {choking, combat.HoldBreathFor{Elapsed: time.Second}, combat.BreathRejectionNotHoldingBreath},
		"choke while suffocating":     {choking, combat.BeginChoking{}, combat.BreathRejectionAlreadySuffocating},
	} {
		res, err := advance.Run(ctx, combat.AdvanceBreathCommand{State: c.state, Change: c.change, Target: who})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if r, ok := res.(combat.BreathAdvanceRejected); !ok || r.Rejection != c.want {
			t.Errorf("%s: result = %#v, want rejection %v [FR-051]", name, res, c.want)
		}
	}
	if _, err := advance.Run(ctx, combat.AdvanceBreathCommand{State: holding, Change: combat.HoldBreathFor{Elapsed: -time.Second}, Target: who}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative elapsed err = %v, want ErrInvalidInput [FR-052]", err)
	}
	if _, err := advance.Run(ctx, combat.AdvanceBreathCommand{State: holding, Change: combat.BeginChoking{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero target err = %v, want ErrInvalidInput", err)
	}
	if _, err := advance.Run(ctx, combat.AdvanceBreathCommand{State: holding, Target: who}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil change err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewBreathHolding(time.Minute, time.Minute); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("holding with elapsed equal to capacity err = %v, want ErrInvalidInput (that creature is suffocating)", err)
	}
	if _, err := combat.NewBreathSuffocating(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("suffocating with no cause err = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := advance.Run(cancelled, combat.AdvanceBreathCommand{State: holding, Change: combat.BeginChoking{}, Target: who}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context err = %v, want context.Canceled", err)
	}
	if _, err := combat.NewRestoreBreathing().Run(ctx, combat.RestoreBreathingCommand{State: combat.BreathHolding{}, Target: who}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("RestoreBreathing with the zero holding state err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-058, FR-059, FR-110 (SRD 5.2.1, "Suffocation [Hazard]", p. 189) - restoring
// the breathing of a creature that never suffocated changes nothing and calls no reduction, and a record that claims more Suffocation levels
// than the set holds ends the Exhaustion without a negative level.
// Path: unhappy (rule interaction).
func TestRestoreBreathingWithNothingToRestoreOrADriftedRecord(t *testing.T) {
	t.Parallel()
	ctx, who := context.Background(), combat.MustCombatantID("swimmer")
	res, err := combat.NewRestoreBreathing().Run(ctx, combat.RestoreBreathingCommand{State: combat.BreathNormal{}, Target: who})
	if err != nil {
		t.Fatal(err)
	}
	calm := res.(combat.BreathingRestored)
	if calm.LevelsRemoved() != 0 || len(calm.Events()) != 0 || calm.Reduction() != (combat.ExhaustionReductionNotAttempted{}) {
		t.Errorf("normal creature = removed %d events %v reduction %v, want nothing and not attempted [FR-058]", calm.LevelsRemoved(), grpEKinds(calm.Events()), calm.Reduction())
	}

	added, ok := (combat.ConditionSet{}).Apply(who, combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)}).(combat.ConditionAdded)
	if !ok {
		t.Fatal("could not give one Exhaustion level")
	}
	drifted := combat.MustHazardRecord(0, 0, 4, 0) // the record says four levels; the set holds one
	res, err = combat.NewRestoreBreathing().Run(ctx, combat.RestoreBreathingCommand{State: combat.BreathSuffocating{}, Conditions: added.Set(), Target: who, Record: drifted})
	if err == nil {
		t.Fatalf("a zero BreathSuffocating (no cause) was accepted: %#v", res)
	}
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput for the invalid state", err)
	}
	choking, err := combat.NewBreathSuffocating(combat.SuffocationChoking)
	if err != nil {
		t.Fatal(err)
	}
	res, err = combat.NewRestoreBreathing().Run(ctx, combat.RestoreBreathingCommand{State: choking, Conditions: added.Set(), Target: who, Record: drifted})
	if err != nil {
		t.Fatal(err)
	}
	fixed := res.(combat.BreathingRestored)
	if grpEExhaustion(fixed.Conditions()) != 0 || fixed.Record().SuffocationLevels() != 0 || fixed.LevelsRemoved() != 1 {
		t.Errorf("drifted record = level %d suffocation %d removed %d, want the one level ended and no negative level [FR-059, FR-110]",
			grpEExhaustion(fixed.Conditions()), fixed.Record().SuffocationLevels(), fixed.LevelsRemoved())
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-027, FR-030 to FR-034, FR-060, FR-065 (SRD 5.2.1, "Dehydration [Hazard]", p. 181) -
// a Medium traveller (1 gallon a day) drinks a quarter, then three eighths, then exactly half and gains a level only on the first two days,
// then drinks the full amount: the two locked levels unlock but are not removed.
// Path: happy.
func TestTravellerDehydratesOverDaysThenDrinksTheFullAmount(t *testing.T) {
	t.Parallel()
	ctx, who := context.Background(), combat.MustCombatantID("traveller")
	need, err := (combat.WaterNeedsPerDay{}).ForSize(content.SizeMedium)
	if err != nil || need.Eighths() != 8 {
		t.Fatalf("Medium need = %v, %v, want 8 eighths of a gallon [FR-027]", need, err)
	}
	water := func(eighths int) combat.WaterAmount {
		w, err := combat.NewWaterAmount(eighths)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	var conds combat.ConditionSet
	record := combat.EmptyHazardRecord()
	day := func(eighths int) combat.DehydrationDayEnded {
		t.Helper()
		res, err := combat.NewResolveDehydrationDayEnd().Run(ctx, combat.ResolveDehydrationDayEndCommand{
			Size: content.SizeMedium, Water: water(eighths), Conditions: conds, Target: who, Record: record})
		if err != nil {
			t.Fatal(err)
		}
		d := res.(combat.DehydrationDayEnded)
		conds, record = d.Conditions(), d.Record()
		return d
	}
	for i, drunk := range []int{2, 3} { // 2 < half of 8, 3 < half of 8
		if d := day(drunk); d.Outcome() != combat.HazardExhaustionGained || grpEExhaustion(conds) != i+1 || record.DehydrationLevels() != i+1 {
			t.Fatalf("day %d (%d eighths) = %v level %d, want one gained level [FR-031, FR-034]", i+1, drunk, d.Outcome(), grpEExhaustion(conds))
		}
	}
	if d := day(4); d.Outcome() != combat.HazardNoExhaustion || grpEExhaustion(conds) != 2 { // exactly half is not less than half
		t.Fatalf("day 3 (exactly half) = %v level %d, want no new level [FR-032]", d.Outcome(), grpEExhaustion(conds))
	}
	locked := combat.NewRemoveExhaustionLevels()
	blocked, err := locked.Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: conds, Target: who, Record: record, Levels: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := blocked.(combat.RemoveExhaustionLevelsRejected); !ok {
		t.Fatalf("removal while dehydrated = %#v, want a rejection: the levels are locked [FR-060, FR-062]", blocked)
	}
	full := day(8)
	kinds := grpEKinds(full.Events())
	if full.Outcome() != combat.HazardNoExhaustion || record.DehydrationLevels() != 0 || grpEExhaustion(conds) != 2 || !grpEKindsEqual(kinds, []string{"dehydration_recovered"}) {
		t.Fatalf("full day = %v record %d level %d events %v, want the 2 levels unlocked but still held [FR-033]", full.Outcome(), record.DehydrationLevels(), grpEExhaustion(conds), kinds)
	}
	if rec := full.Events()[0].(combat.DehydrationRecovered); rec.Levels() != 2 {
		t.Errorf("DehydrationRecovered.Levels() = %d, want 2", rec.Levels())
	}
	freed, err := locked.Run(ctx, combat.RemoveExhaustionLevelsCommand{Conditions: conds, Target: who, Record: record, Levels: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := freed.(combat.ExhaustionLevelsRemoved); !ok || r.Removed() != 1 || grpEExhaustion(r.Conditions()) != 1 {
		t.Errorf("removal after a full day = %#v, want one level removed [FR-065]", freed)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-027, FR-028, section 7 error rows - an invalid size, a zero target, a
// negative water amount and a cancelled context are errors, and an Exhaustion-immune creature is unharmed by thirst.
// Path: unhappy (invalid input).
func TestDehydrationDayEndRejectsBadInputAndImmuneCreaturesAreUnharmed(t *testing.T) {
	t.Parallel()
	ctx, who := context.Background(), combat.MustCombatantID("traveller")
	if _, err := combat.NewWaterAmount(-1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative water err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewFoodAmount(-1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative food err = %v, want ErrInvalidInput", err)
	}
	if _, err := (combat.WaterNeedsPerDay{}).ForSize(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("water need for the zero size err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewResolveDehydrationDayEnd().Run(ctx, combat.ResolveDehydrationDayEndCommand{Target: who}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero size err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewResolveDehydrationDayEnd().Run(ctx, combat.ResolveDehydrationDayEndCommand{Size: content.SizeMedium}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero target err = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := combat.NewResolveDehydrationDayEnd().Run(cancelled, combat.ResolveDehydrationDayEndCommand{Size: content.SizeMedium, Target: who}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context err = %v, want context.Canceled", err)
	}
	immune := combat.MustConditionSet(nil, []content.ConditionKind{content.ConditionExhaustion})
	res, err := combat.NewResolveDehydrationDayEnd().Run(ctx, combat.ResolveDehydrationDayEndCommand{Size: content.SizeMedium, Conditions: immune, Target: who})
	if err != nil {
		t.Fatal(err)
	}
	if d := res.(combat.DehydrationDayEnded); d.Outcome() != combat.HazardImmune || grpEExhaustion(d.Conditions()) != 0 || d.Record().DehydrationLevels() != 0 {
		t.Errorf("immune creature with no water = %v level %d, want immune and nothing recorded", d.Outcome(), grpEExhaustion(d.Conditions()))
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-036 to FR-044 (SRD 5.2.1, "Malnutrition [Hazard]", p. 185) - a Medium scout eats
// nothing for five days (a level on the fifth, no save), then eats a little: the host is asked for a DC 10 Constitution save, rolls it with
// d20, and a failure adds a second level; a full ration then unlocks both levels.
// Path: happy.
func TestScoutStarvesForFiveDaysThenFailsTheSaveAndEatsAgain(t *testing.T) {
	t.Parallel()
	ctx, who := context.Background(), combat.MustCombatantID("scout")
	food := func(eighths int) combat.FoodAmount {
		f, err := combat.NewFoodAmount(eighths)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	var conds combat.ConditionSet
	record := combat.EmptyHazardRecord()
	use := combat.NewResolveMalnutritionDayEnd()
	run := func(eaten int, save combat.FoodSavingThrow) combat.ResolveMalnutritionDayEndResult {
		t.Helper()
		res, err := use.Run(ctx, combat.ResolveMalnutritionDayEndCommand{Size: content.SizeMedium, Food: food(eaten), SavingThrow: save, Conditions: conds, Target: who, Record: record})
		if err != nil {
			t.Fatal(err)
		}
		if d, ok := res.(combat.MalnutritionDayEnded); ok {
			conds, record = d.Conditions(), d.Record()
		}
		return res
	}
	for day := 1; day <= 4; day++ {
		d := run(0, nil).(combat.MalnutritionDayEnded)
		if d.Outcome() != combat.HazardNoExhaustion || record.DaysWithoutFood() != day || grpEExhaustion(conds) != 0 {
			t.Fatalf("day %d without food = %v days %d level %d, want no level yet [FR-042]", day, d.Outcome(), record.DaysWithoutFood(), grpEExhaustion(conds))
		}
	}
	fifth := run(0, nil).(combat.MalnutritionDayEnded)
	if fifth.Outcome() != combat.HazardExhaustionGained || grpEExhaustion(conds) != 1 || record.MalnutritionLevels() != 1 {
		t.Fatalf("day 5 = %v level %d, want a level without a save [FR-043]", fifth.Outcome(), grpEExhaustion(conds))
	}

	// Day 6: a little food. The first call asks for the save and changes nothing.
	ask := run(3, combat.FoodSaveNotRolled{})
	need, ok := ask.(combat.MalnutritionSaveRequired)
	if !ok || need.DC() != d20.DCEasy || need.Ability() != ability.Constitution || grpEExhaustion(conds) != 1 || record.DaysWithoutFood() != 5 {
		t.Fatalf("first call = %#v, want a DC 10 Constitution save and no change [FR-038]", ask)
	}
	failed := run(3, combat.FoodSaveRolled{Outcome: d20.Failure}).(combat.MalnutritionDayEnded)
	if failed.Outcome() != combat.HazardExhaustionGained || grpEExhaustion(conds) != 2 || record.MalnutritionLevels() != 2 || record.DaysWithoutFood() != 0 {
		t.Fatalf("failed save = %v level %d record %+v, want a second level and the run of fasting days ended [FR-039, FR-044]", failed.Outcome(), grpEExhaustion(conds), record)
	}
	full := run(8, nil).(combat.MalnutritionDayEnded)
	if full.Outcome() != combat.HazardNoExhaustion || record.MalnutritionLevels() != 0 || grpEExhaustion(conds) != 2 ||
		!grpEKindsEqual(grpEKinds(full.Events()), []string{"malnutrition_recovered"}) {
		t.Fatalf("full ration = %v record %+v level %d, want both levels unlocked but still held [FR-037]", full.Outcome(), record, grpEExhaustion(conds))
	}
	if rec := full.Events()[0].(combat.MalnutritionRecovered); rec.Levels() != 2 {
		t.Errorf("MalnutritionRecovered.Levels() = %d, want 2", rec.Levels())
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-040, FR-041, FR-044 and section 7 error rows (SRD 5.2.1, "Malnutrition
// [Hazard]", p. 185) - a passed save adds no level, a save reported when none is needed is ignored, and an invalid size, a zero target or a
// negative amount is an error.
// Path: unhappy (rule interaction, invalid input).
func TestMalnutritionSavePassedIgnoredAndInvalidInputs(t *testing.T) {
	t.Parallel()
	ctx, who := context.Background(), combat.MustCombatantID("scout")
	use := combat.NewResolveMalnutritionDayEnd()
	three, _ := combat.NewFoodAmount(3)
	passed, err := use.Run(ctx, combat.ResolveMalnutritionDayEndCommand{Size: content.SizeMedium, Food: three, SavingThrow: combat.FoodSaveRolled{Outcome: d20.Success}, Target: who})
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := passed.(combat.MalnutritionDayEnded); !ok || d.Outcome() != combat.HazardNoExhaustion || grpEExhaustion(d.Conditions()) != 0 {
		t.Errorf("passed save = %#v, want no level [FR-040]", passed)
	}
	half, _ := combat.NewFoodAmount(4)
	ignored, err := use.Run(ctx, combat.ResolveMalnutritionDayEndCommand{Size: content.SizeMedium, Food: half, SavingThrow: combat.FoodSaveRolled{Outcome: d20.Failure}, Target: who})
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := ignored.(combat.MalnutritionDayEnded); !ok || d.Outcome() != combat.HazardNoExhaustion || grpEExhaustion(d.Conditions()) != 0 {
		t.Errorf("a save reported after eating half = %#v, want it ignored [FR-041]", ignored)
	}
	if _, err := use.Run(ctx, combat.ResolveMalnutritionDayEndCommand{Target: who}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero size err = %v, want ErrInvalidInput", err)
	}
	if _, err := use.Run(ctx, combat.ResolveMalnutritionDayEndCommand{Size: content.SizeMedium}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero target err = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := use.Run(cancelled, combat.ResolveMalnutritionDayEndCommand{Size: content.SizeMedium, Target: who}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context err = %v, want context.Canceled", err)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-067, FR-070, FR-075, FR-081 (SRD 5.2.1, "Breaking Objects", Object Hit Points,
// p. 178) - a Resilient Medium chest rolls 4d8 Hit Points instead of taking the average, then two blows of 10 bludgeoning (the second with a
// Damage Threshold of 5 met) leave it with 4 and then destroy it.
// Path: happy.
func TestChestRollsHitPointsThenIsBrokenByTwoBlows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rolls, err := combat.NewRollObjectHitPoints(dicetest.NewFixedRoller(2, 3, 4, 5))
	if err != nil {
		t.Fatal(err)
	}
	res, err := rolls.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeMedium, Durability: combat.ObjectResilient})
	if err != nil {
		t.Fatal(err)
	}
	rolled, ok := res.(combat.ObjectHitPointsRolled)
	if !ok || rolled.HitPoints() != combat.MustHitPoints(14, 14, 0) || rolled.Roll().Total() != 14 || rolled.Entry().HitPoints() != 18 {
		t.Fatalf("result = %#v, want 14 Hit Points rolled against the table average 18 [FR-067, FR-070]", res)
	}
	threshold, err := combat.NewDamageThreshold(5)
	if err != nil {
		t.Fatal(err)
	}
	chest, err := combat.NewObjectStatus(content.SizeMedium, 15, rolled.HitPoints(), threshold, combat.ObjectAffinities{}.Standard(), nil)
	if err != nil {
		t.Fatal(err)
	}
	blow := func() combat.ObjectHarm {
		h, err := combat.NewObjectDamageInstance([]combat.Damage{combat.MustDamage(10, content.DamageBludgeoning)}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	hit := combat.NewApplyDamageToObject()
	first, err := hit.Run(ctx, combat.ApplyDamageToObjectCommand{Status: chest, Harm: blow()})
	if err != nil {
		t.Fatal(err)
	}
	damaged := first.(combat.ObjectDamageApplied)
	if damaged.Status().HitPoints().Current() != 4 || damaged.Status().IsDestroyed() || damaged.Events()[0].Kind() != "object_damaged" {
		t.Fatalf("first blow = %#v, want 4 Hit Points left [FR-075]", damaged)
	}
	second, err := hit.Run(ctx, combat.ApplyDamageToObjectCommand{Status: damaged.Status(), Harm: blow()})
	if err != nil {
		t.Fatal(err)
	}
	if s := second.(combat.ObjectDamageApplied); !s.Status().IsDestroyed() || s.Events()[len(s.Events())-1].Kind() != "object_destroyed" {
		t.Errorf("second blow = %#v, want the chest destroyed", s)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-071, FR-094, FR-095 and section 7 error rows (SRD 5.2.1, "Breaking Objects",
// pp. 177-178) - a Huge object is a split-required result with no roll, a failing roller is a port error recorded once, a roller that answers
// another expression is a contract violation, and a nil roller or an invalid durability is invalid input.
// Path: unhappy (boundary, failure of collaborators).
func TestObjectHitPointsHugeFailingRollerAndInvalidInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	asked := 0
	counting := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		asked++
		return dice.Result{}, errors.New("unreachable")
	})
	uc, err := combat.NewRollObjectHitPoints(counting)
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeHuge, Durability: combat.ObjectFragile})
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(combat.ObjectHitPointsRollSplitRequired); !ok || s.Size != content.SizeHuge || asked != 0 {
		t.Errorf("Huge object = %#v after %d rolls, want split-required and no roll [FR-071]", res, asked)
	}

	backendDown := errors.New("backend down")
	failing, err := combat.NewRollObjectHitPoints(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, backendDown }))
	if err != nil {
		t.Fatal(err)
	}
	_, err = failing.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeSmall, Durability: combat.ObjectFragile})
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, backendDown) || srderr.ErrorType(err) != "dice_roller_error" {
		t.Errorf("roller failure err = %v, want a dice port error wrapping the cause [FR-094]", err)
	}
	wrong, err := combat.NewRollObjectHitPoints(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewFixedRoller(3).Roll(dice.MustExpression(1, dice.D20(), 1))
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = wrong.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeSmall, Durability: combat.ObjectFragile}); !errors.Is(err, srderr.ErrContractViolation) {
		t.Errorf("wrong expression err = %v, want ErrContractViolation [FR-095]", err)
	}
	if _, err := combat.NewRollObjectHitPoints(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller err = %v, want ErrInvalidInput", err)
	}
	if _, err := uc.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeSmall}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero durability err = %v, want ErrInvalidInput", err)
	}
	if _, err := uc.Run(ctx, combat.RollObjectHitPointsCommand{Durability: combat.ObjectFragile}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero size err = %v, want ErrInvalidInput", err)
	}
}
