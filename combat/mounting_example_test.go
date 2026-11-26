package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// A rider with Speed 30 mounts a trained, willing Large horse during their move. Mounting
// costs half the Speed, 15 feet, which the caller charges.
func ExampleMountCreature() {
	facts, _ := combat.NewMountFacts(combat.MountWillingnessWilling, combat.MountAnatomyAppropriate,
		combat.MountTrainingTrainedToAcceptRider, combat.MountHabitatOther, combat.ExoticSaddleNotFitted)
	turn, _ := combat.NewMountingTurnFacts(combat.RiderTurnActive, 30, 30)
	cmd, _ := combat.NewMountCreatureCommand(combat.MountCreatureInput{
		Rider: "rider", Mount: "horse", RiderSize: content.SizeMedium, MountSize: content.SizeLarge,
		Facts: facts, Turn: turn, DistanceFeet: 5, Control: combat.MountControlControlled,
	})
	res, _ := combat.NewMountCreature().Run(context.Background(), cmd)
	applied := res.(combat.MountCreatureApplied)
	control, _ := applied.RiderStatus().Control()
	fmt.Println(applied.MovementCostFeet(), control)
	// Output: 15 controlled
}
