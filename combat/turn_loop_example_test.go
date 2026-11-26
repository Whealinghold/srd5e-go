package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// exampleTurnLoopEncounter starts an encounter of two combatants with 12 Hit Points each, the hero acting first. It
// takes no dice: the turn loop of the examples below rolls none.
func exampleTurnLoopEncounter() (combat.Encounter, combat.CombatantID, combat.CombatantID) {
	ctx := context.Background()
	hero, orc := combat.MustCombatantID("hero"), combat.MustCombatantID("orc")
	walk, _ := combat.NewMovementSpeeds(30) // SRD 5.2.1, "Speed", p. 188
	est, _ := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: hero, Count: 18}, {Combatant: orc, Count: 10}},
		[]combat.CombatantProfile{{ID: hero, Size: content.SizeMedium, Speeds: walk}, {ID: orc, Size: content.SizeMedium, Speeds: walk}}, nil)
	order, _ := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	member := func(id combat.CombatantID, side combat.Side) combat.Combatant {
		hp, _ := combat.RestoreHitPointStatus(combat.MustHitPoints(12, 12, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
		c, _ := combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: "fixture"}}, side, hp, combat.ConditionSet{})
		return c
	}
	started, _ := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("example-turns"),
		order.(combat.InitiativeOrderEstablished), []combat.Combatant{member(hero, combat.SideA), member(orc, combat.SideB)}))
	return started.(combat.StartEncounterApplied).Encounter(), hero, orc
}

// ExampleTakeCombatantAction has the hero take the Dodge action (SRD 5.2.1, "Dodge", p. 190) and then try to take a
// second action in the same turn, which the turn tracker refuses. The result of each call is the next call's input.
func ExampleTakeCombatantAction() {
	encounter, hero, _ := exampleTurnLoopEncounter()
	var ledger combat.ActionLedger
	dodge := combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}}
	for range 2 {
		cmd, _ := combat.NewTakeCombatantActionCommand(encounter, ledger, hero, dodge)
		res, _ := combat.NewTakeCombatantAction().Run(context.Background(), cmd)
		switch r := res.(type) {
		case combat.TakeCombatantActionCompleted:
			encounter, ledger = r.Encounter(), r.Ledger()
			fmt.Println("completed:", kindsOf(r.Events()))
		case combat.TakeCombatantActionRejected:
			fmt.Println("refused:", r.Rejection())
		case combat.TakeCombatantActionInvalid:
			fmt.Println("invalid:", r.Problem())
		}
	}
	// Output:
	// completed: [action_taken]
	// refused: turn_refused
}

// ExampleAdvanceCombatTurn ends the hero's turn and starts the orc's: the result names the events of the turn change and
// the encounter it returns has the orc active.
func ExampleAdvanceCombatTurn() {
	encounter, _, _ := exampleTurnLoopEncounter()
	cmd, _ := combat.NewAdvanceCombatTurnCommand(encounter, combat.ActionLedger{}, nil, nil)
	res, _ := combat.NewAdvanceCombatTurn().Run(context.Background(), cmd)
	switch r := res.(type) {
	case combat.AdvanceCombatTurnCompleted:
		fmt.Println("completed:", kindsOf(r.Events()))
		fmt.Println("active:", r.Encounter().Tracker().Active().Combatant)
	case combat.AdvanceCombatTurnRejected:
		fmt.Println("refused:", r.Rejection())
	case combat.AdvanceCombatTurnInvalid:
		fmt.Println("invalid:", r.Problem())
	}
	// Output:
	// completed: [tracker_turn_ended tracker_turn_started]
	// active: orc
}
