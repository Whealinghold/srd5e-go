package character_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

// A Wish that does more than duplicate a spell weakens the caster: Strength becomes 3 for the rolled days, and a day of
// rest with light activity takes two days off.
func ExampleWishAfterDay() {
	stress := character.IncurWishStress(character.WishStress{}, must(character.NewWishStressRoll(5, 50)))
	scores := must(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(14), ability.Dexterity: ability.MustScore(10), ability.Constitution: ability.MustScore(10),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(10),
	}))
	fmt.Println("Strength while weakened:", character.WishEffectiveScores(scores, stress).Score(ability.Strength))
	next := character.WishAfterDay(character.MustIdentifier("wizard"), stress, character.WishRestingDayWithLightActivity).Stress()
	fmt.Println("days remaining:", next.Weakness().(character.WishWeakened).DaysRemaining())
	// Output:
	// Strength while weakened: 3
	// days remaining: 3
}
