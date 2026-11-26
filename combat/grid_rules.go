package combat

// GridRules is the stateless service of the optional "Playing on a Grid" rules (SRD 5.2.1, p. 13): ranges, reach
// and the step and path rules of grid_step.go and grid_path.go. Every method takes the SquareGrid it works on, so
// code without a grid cannot reach it (FR-001). The zero GridRules is ready to use. Pure and silent: it reads no
// port and emits no signal. Safe for concurrent use.
type GridRules struct{}

// rangeSquares is the range of FR-010: the larger of the column gap and the row gap between the rectangles, each
// 0 when they overlap on that axis. Corner arithmetic only; both footprints must be valid.
func rangeSquares(a, b GridFootprint) (int, error) {
	if a.IsZero() || b.IsZero() {
		return 0, invalidInput("a range needs two footprints")
	}
	return squaresBetween(a, b), nil
}

// squaresBetween is rangeSquares for two valid footprints.
func squaresBetween(a, b GridFootprint) int {
	dx := max(a.column-b.maxColumn(), b.column-a.maxColumn(), 0)
	dy := max(a.row-b.maxRow(), b.row-a.maxRow(), 0)
	return max(dx, dy)
}

// RangeInSquares returns the number of squares counted from a square adjacent to a to, and including, the first
// square in b's space, by the shortest route, a diagonal step counting 1 (SRD 5.2.1, "Ranges", p. 13): adjacent
// spaces are 1 and overlapping or identical spaces 0. The result is the same with a and b swapped and ignores
// walls, filled squares, corners and creatures between them (FR-013). It accepts any two footprints, creature,
// object or location. It returns an error matching srderr.ErrInvalidInput for a zero footprint. It never panics.
func (GridRules) RangeInSquares(_ SquareGrid, a, b GridFootprint) (int, error) {
	return rangeSquares(a, b)
}

// RangeInFeet returns RangeInSquares times the feet of a square. It has the errors of RangeInSquares.
func (GridRules) RangeInFeet(grid SquareGrid, a, b GridFootprint) (int, error) {
	n, err := rangeSquares(a, b)
	return n * grid.FeetPerSquare(), err
}

// IsWithinFeet reports whether the range from a to b is at most feet, exactly equal counting; a feet that is not
// a multiple of 5 acts as its multiple of 5 rounded down (SRD 5.2.1, "Round Down", p. 187). It returns an error
// matching srderr.ErrInvalidInput for negative feet or a zero footprint.
func (GridRules) IsWithinFeet(grid SquareGrid, a, b GridFootprint, feet int) (bool, error) {
	squares, err := grid.SpeedInSquares(feet)
	if err != nil {
		return false, err
	}
	n, err := rangeSquares(a, b)
	return err == nil && n <= squares, err
}

// DistancesFeetFrom returns, for every creature on the board other than subject (by id), the range in feet from
// the subject's footprint, as a fresh map; the map is empty when nobody else is on the board. It does not filter by
// enemy, sight or Incapacitated: the host maps ids to the facts attacks need (SRD 5.2.1, "Ranged Attacks in Close
// Combat", p. 15). It returns an error matching srderr.ErrInvalidInput for a zero subject.
func (GridRules) DistancesFeetFrom(grid SquareGrid, board GridBoard, subject GridCreature) (map[CombatantID]int, error) {
	if subject.IsZero() {
		return nil, invalidInput("distances need a subject")
	}
	out := make(map[CombatantID]int, len(board.creatures))
	for _, c := range board.creatures {
		if c.id == subject.id {
			continue
		}
		out[c.id] = squaresBetween(subject.footprint, c.footprint) * grid.FeetPerSquare()
	}
	return out, nil
}

// LeavesReach reports whether a creature that moves from before to after leaves the reach of observer: before is
// within reachFeet of observer and after is not (SRD 5.2.1, "Reach", "Making an Opportunity Attack", p. 15).
// Whether the observer can see the creature and has its Reaction is the caller's. It returns an error matching
// srderr.ErrInvalidInput for a reachFeet below 5 or a zero footprint.
func (r GridRules) LeavesReach(grid SquareGrid, observer, before, after GridFootprint, reachFeet int) (bool, error) {
	if reachFeet < grid.FeetPerSquare() {
		return false, invalidInput("a reach is at least one square")
	}
	was, err := r.IsWithinFeet(grid, observer, before, reachFeet)
	if err != nil {
		return false, err
	}
	is, err := r.IsWithinFeet(grid, observer, after, reachFeet)
	return was && !is, err
}
