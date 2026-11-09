package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/exploration.md FR-001 to FR-005, FR-047 - the Travel Pace
// table figures are exactly the printed ones, per pace and unit, and are not derived
// from one another (SRD 5.2.1, "Travel Pace", p. 12).
// Path: per case.
func TestTravelPaceRow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path                  string
		pace                        character.TravelPace
		feetPerMinute, perHour, day int
	}{
		{"Fast", "happy", character.PaceFast, 400, 4, 30},
		{"Normal", "happy", character.PaceNormal, 300, 3, 24},
		{"Slow", "happy", character.PaceSlow, 200, 2, 18},
		{"invalid pace has no row", "unhappy (invalid input)", character.TravelPace(0), 0, 0, 0},
		{"pace above Fast has no row", "unhappy (boundary)", character.PaceFast + 1, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := c.pace.Row()
			if r.FeetPerMinute() != c.feetPerMinute || r.MilesPerHour() != c.perHour || r.MilesPerDay() != c.day {
				t.Errorf("Row(%v) = %d/%d/%d, want %d/%d/%d", c.pace, r.FeetPerMinute(), r.MilesPerHour(), r.MilesPerDay(), c.feetPerMinute, c.perHour, c.day)
			}
		})
	}
	// FR-047: the printed figures are not multiples of one another.
	fast, slow := character.PaceFast.Row(), character.PaceSlow.Row()
	if fast.MilesPerDay() == fast.MilesPerHour()*8 || slow.MilesPerDay() == slow.MilesPerHour()*8 || fast.FeetPerMinute()*60 == fast.MilesPerHour()*5280 {
		t.Error("FR-047: the pace figures must not be derived from one another")
	}
}

// Requirement: doc/specs/exploration.md FR-001, FR-006 - three paces ordered slowest
// first, and every figure positive and strictly falling from Fast to Slow (SRD 5.2.1,
// "Travel Pace", p. 12).
// Path: unhappy (boundary).
func TestTravelPaceOrderAndTable(t *testing.T) {
	t.Parallel()
	paces := character.TravelPaces()
	if !slices.Equal(paces, []character.TravelPace{character.PaceSlow, character.PaceNormal, character.PaceFast}) {
		t.Fatalf("TravelPaces() = %v", paces)
	}
	paces[0] = character.PaceFast // a copy: later calls are unaffected
	if character.TravelPaces()[0] != character.PaceSlow {
		t.Error("TravelPaces must return a fresh slice")
	}
	for i, p := range character.TravelPaces() {
		r := p.Row()
		if r.FeetPerMinute() <= 0 || r.MilesPerHour() <= 0 || r.MilesPerDay() <= 0 || !p.Valid() {
			t.Errorf("%v: a figure is not positive", p)
		}
		if i > 0 {
			q := character.TravelPaces()[i-1].Row()
			if r.FeetPerMinute() <= q.FeetPerMinute() || r.MilesPerHour() <= q.MilesPerHour() || r.MilesPerDay() <= q.MilesPerDay() {
				t.Errorf("%v is not faster than %v in every unit", p, character.TravelPaces()[i-1])
			}
		}
	}
	if character.TravelPace(0).Valid() || (character.PaceFast + 1).Valid() {
		t.Error("an undefined pace must be invalid")
	}
}

// Requirement: doc/specs/exploration.md FR-046 - RaisedOneStep moves one step and
// stops at Fast (SRD 5.2.1, "Good Roads", p. 192).
// Path: per case.
func TestTravelPaceRaisedOneStep(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		in, want   character.TravelPace
	}{
		{"slow gives normal", "happy", character.PaceSlow, character.PaceNormal},
		{"normal gives fast", "happy", character.PaceNormal, character.PaceFast},
		{"fast stays fast", "unhappy (boundary)", character.PaceFast, character.PaceFast},
		{"invalid stays invalid", "unhappy (invalid input)", character.TravelPace(0), character.TravelPace(0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.in.RaisedOneStep(); got != c.want {
				t.Errorf("%v.RaisedOneStep() = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// Requirement: none - pins the String values of TravelPace.
// Path: happy.
func TestTravelPaceString(t *testing.T) {
	t.Parallel()
	if got := character.PaceSlow.String() + character.PaceNormal.String() + character.PaceFast.String() + character.TravelPace(0).String(); got != "slownormalfastunknown" {
		t.Errorf("strings = %q", got)
	}
}
