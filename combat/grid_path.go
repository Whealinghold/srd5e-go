package combat

// GridPathPlan is the closed set of outcomes of GridRules.PlanPath. Variants: GridPathPlanned, GridPathBlocked.
type GridPathPlan interface{ isGridPathPlan() }

// GridPathPlanned is a path whose every step the grid rules allow: where the mover ends and one MoveSegment per
// step for TurnTracker.Move to judge. Segments is a fresh slice.
type GridPathPlanned struct {
	FinalFootprint GridFootprint
	Segments       []MoveSegment
}

// GridPathBlocked is a path stopped at the step with the given zero-based index; it carries no segments.
type GridPathBlocked struct {
	StepIndex int
	Reason    GridStepBlockReason
}

func (GridPathPlanned) isGridPathPlan() {}
func (GridPathBlocked) isGridPathPlan() {}

// PlanPath applies AssessStep to each direction in order, the mover's footprint following each allowed step (SRD
// 5.2.1, "Playing on a Grid", "Speed", "Entering a Square", p. 13). It returns GridPathPlanned with one 5-foot
// MoveSegment per step (the given mode, speedUse and extraCostPerFoot, and the step's terrain, occupants and end),
// or GridPathBlocked at the first blocked step. It does not judge creatures that cannot be passed through, ending in
// an occupied space or whether the Speed suffices: TurnTracker.Move refuses those, so a plan with an enemy square is
// returned as planned. A positive extraCostPerFoot makes each square cost that many extra squares. The grid is
// unbounded; a host bounds the play area with filled squares. Nothing is changed: the host applies a plan with
// board.Place. It returns an error matching srderr.ErrInvalidInput for the zero mover, no directions, an invalid
// direction, mode or speedUse, a negative extraCostPerFoot, or a step that leaves the grid range. The cost is linear
// in the number of directions (a path is capped at 4096 directions, plan P10). It returns a LimitExceededError for more than 4096
// directions, before any step is assessed. It never panics.
func (r GridRules) PlanPath(grid SquareGrid, board GridBoard, mover GridCreature, directions []GridDirection,
	allies []CombatantID, mode MovementMode, speedUse SpeedUse, extraCostPerFoot int) (GridPathPlan, error) {
	if len(directions) == 0 {
		return nil, invalidInput("a path needs at least one step")
	}
	if err := overCap("path directions", len(directions), maxPathDirections); err != nil {
		return nil, err
	}
	// The segment is built from validated parts only: the template checks the host's numbers once, so a blocked first step still reports them.
	if _, err := NewMoveSegment(MoveSegmentInput{DistanceFeet: grid.FeetPerSquare(), Mode: mode, SpeedUse: speedUse, ExtraCostPerFoot: extraCostPerFoot}); err != nil {
		return nil, err
	}
	segments := make([]MoveSegment, 0, len(directions))
	for i, d := range directions {
		assessed, err := r.AssessStep(grid, board, mover, d, allies)
		if err != nil {
			return nil, err
		}
		switch step := assessed.(type) {
		case GridStepBlocked:
			return GridPathBlocked{i, step.Reason}, nil
		case GridStepAllowed:
			segments = append(segments, MoveSegment{distance: grid.FeetPerSquare(), extra: extraCostPerFoot, mode: mode, use: speedUse,
				terrain: step.Terrain, occupants: step.Occupants, end: step.End})
			mover.footprint = step.Footprint
		}
	}
	return GridPathPlanned{mover.footprint, segments}, nil
}
