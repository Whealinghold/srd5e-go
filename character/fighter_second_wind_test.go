package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-fighter.md Table C, FR-053, FR-054, FR-055, FR-058, FR-162 - SecondWind.Resolve returns the d10 roll plus
// the Fighter level of the stats, uncapped, and expends exactly one Second Wind use, leaving Action Surge and Indomitable alone (rows
// SW1 to SW6; SW6 is a Fighter 3 with Wizard 2, whose stats are those of Fighter level 3).
// Path: per case.
func TestSecondWindResolveTableC(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name        string
		level, roll int
		hitPoints   int
	}{
		{"SW1", 1, 1, 2}, {"SW2", 1, 10, 11}, {"SW3", 5, 6, 11}, {"SW4", 20, 1, 21}, {"SW5", 20, 10, 30}, {"SW6", 3, 4, 7},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := fgtState(0, 1, 1)
			res, err := character.SecondWind{}.Resolve(fgtStats(t, c.level), in, c.roll)
			if err != nil || res.HitPointsToRegain() != c.hitPoints || res.Rolled() != c.roll || res.State() != fgtState(1, 1, 1) {
				t.Errorf("Resolve = %v, %v; want %d Hit Points and one use expended", res, err, c.hitPoints)
			}
			if in != fgtState(0, 1, 1) {
				t.Error("Resolve altered its input state")
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-056, FR-162 - SecondWind.Availability is unavailable with no use remaining or with zero
// stats (reason noUsesRemaining), and Resolve refuses a roll of 0 or 11, zero stats and an unavailable use with an error matching
// srderr.ErrInvalidInput.
// Path: per case (unhappy: boundary, invalid input, state error).
func TestSecondWindAvailabilityAndInvalidInput(t *testing.T) {
	t.Parallel()
	l1 := fgtStats(t, 1)
	if _, ok := (character.SecondWind{}).Availability(l1, fgtState(1, 0, 0)).(character.SecondWindAvailable); !ok {
		t.Error("one use remaining must be available")
	}
	for _, c := range []struct {
		name  string
		stats character.FighterLevelStats
		state character.FighterState
	}{{"all uses expended", l1, fgtState(2, 0, 0)}, {"more expended than the maximum", l1, fgtState(7, 0, 0)}, {"zero stats", character.FighterLevelStats{}, fgtState(0, 0, 0)}} {
		got, ok := (character.SecondWind{}).Availability(c.stats, c.state).(character.SecondWindUnavailable)
		if !ok || got.Reason != character.FighterNoUsesRemaining {
			t.Errorf("%s: availability = %v, want unavailable with noUsesRemaining", c.name, got)
		}
		_, err := character.SecondWind{}.Resolve(c.stats, c.state, 5)
		wantInvalid(t, err, c.name)
	}
	for _, roll := range []int{0, 11, -3} {
		_, err := character.SecondWind{}.Resolve(l1, fgtState(0, 0, 0), roll)
		wantInvalid(t, err, "roll")
	}
}

// Requirement: doc/specs/class-fighter.md FR-059, FR-060 - FighterEffects.TacticalShift offers the move from Fighter level 5 and not at
// level 4, and the offer carries the fraction one half: HalfOf rounds down (30 gives 15, 25 gives 12, 0 gives 0) and refuses a negative
// Speed.
// Path: per case (unhappy: boundary).
func TestTacticalShiftOfferAndHalf(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 20; n++ {
		_, offered := character.FighterEffects{}.TacticalShift(character.MustFighterLevel(n)).(character.TacticalShiftOffered)
		if offered != (n >= 5) {
			t.Errorf("level %d: offered = %v, want %v", n, offered, n >= 5)
		}
	}
	if _, none := (character.FighterEffects{}).TacticalShift(character.MustFighterLevel(4)).(character.NoTacticalShift); !none {
		t.Error("level 4 must offer NoTacticalShift")
	}
	for speed, want := range map[int]int{30: 15, 25: 12, 0: 0, 1: 0, 31: 15} {
		if got, err := (character.TacticalShiftOffered{}).HalfOf(speed); err != nil || got != want {
			t.Errorf("HalfOf(%d) = %d, %v; want %d", speed, got, err, want)
		}
	}
	_, err := character.TacticalShiftOffered{}.HalfOf(-5)
	wantInvalid(t, err, "negative Speed")
}
