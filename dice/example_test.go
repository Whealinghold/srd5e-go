package dice_test

import (
	"fmt"
	"math/rand/v2"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

func ExampleParseExpression() {
	e, err := dice.ParseExpression("2d8 + 2")
	fmt.Println(e, err, e.Minimum(), e.Maximum(), e.Average())

	_, err = dice.ParseExpression("2d")
	fmt.Println(err)
	// Output:
	// 2d8+2 <nil> 4 18 11
	// dice: malformed
}

func ExampleRoll() {
	roller := dicetest.NewFixedRoller(3, 6, 1, 4)
	res, err := dice.Roll(roller, dice.MustParseExpression("4d6"))
	best, _ := res.SumOfHighest(3)
	fmt.Println(res, err, res.Total(), best)
	// Output: 4d6 [d6=3 d6=6 d6=1 d6=4] <nil> 14 13
}

func ExampleNewSeededRoller() {
	roller := dice.NewSeededRoller(rand.New(rand.NewPCG(1, 2)))
	res, err := dice.Roll(roller, dice.MustParseExpression("1d20"))
	fmt.Println(err, res.Total())
	// Output: <nil> 16
}
