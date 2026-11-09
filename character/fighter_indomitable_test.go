package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-fighter.md Table F, FR-080, FR-081, FR-082, FR-126 - at Fighter level 9, save modifier +4 and DC 15 the
// new total is the used die of the recomputed Roll plus the modifiers plus the Fighter level; only the named die is replaced and the
// other keeps its value (rows IN1 to IN7: one d20, Advantage with the used and the other die, Disadvantage with the discarded and
// the used die); one use is expended whether the reroll succeeds or not.
// Path: per case (unhappy: boundary at the DC, rule interaction).
func TestIndomitableResolveTableF(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		mode     modifier.RollMode
		original []int
		replaced int
		rolled   int
		dice     []int
		total    int
		succeeds bool
	}{
		{"IN1", modifier.Normal, []int{8}, 0, 1, []int{1}, 14, false},
		{"IN2", modifier.Normal, []int{8}, 0, 2, []int{2}, 15, true},
		{"IN3", modifier.Normal, []int{8}, 0, 20, []int{20}, 33, true},
		{"IN4", modifier.WithAdvantage, []int{3, 8}, 1, 2, []int{3, 2}, 16, true},
		{"IN5", modifier.WithAdvantage, []int{3, 8}, 0, 20, []int{20, 8}, 33, true},
		{"IN6", modifier.WithDisadvantage, []int{1, 12}, 1, 15, []int{1, 15}, 14, false},
		{"IN7", modifier.WithDisadvantage, []int{1, 12}, 0, 10, []int{10, 12}, 23, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			save := fgtCheck(t, d20.SavingThrow, c.mode, c.original, 4, 15)
			res, err := character.Indomitable{}.Resolve(fgtStats(t, 9), fgtState(0, 1, 0), save, c.replaced, c.rolled)
			if err != nil {
				t.Fatal(err)
			}
			var check d20.CheckResult
			var state character.FighterState
			var rolled int
			switch r := res.(type) {
			case character.IndomitableResolutionSucceeded:
				check, state, rolled = r.Check(), r.State(), r.Rolled()
				if !c.succeeds {
					t.Error("resolved as succeeded, want still failed")
				}
			case character.IndomitableResolutionStillFailed:
				check, state, rolled = r.Check(), r.State(), r.Rolled()
				if c.succeeds {
					t.Error("resolved as still failed, want succeeded")
				}
			}
			if check.Total() != c.total || rolled != c.rolled || state != fgtState(0, 1, 1) || !equalInts(check.Roll().Dice(), c.dice) || check.Target() != save.Target() {
				t.Errorf("dice %v total %d rolled %d state %v; want %v %d %d one Indomitable use expended", check.Roll().Dice(), check.Total(), rolled, state, c.dice, c.total, c.rolled)
			}
			if !equalInts(save.Roll().Dice(), c.original) {
				t.Error("Resolve altered the original roll")
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Requirement: doc/specs/class-fighter.md FR-077, FR-078, FR-079, FR-127, IN8 - Indomitable is unavailable below Fighter level 9
// (featureNotGained), for an ability check (notASavingThrow), for a die the roll does not hold (dieNotRolled, IN8, also a negative
// index), for a save that succeeded (saveNotFailed) and with no use remaining (noUsesRemaining), checked in that order; Resolve refuses
// an unavailable use, zero stats and a roll outside 1 to 20.
// Path: per case (unhappy: boundary, invalid input, state error).
func TestIndomitableAvailability(t *testing.T) {
	t.Parallel()
	failed := fgtCheck(t, d20.SavingThrow, modifier.WithAdvantage, []int{3, 8}, 4, 15)
	passed := fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{18}, 4, 15)
	check := fgtCheck(t, d20.AbilityCheck, modifier.Normal, []int{3}, 4, 15)
	for _, c := range []struct {
		name     string
		level    int
		state    character.FighterState
		save     d20.CheckResult
		replaced int
		want     character.FighterReason
	}{
		{"level 8", 8, fgtState(0, 0, 0), failed, 0, character.FighterFeatureNotGained},
		{"an ability check", 9, fgtState(0, 0, 0), check, 0, character.FighterNotASavingThrow},
		{"IN8 a die the roll does not hold", 9, fgtState(0, 0, 0), failed, 2, character.FighterDieNotRolled},
		{"a die far beyond the roll", 9, fgtState(0, 0, 0), failed, 5, character.FighterDieNotRolled},
		{"a negative index", 9, fgtState(0, 0, 0), failed, -1, character.FighterDieNotRolled},
		{"the die before the outcome", 9, fgtState(0, 0, 1), passed, 1, character.FighterDieNotRolled},
		{"a passed save", 9, fgtState(0, 0, 0), passed, 0, character.FighterSaveNotFailed},
		{"a passed save before the uses", 9, fgtState(0, 0, 1), passed, 0, character.FighterSaveNotFailed},
		{"no uses remaining", 9, fgtState(0, 0, 1), failed, 1, character.FighterNoUsesRemaining},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := (character.Indomitable{}).Availability(fgtStats(t, c.level), c.state, c.save, c.replaced).(character.IndomitableUnavailable)
			if !ok || got.Reason != c.want {
				t.Errorf("availability = %v, want unavailable %v", got, c.want)
			}
			_, err := character.Indomitable{}.Resolve(fgtStats(t, c.level), c.state, c.save, c.replaced, 5)
			wantInvalid(t, err, "unavailable use")
		})
	}
	if _, ok := (character.Indomitable{}).Availability(fgtStats(t, 13), fgtState(0, 0, 1), failed, 1).(character.IndomitableAvailable); !ok {
		t.Error("level 13 with one of two uses expended must be available")
	}
	if _, ok := (character.Indomitable{}).Availability(fgtStats(t, 17), fgtState(0, 0, 0), failed, 0).(character.IndomitableAvailable); !ok {
		t.Error("level 17 with three uses remaining must be available")
	}
	if zero, ok := (character.Indomitable{}).Availability(fgtStats(t, 9), fgtState(0, 0, 0), d20.CheckResult{}, 0).(character.IndomitableUnavailable); !ok || zero.Reason != character.FighterNotASavingThrow {
		t.Errorf("a zero check result: availability = %v, want notASavingThrow", zero)
	}
	for _, roll := range []int{0, 21} {
		_, err := character.Indomitable{}.Resolve(fgtStats(t, 9), fgtState(0, 0, 0), failed, 0, roll)
		wantInvalid(t, err, "roll")
	}
	_, err := character.Indomitable{}.Resolve(character.FighterLevelStats{}, fgtState(0, 0, 0), failed, 0, 5)
	wantInvalid(t, err, "zero stats")
}
