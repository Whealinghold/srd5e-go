package combat

import "github.com/Whealinghold/srd5e-go/content"

// gridSideLimit is the largest width or height of a footprint (FR-004, D-8).
const gridSideLimit = 1000

// GridFootprint is the space of a creature or object on the grid: a rectangle of squares given by its origin
// (lowest column and row), width and height (SRD 5.2.1, "Creature Size", p. 14). The zero GridFootprint is
// invalid (IsZero). Comparable with ==. Safe for concurrent use.
//
// Contains, Overlaps and the range maths use the corners of the rectangle, never the list of squares, so a
// footprint of 1,000 by 1,000 costs no more than a single square. Only Squares lists them.
type GridFootprint struct{ column, row, width, height int }

// NewGridFootprint returns the rectangle with its lowest column and row at origin, width squares wide and
// height squares tall. It returns an error matching srderr.ErrInvalidInput when width or height is outside 1 to
// 1,000 or when a square it covers is outside the range of NewGridSquare. It never panics.
func NewGridFootprint(origin GridSquare, width, height int) (GridFootprint, error) {
	return footprintAt(origin.column, origin.row, width, height)
}

// footprintAt validates the four numbers with arithmetic that cannot overflow: the sides are at most 1,000 and
// the coordinates at most about a million, whatever the caller passes first.
func footprintAt(column, row, width, height int) (GridFootprint, error) {
	if width < 1 || width > gridSideLimit || height < 1 || height > gridSideLimit ||
		!inGridRange(column) || !inGridRange(row) || !inGridRange(column+width-1) || !inGridRange(row+height-1) {
		return GridFootprint{}, invalidInput("a footprint needs sides of 1 to 1,000 squares inside the grid range")
	}
	return GridFootprint{column, row, width, height}, nil
}

// GridFootprintForCreature returns the square footprint of a creature of the given size with its lowest column
// and row at origin: one square for Tiny, Small and Medium, then 2, 3 and 4 squares a side (SRD 5.2.1, Creature
// Size and Space table, p. 14; the side is taken from content.CreatureSize.Space). It returns an error matching
// srderr.ErrInvalidInput for an invalid size or a footprint that leaves the grid range. It never panics.
func GridFootprintForCreature(size content.CreatureSize, origin GridSquare) (GridFootprint, error) {
	side := 0
	switch s := size.Space().(type) {
	case content.SharedSquareSpace:
		side = 1
	case content.WholeSquareSpace:
		side = s.SquaresPerSide()
	default:
		return GridFootprint{}, invalidInput("a creature footprint needs a valid size")
	}
	return footprintAt(origin.column, origin.row, side, side)
}

// IsZero reports whether f is the invalid zero footprint.
func (f GridFootprint) IsZero() bool { return f.width == 0 }

// Origin returns the square with the lowest column and row.
func (f GridFootprint) Origin() GridSquare { return GridSquare{f.column, f.row} }

// Width returns the number of columns.
func (f GridFootprint) Width() int { return f.width }

// Height returns the number of rows.
func (f GridFootprint) Height() int { return f.height }

func (f GridFootprint) maxColumn() int { return f.column + f.width - 1 }
func (f GridFootprint) maxRow() int    { return f.row + f.height - 1 }

// Squares returns every square of f row by row from the origin, as a fresh slice on each call.
func (f GridFootprint) Squares() []GridSquare {
	out := make([]GridSquare, 0, f.width*f.height)
	for r := f.row; r <= f.maxRow(); r++ {
		for c := f.column; c <= f.maxColumn(); c++ {
			out = append(out, GridSquare{c, r})
		}
	}
	return out
}

// Contains reports whether square lies in f.
func (f GridFootprint) Contains(square GridSquare) bool {
	return square.column >= f.column && square.column <= f.maxColumn() && square.row >= f.row && square.row <= f.maxRow()
}

// Overlaps reports whether f and other share a square. The zero footprint overlaps nothing.
func (f GridFootprint) Overlaps(other GridFootprint) bool {
	return !f.IsZero() && !other.IsZero() && f.column <= other.maxColumn() && other.column <= f.maxColumn() &&
		f.row <= other.maxRow() && other.row <= f.maxRow()
}

// Shifted returns the same rectangle one square on in direction. It returns an error matching
// srderr.ErrInvalidInput for an invalid direction, the zero footprint, or a square that leaves the range of
// NewGridSquare.
func (f GridFootprint) Shifted(direction GridDirection) (GridFootprint, error) {
	if !direction.Valid() || f.IsZero() {
		return GridFootprint{}, invalidInput("a shift needs a direction and a footprint")
	}
	c, r := direction.Offsets()
	return footprintAt(f.column+c, f.row+r, f.width, f.height)
}

// enteredBy returns the squares a step of (dc, dr) enters, given the footprint f it ends in, as at most two
// strips whose union is f minus the footprint before the step (D-5): the leading column (rows of f) when
// dc is not 0 and the leading row (columns of f) when dr is not 0. The strips overlap at a corner of a diagonal
// step. A strip of f is always inside the grid range.
func (f GridFootprint) enteredBy(dc, dr int) []GridFootprint {
	var out []GridFootprint
	if dc != 0 {
		col := f.column
		if dc > 0 {
			col = f.maxColumn()
		}
		out = append(out, GridFootprint{col, f.row, 1, f.height})
	}
	if dr != 0 {
		row := f.row
		if dr > 0 {
			row = f.maxRow()
		}
		out = append(out, GridFootprint{f.column, row, f.width, 1})
	}
	return out
}
