package character_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A Long Rest that is interrupted by damage after 3 hours still confers the
// benefits of a Short Rest.
func ExampleRestTaker_TakeRest() {
	report := character.MustRestReport(character.LongRest, character.StandardLongRest,
		character.NewRestStopped(character.TakingDamage, 3, 0))
	cmd := character.MustTakeRestCommand(character.MustIdentifier("hero"), report, 12, 0, nil, character.SpentHitDice{})
	res := character.NewRestTaker().TakeRest(context.Background(), cmd)
	applied := res.(character.TakeRestAppliedAfterInterruption).Applied()
	fmt.Println(applied.BenefitsOf(), applied.Source() == character.LongRestInterrupted, len(applied.Effects()))
	// Output: short true 2
}

// Spending a d8 that rolls 4 with a Constitution modifier of +2 regains 6 Hit Points.
func ExampleHitDieSpender_SpendHitPointDie() {
	spender, _ := character.NewHitDieSpender(dicetest.NewFixedRoller(4))
	pool, _ := character.NewHitDicePool(map[content.HitPointDie]int{content.HitDieD8: 3})
	cmd, _ := character.NewSpendHitPointDieCommand(character.MustIdentifier("hero"), pool, nil, content.HitDieD8,
		ability.MustScore(14).Modifier(), character.ShortRest)
	res, _ := spender.SpendHitPointDie(context.Background(), cmd)
	fmt.Println(res.(character.SpendHitPointDieApplied).HitPointsRegained())
	// Output: 6
}
