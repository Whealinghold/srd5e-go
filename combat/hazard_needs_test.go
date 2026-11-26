package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/environment-hazards-and-objects.md FR-027, FR-028 (table 4.4a) - the daily water and food need of each of the six sizes in eighths; an invalid size is an ErrInvalidInput error.
// Path: per case.
func TestDailyNeedsBySize(t *testing.T) {
	t.Parallel()
	want := map[content.CreatureSize]int{content.SizeTiny: 2, content.SizeSmall: 8, content.SizeMedium: 8, content.SizeLarge: 32, content.SizeHuge: 128, content.SizeGargantuan: 512}
	for size, eighths := range want {
		if w, err := (combat.WaterNeedsPerDay{}).ForSize(size); err != nil || w.Eighths() != eighths {
			t.Errorf("water for %v = %v, %v, want %d eighths (FR-027)", size, w, err, eighths)
		}
		if f, err := (combat.FoodNeedsPerDay{}).ForSize(size); err != nil || f.Eighths() != eighths {
			t.Errorf("food for %v = %v, %v, want %d eighths (FR-028)", size, f, err, eighths)
		}
	}
	for _, size := range []content.CreatureSize{0, 7, -1} {
		if _, err := (combat.WaterNeedsPerDay{}).ForSize(size); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("water for size %d = %v, want ErrInvalidInput", int(size), err)
		}
		if _, err := (combat.FoodNeedsPerDay{}).ForSize(size); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("food for size %d = %v, want ErrInvalidInput", int(size), err)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-029, FR-121 - amounts of water and food are 0 or more, comparable, and distinct types.
// Path: unhappy (invalid input).
func TestWaterAndFoodAmounts(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 1, 8, math.MaxInt} {
		w, werr := combat.NewWaterAmount(n)
		f, ferr := combat.NewFoodAmount(n)
		if werr != nil || ferr != nil || w.Eighths() != n || f.Eighths() != n {
			t.Errorf("amount %d = %v/%v, %v/%v", n, w, f, werr, ferr)
		}
	}
	for _, n := range []int{-1, math.MinInt} {
		if _, err := combat.NewWaterAmount(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewWaterAmount(%d) = %v", n, err)
		}
		if _, err := combat.NewFoodAmount(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewFoodAmount(%d) = %v", n, err)
		}
	}
	a, _ := combat.NewWaterAmount(8)
	b, _ := combat.NewWaterAmount(8)
	c, _ := combat.NewWaterAmount(7)
	if a != b || a == c {
		t.Error("water amounts do not compare by value")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-029 - negative or oversized amounts never panic.
// Path: unhappy (invalid input).
func FuzzAmounts(f *testing.F) {
	for _, n := range []int{0, 1, -1, math.MaxInt, math.MinInt} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		w, werr := combat.NewWaterAmount(n)
		fd, ferr := combat.NewFoodAmount(n)
		fall, fallErr := combat.NewFallDistance(n)
		th, thErr := combat.NewDamageThreshold(n)
		if (werr == nil) != (n >= 0) || (ferr == nil) != (n >= 0) || (fallErr == nil) != (n >= 0) || (thErr == nil) != (n >= 1) {
			t.Fatalf("n = %d: errors %v %v %v %v", n, werr, ferr, fallErr, thErr)
		}
		if werr == nil && (w.Eighths() != n || fd.Eighths() != n || fall.Feet() != n || fall.Dice() > 20) {
			t.Fatalf("n = %d: values %v %v %v", n, w, fd, fall)
		}
		if thErr == nil && th.Amount() != n {
			t.Fatalf("threshold %v for %d", th, n)
		}
	})
}
