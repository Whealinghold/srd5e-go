package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewGridFootprintBounds(t *testing.T) {
	t.Parallel()
	const fr = "doc/specs/grid-positions-and-movement.md FR-004 (D-8)"
	cases := []struct {
		name, requirement, path    string
		column, row, width, height int
		ok                         bool
	}{
		{"one square", fr, "happy", 0, 0, 1, 1, true},
		{"1,000 per side", fr, "unhappy (boundary)", 0, 0, 1000, 1000, true},
		{"1,001 wide", fr, "unhappy (boundary)", 0, 0, 1001, 1, false},
		{"1,001 tall", fr, "unhappy (boundary)", 0, 0, 1, 1001, false},
		{"width 0", fr, "unhappy (boundary)", 0, 0, 0, 1, false},
		{"height 0", fr, "unhappy (boundary)", 0, 0, 1, 0, false},
		{"negative width", fr, "unhappy (invalid input)", 0, 0, -1, 1, false},
		{"largest side", fr, "unhappy (invalid input)", 0, 0, math.MaxInt, math.MaxInt, false},
		{"smallest side", fr, "unhappy (invalid input)", 0, 0, math.MinInt, math.MinInt, false},
		{"last square exactly at the upper limit", fr, "unhappy (boundary)", 1_000_000 - 999, 1_000_000 - 999, 1000, 1000, true},
		{"last square one beyond the upper limit", fr, "unhappy (boundary)", 1_000_000 - 998, 0, 1000, 1, false},
		{"last row one beyond the upper limit", fr, "unhappy (boundary)", 0, 1_000_000 - 998, 1, 1000, false},
		{"first square at the lower limit", fr, "unhappy (boundary)", -1_000_000, -1_000_000, 1000, 1000, true},
		{"single square at the upper limit", fr, "unhappy (boundary)", 1_000_000, 1_000_000, 1, 1, true},
		{"single square one beyond", fr, "unhappy (boundary)", 1_000_000, 1_000_000, 2, 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			origin := combat.GridSquare{}
			if c.column != 0 || c.row != 0 {
				origin = sq(t, c.column, c.row)
			}
			got, err := combat.NewGridFootprint(origin, c.width, c.height)
			if c.ok != (err == nil) || (err != nil && (!errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero())) || (err == nil && got.IsZero()) {
				t.Errorf("NewGridFootprint(%d,%d %dx%d) = %v, %v, ok want %v [%s]", c.column, c.row, c.width, c.height, got, err, c.ok, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-004 - the accessors, Squares row by row from the origin as a fresh copy, Contains, and comparison by value.
// Path: happy.
func TestGridFootprintSquaresContainsAndEquality(t *testing.T) {
	t.Parallel()
	f := fp(t, -1, 2, 3, 2)
	want := squares(t, -1, 2, 0, 2, 1, 2, -1, 3, 0, 3, 1, 3)
	got := f.Squares()
	if !slices.Equal(got, want) || f.Origin() != sq(t, -1, 2) || f.Width() != 3 || f.Height() != 2 {
		t.Fatalf("Squares() = %v origin %v %dx%d, want %v", got, f.Origin(), f.Width(), f.Height(), want)
	}
	if zero := (combat.GridFootprint{}).Squares(); len(zero) != 0 {
		t.Errorf("the zero footprint lists %v squares, want none", zero)
	}
	got[0] = sq(t, 9, 9)
	if !slices.Equal(f.Squares(), want) {
		t.Error("Squares() is not a fresh copy: the caller changed the footprint")
	}
	for _, s := range want {
		if !f.Contains(s) {
			t.Errorf("Contains(%v) = false", s)
		}
	}
	for _, s := range squares(t, -2, 2, 2, 2, -1, 1, -1, 4, 5, 5) {
		if f.Contains(s) {
			t.Errorf("Contains(%v) = true, want false", s)
		}
	}
	if f != fp(t, -1, 2, 3, 2) || f == fp(t, -1, 2, 2, 3) {
		t.Error("footprints do not compare by value")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-004 (SRD 5.2.1, "Creature Size", p. 14) - Overlaps is true exactly when the rectangles share a square, by corner arithmetic: touching edges, a gap, containment and the zero footprint.
// Path: unhappy (boundary).
func TestGridFootprintOverlaps(t *testing.T) {
	t.Parallel()
	a := fp(t, 0, 0, 2, 2)
	cases := []struct {
		name string
		b    combat.GridFootprint
		want bool
	}{
		{"identical", fp(t, 0, 0, 2, 2), true},
		{"sharing one corner square", fp(t, 1, 1, 2, 2), true},
		{"inside", fp(t, 1, 1, 1, 1), true},
		{"containing", fp(t, -1, -1, 4, 4), true},
		{"touching on the east edge", fp(t, 2, 0, 1, 2), false},
		{"touching on the south edge", fp(t, 0, 2, 2, 1), false},
		{"touching at a corner", fp(t, 2, 2, 1, 1), false},
		{"touching on the west edge", fp(t, -1, 0, 1, 2), false},
		{"touching on the north edge", fp(t, 0, -1, 2, 1), false},
		{"far away", fp(t, 50, 50, 1, 1), false},
		{"the zero footprint", combat.GridFootprint{}, false},
	}
	for _, c := range cases {
		if got, back := a.Overlaps(c.b), c.b.Overlaps(a); got != c.want || back != c.want {
			t.Errorf("%s: Overlaps = %v, swapped %v, want %v", c.name, got, back, c.want)
		}
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-004, D-8, plan 5.7 step 7d - the overlap test is arithmetic, not an enumeration: two footprints of 1,000 by 1,000 squares answer at the limits of the grid (the squared-distance arithmetic of the plan stays inside a 32-bit int).
// Path: unhappy (boundary).
func TestGridFootprintOverlapsAtTheLimits(t *testing.T) {
	t.Parallel()
	low := fp(t, -1_000_000, -1_000_000, 1000, 1000)
	high := fp(t, 1_000_000-999, 1_000_000-999, 1000, 1000)
	if low.Overlaps(high) || !low.Overlaps(low) || !high.Overlaps(high) {
		t.Error("Overlaps is wrong for 1,000 by 1,000 footprints at the limits")
	}
	if !low.Overlaps(fp(t, -1_000_000+999, -1_000_000+999, 1, 1)) || low.Overlaps(fp(t, -1_000_000+1000, -1_000_000+1000, 1, 1)) {
		t.Error("Overlaps is wrong at the far corner of a 1,000 by 1,000 footprint")
	}
	if !low.Contains(sq(t, -1_000_000+999, -1_000_000+999)) || low.Contains(sq(t, -1_000_000+1000, -1_000_000)) {
		t.Error("Contains is wrong at the far corner of a 1,000 by 1,000 footprint")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-004 - Shifted moves the same rectangle one square in each of the eight directions and refuses a square that leaves the grid range, at each edge, with the receiver unchanged.
// Path: unhappy (boundary).
func TestGridFootprintShifted(t *testing.T) {
	t.Parallel()
	f := fp(t, 5, 5, 2, 3)
	for _, d := range []combat.GridDirection{combat.GridDirectionNorth, combat.GridDirectionNorthEast, combat.GridDirectionEast, combat.GridDirectionSouthEast,
		combat.GridDirectionSouth, combat.GridDirectionSouthWest, combat.GridDirectionWest, combat.GridDirectionNorthWest} {
		c, r := d.Offsets()
		got, err := f.Shifted(d)
		if err != nil || got != fp(t, 5+c, 5+r, 2, 3) {
			t.Errorf("Shifted(%v) = %v, %v", d, got, err)
		}
	}
	edges := []struct {
		name string
		f    combat.GridFootprint
		d    combat.GridDirection
	}{
		{"east past the upper column", fp(t, 1_000_000-1, 0, 2, 1), combat.GridDirectionEast},
		{"south past the upper row", fp(t, 0, 1_000_000-1, 1, 2), combat.GridDirectionSouth},
		{"west past the lower column", fp(t, -1_000_000, 0, 1, 1), combat.GridDirectionWest},
		{"north past the lower row", fp(t, 0, -1_000_000, 1, 1), combat.GridDirectionNorth},
		{"south east out of the corner", fp(t, 1_000_000, 1_000_000, 1, 1), combat.GridDirectionSouthEast},
		{"north west out of the corner", fp(t, -1_000_000, -1_000_000, 1, 1), combat.GridDirectionNorthWest},
	}
	for _, e := range edges {
		if got, err := e.f.Shifted(e.d); !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
			t.Errorf("%s: Shifted = %v, %v, want ErrInvalidInput", e.name, got, err)
		}
	}
	if got, err := fp(t, 1_000_000-2, 0, 2, 1).Shifted(combat.GridDirectionEast); err != nil || got != fp(t, 1_000_000-1, 0, 2, 1) {
		t.Errorf("a shift ending exactly on the limit = %v, %v", got, err)
	}
	if _, err := f.Shifted(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Error("an invalid direction was accepted")
	}
	if _, err := (combat.GridFootprint{}).Shifted(combat.GridDirectionEast); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Error("the zero footprint was shifted")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-005, table 4.1a (SRD 5.2.1, Creature Size and Space table, p. 14) - the footprint of each size is a square of 1, 1, 1, 2, 3 or 4 squares a side, 1, 1, 1, 4, 9 or 16 squares in all, taken from content.CreatureSize.Space.
// Path: per case.
func TestGridFootprintForCreatureTakesTheSideFromTheSize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		size    content.CreatureSize
		side    int
		squares int
		path    string
	}{
		{content.SizeTiny, 1, 1, "unhappy (boundary)"},
		{content.SizeSmall, 1, 1, "happy"},
		{content.SizeMedium, 1, 1, "happy"},
		{content.SizeLarge, 2, 4, "happy"},
		{content.SizeHuge, 3, 9, "happy"},
		{content.SizeGargantuan, 4, 16, "happy"},
	}
	for _, c := range cases {
		t.Run(c.size.String(), func(t *testing.T) {
			t.Parallel()
			got, err := combat.GridFootprintForCreature(c.size, sq(t, 0, 0))
			if err != nil || got != fp(t, 0, 0, c.side, c.side) || len(got.Squares()) != c.squares {
				t.Errorf("GridFootprintForCreature(%v) = %v, %v, want %d x %d (%d squares) [%s]", c.size, got, err, c.side, c.side, c.squares, c.path)
			}
		})
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-005, FR-040 - an invalid size and a footprint that would leave the grid range are ErrInvalidInput errors.
// Path: unhappy (invalid input).
func TestGridFootprintForCreatureRejects(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		size   content.CreatureSize
		origin combat.GridSquare
	}{
		{"the zero size", 0, sq(t, 0, 0)},
		{"a size past the last", content.SizeGargantuan + 1, sq(t, 0, 0)},
		{"Gargantuan at the upper limit", content.SizeGargantuan, sq(t, 1_000_000, 0)},
		{"Gargantuan last square at the limit plus one", content.SizeGargantuan, sq(t, 1_000_000-2, 0)},
	} {
		if got, err := combat.GridFootprintForCreature(c.size, c.origin); !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
			t.Errorf("%s: = %v, %v, want ErrInvalidInput", c.name, got, err)
		}
	}
	if _, err := combat.GridFootprintForCreature(content.SizeGargantuan, sq(t, 1_000_000-3, 0)); err != nil {
		t.Errorf("Gargantuan ending on the limit: %v", err)
	}
}
