package combat

// PlayingOnAGrid is the SRD's "Playing on a Grid" choice (SRD 5.2.1, p. 13).
// Variants: NoGrid, SquareGrid.
type PlayingOnAGrid interface{ playingOnAGrid() }

// NoGrid is the baseline: distances are counted in feet. The default.
type NoGrid struct{}

// SquareGrid plays on a square grid where each square is 5 feet.
type SquareGrid struct{}

func (NoGrid) playingOnAGrid()     {}
func (SquareGrid) playingOnAGrid() {}

// FeetPerSquare returns 5 (SRD 5.2.1, "Playing on a Grid", p. 13).
func (SquareGrid) FeetPerSquare() int { return 5 }

// SpeedInSquares returns feet divided by 5, rounded down (SRD 5.2.1, "Playing on
// a Grid", p. 13; "Round Down", p. 5). It returns 0 and an error matching
// srderr.ErrInvalidInput for negative feet. It never panics.
func (g SquareGrid) SpeedInSquares(feet int) (int, error) {
	if feet < 0 {
		return 0, invalidInput("negative feet")
	}
	return feet / g.FeetPerSquare(), nil
}

// RuleOptions are the optional rules of combat the SRD itself presents.
// It is a value passed in a command, never a global setting. The zero value is
// the standard value (NoGrid) and equals StandardRuleOptions(). Comparable
// and safe for concurrent use.
type RuleOptions struct{ squares bool }

// StandardRuleOptions returns the SRD baseline: NoGrid.
func StandardRuleOptions() RuleOptions { return RuleOptions{} }

// NewRuleOptions returns options that play on grid. A nil grid is NoGrid.
func NewRuleOptions(grid PlayingOnAGrid) RuleOptions {
	return StandardRuleOptions().WithGrid(grid)
}

// WithGrid returns a copy of o with the grid choice changed.
func (o RuleOptions) WithGrid(grid PlayingOnAGrid) RuleOptions {
	switch grid.(type) {
	case SquareGrid:
		o.squares = true
	case NoGrid, nil:
		o.squares = false
	}
	return o
}

// Grid returns the grid choice.
func (o RuleOptions) Grid() PlayingOnAGrid {
	if o.squares {
		return SquareGrid{}
	}
	return NoGrid{}
}
