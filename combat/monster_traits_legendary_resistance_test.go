package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
)

func mtLR(tb testing.TB, s combat.LimitedUseState, tr combat.LegendaryResistanceTrait, f combat.SaveFailure) combat.LegendaryResistanceOutcome {
	tb.Helper()
	out, err := combat.UseLegendaryResistance(s, tr, mtID(), f)
	if err != nil {
		tb.Fatal(err)
	}
	return out
}

func mtFailedSave(tb testing.TB) combat.RolledSaveFailure {
	tb.Helper()
	return combat.RolledSaveFailure{Result: mtSave(tb, 5, 0, 15)}
}

// Requirement: doc/specs/monster-traits.md FR-010 - a failed save with a use left gives Applied: one use spent, the failure kept, the effective outcome Success, events LimitedUsePartSpent then LegendaryResistanceUsed for [subject].
// Path: happy.
func TestLegendaryResistanceApplied(t *testing.T) {
	t.Parallel()
	state, tr := mtLimited(t)
	failure := mtFailedSave(t)
	a, ok := mtLR(t, state, tr, failure).(combat.LegendaryResistanceApplied)
	if !ok || a.EffectiveOutcome() != d20.Success {
		t.Fatalf("outcome = %#v", a)
	}
	if got := mtKindsOf(a.Events()); len(got) != 2 || got[0] != "limited_use_part_spent" || got[1] != "legendary_resistance_used" {
		t.Errorf("events = %v", got)
	}
	if left := luPerDayOf(t, a.State(), tr.Part()).Remaining(); left != 2 {
		t.Errorf("uses left = %d, want 2", left)
	}
	if luPerDayOf(t, state, tr.Part()).Remaining() != 3 {
		t.Error("the input state changed")
	}
	if f, ok := a.Failure().(combat.RolledSaveFailure); !ok || !f.Result.Equal(failure.Result) || failure.Result.Outcome() != d20.Failure {
		t.Error("the original failure was edited")
	}
	ev := a.Events()
	ev[0] = nil
	if a.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}

// Requirement: doc/specs/monster-traits.md FR-011 - a chosen or automatic failure is accepted like a rolled one.
// Path: unhappy (rule interaction).
func TestLegendaryResistanceAcceptsForcedFailures(t *testing.T) {
	t.Parallel()
	state, tr := mtLimited(t)
	for _, reason := range []d20.ForcedFailureReason{d20.ChosenToFail, d20.Automatic} {
		f := combat.ForcedSaveFailure{Result: luMust(d20.FailWithoutRoll(d20.SavingThrow, reason))}
		if _, ok := mtLR(t, state, tr, f).(combat.LegendaryResistanceApplied); !ok {
			t.Errorf("%v was refused", reason)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-012 - refusals leave the state unchanged with no event, first match wins: NotTracked, then NotAFailure, then Exhausted.
// Path: unhappy (boundary).
func TestLegendaryResistanceRefusalOrder(t *testing.T) {
	t.Parallel()
	state, tr := mtLimited(t)
	passed := combat.RolledSaveFailure{Result: mtSave(t, 15, 0, 15)}
	untracked := luMust(combat.NewLegendaryResistanceTrait(luRef(t, combat.MonsterPartTraits, 5)))
	exhausted := state
	for range 3 {
		exhausted = mtLR(t, exhausted, tr, mtFailedSave(t)).(combat.LegendaryResistanceApplied).State()
	}
	cases := []struct {
		name  string
		state combat.LimitedUseState
		trait combat.LegendaryResistanceTrait
		f     combat.SaveFailure
		want  combat.LegendaryResistanceRefusal
	}{
		{"not tracked beats success", state, untracked, passed, combat.LegendaryResistanceNotTracked},
		{"success beats exhausted", exhausted, tr, passed, combat.LegendaryResistanceNotAFailure},
		{"success", state, tr, passed, combat.LegendaryResistanceNotAFailure},
		{"fourth use outside the lair", exhausted, tr, mtFailedSave(t), combat.LegendaryResistanceExhausted},
	}
	for _, c := range cases {
		got, ok := mtLR(t, c.state, c.trait, c.f).(combat.LegendaryResistanceRefused)
		if !ok || got.Reason != c.want {
			t.Errorf("%s: got %#v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-014 - 3 uses outside the lair and 4 in it, and a Long Rest restores them through LimitedUseState.
// Path: unhappy (boundary).
func TestLegendaryResistanceUsesFollowTheLair(t *testing.T) {
	t.Parallel()
	state, tr := mtLimited(t)
	lair := luStart(t, mtMonsterOf(t), combat.InLair)
	count := func(s combat.LimitedUseState) int {
		n := 0
		for {
			a, ok := mtLR(t, s, tr, mtFailedSave(t)).(combat.LegendaryResistanceApplied)
			if !ok {
				return n
			}
			s, n = a.State(), n+1
		}
	}
	if count(state) != 3 || count(lair) != 4 {
		t.Errorf("uses outside %d, in lair %d, want 3 and 4", count(state), count(lair))
	}
}

// Requirement: doc/specs/monster-traits.md FR-014 - a Long Rest restores the uses.
// Path: unhappy (rule interaction).
func TestLegendaryResistanceRestoredByLongRest(t *testing.T) {
	t.Parallel()
	state, tr := mtLimited(t)
	used := mtLR(t, state, tr, mtFailedSave(t)).(combat.LegendaryResistanceApplied).State()
	rested := used.AfterRest(restLong()).State()
	if luPerDayOf(t, rested, tr.Part()).Remaining() != 3 {
		t.Error("a Long Rest did not restore the uses")
	}
}

// Requirement: doc/specs/monster-traits.md FR-013 - invalid input is an error: an ability check, a zero trait, state, subject or failure, a nil failure, and a state of another combatant.
// Path: unhappy (invalid input).
func TestLegendaryResistanceInvalidInput(t *testing.T) {
	t.Parallel()
	state, tr := mtLimited(t)
	check := combat.RolledSaveFailure{Result: luMust(d20.ResolveCheck(d20.AbilityCheck, luMust(d20.NewRoll(0, []int{3})), d20.NewRollModifiers(modifierZero()), 15))}
	forcedCheck := combat.ForcedSaveFailure{Result: luMust(d20.FailWithoutRoll(d20.AbilityCheck, d20.Automatic))}
	cases := []struct {
		name  string
		state combat.LimitedUseState
		trait combat.LegendaryResistanceTrait
		who   combat.CombatantID
		f     combat.SaveFailure
	}{
		{"ability check", state, tr, mtID(), check},
		{"forced ability check", state, tr, mtID(), forcedCheck},
		{"zero rolled result", state, tr, mtID(), combat.RolledSaveFailure{}},
		{"zero forced result", state, tr, mtID(), combat.ForcedSaveFailure{}},
		{"nil failure", state, tr, mtID(), nil},
		{"zero trait", state, combat.LegendaryResistanceTrait{}, mtID(), mtFailedSave(t)},
		{"zero state", combat.LimitedUseState{}, tr, mtID(), mtFailedSave(t)},
		{"zero subject", state, tr, "", mtFailedSave(t)},
		{"another combatant", state, tr, "other", mtFailedSave(t)},
		{"another combatant sorting first", state, tr, "aaa", mtFailedSave(t)},
	}
	for _, c := range cases {
		out, err := combat.UseLegendaryResistance(c.state, c.trait, c.who, c.f)
		if !isInvalid(err) || out != nil {
			t.Errorf("%s: got %#v, %v", c.name, out, err)
		}
	}
}
