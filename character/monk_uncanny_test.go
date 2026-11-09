package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

func wantReason(t *testing.T, got character.MonkAvailability, want character.MonkReason, what string) {
	t.Helper()
	if want == 0 {
		if _, ok := got.(character.MonkAvailable); !ok {
			t.Errorf("%s: availability = %#v, want available", what, got)
		}
		return
	}
	if un, ok := got.(character.MonkUnavailable); !ok || un.Reason != want {
		t.Errorf("%s: availability = %#v, want unavailable %v", what, got, want)
	}
}

// Requirement: doc/specs/class-monk.md Table 4.5a, FR-046, FR-047 - Resolve returns the Monk level plus the Martial Arts die roll, uncapped,
// regains every expended Focus Point and marks the feature used; rows: level 2 d6 roll 1 gives 3, roll 6 gives 8; level 5 d8 roll 8 gives 13;
// level 11 d10 roll 10 gives 21; level 20 d12 roll 12 gives 32.
// Path: per case (rows of Table 4.5a).
func TestUncannyMetabolismResolve(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name              string
		level, roll, want int
	}{{"lowest level, lowest roll", 2, 1, 3}, {"lowest level, highest roll", 2, 6, 8}, {"d8 tier", 5, 8, 13}, {"d10 tier", 11, 10, 21}, {"top level, highest roll", 20, 12, 32}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mnkState(2, false, 1)
			res, err := character.UncannyMetabolism{}.Resolve(mnkStats(t, c.level), in, true, c.roll)
			if err != nil || res.HitPointsToRegain() != c.want || res.Rolled() != c.roll || res.FocusPointsRegained() != 2 || res.State() != mnkState(0, true, 1) {
				t.Errorf("Resolve = %+v, %v; want %d Hit Points, 2 points regained, used", res, err, c.want)
			}
			if in != mnkState(2, false, 1) {
				t.Error("Resolve altered its input")
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-048, FR-050, FR-135, Table 4.11a - Uncanny Metabolism is levelTooLow below 2, notEligible when the trigger is not
// an Initiative roll (before alreadyUsed), alreadyUsed after a use; Resolve refuses a roll outside the die, zero stats and an unavailable use.
// Path: per case (unhappy: state error, boundary, invalid input).
func TestUncannyMetabolismAvailability(t *testing.T) {
	t.Parallel()
	u := character.UncannyMetabolism{}
	wantReason(t, u.Availability(mnkStats(t, 2), mnkState(0, false, 0), true), 0, "level 2")
	wantReason(t, u.Availability(mnkStats(t, 1), mnkState(0, false, 0), true), character.MonkLevelTooLow, "level 1")
	wantReason(t, u.Availability(mnkStats(t, 2), mnkState(0, false, 0), false), character.MonkNotEligible, "not initiative")
	wantReason(t, u.Availability(mnkStats(t, 2), mnkState(0, true, 0), true), character.MonkAlreadyUsed, "used")
	wantReason(t, u.Availability(mnkStats(t, 2), mnkState(0, true, 0), false), character.MonkNotEligible, "used, wrong trigger")
	wantReason(t, u.Availability(character.MonkLevelStats{}, mnkState(0, false, 0), true), character.MonkNotAMonk, "zero stats")
	for _, roll := range []int{0, 7, -1} {
		_, err := u.Resolve(mnkStats(t, 2), mnkState(0, false, 0), true, roll)
		wantInvalid(t, err, "roll")
	}
	_, err := u.Resolve(mnkStats(t, 2), mnkState(0, true, 0), true, 3)
	wantInvalid(t, err, "used")
	_, err = u.Resolve(character.MonkLevelStats{}, mnkState(0, false, 0), true, 3)
	wantInvalid(t, err, "zero stats")
}

// Requirement: doc/specs/class-monk.md Table 4.9a, FR-097, FR-098, FR-099 - Perfect Focus regains Focus Points until 4 are available when 3 or fewer are,
// changes nothing with 4 or more, does not apply when Uncanny Metabolism was used for that Initiative roll or the trigger is not Initiative, and is levelTooLow below 15.
// Path: per case (rows of Table 4.9a).
func TestPerfectFocus(t *testing.T) {
	t.Parallel()
	stats := mnkStats(t, 15) // 15 Focus Points
	for _, c := range []struct {
		name                 string
		available, wantAvail int
		regained             int
	}{{"none left", 0, 4, 4}, {"three left", 3, 4, 1}, {"exactly four", 4, 4, 0}, {"five left", 5, 5, 0}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mnkState(15-c.available, false, 0)
			res, err := character.PerfectFocus{}.Resolve(stats, in, true, false)
			if err != nil || res.State().FocusPoints(stats).Available() != c.wantAvail || res.FocusPointsRegained() != c.regained {
				t.Errorf("Resolve = %+v, %v; want %d available, %d regained", res, err, c.wantAvail, c.regained)
			}
			if c.regained == 0 && res.State() != in {
				t.Error("a change with 4 or more available")
			}
		})
	}
	p := character.PerfectFocus{}
	wantReason(t, p.Availability(stats, true, true), character.MonkNotEligible, "Uncanny Metabolism used")
	wantReason(t, p.Availability(stats, false, false), character.MonkNotEligible, "not Initiative")
	wantReason(t, p.Availability(mnkStats(t, 14), true, false), character.MonkLevelTooLow, "level 14")
	wantReason(t, p.Availability(stats, true, false), 0, "level 15")
	wantReason(t, p.Availability(character.MonkLevelStats{}, true, false), character.MonkNotAMonk, "zero stats")
	_, err := p.Resolve(stats, mnkState(15, false, 0), true, true)
	wantInvalid(t, err, "Uncanny Metabolism used")
	res, err := p.Resolve(stats, mnkState(15, false, 0), true, false)
	if err != nil || res.State().FocusExpended() != 11 {
		t.Errorf("0 left: %+v, %v", res, err)
	}
}
