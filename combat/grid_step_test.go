package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type stepWant struct {
	blocked   combat.GridStepBlockReason // 0: the step is allowed
	terrain   combat.Terrain
	occupants []combat.Occupant
	end       combat.SegmentEnd
}

// Requirement: per case
// Path: per case.
func TestAssessStep(t *testing.T) {
	t.Parallel()
	med, tiny, large := content.SizeMedium, content.SizeTiny, content.SizeLarge
	mover := func(size content.CreatureSize) combat.GridCreature { return gridCreature(t, "mover", size, 5, 5) }
	other := func(id string, size content.CreatureSize, c, r int) combat.GridCreature {
		return gridCreature(t, id, size, c, r)
	}
	none := combat.ConditionActionAvailabilityNone
	avail := combat.ConditionActionAvailabilityAvailable
	occ := func(size content.CreatureSize, rel combat.OccupantRelation, a combat.ConditionActionAvailability) []combat.Occupant {
		return []combat.Occupant{{Size: size, Relation: rel, Availability: a}}
	}
	cases := []struct {
		name, requirement, path string
		mover                   combat.GridCreature
		dir                     combat.GridDirection
		difficult, filled       []int
		creatures               []combat.GridCreature
		allies                  []combat.CombatantID
		want                    stepWant
	}{
		{"open, orthogonal", "FR-020", "happy", mover(med), combat.GridDirectionEast, nil, nil, nil, nil, stepWant{}},
		{"open, diagonal", "FR-020", "happy", mover(med), combat.GridDirectionNorthEast, nil, nil, nil, nil, stepWant{}},
		{"difficult entered square", "FR-024", "happy", mover(med), combat.GridDirectionEast, []int{6, 5}, nil, nil, nil, stepWant{terrain: combat.TerrainDifficult}},
		{"filled entered square", "FR-021", "unhappy (rule interaction)", mover(med), combat.GridDirectionEast, nil, []int{6, 5}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"diagonal, destination filled", "FR-021", "unhappy (rule interaction)", mover(med), combat.GridDirectionSouthEast, nil, []int{6, 6}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"diagonal, east neighbour filled", "FR-022", "unhappy (boundary)", mover(med), combat.GridDirectionSouthEast, nil, []int{6, 5}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"diagonal, south neighbour filled", "FR-022", "unhappy (boundary)", mover(med), combat.GridDirectionSouthEast, nil, []int{5, 6}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"diagonal, both neighbours filled", "FR-022", "unhappy (boundary)", mover(med), combat.GridDirectionSouthEast, nil, []int{6, 5, 5, 6}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"diagonal north west, north neighbour filled", "FR-022", "unhappy (boundary)", mover(med), combat.GridDirectionNorthWest, nil, []int{5, 4}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"diagonal, filled destination and neighbour: FR-021 first", "FR-022", "unhappy (rule interaction)", mover(med), combat.GridDirectionSouthEast, nil, []int{6, 6, 6, 5}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"diagonal, neighbour only difficult", "FR-024", "unhappy (boundary)", mover(med), combat.GridDirectionSouthEast, []int{6, 5}, nil, nil, nil, stepWant{}},
		{"orthogonal beside a filled square", "FR-023", "unhappy (boundary)", mover(med), combat.GridDirectionEast, nil, []int{6, 6}, nil, nil, stepWant{}},
		{"creature beside the corner", "FR-023", "unhappy (rule interaction)", mover(med), combat.GridDirectionSouthEast, nil, nil,
			[]combat.GridCreature{other("ally", med, 6, 5)}, []combat.CombatantID{"ally"}, stepWant{}},
		{"ally in the entered square", "FR-025, FR-026", "happy", mover(med), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("ally", med, 6, 5)}, []combat.CombatantID{"ally"}, stepWant{occupants: occ(med, combat.OccupantAlly, avail), end: combat.SegmentEndOccupiedByCreature}},
		{"Tiny enemy in the square", "FR-025", "unhappy (boundary)", mover(med), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("foe", tiny, 6, 5)}, nil, stepWant{occupants: occ(tiny, combat.OccupantOther, avail), end: combat.SegmentEndOccupiedByCreature}},
		{"Incapacitated creature", "FR-025", "unhappy (rule interaction)", mover(med), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{gridCreatureAs(t, "down", med, 6, 5, none)}, nil, stepWant{occupants: occ(med, combat.OccupantOther, none), end: combat.SegmentEndOccupiedByCreature}},
		{"the mover's own board entry", "FR-025", "unhappy (state error)", mover(med), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{mover(med)}, nil, stepWant{}},
		{"same creature on two entered squares of a Large mover", "FR-025", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("foe", large, 7, 5)}, nil, stepWant{occupants: occ(large, combat.OccupantOther, avail), end: combat.SegmentEndOccupiedByCreature}},
		{"Large mover, difficult in one entered square", "FR-024", "unhappy (rule interaction)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionEast, []int{7, 6}, nil, nil, nil, stepWant{terrain: combat.TerrainDifficult}},
		{"Large mover, difficult in a kept square", "FR-024", "unhappy (rule interaction)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionEast, []int{6, 5, 5, 5}, nil, nil, nil, stepWant{}},
		{"Large mover, filled square in an entered square", "FR-021", "unhappy (rule interaction)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionSouthEast, nil, []int{7, 7}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"Large mover, filled square beside the corner (east)", "FR-022", "unhappy (rule interaction)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionSouthEast, nil, []int{7, 5}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"Large mover, filled square beside the corner (south)", "FR-022", "unhappy (rule interaction)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionSouthEast, nil, []int{5, 7}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"Tiny mover into a Tiny creature's square", "FR-026", "unhappy (boundary)", mover(tiny), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("foe", tiny, 6, 5)}, nil, stepWant{occupants: occ(tiny, combat.OccupantOther, avail)}},
		{"Tiny mover into a Medium creature's square", "FR-026", "unhappy (boundary)", mover(tiny), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("foe", med, 6, 5)}, nil, stepWant{occupants: occ(med, combat.OccupantOther, avail), end: combat.SegmentEndOccupiedByCreature}},
		{"Medium mover into a Tiny creature's square with a second Tiny one", "FR-026", "unhappy (boundary)", mover(med), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("a", tiny, 6, 5), other("b", tiny, 6, 5)},
			nil, stepWant{occupants: []combat.Occupant{{Size: tiny, Availability: avail}, {Size: tiny, Availability: avail}}, end: combat.SegmentEndOccupiedByCreature}},
		{"Tiny mover among a Tiny and a Medium creature", "FR-026", "unhappy (boundary)", mover(tiny), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("a", tiny, 6, 5), other("b", med, 6, 5)},
			nil, stepWant{occupants: []combat.Occupant{{Size: tiny, Availability: avail}, {Size: med, Availability: avail}}, end: combat.SegmentEndOccupiedByCreature}},
		{"Large mover, creature in a kept square", "FR-025, FR-026", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("foe", tiny, 6, 5)}, nil, stepWant{end: combat.SegmentEndOccupiedByCreature}},
		{"a creature behind the mover", "FR-025", "unhappy (boundary)", mover(med), combat.GridDirectionEast, nil, nil,
			[]combat.GridCreature{other("behind", med, 4, 5)}, nil, stepWant{}},
		{"Large mover south, filled square in a kept square", "FR-021", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionSouth, nil, []int{5, 6}, nil, nil, stepWant{}},
		{"Large mover south, difficult square in a kept square", "FR-024", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionSouth, []int{6, 6}, nil, nil, nil, stepWant{}},
		{"Large mover south, filled square in the entered row", "FR-021", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionSouth, nil, []int{6, 7}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"Large mover west, filled square in the entered column", "FR-021", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionWest, nil, []int{4, 6}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"Large mover west, filled square in a kept square", "FR-021", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionWest, nil, []int{5, 5}, nil, nil, stepWant{}},
		{"Large mover north, difficult square in the entered row", "FR-024", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionNorth, []int{6, 4}, nil, nil, nil, stepWant{terrain: combat.TerrainDifficult}},
		{"Large mover north, difficult square in a kept square", "FR-024", "unhappy (boundary)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionNorth, []int{6, 5}, nil, nil, nil, stepWant{}},
		{"Gargantuan, 4 squares entered", "FR-024", "unhappy (boundary)", gridCreature(t, "mover", content.SizeGargantuan, 5, 5), combat.GridDirectionEast, []int{9, 5, 9, 6, 9, 7, 9, 8}, nil, nil, nil, stepWant{terrain: combat.TerrainDifficult}},
		{"Gargantuan, filled square one row beyond the entered column", "FR-021", "unhappy (boundary)", gridCreature(t, "mover", content.SizeGargantuan, 5, 5), combat.GridDirectionEast, nil, []int{9, 9}, nil, nil, stepWant{}},
		{"filled and difficult entered together", "FR-021", "unhappy (rule interaction)", mover(med), combat.GridDirectionEast, []int{6, 5}, []int{6, 5}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"all four orthogonal directions open: south", "FR-020", "happy", mover(med), combat.GridDirectionSouth, nil, nil, nil, nil, stepWant{}},
		{"all four orthogonal directions open: west", "FR-020", "happy", mover(med), combat.GridDirectionWest, nil, nil, nil, nil, stepWant{}},
		{"all four orthogonal directions open: north", "FR-020", "happy", mover(med), combat.GridDirectionNorth, nil, nil, nil, nil, stepWant{}},
		{"south west diagonal, west neighbour filled", "FR-022", "unhappy (boundary)", mover(med), combat.GridDirectionSouthWest, nil, []int{4, 5}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"south west diagonal, south neighbour filled", "FR-022", "unhappy (boundary)", mover(med), combat.GridDirectionSouthWest, nil, []int{5, 6}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"north east diagonal, east neighbour filled", "FR-022", "unhappy (boundary)", mover(med), combat.GridDirectionNorthEast, nil, []int{6, 5}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
		{"Large mover north west, filled square beside the corner", "FR-022", "unhappy (rule interaction)", gridCreature(t, "mover", large, 5, 5), combat.GridDirectionNorthWest, nil, []int{6, 4}, nil, nil, stepWant{blocked: combat.GridStepBlockReasonCuttingCorner}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			board := gridBoard(t, combat.MustGridTerrain(squares(t, c.difficult...), squares(t, c.filled...)), c.creatures...)
			got, err := gridRules.AssessStep(squareGrid, board, c.mover, c.dir, c.allies)
			if err != nil {
				t.Fatalf("AssessStep: %v", err)
			}
			if c.want.blocked != 0 {
				if got != (combat.GridStepBlocked{Reason: c.want.blocked}) {
					t.Errorf("AssessStep = %#v, want blocked %v [%s]", got, c.want.blocked, c.requirement)
				}
				return
			}
			c0, r0 := c.dir.Offsets()
			wantStep := combat.GridStepAllowed{Footprint: fp(t, c.mover.Footprint().Origin().Column()+c0, c.mover.Footprint().Origin().Row()+r0, c.mover.Footprint().Width(), c.mover.Footprint().Height()),
				Terrain: c.want.terrain, Occupants: c.want.occupants, End: c.want.end}
			allowed, ok := got.(combat.GridStepAllowed)
			if !ok || allowed.Footprint != wantStep.Footprint || allowed.Terrain != wantStep.Terrain || allowed.End != wantStep.End ||
				!slices.Equal(allowed.Occupants, wantStep.Occupants) {
				t.Errorf("AssessStep = %#v, want %#v [%s]", got, wantStep, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-025 - occupants are listed once each in board order, and an ally is told by the id set alone.
// Path: happy.
func TestAssessStepListsOccupantsInBoardOrder(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "mover", content.SizeLarge, 5, 5)
	first, second := gridCreature(t, "first", content.SizeMedium, 7, 6), gridCreature(t, "second", content.SizeLarge, 7, 5)
	board := gridBoard(t, combat.GridTerrain{}, second, first)
	got, err := gridRules.AssessStep(squareGrid, board, mover, combat.GridDirectionEast, []combat.CombatantID{"first", "stranger"})
	allowed, ok := got.(combat.GridStepAllowed)
	want := []combat.Occupant{{Size: content.SizeLarge, Relation: combat.OccupantOther, Availability: combat.ConditionActionAvailabilityAvailable},
		{Size: content.SizeMedium, Relation: combat.OccupantAlly, Availability: combat.ConditionActionAvailabilityAvailable}}
	if err != nil || !ok || !slices.Equal(allowed.Occupants, want) {
		t.Errorf("AssessStep = %#v, %v, want occupants %v in board order", got, err, want)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-032 - AssessStep changes neither the board nor the mover, returns a fresh occupants slice, and gives equal outputs for equal inputs.
// Path: unhappy (immutability).
func TestAssessStepChangesNothing(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "mover", content.SizeMedium, 5, 5)
	foe := gridCreature(t, "foe", content.SizeMedium, 6, 5)
	board := gridBoard(t, combat.MustGridTerrain(squares(t, 6, 5), nil), foe)
	allies := []combat.CombatantID{"foe"}
	first, err := gridRules.AssessStep(squareGrid, board, mover, combat.GridDirectionEast, allies)
	if err != nil {
		t.Fatal(err)
	}
	first.(combat.GridStepAllowed).Occupants[0].Size = content.SizeHuge
	second, _ := gridRules.AssessStep(squareGrid, board, mover, combat.GridDirectionEast, allies)
	third, _ := gridRules.AssessStep(squareGrid, board, mover, combat.GridDirectionEast, allies)
	if !reflect.DeepEqual(second, third) || second.(combat.GridStepAllowed).Occupants[0].Size != content.SizeMedium {
		t.Errorf("AssessStep is not deterministic or shares its occupants: %#v %#v", second, third)
	}
	if !slices.Equal(board.Creatures(), []combat.GridCreature{foe}) || mover.Footprint() != fp(t, 5, 5, 1, 1) || !slices.Equal(allies, []combat.CombatantID{"foe"}) ||
		board.Terrain().TerrainOf(sq(t, 6, 5)) != combat.SquareTerrainDifficult {
		t.Error("AssessStep changed its inputs")
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-031, FR-040 - a step that leaves the grid range, an invalid direction and the zero mover are ErrInvalidInput errors, not blocked steps.
// Path: unhappy (invalid input).
func TestAssessStepRejects(t *testing.T) {
	t.Parallel()
	edge := gridCreature(t, "edge", content.SizeMedium, 1_000_000, 0)
	cases := []struct {
		name  string
		mover combat.GridCreature
		dir   combat.GridDirection
	}{
		{"east past the upper column", edge, combat.GridDirectionEast},
		{"north east past the upper column", edge, combat.GridDirectionNorthEast},
		{"north past the lower row of a Large mover at the edge", gridCreature(t, "edge", content.SizeLarge, 0, -1_000_000), combat.GridDirectionNorth},
		{"an invalid direction", edge, 0},
		{"the zero mover", combat.GridCreature{}, combat.GridDirectionEast},
	}
	for _, c := range cases {
		if got, err := gridRules.AssessStep(squareGrid, combat.GridBoard{}, c.mover, c.dir, nil); !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
			t.Errorf("%s: AssessStep = %v, %v, want ErrInvalidInput", c.name, got, err)
		}
	}
	if got, err := gridRules.AssessStep(squareGrid, combat.GridBoard{}, edge, combat.GridDirectionWest, nil); err != nil || got == nil {
		t.Errorf("a step back from the edge: %v, %v", got, err)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-002, FR-031, D-8, plan 5.7 step 7d - the largest mover (Gargantuan, 4 squares a side) steps to exactly the last column of the grid, is judged on that column, and one step further is an ErrInvalidInput error; a creature footprint never reaches 1,000 squares a side, which only objects and locations of a range have.
// Path: unhappy (boundary).
func TestAssessStepAtTheLimitOfTheGrid(t *testing.T) {
	t.Parallel()
	big := gridCreature(t, "big", content.SizeGargantuan, 1_000_000-4, -1_000_000)
	board := gridBoard(t, combat.MustGridTerrain(squares(t, 1_000_000, -1_000_000+3), nil))
	got, err := gridRules.AssessStep(squareGrid, board, big, combat.GridDirectionEast, nil)
	if err != nil || !stepIsDifficult(got) || got.(combat.GridStepAllowed).Footprint != fp(t, 1_000_000-3, -1_000_000, 4, 4) {
		t.Fatalf("the step onto the last column = %#v, %v", got, err)
	}
	at, err := big.WithFootprint(got.(combat.GridStepAllowed).Footprint)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gridRules.AssessStep(squareGrid, board, at, combat.GridDirectionEast, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a step one column past the limit: %v, want ErrInvalidInput", err)
	}
	if _, err := gridRules.AssessStep(squareGrid, board, at, combat.GridDirectionNorth, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a step one row past the lower limit: %v, want ErrInvalidInput", err)
	}
}
func stepIsDifficult(a combat.GridStepAssessment) bool {
	s, ok := a.(combat.GridStepAllowed)
	return ok && s.Terrain == combat.TerrainDifficult
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-021, FR-022 - the block reasons name themselves.
// Path: happy.
func TestGridStepBlockReasonString(t *testing.T) {
	t.Parallel()
	for r, want := range map[combat.GridStepBlockReason]string{combat.GridStepBlockReasonEnteringFilledSquare: "entering_filled_square", combat.GridStepBlockReasonCuttingCorner: "cutting_corner", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("GridStepBlockReason(%d).String() = %q, want %q", int(r), r.String(), want)
		}
	}
}
