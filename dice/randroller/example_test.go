package randroller_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/randroller"
)

func ExampleRoller() {
	e := dice.MustParseExpression("2d8+2")
	res, err := dice.Roll(randroller.New(), e)
	fmt.Println(err, res.Total() >= e.Minimum() && res.Total() <= e.Maximum())
	// Output: <nil> true
}
