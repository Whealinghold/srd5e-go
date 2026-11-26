package combat_test

import (
	"context"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// FuzzNewMountCreatureCommand feeds the constructors host numbers. It asserts properties: no
// constructor panics, a command that was built runs without error, and an accepted mount costs
// half the Speed rounded down.
func FuzzNewMountCreatureCommand(f *testing.F) {
	for _, s := range [][6]int{{1, 3, 30, 30, 5, 1}, {2, 2, 0, 0, 0, 2}, {6, 6, -1, 0, -1, 1}, {0, 9, 1, -5, 6, 3}} {
		f.Add(s[0], s[1], s[2], s[3], s[4], s[5])
	}
	f.Add(math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, 7)
	f.Fuzz(func(t *testing.T, riderSize, mountSize, speed, left, distance, control int) {
		facts, err := combat.NewMountFacts(combat.MountWillingnessWilling, combat.MountAnatomyAppropriate, combat.MountTrainingTrainedToAcceptRider, combat.MountHabitatOther, combat.ExoticSaddleNotFitted)
		if err != nil {
			t.Fatal(err)
		}
		turn, err := combat.NewMountingTurnFacts(combat.RiderTurnActive, speed, left)
		if err != nil {
			return
		}
		cmd, err := combat.NewMountCreatureCommand(combat.MountCreatureInput{Rider: gid("a"), Mount: gid("b"), RiderSize: content.CreatureSize(riderSize),
			MountSize: content.CreatureSize(mountSize), Facts: facts, Turn: turn, DistanceFeet: distance, Control: combat.MountControl(control)})
		if err != nil {
			return
		}
		res, err := combat.NewMountCreature().Run(context.Background(), cmd)
		if err != nil {
			t.Fatalf("Run on a built command: %v", err)
		}
		if a, ok := res.(combat.MountCreatureApplied); ok && a.MovementCostFeet() != speed/2 {
			t.Fatalf("cost = %d for Speed %d, want %d", a.MovementCostFeet(), speed, speed/2)
		}
	})
}
