package combat_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func conMod(score int) ability.Modifier { return ability.MustScore(score).Modifier() }

func sec(n int) time.Duration { return time.Duration(n) * time.Second }

func holding(t *testing.T, elapsed, capacity time.Duration) combat.BreathHolding {
	t.Helper()
	h, err := combat.NewBreathHolding(elapsed, capacity)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-046 (table 4.7a) - a creature can hold its breath for the longer of 30 seconds and 1 + its Constitution modifier minutes.
// Path: per case.
func TestBreathCapacityForConstitution(t *testing.T) {
	t.Parallel()
	cases := []struct {
		score, modifier int
		want            time.Duration
	}{{1, -5, sec(30)}, {6, -2, sec(30)}, {8, -1, sec(30)}, {10, 0, sec(60)}, {12, 1, sec(120)}, {16, 3, sec(240)}, {30, 10, sec(660)}}
	for _, c := range cases {
		if m := conMod(c.score); m.Value() != c.modifier {
			t.Fatalf("score %d has modifier %d, test table is wrong", c.score, m.Value())
		}
		if got := (combat.BreathCapacity{}).ForConstitution(conMod(c.score)); got != c.want {
			t.Errorf("capacity for modifier %d = %v, want %v", c.modifier, got, c.want)
		}
	}
}

func advanceBreath(t *testing.T, state combat.BreathState, change combat.BreathChange) (combat.AdvanceBreathResult, error) {
	t.Helper()
	return combat.NewAdvanceBreath().Run(context.Background(), combat.AdvanceBreathCommand{State: state, Change: change, Target: subject()})
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-047 to FR-052 (table 4.7b) - the breath state machine: holding starts at zero with the capacity, a hold adds time, reaching the capacity means suffocating, choking needs no wait, and a change that does not fit the state is a rejection with no event.
// Path: per case.
func TestAdvanceBreath(t *testing.T) {
	t.Parallel()
	choking := mustSuffocating(t, combat.SuffocationChoking)
	cases := []struct {
		name, requirement, path string
		state                   combat.BreathState
		change                  combat.BreathChange
		want                    combat.BreathState
		reject                  combat.BreathRejection
		events                  []string
	}{
		{"begin holding", "FR-047", "happy", combat.BreathNormal{}, combat.BeginHoldingBreath{ConstitutionModifier: conMod(10)}, holding(t, 0, sec(60)), 0, []string{"breath_held_started"}},
		{"begin holding from nil", "FR-047", "unhappy (empty and minimal)", nil, combat.BeginHoldingBreath{ConstitutionModifier: conMod(12)}, holding(t, 0, sec(120)), 0, []string{"breath_held_started"}},
		{"partway", "FR-048", "happy", holding(t, 0, sec(60)), combat.HoldBreathFor{Elapsed: sec(30)}, holding(t, sec(30), sec(60)), 0, []string{}},
		{"zero elapsed", "FR-048", "unhappy (boundary)", holding(t, sec(30), sec(60)), combat.HoldBreathFor{}, holding(t, sec(30), sec(60)), 0, []string{}},
		{"one second below capacity", "FR-048", "unhappy (boundary)", holding(t, sec(30), sec(60)), combat.HoldBreathFor{Elapsed: sec(29)}, holding(t, sec(59), sec(60)), 0, []string{}},
		{"exactly capacity", "FR-049", "unhappy (boundary)", holding(t, sec(30), sec(60)), combat.HoldBreathFor{Elapsed: sec(30)}, mustSuffocating(t, combat.SuffocationRanOutOfBreath), 0, []string{"ran_out_of_breath"}},
		{"beyond capacity", "FR-049", "unhappy (boundary)", holding(t, sec(30), sec(60)), combat.HoldBreathFor{Elapsed: sec(600)}, mustSuffocating(t, combat.SuffocationRanOutOfBreath), 0, []string{"ran_out_of_breath"}},
		{"enormous elapsed does not overflow", "FR-049", "unhappy (boundary)", holding(t, sec(30), sec(60)), combat.HoldBreathFor{Elapsed: time.Duration(math.MaxInt64)}, mustSuffocating(t, combat.SuffocationRanOutOfBreath), 0, []string{"ran_out_of_breath"}},
		{"choke while breathing", "FR-050", "happy", combat.BreathNormal{}, combat.BeginChoking{}, choking, 0, []string{"began_choking"}},
		{"choke while holding", "FR-050", "unhappy (rule interaction)", holding(t, sec(10), sec(60)), combat.BeginChoking{}, choking, 0, []string{"began_choking"}},
		{"begin holding while holding", "FR-051", "unhappy (state error)", holding(t, 0, sec(60)), combat.BeginHoldingBreath{}, nil, combat.BreathRejectionNotBreathing, nil},
		{"begin holding while suffocating", "FR-051", "unhappy (state error)", choking, combat.BeginHoldingBreath{}, nil, combat.BreathRejectionNotBreathing, nil},
		{"hold while breathing", "FR-051", "unhappy (state error)", combat.BreathNormal{}, combat.HoldBreathFor{Elapsed: sec(10)}, nil, combat.BreathRejectionNotHoldingBreath, nil},
		{"hold while suffocating", "FR-051", "unhappy (state error)", choking, combat.HoldBreathFor{Elapsed: sec(10)}, nil, combat.BreathRejectionNotHoldingBreath, nil},
		{"choke while suffocating", "FR-051", "unhappy (state error)", choking, combat.BeginChoking{}, nil, combat.BreathRejectionAlreadySuffocating, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := advanceBreath(t, c.state, c.change)
			if err != nil {
				t.Fatal(err)
			}
			if c.reject != 0 {
				if res != (combat.BreathAdvanceRejected{Rejection: c.reject}) {
					t.Errorf("= %#v, want the rejection %v [%s]", res, c.reject, c.requirement)
				}
				return
			}
			got, ok := res.(combat.BreathAdvanced)
			if !ok || got.State() != c.want || !equalStrings(envKinds(got.Events()), c.events) {
				t.Errorf("= %#v, want state %#v and events %v [%s]", res, c.want, c.events, c.requirement)
			}
		})
	}
}

func mustSuffocating(t *testing.T, cause combat.SuffocationCause) combat.BreathSuffocating {
	t.Helper()
	s, err := combat.NewBreathSuffocating(cause)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-052, FR-051 - a negative elapsed time is an ErrInvalidInput error the span records; a state or change that is not valid, a zero target and a cancelled context are errors before any signal.
// Path: unhappy (invalid input).
func TestAdvanceBreathInvalidInput(t *testing.T) {
	t.Parallel()
	uc := combat.NewAdvanceBreath()
	h := holding(t, sec(30), sec(60))
	if _, err := uc.Run(context.Background(), combat.AdvanceBreathCommand{State: h, Change: combat.HoldBreathFor{Elapsed: -1}, Target: subject()}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative elapsed = %v, want ErrInvalidInput", err)
	}
	bad := map[string]combat.AdvanceBreathCommand{
		"zero target": {State: h, Change: combat.BeginChoking{}}, "nil change": {State: h, Target: subject()},
		"zero BreathHolding":     {State: combat.BreathHolding{}, Change: combat.BeginChoking{}, Target: subject()},
		"zero BreathSuffocating": {State: combat.BreathSuffocating{}, Change: combat.BeginChoking{}, Target: subject()},
		"unknown state":          {State: badBreath{}, Change: combat.BeginChoking{}, Target: subject()},
		"unknown change":         {State: h, Change: badChange{}, Target: subject()},
	}
	for name, cmd := range bad {
		if _, err := uc.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, combat.AdvanceBreathCommand{State: h, Change: combat.BeginChoking{}, Target: subject()}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}

type badBreath struct{ combat.BreathState }
type badChange struct{ combat.BreathChange }

// Requirement: doc/specs/environment-hazards-and-objects.md FR-047, FR-049 - BreathHolding and BreathSuffocating are built only from valid values: elapsed from 0 to below the capacity, and a known cause.
// Path: unhappy (invalid input).
func TestBreathStateConstructors(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewBreathHolding(0, 1); err != nil {
		t.Errorf("elapsed 0 of capacity 1 = %v, want valid", err)
	}
	for name, args := range map[string][2]time.Duration{"negative elapsed": {-1, sec(60)}, "zero capacity": {0, 0}, "negative capacity": {0, -1},
		"elapsed equals capacity": {sec(60), sec(60)}, "elapsed above capacity": {sec(61), sec(60)}} {
		if _, err := combat.NewBreathHolding(args[0], args[1]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	h := holding(t, sec(59), sec(60))
	if h.Elapsed() != sec(59) || h.Capacity() != sec(60) {
		t.Errorf("accessors = %v/%v", h.Elapsed(), h.Capacity())
	}
	for _, c := range []combat.SuffocationCause{0, 3, -1} {
		if _, err := combat.NewBreathSuffocating(c); !errors.Is(err, srderr.ErrInvalidInput) || c.Valid() || c.String() != "unknown" {
			t.Errorf("cause %d = %v", int(c), err)
		}
	}
	if s := mustSuffocating(t, combat.SuffocationChoking); s.Cause() != combat.SuffocationChoking || s.Cause().String() != "choking" ||
		combat.SuffocationRanOutOfBreath.String() != "ran_out_of_breath" {
		t.Error("cause accessor or names are wrong")
	}
	for r, n := range map[combat.BreathRejection]string{combat.BreathRejectionNotBreathing: "not_breathing", combat.BreathRejectionNotHoldingBreath: "not_holding_breath",
		combat.BreathRejectionAlreadySuffocating: "already_suffocating", 0: "unknown"} {
		if r.String() != n {
			t.Errorf("rejection %d = %q, want %q", int(r), r, n)
		}
	}
}

func suffocationTurnEnd(t *testing.T, state combat.BreathState, set combat.ConditionSet, rec combat.HazardRecord) combat.SuffocationTurnEnded {
	t.Helper()
	res, err := combat.NewResolveSuffocationTurnEnd().Run(context.Background(), combat.ResolveSuffocationTurnEndCommand{State: state, Conditions: set, Target: subject(), Record: rec})
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.SuffocationTurnEnded)
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-053 to FR-055, FR-111 (table 4.10a) - a suffocating creature gains one Exhaustion level at the end of its turn and the record counts it; a breathing or holding one gains nothing; level 6 and Immunity change nothing.
// Path: per case.
func TestResolveSuffocationTurnEnd(t *testing.T) {
	t.Parallel()
	choking := mustSuffocating(t, combat.SuffocationChoking)
	cases := []struct {
		name, requirement, path string
		state                   combat.BreathState
		set                     combat.ConditionSet
		rec                     combat.HazardRecord
		outcome                 combat.HazardExhaustionOutcome
		levelAfter, recAfter    int
	}{
		{"suffocating", "FR-053", "happy", choking, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), combat.HazardExhaustionGained, 1, 1},
		{"suffocating, second turn", "FR-053", "happy", choking, envExhaustion(t, 1), envRecord(t, 0, 0, 1, 0, 0), combat.HazardExhaustionGained, 2, 2},
		{"breathing", "FR-054", "unhappy (state error)", combat.BreathNormal{}, envExhaustion(t, 1), envRecord(t, 0, 0, 0, 0, 0), combat.HazardNoExhaustion, 1, 0},
		{"nil state", "FR-054", "unhappy (empty and minimal)", nil, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), combat.HazardNoExhaustion, 0, 0},
		{"holding", "FR-054", "unhappy (state error)", holding(t, sec(5), sec(60)), envExhaustion(t, 1), envRecord(t, 0, 0, 0, 0, 0), combat.HazardNoExhaustion, 1, 0},
		{"level 5 to 6", "FR-053, FR-111", "unhappy (boundary)", choking, envExhaustion(t, 5), envRecord(t, 0, 0, 5, 0, 0), combat.HazardExhaustionGained, 6, 6},
		{"level 6", "FR-055", "unhappy (rule interaction)", choking, envExhaustion(t, 6), envRecord(t, 0, 0, 6, 0, 0), combat.HazardExhaustionAtMaximum, 6, 6},
		{"level 6, smaller record", "FR-055", "unhappy (rule interaction)", choking, envExhaustion(t, 6), envRecord(t, 0, 0, 3, 0, 0), combat.HazardExhaustionAtMaximum, 6, 3},
		{"immune with a stored level", "FR-055", "unhappy (rule interaction)", choking, condSet(t, condConds(condLevel(2)), content.ConditionExhaustion), envRecord(t, 0, 0, 1, 0, 0), combat.HazardImmune, 2, 1},
		{"immune", "FR-055", "unhappy (rule interaction)", choking, condSet(t, nil, content.ConditionExhaustion), envRecord(t, 0, 0, 0, 0, 0), combat.HazardImmune, 0, 0},
		{"record above the set is clamped", "FR-110", "unhappy (state error)", combat.BreathNormal{}, envExhaustion(t, 1), envRecord(t, 0, 0, 3, 0, 0), combat.HazardNoExhaustion, 1, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := suffocationTurnEnd(t, c.state, c.set, c.rec)
			if got.Outcome() != c.outcome || envLevel(got.Conditions()) != c.levelAfter || got.Record().SuffocationLevels() != c.recAfter {
				t.Errorf("= %v level %d record %d, want %v %d %d [%s]", got.Outcome(), envLevel(got.Conditions()), got.Record().SuffocationLevels(),
					c.outcome, c.levelAfter, c.recAfter, c.requirement)
			}
			if (got.Outcome() == combat.HazardExhaustionGained) != (len(got.Events()) == 1) {
				t.Errorf("events = %v for outcome %v", envKinds(got.Events()), got.Outcome())
			}
		})
	}
	uc := combat.NewResolveSuffocationTurnEnd()
	for name, cmd := range map[string]combat.ResolveSuffocationTurnEndCommand{"zero target": {}, "bad state": {State: combat.BreathHolding{}, Target: subject()}} {
		if _, err := uc.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, combat.ResolveSuffocationTurnEndCommand{Target: subject()}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}

func restoreBreathing(t *testing.T, state combat.BreathState, set combat.ConditionSet, rec combat.HazardRecord) combat.BreathingRestored {
	t.Helper()
	res, err := combat.NewRestoreBreathing().Run(context.Background(), combat.RestoreBreathingCommand{State: state, Conditions: set, Target: subject(), Record: rec})
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.BreathingRestored)
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-056 to FR-059, FR-107, FR-108, FR-110, FR-120 (tables 4.7c, 4.10a) - restoring breath removes only the levels the record attributes to Suffocation, keeps the other sources, tolerates a record larger than the set, and reports BreathRestored with the levels actually removed, or nothing in the no-op case.
// Path: per case.
func TestRestoreBreathing(t *testing.T) {
	t.Parallel()
	choking := mustSuffocating(t, combat.SuffocationChoking)
	cases := []struct {
		name, requirement, path string
		state                   combat.BreathState
		set                     combat.ConditionSet
		rec                     combat.HazardRecord
		levelAfter, recAfter    int
		reduction               string // "reduced", "ended", "not_present", "not_attempted"
		removed                 int
		events                  []string
	}{
		{"all suffocation levels", "FR-056", "happy", choking, envExhaustion(t, 2), envRecord(t, 0, 0, 2, 0, 0), 0, 0, "ended", 2, []string{"condition_removed", "breath_restored"}},
		{"some from suffocation", "FR-057", "unhappy (rule interaction)", choking, envExhaustion(t, 3), envRecord(t, 0, 0, 2, 0, 0), 1, 0, "reduced", 2, []string{"condition_level_reduced", "breath_restored"}},
		{"dehydration level kept", "FR-057", "unhappy (rule interaction)", choking, envExhaustion(t, 3), envRecord(t, 2, 0, 1, 0, 0), 2, 0, "reduced", 1, []string{"condition_level_reduced", "breath_restored"}},
		{"malnutrition level kept", "FR-057", "unhappy (rule interaction)", choking, envExhaustion(t, 2), envRecord(t, 0, 1, 1, 0, 0), 1, 0, "reduced", 1, []string{"condition_level_reduced", "breath_restored"}},
		{"holding, nothing gained", "FR-058", "unhappy (empty and minimal)", holding(t, sec(5), sec(60)), envExhaustion(t, 1), envRecord(t, 1, 0, 0, 0, 0), 1, 0, "not_attempted", 0, []string{}},
		{"breathing, nothing gained", "FR-058", "unhappy (empty and minimal)", combat.BreathNormal{}, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), 0, 0, "not_attempted", 0, []string{}},
		{"record exceeds the level", "FR-059, FR-107", "unhappy (state error)", choking, envExhaustion(t, 1), envRecord(t, 0, 0, 3, 0, 0), 0, 0, "ended", 1, []string{"condition_removed", "breath_restored"}},
		{"record but no Exhaustion", "FR-059", "unhappy (state error)", choking, envExhaustion(t, 0), envRecord(t, 0, 0, 2, 0, 0), 0, 0, "not_present", 0, []string{"breath_restored"}},
		{"from level 6", "FR-056", "unhappy (boundary)", choking, envExhaustion(t, 6), envRecord(t, 0, 0, 6, 0, 0), 0, 0, "ended", 6, []string{"condition_removed", "breath_restored"}},
		{"leftover record, breathing", "FR-108", "unhappy (state error)", combat.BreathNormal{}, envExhaustion(t, 2), envRecord(t, 0, 0, 2, 0, 0), 0, 0, "ended", 2, []string{"condition_removed", "breath_restored"}},
		{"leftover record, holding", "FR-108", "unhappy (state error)", holding(t, sec(5), sec(60)), envExhaustion(t, 3), envRecord(t, 2, 0, 1, 0, 0), 2, 0, "reduced", 1, []string{"condition_level_reduced", "breath_restored"}},
		{"choking, restored at once", "FR-107", "unhappy (boundary)", choking, envExhaustion(t, 0), envRecord(t, 0, 0, 0, 0, 0), 0, 0, "not_attempted", 0, []string{"breath_restored"}},
		{"other sources clamped to the level", "FR-110", "unhappy (state error)", choking, envExhaustion(t, 3), envRecord(t, 2, 2, 0, 0, 0), 3, 0, "not_attempted", 0, []string{"breath_restored"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := restoreBreathing(t, c.state, c.set, c.rec)
			red := "not_attempted"
			if m, ok := got.Reduction().(combat.ExhaustionReductionMade); ok {
				switch m.Result.(type) {
				case combat.ExhaustionReduced:
					red = "reduced"
				case combat.ExhaustionEnded:
					red = "ended"
				case combat.ExhaustionReductionNotPresent:
					red = "not_present"
				}
			}
			if got.State() != (combat.BreathNormal{}) || envLevel(got.Conditions()) != c.levelAfter || got.Record().SuffocationLevels() != c.recAfter ||
				red != c.reduction || got.LevelsRemoved() != c.removed || !equalStrings(envKinds(got.Events()), c.events) {
				t.Errorf("= level %d suffocation %d %s removed %d events %v, want %d %d %s %d %v [%s]", envLevel(got.Conditions()), got.Record().SuffocationLevels(),
					red, got.LevelsRemoved(), envKinds(got.Events()), c.levelAfter, c.recAfter, c.reduction, c.removed, c.events, c.requirement)
			}
			if got.Record().DehydrationLevels() != min(c.rec.DehydrationLevels(), envLevel(got.Conditions())) && c.recAfter == 0 && c.rec.DehydrationLevels() <= envLevel(got.Conditions()) {
				t.Error("the Dehydration levels changed")
			}
			if n := len(got.Events()); n > 0 {
				if br := got.Events()[n-1].(combat.BreathRestored); br.LevelsRemoved() != c.removed || br.Combatants()[0] != subject() {
					t.Errorf("BreathRestored = %+v", br)
				}
			}
		})
	}
	uc := combat.NewRestoreBreathing()
	for name, cmd := range map[string]combat.RestoreBreathingCommand{"zero target": {}, "bad state": {State: combat.BreathSuffocating{}, Target: subject()}} {
		if _, err := uc.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, combat.RestoreBreathingCommand{Target: subject()}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
