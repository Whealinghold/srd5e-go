package character_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

// Two classes that each give two attacks give two, not four: Extra Attack does
// not stack (SRD 5.2.1, "Multiclassing", p. 25).
func ExampleExtraAttackCombination() {
	n, err := character.ExtraAttackCombination([]int{2, 2})
	fmt.Println(n, err)
	// Output: 2 <nil>
}

// An Ability Score Improvement raises two abilities by one, to at most 20 (SRD
// 5.2.1, "Ability Score Improvement", p. 87).
func ExampleNewAbilityScoreImprovementTwoByOne() {
	inc, err := character.NewAbilityScoreImprovementTwoByOne(ability.Strength, ability.Dexterity)
	fmt.Println(inc.Increases()[ability.Strength], inc.Increases()[ability.Dexterity], inc.MaximumScore(), err)
	// Output: 1 1 20 <nil>
}
