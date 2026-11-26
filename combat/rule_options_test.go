package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-foundations.md FR-042, FR-043 - the standard
// options use NoGrid and equal the zero value; WithGrid changes only the grid
// and leaves the receiver unchanged; NewRuleOptions agrees; a nil grid is NoGrid.
// Path: happy / unhappy (rule interaction).
func TestRuleOptions(t *testing.T) {
	t.Parallel()
	std := combat.StandardRuleOptions()
	if std != (combat.RuleOptions{}) || std.Grid() != (combat.NoGrid{}) {
		t.Errorf("standard options = %v, want the zero value with NoGrid", std)
	}
	sq := std.WithGrid(combat.SquareGrid{})
	if sq.Grid() != (combat.SquareGrid{}) || std.Grid() != (combat.NoGrid{}) {
		t.Errorf("WithGrid(SquareGrid) = %v, receiver %v: want a changed copy and an unchanged receiver", sq.Grid(), std.Grid())
	}
	if sq != combat.NewRuleOptions(combat.SquareGrid{}) || sq == std {
		t.Error("NewRuleOptions(SquareGrid) must equal WithGrid(SquareGrid) and differ from standard")
	}
	if back := sq.WithGrid(combat.NoGrid{}); back != std {
		t.Errorf("WithGrid(NoGrid) = %v, want the standard options", back)
	}
	if nilGrid := sq.WithGrid(nil); nilGrid != std {
		t.Errorf("WithGrid(nil) = %v, want the standard options", nilGrid)
	}
	switch combat.NewRuleOptions(combat.SquareGrid{}).Grid().(type) {
	case combat.NoGrid:
		t.Error("exhaustive switch reached NoGrid for a square grid")
	case combat.SquareGrid:
	}
}

// Requirement: doc/specs/combat-foundations.md FR-044, FR-045 - SquareGrid is 5
// feet a square and Speed divides by 5, rounding down; negative feet are
// rejected (SRD 5.2.1, "Playing on a Grid", p. 13; "Round Down", p. 5).
// Path: per case.
func TestSquareGridSpeedInSquares(t *testing.T) {
	t.Parallel()
	g := combat.SquareGrid{}
	if g.FeetPerSquare() != 5 {
		t.Errorf("FeetPerSquare() = %d, want 5", g.FeetPerSquare())
	}
	cases := []struct {
		name, path string
		feet, want int
	}{
		{"the SRD's own example", "happy", 30, 6},
		{"zero Speed", "unhappy (boundary, minimal)", 0, 0},
		{"one square", "unhappy (boundary)", 5, 1},
		{"one foot short of a square", "unhappy (boundary)", 4, 0},
		{"one foot over a square", "unhappy (boundary)", 6, 1},
		{"non-multiple rounds down", "unhappy (rule interaction)", 32, 6},
		{"large Speed", "happy", 120, 24},
		{"maximum int", "unhappy (boundary)", math.MaxInt, math.MaxInt / 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := g.SpeedInSquares(c.feet)
			if err != nil || got != c.want {
				t.Errorf("SpeedInSquares(%d) = %d, %v, want %d", c.feet, got, err, c.want)
			}
		})
	}
	for _, feet := range []int{-1, math.MinInt} {
		got, err := g.SpeedInSquares(feet)
		if got != 0 || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("SpeedInSquares(%d) = %d, %v, want 0 and ErrInvalidInput", feet, got, err)
		}
	}
}
