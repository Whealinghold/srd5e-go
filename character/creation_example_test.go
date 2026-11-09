package character_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Assigning the Standard Array: a permutation is accepted, a repeated score is not.
func ExampleNewAbilityScoreAssignment() {
	scores := func(str int) map[ability.Ability]int {
		return map[ability.Ability]int{ability.Strength: str, ability.Dexterity: 14, ability.Constitution: 13, ability.Intelligence: 12, ability.Wisdom: 10, ability.Charisma: 8}
	}
	_, err := character.NewAbilityScoreAssignment(character.StandardArray, scores(15))
	fmt.Println("permutation:", err)
	_, err = character.NewAbilityScoreAssignment(character.StandardArray, scores(14))
	fmt.Println("repeated score is NotStandardArray:", err != nil)
	// Output:
	// permutation: <nil>
	// repeated score is NotStandardArray: true
}

// Rolling two languages: the d12 rolls 3 and 5 give Dwarvish and Elvish.
func ExampleRollLanguages() {
	roller := dicetest.NewFixedRoller(3, 5)
	uc, _ := character.NewRollLanguages(roller)
	res, _ := uc.Run(context.Background(), character.RollLanguagesCommand{})
	rolled := res.(character.RollLanguagesSucceeded)
	fmt.Println(rolled.Languages(), rolled.Rolls())
	// Output: [Dwarvish Elvish] [3 5]
}

// Rolling the six Random Generation scores with the maximum roller: every total is 18.
func ExampleRollAbilityScores() {
	roller := dicetest.NewMaximumRoller()
	uc, _ := character.NewRollAbilityScores(roller)
	res, _ := uc.Run(context.Background(), character.RollAbilityScoresCommand{})
	fmt.Println(res.(character.RollAbilityScoresSucceeded).Scores().Totals())
	// Output: [18 18 18 18 18 18]
}
