package character_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Proficiency Bonus comes from the total level, whatever the classes.
func ExampleProficiencyBonusFor() {
	levels := character.MustClassLevels([]character.ClassLevel{
		{Class: content.MustClassID("fighter"), Level: 3},
		{Class: content.MustClassID("rogue"), Level: 2},
	})
	fmt.Println(character.ProficiencyBonusFor(levels.TotalLevel()).Value())
	// Output: 3
}

// A creature is Bloodied at half its Hit Point maximum or fewer.
func ExampleIsBloodied() {
	maximum := character.MustHitPointMaximum(15)
	for _, current := range []int{7, 8} {
		bloodied, _ := character.IsBloodied(current, maximum)
		fmt.Println(current, bloodied)
	}
	// Output:
	// 7 true
	// 8 false
}
