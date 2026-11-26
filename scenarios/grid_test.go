package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// gridTracker establishes a tracker of "hero" (active) and "foe", both Medium, hero with the given walking Speed.
func gridTracker(t *testing.T, heroWalk int) combat.TurnTracker {
	t.Helper()
	profile := func(id string, walk int) combat.CombatantProfile {
		s, err := combat.NewMovementSpeeds(walk)
		if err != nil {
			t.Fatal(err)
		}
		return combat.CombatantProfile{ID: combat.MustCombatantID(id), Size: content.SizeMedium, Speeds: s}
	}
	standings := []combat.InitiativeStanding{{Combatant: "hero", Count: 20}, {Combatant: "foe", Count: 10}}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, []combat.CombatantProfile{profile("hero", heroWalk), profile("foe", 30)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.InitiativeOrderEstablished).Tracker()
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-001, FR-028, FR-029, section 8 scenario (SRD 5.2.1, "Playing on a Grid", "Speed", "Entering a Square", p. 13; "Round Down", p. 187) - a host
// that plays on a square grid plans a move square by square and gives it to TurnTracker.Move: six open squares are accepted at Speed 30 and a seventh
// is refused noMovementRemaining; one Difficult Terrain square costs two; a Speed of 32 rounds down to 30; an enemy in the path is refused
// blockedByCreature; and without a grid the same host takes the feet path.
// Path: happy.
func TestGridPlanAgainstTheTurnTracker(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	options := combat.NewRuleOptions(combat.SquareGrid{})
	grid, ok := options.Grid().(combat.SquareGrid)
	if !ok {
		t.Fatalf("Grid() = %#v, want SquareGrid", options.Grid())
	}
	if _, ok := combat.StandardRuleOptions().Grid().(combat.SquareGrid); ok {
		t.Fatal("the standard options play on a grid, want NoGrid and the feet path")
	}
	hero := combat.MustCombatantID("hero")
	footprint, err := combat.GridFootprintForCreature(content.SizeMedium, combat.MustGridSquare(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	mover, err := combat.NewGridCreature(hero, content.SizeMedium, footprint, combat.ConditionActionAvailabilityAvailable)
	if err != nil {
		t.Fatal(err)
	}
	east := func(n int) []combat.GridDirection {
		out := make([]combat.GridDirection, n)
		for i := range out {
			out[i] = combat.GridDirectionEast
		}
		return out
	}
	move := func(board combat.GridBoard, speed, steps int) combat.MoveResult {
		t.Helper()
		plan, err := combat.GridRules{}.PlanPath(grid, board, mover, east(steps), nil, combat.MovementModeWalk, combat.SpeedUseWalking, 0)
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := combat.NewMoveCommand(gridTracker(t, speed), hero, plan.(combat.GridPathPlanned).Segments)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewMove().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	open, err := combat.NewGridBoard(combat.GridTerrain{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := move(open, 30, 6).(combat.MoveApplied).Tracker().Resources().MovedFeet(); got != 30 {
		t.Errorf("six open squares at Speed 30 moved %d feet, want 30", got)
	}
	if r, ok := move(open, 30, 7).(combat.MoveRejected); !ok || r.Reason != (combat.TurnNoMovementRemainingRejection{SegmentIndex: 6, MovementLeft: 0}) {
		t.Errorf("seven open squares at Speed 30 = %#v, want noMovementRemaining at the seventh (nothing is left of 30 feet)", r)
	}
	if got := move(open, 32, 6).(combat.MoveApplied).Tracker().Resources().MovedFeet(); got != 30 {
		t.Errorf("six squares at Speed 32 moved %d feet, want 30 (32 rounds down to 6 squares)", got)
	}
	if _, ok := move(open, 32, 7).(combat.MoveRejected); !ok {
		t.Error("a seventh square at Speed 32 was accepted")
	}

	rough, _ := combat.NewGridBoard(combat.MustGridTerrain([]combat.GridSquare{combat.MustGridSquare(2, 0)}, nil), nil)
	if got := move(rough, 30, 5).(combat.MoveApplied).Tracker().Resources().MovedFeet(); got != 30 {
		t.Errorf("five squares over one Difficult Terrain square moved %d feet, want 30 (6 squares of cost)", got)
	}
	if _, ok := move(rough, 30, 6).(combat.MoveRejected); !ok {
		t.Error("six squares over one Difficult Terrain square were accepted at Speed 30")
	}

	foeFootprint, _ := combat.GridFootprintForCreature(content.SizeMedium, combat.MustGridSquare(1, 0))
	foe, err := combat.NewGridCreature(combat.MustCombatantID("foe"), content.SizeMedium, foeFootprint, combat.ConditionActionAvailabilityAvailable)
	if err != nil {
		t.Fatal(err)
	}
	crowded, _ := combat.NewGridBoard(combat.GridTerrain{}, []combat.GridCreature{foe})
	if r, ok := move(crowded, 30, 2).(combat.MoveRejected); !ok || r.Reason != (combat.TurnBlockedByCreatureRejection{}) {
		t.Errorf("an enemy in the path = %#v, want blockedByCreature", r)
	}
}
