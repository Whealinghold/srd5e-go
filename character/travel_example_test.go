package character_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A group travelling at a Slow pace has Advantage on Perception checks.
func ExampleTravelRules_Circumstances() {
	cs := character.TravelRules{}.Circumstances(character.LandTravel{Pace: character.PaceSlow}, content.SkillPerception)
	fmt.Println(modifier.Resolve(cs...))
	// Output: advantage
}
