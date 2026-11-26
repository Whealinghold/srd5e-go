package combat_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// ExampleGridRules_PlanPath plans a Medium creature's move of three squares east past one Difficult Terrain square
// and then asks how far the next creature is.
func ExampleGridRules_PlanPath() {
	options := combat.NewRuleOptions(combat.SquareGrid{})
	grid, ok := options.Grid().(combat.SquareGrid)
	if !ok {
		return
	}
	origin := combat.MustGridSquare(0, 0)
	footprint, _ := combat.GridFootprintForCreature(content.SizeMedium, origin)
	hero, _ := combat.NewGridCreature(combat.MustCombatantID("hero"), content.SizeMedium, footprint, combat.ConditionActionAvailabilityAvailable)
	board, _ := combat.NewGridBoard(combat.MustGridTerrain([]combat.GridSquare{combat.MustGridSquare(2, 0)}, nil), nil)

	plan, _ := combat.GridRules{}.PlanPath(grid, board, hero,
		[]combat.GridDirection{combat.GridDirectionEast, combat.GridDirectionEast, combat.GridDirectionEast},
		nil, combat.MovementModeWalk, combat.SpeedUseWalking, 0)
	planned := plan.(combat.GridPathPlanned)
	far, _ := combat.NewGridFootprint(combat.MustGridSquare(9, 0), 1, 1)
	feet, _ := combat.GridRules{}.RangeInFeet(grid, planned.FinalFootprint, far)
	fmt.Println(len(planned.Segments), planned.FinalFootprint.Origin().Column(), feet)
	// Output: 3 3 30
}
