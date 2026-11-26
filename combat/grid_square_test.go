package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewGridSquareAcceptsOnlyTheGridRange(t *testing.T) {
	t.Parallel()
	const fr = "doc/specs/grid-positions-and-movement.md FR-002 (D-8)"
	cases := []struct {
		name, requirement, path string
		column, row             int
		ok                      bool
	}{
		{"origin", fr, "happy", 0, 0, true},
		{"column at the upper limit", fr, "unhappy (boundary)", 1_000_000, 0, true},
		{"row at the lower limit", fr, "unhappy (boundary)", 0, -1_000_000, true},
		{"both corners", fr, "unhappy (boundary)", -1_000_000, 1_000_000, true},
		{"column one beyond the upper limit", fr, "unhappy (boundary)", 1_000_001, 0, false},
		{"column one beyond the lower limit", fr, "unhappy (boundary)", -1_000_001, 0, false},
		{"row one beyond the upper limit", fr, "unhappy (boundary)", 0, 1_000_001, false},
		{"row one beyond the lower limit", fr, "unhappy (boundary)", 0, -1_000_001, false},
		{"largest int", fr, "unhappy (invalid input)", math.MaxInt, math.MaxInt, false},
		{"smallest int", fr, "unhappy (invalid input)", math.MinInt, math.MinInt, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewGridSquare(c.column, c.row)
			if c.ok {
				if err != nil || got.Column() != c.column || got.Row() != c.row {
					t.Errorf("NewGridSquare(%d, %d) = %v, %v, want the square [%s]", c.column, c.row, got, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || got != (combat.GridSquare{}) {
				t.Errorf("NewGridSquare(%d, %d) = %v, %v, want the zero square and ErrInvalidInput [%s]", c.column, c.row, got, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-002 - GridSquare is comparable and usable as a map key.
// Path: happy.
func TestGridSquareIsComparableAndAMapKey(t *testing.T) {
	t.Parallel()
	m := map[combat.GridSquare]string{sq(t, 1, 2): "a"}
	again := sq(t, 1, 2)
	if m[again] != "a" || m[sq(t, 2, 1)] != "" || sq(t, 1, 2) != again {
		t.Error("GridSquare does not compare by value")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-040 - MustGridSquare panics with the ErrInvalidInput error and only then.
// Path: unhappy (invalid input).
func TestMustGridSquare(t *testing.T) {
	t.Parallel()
	if got := combat.MustGridSquare(3, -4); got.Column() != 3 || got.Row() != -4 {
		t.Errorf("MustGridSquare(3, -4) = %v", got)
	}
	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("recovered %v, want an ErrInvalidInput error", err)
		}
	}()
	combat.MustGridSquare(1_000_001, 0)
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-003 (SRD 5.2.1, "Entering a Square", p. 13) - eight directions with the offsets of the spec; exactly the four with both offsets non-zero are diagonal.
// Path: happy.
func TestGridDirectionOffsetsAndDiagonals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		d        combat.GridDirection
		name     string
		col, row int
		diagonal bool
	}{
		{combat.GridDirectionNorth, "north", 0, -1, false},
		{combat.GridDirectionNorthEast, "north_east", 1, -1, true},
		{combat.GridDirectionEast, "east", 1, 0, false},
		{combat.GridDirectionSouthEast, "south_east", 1, 1, true},
		{combat.GridDirectionSouth, "south", 0, 1, false},
		{combat.GridDirectionSouthWest, "south_west", -1, 1, true},
		{combat.GridDirectionWest, "west", -1, 0, false},
		{combat.GridDirectionNorthWest, "north_west", -1, -1, true},
	}
	for _, c := range cases {
		col, row := c.d.Offsets()
		if !c.d.Valid() || col != c.col || row != c.row || c.d.IsDiagonal() != c.diagonal || c.d.String() != c.name {
			t.Errorf("%s = (%d, %d) diagonal %v valid %v, want (%d, %d) diagonal %v", c.name, col, row, c.d.IsDiagonal(), c.d.Valid(), c.col, c.row, c.diagonal)
		}
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-003 - the zero direction and one past the last are invalid, move nowhere and are not diagonal.
// Path: unhappy (invalid input).
func TestInvalidGridDirections(t *testing.T) {
	t.Parallel()
	for _, d := range []combat.GridDirection{0, combat.GridDirectionNorthWest + 1, -1} {
		col, row := d.Offsets()
		if d.Valid() || col != 0 || row != 0 || d.IsDiagonal() || d.String() != "unknown" {
			t.Errorf("direction %d = valid %v offsets (%d, %d) diagonal %v %q, want invalid and still", int(d), d.Valid(), col, row, d.IsDiagonal(), d.String())
		}
	}
}
