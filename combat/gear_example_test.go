package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A creature steps on Caltrops, fails the Dexterity saving throw and the caller reads the
// effects: the use case applies none of them.
func ExampleResolveGearEntry() {
	owner, walker := combat.MustCombatantID("owner"), combat.MustCombatantID("walker")
	placed, _ := combat.DeployedGears{}.Deploy(combat.MustDeployedGearID("caltrops-1"), combat.DeployedGearKindCaltrops, owner, 5, combat.GroundLevelLevel)
	gears := placed.(combat.DeployedGearsApplied).Gears()

	cmd, _ := combat.NewResolveGearEntryCommand(combat.ResolveGearEntryInput{
		Gears: gears, GearID: combat.MustDeployedGearID("caltrops-1"), Creature: walker, Trigger: combat.AreaTriggerEnters,
		Stamp: combat.MustTurnStamp(1, walker), HoldID: combat.MustHoldID("unused"), Attempt: d20.RollAttempt{},
	})
	use, _ := combat.NewResolveGearEntry(dicetest.NewFixedRoller(9))
	res, _ := use.Run(context.Background(), cmd)

	failed := res.(combat.ResolveGearEntryFailed)
	for _, effect := range failed.Outcome().(combat.GearEntryFailed).Effects() {
		switch e := effect.(type) {
		case combat.GearDamageEffect:
			fmt.Println("damage", e.Damage.Amount(), e.Damage.Type())
		case combat.GearSpeedZeroEffect:
			fmt.Println("speed 0 until the start of its next turn")
		}
	}
	// Output:
	// damage 1 Piercing
	// speed 0 until the start of its next turn
}
