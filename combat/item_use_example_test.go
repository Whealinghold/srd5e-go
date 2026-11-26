package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A thrown flask of Acid: the target fails its Dexterity saving throw against DC 13 and takes
// 2d6 Acid damage, which the host passes to ApplyDamage.
func ExampleResolveThrownItem() {
	cmd := combat.MustResolveThrownItemCommand(combat.ResolveThrownItemInput{
		Kind: combat.ThrownAcid, Thrower: combat.MustCombatantID("rogue"),
		DexterityModifier: ability.MustScore(16).Modifier(), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)),
		Target:       combat.ThrownAtCreature{ID: combat.MustCombatantID("ogre"), Size: content.SizeLarge, Type: content.CreatureTypeGiant},
		DistanceFeet: 20, Sight: combat.ThrownSightSeen, Cover: combat.ThrownCoverTargetable, Attempt: d20.RollAttempt{},
	})
	throw, _ := combat.NewResolveThrownItem(dicetest.NewFixedRoller(9, 3, 4)) // save 9, then 2d6 = 3 + 4
	res, _ := throw.Run(context.Background(), cmd)
	thrown := res.(combat.ThrownItemResolved)
	fmt.Println(thrown.Save().(combat.ThrownSaveRolled).DifficultyClass(), thrown.Effect().(combat.ThrownDamageDealt).Damage().Amount())
	// Output: 13 7
}

// A Potion of Healing drunk by a creature at 5 of 20 Hit Points: 2d4 + 2 is rolled once.
func ExampleUsePotionOfHealing() {
	hp, _ := combat.RestoreHitPointStatus(combat.MustHitPoints(5, 20, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
	hero := combat.MustCombatantID("hero")
	drink, _ := combat.NewUsePotionOfHealing(dicetest.NewFixedRoller(2, 3)) // 2 + 3 + 2
	res, _ := drink.Run(context.Background(), combat.MustUsePotionOfHealingCommand(hero, hero, hp, combat.DrinkPotion{}))
	used := res.(combat.PotionOfHealingUsed)
	fmt.Println(used.Amount(), used.Status().HitPoints().Current())
	// Output: 7 12
}
