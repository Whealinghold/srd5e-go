package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestRangeBetweenFootprints(t *testing.T) {
	t.Parallel()
	const p13 = `FR-010, FR-014, table 4.2a (SRD 5.2.1, "Ranges", p. 13)`
	cases := []struct {
		name, requirement, path string
		a, b                    combat.GridFootprint
		squares                 int
	}{
		{"orthogonally adjacent", p13, "happy", fp(t, 0, 0, 1, 1), fp(t, 1, 0, 1, 1), 1},
		{"diagonally adjacent", p13, "unhappy (boundary)", fp(t, 0, 0, 1, 1), fp(t, 1, 1, 1, 1), 1},
		{"two squares between", p13, "happy", fp(t, 0, 0, 1, 1), fp(t, 3, 0, 1, 1), 3},
		{"offset in both axes", p13, "happy", fp(t, 0, 0, 1, 1), fp(t, 5, 2, 1, 1), 5},
		{"negative coordinates", p13, "unhappy (boundary)", fp(t, -2, -2, 1, 1), fp(t, 1, 1, 1, 1), 3},
		{"same square", p13, "unhappy (boundary)", fp(t, 4, 4, 1, 1), fp(t, 4, 4, 1, 1), 0},
		{"Large beside Medium", p13, "happy", fp(t, 0, 0, 2, 2), fp(t, 2, 0, 1, 1), 1},
		{"Large, Medium one square off", p13, "happy", fp(t, 0, 0, 2, 2), fp(t, 3, 0, 1, 1), 2},
		{"Large, Medium touching a corner", p13, "unhappy (boundary)", fp(t, 0, 0, 2, 2), fp(t, 2, 2, 1, 1), 1},
		{"Large to Large, two squares between", p13, "happy", fp(t, 0, 0, 2, 2), fp(t, 4, 0, 2, 2), 3},
		{"Medium inside a Large space", p13, "unhappy (boundary)", fp(t, 0, 0, 2, 2), fp(t, 1, 1, 1, 1), 0},
		{"Gargantuan to a far Medium", p13, "happy", fp(t, 0, 0, 4, 4), fp(t, 10, 0, 1, 1), 7},
		{"object 2x1 (a door) to a location", p13, "happy", fp(t, 0, 0, 2, 1), fp(t, 5, 0, 1, 1), 4},
		{"the row gap decides", p13, "unhappy (boundary)", fp(t, 0, 0, 1, 1), fp(t, 2, 6, 1, 1), 6},
		{"overlapping on one axis only", p13, "unhappy (boundary)", fp(t, 0, 0, 3, 1), fp(t, 1, 4, 1, 1), 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := gridRules.RangeInSquares(squareGrid, c.a, c.b)
			swapped, errSwapped := gridRules.RangeInSquares(squareGrid, c.b, c.a)
			feet, errFeet := gridRules.RangeInFeet(squareGrid, c.a, c.b)
			if err != nil || errSwapped != nil || errFeet != nil || got != c.squares || swapped != c.squares || feet != c.squares*5 {
				t.Errorf("range = %d squares (swapped %d), %d feet, errors %v %v %v; want %d squares [%s]", got, swapped, feet, err, errSwapped, errFeet, c.squares, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-010, FR-012, D-8, plan 5.7 step 7d - the range and its feet at the limits of the grid: the whole width of the grid is 2,000,000 squares and 10,000,000 feet, and 1,000 by 1,000 footprints at the corners lose 999 squares on each axis.
// Path: unhappy (boundary).
func TestRangeAtTheLimitsOfTheGrid(t *testing.T) {
	t.Parallel()
	a, b := fp(t, -1_000_000, -1_000_000, 1, 1), fp(t, 1_000_000, 1_000_000, 1, 1)
	if n, err := gridRules.RangeInSquares(squareGrid, a, b); err != nil || n != 2_000_000 {
		t.Errorf("RangeInSquares(corner to corner) = %d, %v, want 2000000", n, err)
	}
	if n, err := gridRules.RangeInFeet(squareGrid, a, b); err != nil || n != 10_000_000 {
		t.Errorf("RangeInFeet(corner to corner) = %d, %v, want 10000000", n, err)
	}
	bigA, bigB := fp(t, -1_000_000, -1_000_000, 1000, 1000), fp(t, 1_000_000-999, 1_000_000-999, 1000, 1000)
	if n, err := gridRules.RangeInSquares(squareGrid, bigA, bigB); err != nil || n != 2_000_000-999-999 {
		t.Errorf("RangeInSquares(1,000 square footprints at the corners) = %d, %v, want %d", n, err, 2_000_000-999-999)
	}
	if ok, err := gridRules.IsWithinFeet(squareGrid, a, b, 10_000_000); err != nil || !ok {
		t.Errorf("IsWithinFeet(exactly the range at the limit) = %v, %v", ok, err)
	}
	if ok, err := gridRules.IsWithinFeet(squareGrid, a, b, 9_999_995); err != nil || ok {
		t.Errorf("IsWithinFeet(one square short at the limit) = %v, %v", ok, err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-010, FR-012, FR-040 - a range needs two valid footprints: the zero footprint is an ErrInvalidInput error, whichever side it is on and whichever operation.
// Path: unhappy (invalid input).
func TestRangeRejectsTheZeroFootprint(t *testing.T) {
	t.Parallel()
	one, zero := fp(t, 0, 0, 1, 1), combat.GridFootprint{}
	for _, pair := range [][2]combat.GridFootprint{{zero, one}, {one, zero}, {zero, zero}} {
		_, e1 := gridRules.RangeInSquares(squareGrid, pair[0], pair[1])
		_, e2 := gridRules.RangeInFeet(squareGrid, pair[0], pair[1])
		_, e3 := gridRules.IsWithinFeet(squareGrid, pair[0], pair[1], 5)
		left, e4 := gridRules.LeavesReach(squareGrid, pair[0], pair[1], one, 5)
		if left {
			t.Error("a rejected LeavesReach reported that the creature leaves")
		}
		_, e5 := gridRules.LeavesReach(squareGrid, one, one, pair[1], 5)
		for _, err := range []error{e1, e2, e3, e4} {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("a zero footprint was accepted: %v", err)
			}
		}
		if pair[1].IsZero() && !errors.Is(e5, srderr.ErrInvalidInput) {
			t.Errorf("LeavesReach(after zero) = %v", e5)
		}
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-012, table 4.2b (SRD 5.2.1, "Ranges", p. 13; "Round Down", p. 187) - IsWithinFeet counts an exactly equal range, rounds feet down to a multiple of 5, and refuses negative feet.
// Path: unhappy (boundary).
func TestIsWithinFeet(t *testing.T) {
	t.Parallel()
	origin := fp(t, 0, 0, 1, 1)
	cases := []struct {
		name  string
		other combat.GridFootprint
		feet  int
		want  bool
	}{
		{"at exactly 5 feet", fp(t, 1, 0, 1, 1), 5, true},
		{"10 feet away, within 5", fp(t, 2, 0, 1, 1), 5, false},
		{"7 feet rounds down to 5, adjacent", fp(t, 1, 0, 1, 1), 7, true},
		{"7 feet rounds down to 5, two squares", fp(t, 2, 0, 1, 1), 7, false},
		{"9 feet is still 5", fp(t, 2, 0, 1, 1), 9, false},
		{"10 feet", fp(t, 2, 0, 1, 1), 10, true},
		{"well inside a larger distance", fp(t, 1, 0, 1, 1), 100, true},
		{"zero feet, same square", fp(t, 0, 0, 1, 1), 0, true},
		{"zero feet, adjacent", fp(t, 1, 0, 1, 1), 0, false},
		{"4 feet rounds to 0, same square", fp(t, 0, 0, 1, 1), 4, true},
	}
	for _, c := range cases {
		if got, err := gridRules.IsWithinFeet(squareGrid, origin, c.other, c.feet); err != nil || got != c.want {
			t.Errorf("%s: IsWithinFeet = %v, %v, want %v", c.name, got, err, c.want)
		}
	}
	if got, err := gridRules.IsWithinFeet(squareGrid, origin, origin, -1); !errors.Is(err, srderr.ErrInvalidInput) || got {
		t.Errorf("IsWithinFeet(-1) = %v, %v, want ErrInvalidInput", got, err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-016, table 4.2b (SRD 5.2.1, "Reach", "Making an Opportunity Attack", p. 15; "Reach", p. 186) - a creature leaves the reach iff it was within reach and is not after; staying inside, starting outside and a reach below 5 feet.
// Path: per case.
func TestLeavesReach(t *testing.T) {
	t.Parallel()
	observer := fp(t, 0, 0, 1, 1)
	cases := []struct {
		name, path    string
		before, after combat.GridFootprint
		reach         int
		want          bool
	}{
		{"leaves a 5-foot reach", "happy", fp(t, 1, 0, 1, 1), fp(t, 2, 0, 1, 1), 5, true},
		{"stays inside (circles)", "unhappy (boundary)", fp(t, 1, 0, 1, 1), fp(t, 0, 1, 1, 1), 5, false},
		{"starts outside", "unhappy (boundary)", fp(t, 2, 0, 1, 1), fp(t, 3, 0, 1, 1), 5, false},
		{"10-foot reach, moves from 10 to 15", "happy", fp(t, 2, 0, 1, 1), fp(t, 3, 0, 1, 1), 10, true},
		{"10-foot reach, moves from 5 to 10", "unhappy (boundary)", fp(t, 1, 0, 1, 1), fp(t, 2, 0, 1, 1), 10, false},
		{"moves closer", "unhappy (boundary)", fp(t, 2, 0, 1, 1), fp(t, 1, 0, 1, 1), 5, false},
		{"reach 7 acts as 5", "unhappy (boundary)", fp(t, 1, 0, 1, 1), fp(t, 2, 0, 1, 1), 7, true},
	}
	for _, c := range cases {
		if got, err := gridRules.LeavesReach(squareGrid, observer, c.before, c.after, c.reach); err != nil || got != c.want {
			t.Errorf("%s: LeavesReach = %v, %v, want %v [%s]", c.name, got, err, c.want, c.path)
		}
	}
	for _, reach := range []int{4, 0, -5} {
		if got, err := gridRules.LeavesReach(squareGrid, observer, observer, observer, reach); !errors.Is(err, srderr.ErrInvalidInput) || got {
			t.Errorf("LeavesReach(reach %d) = %v, %v, want ErrInvalidInput", reach, got, err)
		}
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-015 (SRD 5.2.1, "Ranged Attacks in Close Combat", p. 15), FR-013 - DistancesFeetFrom maps every other creature to its range in feet, ignoring walls and allegiance, as a fresh map; an empty map when nobody else is on the board; the zero subject is refused.
// Path: unhappy (empty case).
func TestDistancesFeetFrom(t *testing.T) {
	t.Parallel()
	subject := gridCreature(t, "subject", content.SizeMedium, 0, 0)
	near, far := gridCreature(t, "near", content.SizeMedium, 1, 0), gridCreature(t, "far", content.SizeLarge, 6, 0)
	late := gridCreature(t, "zebra", content.SizeMedium, 2, 0)
	shared := gridCreature(t, "shared", content.SizeTiny, 0, 0)
	walled := combat.MustGridTerrain(nil, squares(t, 1, 0, 2, 0, 3, 0))
	board := gridBoard(t, walled, subject, near, far, shared, late)

	got, err := gridRules.DistancesFeetFrom(squareGrid, board, subject)
	want := map[combat.CombatantID]int{"near": 5, "far": 30, "shared": 0, "zebra": 10}
	if err != nil || len(got) != 4 || got["near"] != want["near"] || got["far"] != want["far"] || got["shared"] != want["shared"] || got["zebra"] != want["zebra"] {
		t.Errorf("DistancesFeetFrom = %v, %v, want %v (walls ignored, subject excluded)", got, err, want)
	}
	got["near"] = 99
	if again, _ := gridRules.DistancesFeetFrom(squareGrid, board, subject); again["near"] != 5 {
		t.Error("DistancesFeetFrom returned shared state")
	}
	if got, err := gridRules.DistancesFeetFrom(squareGrid, gridBoard(t, combat.GridTerrain{}, subject), subject); err != nil || got == nil || len(got) != 0 {
		t.Errorf("a board with only the subject = %v, %v, want an empty map", got, err)
	}
	if got, err := gridRules.DistancesFeetFrom(squareGrid, combat.GridBoard{}, subject); err != nil || len(got) != 0 {
		t.Errorf("an empty board = %v, %v, want an empty map", got, err)
	}
	if got, err := gridRules.DistancesFeetFrom(squareGrid, board, combat.GridCreature{}); !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
		t.Errorf("the zero subject = %v, %v, want ErrInvalidInput", got, err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-013 - range measures and does not move: walls and filled squares between two spaces do not change it (RangeInSquares takes no board, and DistancesFeetFrom gives the same answer with and without a wall).
// Path: unhappy (rule interaction).
func TestRangeIgnoresObstacles(t *testing.T) {
	t.Parallel()
	subject, other := gridCreature(t, "subject", content.SizeMedium, 0, 0), gridCreature(t, "other", content.SizeMedium, 4, 0)
	open := gridBoard(t, combat.GridTerrain{}, subject, other)
	walled := gridBoard(t, combat.MustGridTerrain(nil, squares(t, 1, 0, 2, 0, 3, 0)), subject, other)
	a, _ := gridRules.DistancesFeetFrom(squareGrid, open, subject)
	b, _ := gridRules.DistancesFeetFrom(squareGrid, walled, subject)
	if a["other"] != 20 || b["other"] != 20 {
		t.Errorf("distances = %v and %v, want 20 feet with and without the wall", a, b)
	}
}
