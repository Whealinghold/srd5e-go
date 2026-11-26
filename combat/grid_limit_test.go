package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// gridCreatures builds n distinct Medium creatures on a row.
func gridCreatures(tb testing.TB, n int) []combat.GridCreature {
	tb.Helper()
	out := make([]combat.GridCreature, n)
	for i := range out {
		out[i] = gridCreature(tb, fmt.Sprintf("c%d", i), content.SizeMedium, i, 0)
	}
	return out
}

// Requirement: doc/specs/grid-positions-and-movement.md Limits (plan P10) - a terrain holds at most 65536 squares, difficult and filled together.
// Path: unhappy (boundary).
func TestNewGridTerrainCap(t *testing.T) {
	t.Parallel()
	runCap(t, "terrain squares", 65536, func(n int) error {
		difficult := make([]combat.GridSquare, n)
		for i := range difficult {
			difficult[i] = combat.MustGridSquare(i%256, i/256)
		}
		_, err := combat.NewGridTerrain(difficult[:n/2], difficult[n/2:])
		return err
	})
}

// Requirement: doc/specs/grid-positions-and-movement.md Limits (plan P10) - a board holds at most 256 creatures: NewGridBoard refuses 257, Place refuses a 257th creature, and Place replacing a creature by id at the cap is accepted.
// Path: unhappy (boundary).
func TestGridBoardCreatureCap(t *testing.T) {
	t.Parallel()
	t.Run("NewGridBoard", func(t *testing.T) {
		t.Parallel()
		runCap(t, "grid creatures", 256, func(n int) error {
			_, err := combat.NewGridBoard(combat.GridTerrain{}, gridCreatures(t, n))
			return err
		})
	})
	t.Run("Place", func(t *testing.T) {
		t.Parallel()
		full := gridBoard(t, combat.GridTerrain{}, gridCreatures(t, 256)...)
		moved := gridCreature(t, "c7", content.SizeMedium, 9, 9)
		if b, err := full.Place(moved); err != nil || len(b.Creatures()) != 256 {
			t.Errorf("Place(replacing c7) = %d creatures, %v, want 256 and no error", len(b.Creatures()), err)
		}
		_, err := full.Place(gridCreature(t, "extra", content.SizeMedium, 0, 5))
		wantLimit(t, err, "grid creatures", 256)
		if got := len(full.Creatures()); got != 256 {
			t.Errorf("the board changed: %d creatures", got)
		}
		if b, err := gridBoard(t, combat.GridTerrain{}, gridCreatures(t, 255)...).Place(gridCreature(t, "last", content.SizeMedium, 0, 5)); err != nil || len(b.Creatures()) != 256 {
			t.Errorf("Place(the 256th) = %d creatures, %v", len(b.Creatures()), err)
		}
	})
}

// Requirement: doc/specs/grid-positions-and-movement.md Limits (plan P10) - PlanPath reads at most 4096 directions; 4097 is a LimitExceededError before any step is assessed.
// Path: unhappy (boundary).
func TestPlanPathDirectionCap(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "mover", content.SizeMedium, 0, 0)
	board := gridBoard(t, combat.GridTerrain{})
	runCap(t, "path directions", 4096, func(n int) error {
		dirs := make([]combat.GridDirection, n)
		for i := range dirs {
			dirs[i] = combat.GridDirectionEast
		}
		_, err := gridRules.PlanPath(squareGrid, board, mover, dirs, nil, combat.MovementModeWalk, combat.SpeedUseSpecial, 0)
		return err
	})
}

// Requirement: doc/specs/grid-positions-and-movement.md Limits (plan P10), doc/rules/testing.md section 4 - MustGridTerrain given 65537 squares panics with the LimitExceededError NewGridTerrain returns, an error matching srderr.ErrInvalidInput.
// Path: unhappy (boundary).
func TestMustGridTerrainPanicsPastTheCap(t *testing.T) {
	t.Parallel()
	squares := make([]combat.GridSquare, 65537)
	for i := range squares {
		squares[i] = combat.MustGridSquare(i%256, i/256)
	}
	defer func() {
		err, ok := recover().(error)
		if !ok {
			t.Fatalf("recovered a non-error value or nothing")
		}
		wantLimit(t, err, "terrain squares", 65536)
	}()
	combat.MustGridTerrain(squares, nil)
	t.Error("MustGridTerrain did not panic")
}
