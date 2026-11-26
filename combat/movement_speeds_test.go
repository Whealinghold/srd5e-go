package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func fly(feet int) combat.MovementSpeedEntry {
	return combat.MovementSpeedEntry{Name: combat.SpeedNameFly, Feet: feet}
}
func climb(feet int) combat.MovementSpeedEntry {
	return combat.MovementSpeedEntry{Name: combat.SpeedNameClimb, Feet: feet}
}

// Requirement: doc/specs/initiative-and-actions.md FR-075 - walk and each special speed must be 0 or more; a negative, repeated or undefined speed is an invalid-input error.
// Path: unhappy (invalid input).
func TestNewMovementSpeedsRejectsInvalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		walk  int
		extra []combat.MovementSpeedEntry
	}{
		{"negative walk", -1, nil},
		{"negative fly", 30, []combat.MovementSpeedEntry{fly(-1)}},
		{"walk named as special", 30, []combat.MovementSpeedEntry{{Name: combat.SpeedNameWalk, Feet: 10}}},
		{"unknown name", 30, []combat.MovementSpeedEntry{{Name: 0, Feet: 10}}},
		{"name past the last", 30, []combat.MovementSpeedEntry{{Name: combat.SpeedNameSwim + 1, Feet: 10}}},
		{"fly twice", 30, []combat.MovementSpeedEntry{fly(10), fly(20)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, err := combat.NewMovementSpeeds(c.walk, c.extra...); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("NewMovementSpeeds = %v, want invalid input [FR-075]", err)
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-075 - Of reports each speed and whether the creature has it; the walking Speed is always had and a special speed of 0 is still had.
// Path: happy.
func TestMovementSpeedsOf(t *testing.T) {
	t.Parallel()
	s := trkSpeeds(t, 30, climb(0), fly(40))
	for _, c := range []struct {
		name combat.SpeedName
		feet int
		has  bool
	}{{combat.SpeedNameWalk, 30, true}, {combat.SpeedNameClimb, 0, true}, {combat.SpeedNameFly, 40, true},
		{combat.SpeedNameBurrow, 0, false}, {combat.SpeedNameSwim, 0, false}, {0, 0, false}, {99, 0, false}} {
		if feet, has := s.Of(c.name); feet != c.feet || has != c.has {
			t.Errorf("Of(%v) = %d, %v; want %d, %v", c.name, feet, has, c.feet, c.has)
		}
	}
	if s.Walk() != 30 {
		t.Errorf("Walk = %d, want 30", s.Walk())
	}
	var zero combat.MovementSpeeds
	if zero.Walk() != 0 || zero != trkSpeeds(t, 0) {
		t.Errorf("zero MovementSpeeds = %+v, want walking Speed 0 and no special speed [FR-075]", zero)
	}
}

// Requirement: per case
// Path: per case.
func TestMovementSpeedsChanges(t *testing.T) {
	t.Parallel()
	sp := func(walk int, e ...combat.MovementSpeedEntry) combat.MovementSpeeds { return trkSpeeds(t, walk, e...) }
	cases := []struct {
		name, requirement, path string
		change                  func(combat.MovementSpeeds) combat.MovementSpeeds
		in, want                combat.MovementSpeeds
	}{
		{"reduce both", "FR-076", "happy", func(s combat.MovementSpeeds) combat.MovementSpeeds { return s.AdjustedBy(-15) }, sp(30, fly(40)), sp(15, fly(25))},
		{"increase both", "FR-076", "happy", func(s combat.MovementSpeeds) combat.MovementSpeeds { return s.AdjustedBy(10) }, sp(30, fly(40)), sp(40, fly(50))},
		{"climb follows to 0", "FR-076", "unhappy (boundary)", func(s combat.MovementSpeeds) combat.MovementSpeeds { return s.AdjustedBy(-30) }, sp(30, climb(20)), sp(0, climb(0))},
		{"floor at 0", "FR-076", "unhappy (boundary)", func(s combat.MovementSpeeds) combat.MovementSpeeds { return s.AdjustedBy(-40) }, sp(30, fly(40)), sp(0, fly(0))},
		{"no special speed", "FR-076", "unhappy (empty case)", func(s combat.MovementSpeeds) combat.MovementSpeeds { return s.AdjustedBy(-10) }, sp(30), sp(20)},
		{"zero adjustment", "FR-076", "unhappy (empty case)", func(s combat.MovementSpeeds) combat.MovementSpeeds { return s.AdjustedBy(0) }, sp(30, fly(5)), sp(30, fly(5))},
		{"adjust saturates", "FR-076", "unhappy (boundary)", func(s combat.MovementSpeeds) combat.MovementSpeeds { return s.AdjustedBy(math.MaxInt) }, sp(30), sp(math.MaxInt)},
		{"halved, the SRD's example", "FR-109", "happy", combat.MovementSpeeds.Halved, sp(30, fly(40)), sp(15, fly(20))},
		{"halved, odd walk rounds down", "FR-109", "unhappy (boundary)", combat.MovementSpeeds.Halved, sp(25), sp(12)},
		{"halved, each speed rounds on its own", "FR-109", "unhappy (boundary)", combat.MovementSpeeds.Halved, sp(25, fly(41)), sp(12, fly(20))},
		{"halved, every special speed", "FR-109", "unhappy (rule interaction)", combat.MovementSpeeds.Halved,
			sp(30, combat.MovementSpeedEntry{Name: combat.SpeedNameBurrow, Feet: 15}, climb(21), combat.MovementSpeedEntry{Name: combat.SpeedNameSwim, Feet: 31}),
			sp(15, combat.MovementSpeedEntry{Name: combat.SpeedNameBurrow, Feet: 7}, climb(10), combat.MovementSpeedEntry{Name: combat.SpeedNameSwim, Feet: 15})},
		{"halved, 1 becomes 0", "FR-109", "unhappy (boundary)", combat.MovementSpeeds.Halved, sp(1, fly(1)), sp(0, fly(0))},
		{"halved, 0 stays 0", "FR-109", "unhappy (boundary)", combat.MovementSpeeds.Halved, sp(0, fly(40)), sp(0, fly(20))},
		{"doubled", "FR-160", "unhappy (boundary)", combat.MovementSpeeds.Doubled, sp(30, fly(40)), sp(60, fly(80))},
		{"doubled, 0 stays 0", "FR-160", "unhappy (boundary)", combat.MovementSpeeds.Doubled, sp(0, climb(15)), sp(0, climb(30))},
		{"doubled, odd", "FR-160", "unhappy (boundary)", combat.MovementSpeeds.Doubled, sp(25), sp(50)},
		{"doubled saturates", "FR-160", "unhappy (boundary)", combat.MovementSpeeds.Doubled, sp(math.MaxInt), sp(math.MaxInt)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.change(c.in); got != c.want {
				t.Errorf("changed %+v = %+v, want %+v [%s %s]", c.in, got, c.want, "doc/specs/initiative-and-actions.md", c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-075 - NewMovementSpeeds never panics for any numbers.
// Path: unhappy (invalid input).
func FuzzNewMovementSpeeds(f *testing.F) {
	f.Add(30, 40, 3, -1)
	f.Add(math.MinInt, math.MaxInt, 0, 99)
	f.Fuzz(func(t *testing.T, walk, feet, name, other int) {
		s, err := combat.NewMovementSpeeds(walk, combat.MovementSpeedEntry{Name: combat.SpeedName(name), Feet: feet}, combat.MovementSpeedEntry{Name: combat.SpeedName(other), Feet: feet})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if s.Walk() < 0 || s.Halved().Walk() < 0 || s.Doubled().Walk() < 0 || s.AdjustedBy(-5).Walk() < 0 {
			t.Fatalf("speeds %+v went negative", s)
		}
	})
}
