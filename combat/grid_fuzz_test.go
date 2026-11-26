package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzGridConstructors feeds the grid constructors and rules host numbers (a column, a row, a footprint
// size, directions, a mode, a speed use and an extra cost). It asserts properties: nothing panics, a rejection
// matches srderr.ErrInvalidInput, an accepted square or footprint lies inside the grid range with sides of 1 to
// 1,000, ranges are never negative and agree when the arguments are swapped, and a planned path has one 5-foot
// segment per direction.
func FuzzGridConstructors(f *testing.F) {
	f.Add(0, 0, 1, 1, 3, 3, 1, 0, 0)
	f.Add(1_000_000, -1_000_000, 1000, 1000, 1, 1, 1, 1, 0)
	f.Add(1_000_001, 0, 1001, 0, 0, 9, 0, 9, -1)
	f.Add(math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MinInt)
	f.Fuzz(func(t *testing.T, column, row, width, height, d1, d2, mode, use, extra int) {
		bad := func(err error) bool { return err != nil && !errors.Is(err, srderr.ErrInvalidInput) }
		s, err := combat.NewGridSquare(column, row)
		if bad(err) || (err == nil && (s.Column() != column || s.Row() != row || column < -1_000_000 || column > 1_000_000 || row < -1_000_000 || row > 1_000_000)) {
			t.Fatalf("NewGridSquare(%d, %d) = %v, %v", column, row, s, err)
		}
		fpr, err := combat.NewGridFootprint(s, width, height)
		switch {
		case bad(err) || (err != nil && !fpr.IsZero()):
			t.Fatalf("NewGridFootprint = %v, %v", fpr, err)
		case err == nil && (fpr.Width() < 1 || fpr.Width() > 1000 || fpr.Height() < 1 || fpr.Height() > 1000 || !fpr.Contains(s)):
			t.Fatalf("NewGridFootprint accepted %v", fpr)
		case err == nil:
			a, errA := gridRules.RangeInSquares(squareGrid, fpr, fpr)
			if errA != nil || a != 0 || !fpr.Overlaps(fpr) {
				t.Fatalf("a footprint is not 0 away from itself: %d, %v", a, errA)
			}
			one, _ := combat.NewGridFootprint(s, 1, 1)
			if n, err := gridRules.RangeInSquares(squareGrid, fpr, one); err != nil || n != 0 {
				t.Fatalf("range to its origin = %d, %v", n, err)
			}
			shifted, err := fpr.Shifted(combat.GridDirection(d1))
			if bad(err) || (err == nil && shifted.Width() != fpr.Width()) {
				t.Fatalf("Shifted = %v, %v", shifted, err)
			}
		}
		for _, size := range []content.CreatureSize{content.CreatureSize(d1), content.SizeLarge} {
			fp2, err := combat.GridFootprintForCreature(size, s)
			if bad(err) {
				t.Fatalf("GridFootprintForCreature: %v", err)
			}
			if err != nil {
				continue
			}
			mover, err := combat.NewGridCreature(combat.MustCombatantID("m"), size, fp2, combat.ConditionActionAvailabilityAvailable)
			if err != nil {
				t.Fatalf("NewGridCreature for a footprint of its own size: %v", err)
			}
			ds := []combat.GridDirection{combat.GridDirection(d1), combat.GridDirection(d2)}
			plan, err := gridRules.PlanPath(squareGrid, combat.GridBoard{}, mover, ds, nil, combat.MovementMode(mode), combat.SpeedUse(use), extra)
			if bad(err) {
				t.Fatalf("PlanPath: %v", err)
			}
			if p, ok := plan.(combat.GridPathPlanned); ok && (len(p.Segments) != 2 || p.Segments[0].DistanceFeet() != 5) {
				t.Fatalf("PlanPath = %#v", p)
			}
		}
	})
}
