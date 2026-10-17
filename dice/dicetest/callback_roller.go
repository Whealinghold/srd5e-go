package dicetest

import "github.com/Whealinghold/srd5e-go/dice"

var _ dice.Roller = CallbackRoller(nil)

// CallbackRoller adapts a function to dice.Roller, like http.HandlerFunc. Use
// it to script failures, bad answers or panics.
type CallbackRoller func(dice.Expression) (dice.Result, error)

// Roll calls the function.
func (c CallbackRoller) Roll(e dice.Expression) (dice.Result, error) { return c(e) }
