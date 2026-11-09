package character_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/exploration.md FR-016, cases 4.3a - NewVehicleSpeed accepts
// 1 to 2^52 half-miles per hour and reads back exactly (SRD 5.2.1, "Airborne and
// Waterborne Vehicles", p. 101).
// Path: per case.
func TestNewVehicleSpeed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		half       int64
		bad        bool
		mph        float64
	}{
		{"Rowboat", "happy", 3, false, 1.5},
		{"Keelboat", "happy", 2, false, 1},
		{"Airship", "happy", 16, false, 8},
		{"one half-mile", "unhappy (boundary)", 1, false, 0.5},
		{"upper bound", "unhappy (boundary)", 1 << 52, false, 1 << 51},
		{"one over the upper bound", "unhappy (boundary)", 1<<52 + 1, true, 0},
		{"zero", "unhappy (invalid input)", 0, true, 0},
		{"negative", "unhappy (invalid input)", -1, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := character.NewVehicleSpeed(c.half)
			if c.bad {
				if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ExplorationError{Input: character.ExplorationVehicleSpeed}) || !s.IsZero() {
					t.Errorf("NewVehicleSpeed(%d) = %v, %v", c.half, s, err)
				}
				if recovered(t, func() { character.MustVehicleSpeed(c.half) }) == nil {
					t.Error("MustVehicleSpeed must panic")
				}
				return
			}
			if err != nil || s.MilesPerHour() != c.mph || s.HalfMilesPerHour() != c.half || s.IsZero() || character.MustVehicleSpeed(c.half) != s {
				t.Errorf("NewVehicleSpeed(%d) = %v mph, %v", c.half, s.MilesPerHour(), err)
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-016, cases 4.3a - VehicleSpeedFromMilesPerHour
// accepts only a finite positive multiple of 0.5 at most 2^51.
// Path: per case.
func TestVehicleSpeedFromMilesPerHour(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		mph        float64
		want       int64 // 0 means an error
	}{
		{"1.5", "happy", 1.5, 3},
		{"0.5", "unhappy (boundary)", 0.5, 1},
		{"upper bound", "unhappy (boundary)", 1 << 51, 1 << 52},
		{"half a mile over", "unhappy (boundary)", 1<<51 + 0.5, 0},
		{"huge", "unhappy (invalid input)", 1e300, 0},
		{"zero", "unhappy (invalid input)", 0, 0},
		{"negative", "unhappy (invalid input)", -1, 0},
		{"NaN", "unhappy (invalid input)", math.NaN(), 0},
		{"+Inf", "unhappy (invalid input)", math.Inf(1), 0},
		{"-Inf", "unhappy (invalid input)", math.Inf(-1), 0},
		{"0.3", "unhappy (invalid input)", 0.3, 0},
		{"0.25", "unhappy (invalid input)", 0.25, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := character.VehicleSpeedFromMilesPerHour(c.mph)
			if c.want == 0 {
				if !errors.Is(err, srderr.ErrInvalidInput) || !s.IsZero() {
					t.Errorf("From(%v) = %v, %v", c.mph, s, err)
				}
				return
			}
			if err != nil || s != character.MustVehicleSpeed(c.want) {
				t.Errorf("From(%v) = %v, %v, want %d half-miles", c.mph, s, err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-016 and doc/rules/testing.md section 10 -
// neither constructor panics, and an accepted value converts back to itself.
// Path: unhappy (invalid input).
func FuzzVehicleSpeed(f *testing.F) {
	for _, h := range []int64{0, 1, 3, -1, 1 << 52, 1<<52 + 1, math.MaxInt64, math.MinInt64} {
		f.Add(h, float64(h)/2)
	}
	f.Add(int64(0), math.NaN())
	f.Fuzz(func(t *testing.T, half int64, mph float64) {
		if s, err := character.NewVehicleSpeed(half); err == nil && s.HalfMilesPerHour() != half {
			t.Fatalf("NewVehicleSpeed(%d) read back %d", half, s.HalfMilesPerHour())
		}
		if s, err := character.VehicleSpeedFromMilesPerHour(mph); err == nil {
			if s.MilesPerHour() != mph || s.HalfMilesPerHour() < 1 {
				t.Fatalf("From(%v) = %v", mph, s.MilesPerHour())
			}
		}
	})
}
