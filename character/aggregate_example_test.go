package character_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// A level 1 Fighter has the die's top face plus the Constitution modifier as
// Hit Point maximum, and a level is gained only once the XP reaches the next
// minimum.
func ExampleNew() {
	scores, _ := ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(15), ability.Dexterity: ability.MustScore(13), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(12), ability.Charisma: ability.MustScore(8),
	})
	created, err := character.New(character.NewParams{
		ID: character.MustIdentifier("pc-1"), Class: content.MustClassID("fighter"), HitPointDie: content.HitDieD10,
		Species: content.MustSpeciesID("human"), Background: content.MustBackgroundID("soldier"), AbilityScores: scores,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	pc := created.Character()
	fmt.Println(pc.HitPointMaximum().Value(), pc.IsEligibleForNextLevel())
	pc = pc.AwardExperience(character.MustExperienceAmount(300)).Character()
	fmt.Println(pc.ExperiencePoints().Value(), pc.IsEligibleForNextLevel())
	// Output:
	// 12 false
	// 300 true
}
