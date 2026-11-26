package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Add a summoned creature directly after its summoner, then remove it again when it disappears.
func ExampleAddCombatantToEncounter() {
	knight, goblin, wolf := combat.MustCombatantID("knight"), combat.MustCombatantID("goblin"), combat.MustCombatantID("wolf")
	speed, _ := combat.NewMovementSpeeds(30)
	cmd, _ := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: knight, Count: 15}, {Combatant: goblin, Count: 9}},
		[]combat.CombatantProfile{{ID: knight, Size: content.SizeMedium, Speeds: speed}, {ID: goblin, Size: content.SizeSmall, Speeds: speed}}, nil)
	res, _ := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	member := func(id combat.CombatantID, side combat.Side) combat.Combatant {
		hp := combat.MustHitPointStatus(10, combat.FallsUnconscious, combat.NoAffinities())
		c, _ := combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: "example"}}, side, hp, combat.ConditionSet{})
		return c
	}
	enc, _, _ := (combat.Encounter{}).Start(combat.MustEncounterID("example"), res.(combat.InitiativeOrderEstablished),
		[]combat.Combatant{member(knight, combat.SideA), member(goblin, combat.SideB)})

	add, _ := combat.NewAddCombatantToEncounterCommand(enc, member(wolf, combat.SideA),
		combat.CombatantProfile{ID: wolf, Size: content.SizeMedium, Speeds: speed}, combat.RosterAfterCombatant{Anchor: knight}, combat.JoinReasonSummoned)
	added, _ := combat.NewAddCombatantToEncounter().Run(context.Background(), add)
	joined := added.(combat.AddCombatantToEncounterApplied)
	for _, c := range joined.Encounter().Combatants() {
		fmt.Println(c.Ref().ID)
	}
	fmt.Println(joined.Events()[0].Kind())

	// The wolf is not the active combatant, so the host can remove it.
	remove, _ := combat.NewRemoveCombatantFromEncounterCommand(joined.Encounter(), combat.ActionLedger{}, wolf, combat.RemovalCauseDisappeared)
	removed, _ := combat.NewRemoveCombatantFromEncounter().Run(context.Background(), remove)
	fmt.Println(len(removed.(combat.RemoveCombatantFromEncounterApplied).Encounter().Combatants()))
	// Output:
	// knight
	// wolf
	// goblin
	// combatant_joined
	// 2
}
