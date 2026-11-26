package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ExampleApplyConditionToSet adds Grappled to an empty set and then Unconscious, and prints the conditions the set
// holds (Unconscious implies Incapacitated and Prone, SRD 5.2.1, "Unconscious", p. 191). A repeat is refused.
func ExampleApplyConditionToSet() {
	ctx := context.Background()
	hero, orc := combat.MustCombatantID("hero"), combat.MustCombatantID("orc")
	apply := func(set combat.ConditionSet, c combat.Condition) combat.ApplyConditionToSetResult {
		cmd, _ := combat.NewApplyConditionToSetCommand(set, hero, c)
		res, _ := combat.NewApplyConditionToSet().Run(ctx, cmd)
		return res
	}
	res := apply(combat.ConditionSet{}, combat.Grappled{Grappler: orc})
	res = apply(res.Set(), combat.Unconscious{})
	fmt.Println(res.Set().Effective())
	_, again := apply(res.Set(), combat.Unconscious{}).(combat.ConditionAdded)
	fmt.Println("added again:", again)
	// Output:
	// [Grappled Incapacitated Prone Unconscious]
	// added again: false
}

// ExampleRollInitiative rolls one d20 for a hero with a +3 modifier (SRD 5.2.1, "Initiative", p. 13) from a fixed roller.
func ExampleRollInitiative() {
	hero := combat.MustCombatantID("hero")
	entry, _ := combat.NewInitiativeParticipant(combat.InitiativeParticipantInput{IDs: []combat.CombatantID{hero}, Modifier: modifier.New(3)})
	cmd, _ := combat.NewRollInitiativeCommand(combat.InitiativeMethodRolled, []combat.InitiativeParticipant{entry})
	uc, _ := combat.NewRollInitiative(dicetest.NewFixedRoller(14))
	res, _ := uc.Run(context.Background(), cmd)
	for _, b := range res.(combat.RollInitiativeRolled).Breakdowns() {
		fmt.Println(b.IDs(), "count", b.Count())
	}
	// Output:
	// [hero] count 17
}

// ExampleResolveAttack rolls a melee weapon attack of +5 against Armor Class 15 with a fixed d20 of 10: a total of 15 meets
// the Armor Class and hits (SRD 5.2.1, "Making an Attack", pp. 14-15).
func ExampleResolveAttack() {
	cmd, _ := combat.NewResolveAttackCommand(combat.ResolveAttackParams{
		Form: combat.MeleeWeaponAttack, ArmorClass: 15, Distance: 5,
		Modifiers: d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)),
	})
	uc, _ := combat.NewResolveAttack(dicetest.NewFixedRoller(10))
	res, _ := uc.Run(context.Background(), cmd)
	r := res.(combat.AttackResolved)
	fmt.Println("total", r.Attack().Total(), "hit", r.Hit(), "critical", r.CriticalHit())
	// Output:
	// total 15 hit true critical false
}
