package character_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A character who has Heroic Inspiration expends it to reroll the second of two
// d20s; the new roll must be used.
func ExampleExpendHeroicInspiration() {
	hero := pcExample()
	use, _ := character.NewExpendHeroicInspiration(dicetest.NewFixedRoller(17))
	res, _ := use.Expend(context.Background(), character.ExpendHeroicInspirationCommand{
		Character: hero,
		Target:    character.MustRerollTarget(dice.D20(), 2, 1),
	})
	r := res.(character.ExpendHeroicInspirationRerolled)
	fmt.Println(r.Rerolled().Position(), r.Rerolled().NewValue(), r.Character().HeroicInspiration())
	// Output: 1 17 notInspired
}

// A holder who already has Heroic Inspiration loses the second one.
func ExampleResolveExcessInspiration() {
	res := character.ResolveExcessInspiration(pcExample(), character.LoseExcessInspiration{})
	_, lost := res.(character.ExcessResolutionExcessLost)
	fmt.Println(lost)
	// Output: true
}

// pcExample is a level 1 Fighter who has Heroic Inspiration.
func pcExample() character.Character {
	ten := ability.MustScore(10)
	scores, _ := ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ten, ability.Dexterity: ten, ability.Constitution: ten,
		ability.Intelligence: ten, ability.Wisdom: ten, ability.Charisma: ten,
	})
	created, _ := character.New(character.NewParams{
		ID: character.MustIdentifier("pc-1"), Class: content.MustClassID("fighter"), HitPointDie: content.HitDieD10,
		Species: content.MustSpeciesID("human"), Background: content.MustBackgroundID("soldier"), AbilityScores: scores,
	})
	return created.Character().GrantHeroicInspiration().Character()
}
