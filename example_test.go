package srd5e_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Import the package that owns each type: here a saving throw is resolved with
// d20, ability and modifier, and a scripted roller from dicetest stands in for
// real randomness.
func Example() {
	resolver, err := d20.NewResolver(dicetest.NewFixedRoller(14))
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := resolver.Resolve(context.Background(), d20.SavingThrowCommand{
		Ability: ability.Constitution,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(2))},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res.Kind(), res.Outcome())
	// Output: saving_throw success
}
