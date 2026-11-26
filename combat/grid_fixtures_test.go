package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// sq is the square at column and row; it fails the test outside the grid range.
func sq(tb testing.TB, column, row int) combat.GridSquare {
	tb.Helper()
	s, err := combat.NewGridSquare(column, row)
	if err != nil {
		tb.Fatalf("NewGridSquare(%d, %d): %v", column, row, err)
	}
	return s
}

// fp is the footprint with its origin at (column, row), width by height squares.
func fp(tb testing.TB, column, row, width, height int) combat.GridFootprint {
	tb.Helper()
	f, err := combat.NewGridFootprint(sq(tb, column, row), width, height)
	if err != nil {
		tb.Fatalf("NewGridFootprint((%d, %d), %d, %d): %v", column, row, width, height, err)
	}
	return f
}

// gridCreature is a creature of size with its origin at (column, row) that can act.
func gridCreature(tb testing.TB, id string, size content.CreatureSize, column, row int) combat.GridCreature {
	tb.Helper()
	return gridCreatureAs(tb, id, size, column, row, combat.ConditionActionAvailabilityAvailable)
}

func gridCreatureAs(tb testing.TB, id string, size content.CreatureSize, column, row int, a combat.ConditionActionAvailability) combat.GridCreature {
	tb.Helper()
	f, err := combat.GridFootprintForCreature(size, sq(tb, column, row))
	if err != nil {
		tb.Fatalf("GridFootprintForCreature: %v", err)
	}
	c, err := combat.NewGridCreature(combat.MustCombatantID(id), size, f, a)
	if err != nil {
		tb.Fatalf("NewGridCreature: %v", err)
	}
	return c
}

// squares builds the squares from column, row pairs.
func squares(tb testing.TB, pairs ...int) []combat.GridSquare {
	tb.Helper()
	var out []combat.GridSquare
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, sq(tb, pairs[i], pairs[i+1]))
	}
	return out
}

// gridBoard is the board of the terrain and the creatures.
func gridBoard(tb testing.TB, terrain combat.GridTerrain, creatures ...combat.GridCreature) combat.GridBoard {
	tb.Helper()
	b, err := combat.NewGridBoard(terrain, creatures)
	if err != nil {
		tb.Fatalf("NewGridBoard: %v", err)
	}
	return b
}

var squareGrid = combat.SquareGrid{}
var gridRules = combat.GridRules{}
