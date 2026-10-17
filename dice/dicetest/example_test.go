package dicetest_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A scripted roller serves the faces you choose, in order, and records what it
// was asked to roll.
func ExampleNewFixedRoller() {
	roller := dicetest.NewFixedRoller(3, 6, 1)
	res, err := dice.Roll(roller, dice.MustParseExpression("3d6"))
	if err != nil {
		fmt.Println(err)
		return
	}
	left, _ := roller.Remaining()
	fmt.Println(res.Total(), roller.Requests(), left)

	// The script is spent: the next request fails rather than inventing a face.
	_, err = dice.Roll(roller, dice.MustParseExpression("1d6"))
	fmt.Println(err != nil)
	// Output:
	// 10 [3d6] 0
	// true
}
