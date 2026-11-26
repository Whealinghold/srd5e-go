package combat

// The grid vocabulary of grid-positions-and-movement.md: squares and directions. Pure and silent.

// gridLimit is the largest absolute value of a column or a row (FR-002, D-8). It keeps every sum
// the grid rules compute within a 32-bit int on every platform.
const gridLimit = 1_000_000

// GridSquare is one square of the square grid (SRD 5.2.1, "Playing on a Grid", p. 13): a whole-number column
// and row, the row growing southward. The zero GridSquare is the valid square (0, 0). Comparable with ==, so
// it is usable as a map key. Safe for concurrent use.
type GridSquare struct{ column, row int }

// NewGridSquare returns the square at column and row. It returns an error matching srderr.ErrInvalidInput
// when either lies outside -1,000,000 to 1,000,000 inclusive. It never panics.
func NewGridSquare(column, row int) (GridSquare, error) {
	if !inGridRange(column) || !inGridRange(row) {
		return GridSquare{}, invalidInput("a grid square needs a column and a row from -1,000,000 to 1,000,000")
	}
	return GridSquare{column, row}, nil
}

// MustGridSquare is NewGridSquare for literals known to be valid. It panics with the error NewGridSquare
// returns, and only then.
func MustGridSquare(column, row int) GridSquare {
	s, err := NewGridSquare(column, row)
	if err != nil {
		panic(err)
	}
	return s
}

func inGridRange(v int) bool { return v >= -gridLimit && v <= gridLimit }

// Column returns the column of the square.
func (s GridSquare) Column() int { return s.column }

// Row returns the row of the square.
func (s GridSquare) Row() int { return s.row }

// GridDirection is one of the eight directions of a step (SRD 5.2.1, "Entering a Square", p. 13). The zero
// GridDirection is invalid.
type GridDirection int

// The eight directions, in the order of FR-003.
const (
	GridDirectionNorth GridDirection = iota + 1
	GridDirectionNorthEast
	GridDirectionEast
	GridDirectionSouthEast
	GridDirectionSouth
	GridDirectionSouthWest
	GridDirectionWest
	GridDirectionNorthWest
)

// Valid reports whether d is one of the eight directions.
func (d GridDirection) Valid() bool { return d >= GridDirectionNorth && d <= GridDirectionNorthWest }

// String returns "north", "north_east", "east", "south_east", "south", "south_west", "west", "north_west"
// or "unknown".
func (d GridDirection) String() string {
	return enumName([]string{"north", "north_east", "east", "south_east", "south", "south_west", "west", "north_west"}, int(d))
}

// Offsets returns the column and row offsets of d: north (0, -1), east (1, 0), south (0, 1), west (-1, 0) and
// the four diagonals between them. An invalid direction gives (0, 0).
func (d GridDirection) Offsets() (column, row int) {
	switch d {
	case GridDirectionNorth:
		return 0, -1
	case GridDirectionNorthEast:
		return 1, -1
	case GridDirectionEast:
		return 1, 0
	case GridDirectionSouthEast:
		return 1, 1
	case GridDirectionSouth:
		return 0, 1
	case GridDirectionSouthWest:
		return -1, 1
	case GridDirectionWest:
		return -1, 0
	case GridDirectionNorthWest:
		return -1, -1
	}
	return 0, 0
}

// IsDiagonal reports whether both offsets of d are non-zero.
func (d GridDirection) IsDiagonal() bool {
	c, r := d.Offsets()
	return c != 0 && r != 0
}
