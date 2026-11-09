package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-fighter.md Table D, FR-065, FR-067, FR-068, FR-069, FR-070 - at DC 15 a failed check of total 12 with a
// d10 of 3 is TacticalMindSucceeded (equals the DC) and expends a Second Wind use; with 2 it is TacticalMindStillFailed and expends
// none; the other rows TM3 to TM5 follow; only the total changes.
// Path: per case (unhappy: boundary at the DC).
func TestTacticalMindResolveTableD(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name        string
		total, roll int
		newTotal    int
		succeeds    bool
	}{
		{"TM1", 12, 3, 15, true}, {"TM2", 12, 2, 14, false}, {"TM3", 14, 1, 15, true}, {"TM4", 5, 10, 15, true}, {"TM5", 5, 9, 14, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			check := fgtCheck(t, d20.AbilityCheck, modifier.Normal, []int{c.total}, 0, 15)
			res, err := character.TacticalMind{}.Resolve(fgtStats(t, 2), fgtState(0, 0, 0), check, c.roll)
			if err != nil {
				t.Fatal(err)
			}
			switch r := res.(type) {
			case character.TacticalMindResolutionSucceeded:
				if !c.succeeds || r.Check().Total() != c.newTotal || r.Rolled() != c.roll || r.State() != fgtState(1, 0, 0) || r.Check().Outcome() != d20.Success {
					t.Errorf("succeeded = %v/%d/%v", r.Check().Total(), r.Rolled(), r.State())
				}
			case character.TacticalMindResolutionStillFailed:
				if c.succeeds || r.Check().Total() != c.newTotal || r.Rolled() != c.roll || r.State() != fgtState(0, 0, 0) || r.Check().Outcome() != d20.Failure {
					t.Errorf("still failed = %v/%d/%v", r.Check().Total(), r.Rolled(), r.State())
				}
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-128 - Tactical Mind adds to the total and does not reroll or replace the d20: the Roll of the
// carried check is untouched, with Advantage and with Disadvantage alike.
// Path: unhappy (rule interaction).
func TestTacticalMindLeavesTheRollAlone(t *testing.T) {
	t.Parallel()
	for _, mode := range []modifier.RollMode{modifier.Normal, modifier.WithAdvantage, modifier.WithDisadvantage} {
		dice := []int{12}
		if mode != modifier.Normal {
			dice = []int{12, 12}
		}
		check := fgtCheck(t, d20.AbilityCheck, mode, dice, 0, 15)
		res, err := character.TacticalMind{}.Resolve(fgtStats(t, 2), fgtState(0, 0, 0), check, 3)
		got := res.(character.TacticalMindResolutionSucceeded)
		if err != nil || got.Check().Roll() != check.Roll() || got.Check().Total() != 15 {
			t.Errorf("mode %v: roll %v total %d (%v), want the original roll and 15", mode, got.Check().Roll(), got.Check().Total(), err)
		}
	}
}

// Requirement: doc/specs/class-fighter.md FR-064, FR-065, FR-066, TM6 - Tactical Mind is unavailable below Fighter level 2
// (featureNotGained), for a saving throw (notAnAbilityCheck), for a check that succeeded (checkNotFailed, TM6) and with no Second Wind
// use remaining (noUsesRemaining), checked in that order; Resolve refuses an unavailable use and a roll outside 1 to 10.
// Path: per case (unhappy: boundary, state error).
func TestTacticalMindAvailability(t *testing.T) {
	t.Parallel()
	failed := fgtCheck(t, d20.AbilityCheck, modifier.Normal, []int{5}, 0, 15)
	passed := fgtCheck(t, d20.AbilityCheck, modifier.Normal, []int{15}, 0, 15)
	save := fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{5}, 0, 15)
	for _, c := range []struct {
		name  string
		level int
		state character.FighterState
		check d20.CheckResult
		want  character.FighterReason
	}{
		{"level 1", 1, fgtState(0, 0, 0), failed, character.FighterFeatureNotGained},
		{"level 1 before the check kind", 1, fgtState(0, 0, 0), save, character.FighterFeatureNotGained},
		{"a saving throw", 2, fgtState(0, 0, 0), save, character.FighterNotAnAbilityCheck},
		{"a saving throw before the outcome", 2, fgtState(2, 0, 0), save, character.FighterNotAnAbilityCheck},
		{"TM6 a passed check", 2, fgtState(0, 0, 0), passed, character.FighterCheckNotFailed},
		{"a passed check before the uses", 2, fgtState(2, 0, 0), passed, character.FighterCheckNotFailed},
		{"no uses remaining", 2, fgtState(2, 0, 0), failed, character.FighterNoUsesRemaining},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := (character.TacticalMind{}).Availability(fgtStats(t, c.level), c.state, c.check).(character.TacticalMindUnavailable)
			if !ok || got.Reason != c.want {
				t.Errorf("availability = %v, want unavailable %v", got, c.want)
			}
			_, err := character.TacticalMind{}.Resolve(fgtStats(t, c.level), c.state, c.check, 5)
			wantInvalid(t, err, "unavailable use")
		})
	}
	if _, ok := (character.TacticalMind{}).Availability(fgtStats(t, 2), fgtState(1, 0, 0), failed).(character.TacticalMindAvailable); !ok {
		t.Error("level 2 with one use remaining on a failed check must be available")
	}
	if got, ok := (character.TacticalMind{}).Availability(fgtStats(t, 2), fgtState(0, 0, 0), d20.CheckResult{}).(character.TacticalMindUnavailable); !ok || got.Reason != character.FighterNotAnAbilityCheck {
		t.Errorf("a zero check result: availability = %v, want notAnAbilityCheck", got)
	}
	for _, roll := range []int{0, 11} {
		_, err := character.TacticalMind{}.Resolve(fgtStats(t, 2), fgtState(0, 0, 0), failed, roll)
		wantInvalid(t, err, "roll")
	}
	_, err := character.TacticalMind{}.Resolve(character.FighterLevelStats{}, fgtState(0, 0, 0), failed, 5)
	wantInvalid(t, err, "zero stats")
}
