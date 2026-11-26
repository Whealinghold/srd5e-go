package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// GridStepBlockReason says why a step is blocked (SRD 5.2.1, "Corners", p. 13). The zero value is invalid.
type GridStepBlockReason int

// The two reasons.
const (
	// GridStepBlockReasonEnteringFilledSquare: the step enters a filled square.
	GridStepBlockReasonEnteringFilledSquare GridStepBlockReason = iota + 1
	// GridStepBlockReasonCuttingCorner: a diagonal step crosses the corner of a filled square.
	GridStepBlockReasonCuttingCorner
)

// String returns "entering_filled_square", "cutting_corner" or "unknown".
func (r GridStepBlockReason) String() string {
	return enumName([]string{"entering_filled_square", "cutting_corner"}, int(r))
}

// GridStepAssessment is the closed set of outcomes of GridRules.AssessStep. Variants: GridStepAllowed,
// GridStepBlocked.
type GridStepAssessment interface{ isGridStepAssessment() }

// GridStepAllowed is a step the grid rules let a creature take. Footprint is where the mover ends, Terrain is
// TerrainDifficult iff any entered square is Difficult Terrain (once, however many are), Occupants lists once each,
// in board order, the other creatures whose space overlaps an entered square, and End says whether the move would
// end in a space another creature occupies. Build it only through AssessStep; the fields are read-only by
// convention and Occupants is a fresh slice.
type GridStepAllowed struct {
	Footprint GridFootprint
	Terrain   Terrain
	Occupants []Occupant
	End       SegmentEnd
}

// GridStepBlocked is a step the grid rules forbid, with its reason. A blocked step is an expected outcome, not an
// error.
type GridStepBlocked struct{ Reason GridStepBlockReason }

func (GridStepAllowed) isGridStepAssessment() {}
func (GridStepBlocked) isGridStepAssessment() {}

// anySquare reports whether pred holds for a square of any strip.
func anySquare(strips []GridFootprint, pred func(GridSquare) bool) bool {
	for _, s := range strips {
		if slices.ContainsFunc(s.Squares(), pred) {
			return true
		}
	}
	return false
}

// overlapsAny reports whether f overlaps a strip.
func overlapsAny(strips []GridFootprint, f GridFootprint) bool {
	return slices.ContainsFunc(strips, f.Overlaps)
}

// AssessStep judges one step of mover in direction on board (SRD 5.2.1, "Entering a Square", "Corners", p. 13;
// "Moving around Other Creatures", "Difficult Terrain", p. 14). A step in any of the eight directions is one square
// of movement and shifts the whole footprint. It is blocked (GridStepBlocked) when it enters a filled square, or,
// diagonally, when a square entered by either of its two component orthogonal steps is filled, filled entered squares
// first; creatures never block a corner. Otherwise it is GridStepAllowed with the terrain, the occupants (an ally
// iff its id is in allies) and the end of the move. Nothing is changed. It returns an error matching
// srderr.ErrInvalidInput for the zero mover, an invalid direction or a step that leaves the grid range. It never
// panics.
func (GridRules) AssessStep(_ SquareGrid, board GridBoard, mover GridCreature, direction GridDirection, allies []CombatantID) (GridStepAssessment, error) {
	if mover.IsZero() {
		return nil, invalidInput("a step needs a mover")
	}
	to, err := mover.footprint.Shifted(direction)
	if err != nil {
		return nil, err
	}
	dc, dr := direction.Offsets()
	entered := to.enteredBy(dc, dr)
	terrain := func(want SquareTerrain) func(GridSquare) bool {
		return func(s GridSquare) bool { return board.terrain.TerrainOf(s) == want }
	}
	if anySquare(entered, terrain(SquareTerrainFilled)) {
		return GridStepBlocked{GridStepBlockReasonEnteringFilledSquare}, nil
	}
	if direction.IsDiagonal() {
		// The component steps are the same footprint shifted by the column offset alone and by the row offset alone.
		for _, part := range [][2]int{{dc, 0}, {0, dr}} {
			// Both footprints are valid, so the one between them is.
			stepped := GridFootprint{mover.footprint.column + part[0], mover.footprint.row + part[1], to.width, to.height}
			if anySquare(stepped.enteredBy(part[0], part[1]), terrain(SquareTerrainFilled)) {
				return GridStepBlocked{GridStepBlockReasonCuttingCorner}, nil
			}
		}
	}
	allowed := GridStepAllowed{Footprint: to, Terrain: TerrainOpen, End: SegmentEndFree}
	if anySquare(entered, terrain(SquareTerrainDifficult)) {
		allowed.Terrain = TerrainDifficult
	}
	allySet := make(map[CombatantID]bool, len(allies))
	for _, a := range allies {
		allySet[a] = true
	}
	allTiny := mover.size == content.SizeTiny
	touched := false
	for _, c := range board.creatures {
		if c.id == mover.id || !to.Overlaps(c.footprint) {
			continue
		}
		touched = true
		allTiny = allTiny && c.size == content.SizeTiny
		if overlapsAny(entered, c.footprint) {
			rel := OccupantOther
			if allySet[c.id] {
				rel = OccupantAlly
			}
			allowed.Occupants = append(allowed.Occupants, Occupant{Size: c.size, Relation: rel, Availability: c.availability})
		}
	}
	if touched && !allTiny {
		allowed.End = SegmentEndOccupiedByCreature
	}
	return allowed, nil
}
