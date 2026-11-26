package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ExamplePerformAttack runs one round of two combatants through the use cases, each result's encounter and ledger
// being the next call's input: the first attacks and hits, the turn advances, the second attacks and misses. The
// dice are a fixed script and every game number follows SRD 5.2.1.
func ExamplePerformAttack() {
	ctx := context.Background()
	hero, orc := combat.MustCombatantID("hero"), combat.MustCombatantID("orc")
	walk, _ := combat.NewMovementSpeeds(30) // SRD 5.2.1, "Speed", p. 188
	est, _ := combat.NewEstablishInitiativeOrderCommand(
		[]combat.InitiativeStanding{{Combatant: hero, Count: 18}, {Combatant: orc, Count: 10}},
		[]combat.CombatantProfile{{ID: hero, Size: content.SizeMedium, Speeds: walk}, {ID: orc, Size: content.SizeMedium, Speeds: walk}}, nil)
	order, _ := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	member := func(id combat.CombatantID, side combat.Side, hitPoints int) combat.Combatant {
		hp, _ := combat.RestoreHitPointStatus(combat.MustHitPoints(hitPoints, hitPoints, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
		c, _ := combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: "fixture"}}, side, hp, combat.ConditionSet{})
		return c
	}
	started, _ := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID("example"), order.(combat.InitiativeOrderEstablished),
		[]combat.Combatant{member(hero, combat.SideA, 12), member(orc, combat.SideB, 15)}))
	encounter := started.(combat.StartEncounterApplied).Encounter()
	var ledger combat.ActionLedger

	// The attack: +5 against Armor Class 13 for 1d8 plus 3 Slashing (SRD 5.2.1, "Making an Attack", pp. 14-15).
	swing := func(attacker, target combat.CombatantID, roller combat.Roller) {
		attack := combat.MustResolveAttackCommand(combat.ResolveAttackParams{Form: combat.MeleeWeaponAttack, ArmorClass: 13, Distance: 5,
			Modifiers: d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)),
			Damage:    combat.MustDamageSpec(combat.MustDiceDamage(1, dice.MustDie(8), content.DamageSlashing, 0, true))})
		scene, _ := combat.NewAttackScene(5, combat.SightRelationCanSee, combat.SightRelationCanSee, combat.LineOfSightBlocked)
		cmd, _ := combat.NewPerformAttackCommand(combat.PerformAttackParams{Encounter: encounter, Ledger: ledger, AttackerID: attacker, TargetID: target,
			Cost: combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansOtherWeapon}, Scene: scene, Attack: attack})
		uc, _ := combat.NewPerformAttack(roller)
		res, _ := uc.Run(ctx, cmd)
		switch r := res.(type) {
		case combat.PerformAttackCompleted:
			encounter, ledger = r.Encounter(), r.Ledger()
			fmt.Println(attacker, "attacks:", kindsOf(r.Events()))
		case combat.PerformAttackRejected:
			fmt.Println(attacker, "refused:", r.Rejection())
		case combat.PerformAttackInvalid:
			fmt.Println(attacker, "invalid:", r.Problem())
		}
	}
	swing(hero, orc, dicetest.NewFixedRoller(14, 4))
	cmd, _ := combat.NewAdvanceCombatTurnCommand(encounter, ledger, nil, nil)
	advanced, _ := combat.NewAdvanceCombatTurn().Run(ctx, cmd)
	done := advanced.(combat.AdvanceCombatTurnCompleted)
	encounter, ledger = done.Encounter(), done.Ledger()
	fmt.Println("turn:", kindsOf(done.Events()))
	swing(orc, hero, dicetest.NewFixedRoller(3))
	for _, id := range []combat.CombatantID{hero, orc} {
		c, _ := encounter.Find(id)
		fmt.Println(id, "has", c.HitPoints().HitPoints().Current(), "Hit Points and conditions", c.Conditions().Effective())
	}
	// Output:
	// hero attacks: [action_taken attack_made_in_attack_action attack_performed damage_taken]
	// turn: [tracker_turn_ended tracker_turn_started]
	// orc attacks: [action_taken attack_made_in_attack_action attack_performed]
	// hero has 12 Hit Points and conditions []
	// orc has 8 Hit Points and conditions []
}

func kindsOf(events []combat.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}
