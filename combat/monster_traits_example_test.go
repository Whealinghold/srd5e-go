package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// ExampleRecogniseTraits recognises Flyby in a synthetic stat block and checks it against a
// flying and a walking departure.
func ExampleRecogniseTraits() {
	flyby := luMust(content.NewStatBlockPart("Flyby", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: mtFlyby}))
	monster := luMonster(new(testing.T), func(f *content.MonsterFields) { f.Traits = []content.StatBlockPart{flyby} })
	trait := combat.RecogniseTraits(monster).Traits()[0].(combat.FlybyTrait)
	fmt.Println(combat.FlybyExemptsFromOpportunityAttack(trait, combat.MovementModeFly))
	fmt.Println(combat.FlybyExemptsFromOpportunityAttack(trait, combat.MovementModeWalk))
	// Output:
	// true
	// false
}
