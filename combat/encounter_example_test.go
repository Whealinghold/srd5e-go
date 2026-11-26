package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Start an encounter from an established turn order, then advance one turn.
func ExampleEncounter_Start() {
	a, b := combat.MustCombatantID("knight"), combat.MustCombatantID("goblin")
	speed, _ := combat.NewMovementSpeeds(30)
	cmd, _ := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: a, Count: 15}, {Combatant: b, Count: 9}},
		[]combat.CombatantProfile{{ID: a, Size: content.SizeMedium, Speeds: speed}, {ID: b, Size: content.SizeSmall, Speeds: speed}}, nil)
	res, _ := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	member := func(id combat.CombatantID, side combat.Side) combat.Combatant {
		hp := combat.MustHitPointStatus(10, combat.FallsUnconscious, combat.NoAffinities())
		c, _ := combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: "example"}}, side, hp, combat.ConditionSet{})
		return c
	}
	enc, events, err := (combat.Encounter{}).Start(combat.MustEncounterID("example"), res.(combat.InitiativeOrderEstablished),
		[]combat.Combatant{member(b, combat.SideB), member(a, combat.SideA)})
	fmt.Println(err, len(enc.Combatants()), events[0].Kind())

	next, _ := enc.AdvanceTurn(combat.TurnAdvance{})
	for _, e := range next.Events() {
		fmt.Println(e.Kind())
	}
	// Output:
	// <nil> 2 encounter_started
	// tracker_turn_ended
	// tracker_turn_started
}

// Apply damage to a combatant through its use case: the result carries the new encounter and the events.
func ExampleApplyDamageToCombatant() {
	a, b := combat.MustCombatantID("knight"), combat.MustCombatantID("goblin")
	speed, _ := combat.NewMovementSpeeds(30)
	cmd, _ := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: a, Count: 15}, {Combatant: b, Count: 9}},
		[]combat.CombatantProfile{{ID: a, Size: content.SizeMedium, Speeds: speed}, {ID: b, Size: content.SizeSmall, Speeds: speed}}, nil)
	res, _ := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	member := func(id combat.CombatantID, side combat.Side, rule combat.ZeroHitPointsRule) combat.Combatant {
		hp := combat.MustHitPointStatus(6, rule, combat.NoAffinities())
		c, _ := combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: "example"}}, side, hp, combat.ConditionSet{})
		return c
	}
	enc, _, _ := (combat.Encounter{}).Start(combat.MustEncounterID("example"), res.(combat.InitiativeOrderEstablished),
		[]combat.Combatant{member(a, combat.SideA, combat.FallsUnconscious), member(b, combat.SideB, combat.DiesInstantly)})
	dmg, _ := combat.NewApplyDamageToCombatantCommand(enc, b, []combat.Damage{combat.MustDamage(8, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
	out, _ := combat.NewApplyDamageToCombatant().Run(context.Background(), dmg)
	applied := out.(combat.ApplyDamageToCombatantApplied)
	for _, e := range applied.Events() {
		fmt.Println(e.Kind())
	}
	// Output:
	// damage_taken
	// dropped_to_zero_hit_points
	// died
}
