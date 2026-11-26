package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
)

// A wizard casts Giant Insect at level 5 during combat. The cast reads no content, returns the numbers of the wasp from the wizard's
// spell facts, and tells the host to join it directly after the wizard.
func ExampleSummonCreature() {
	uc, _ := combat.NewSummonCreature(contenttest.NewFakeSource())
	facts, _ := combat.NewSummonerSpellFacts(7, 15, 4, 3)
	request := combat.GiantInsectRequest{Creature: "wasp-1", Level: 5, Form: combat.InsectWasp}
	cmd, _ := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "wizard", combat.SideA, request, facts, combat.SummonScene{InCombat: true})
	res, _ := uc.Run(context.Background(), cmd)
	applied := res.(combat.SummonCreatureApplied)
	block := applied.Creatures()[0].StatBlock()
	fmt.Println(block.ArmorClass(), block.HitPointMaximum(), block.MultiattackCount())
	fmt.Println(applied.RosterActions()[0].(combat.SummonedRosterJoin).Position)
	// Output:
	// 16 40 2
	// {wizard}
}

// The wizard's wasp drops to 0 Hit Points on its own turn, so its departure is pending: the host ends the turn, then removes it.
func ExampleApplySummonedChange() {
	facts, _ := combat.NewSummonerSpellFacts(7, 15, 4, 3)
	request := combat.GiantInsectRequest{Creature: "wasp-1", Level: 5, Form: combat.InsectWasp}
	made, _ := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "wizard", combat.SideA, request, facts, combat.SummonScene{})
	roster := (combat.SummonedRoster{}).Summon(made).(combat.SummonedApplied).Roster()

	cmd, _ := combat.NewApplySummonedChangeCommand(roster, combat.ReportZeroHitPoints{Creature: "wasp-1"}, "wasp-1")
	res, _ := combat.NewApplySummonedChange().Run(context.Background(), cmd)
	applied := res.(combat.ApplySummonedChangeApplied)
	leave := applied.RosterActions()[0].(combat.SummonedRosterLeave)
	fmt.Println(leave.Cause, leave.AfterTurn, applied.Roster().Len())
	// Output: disappeared true 0
}
