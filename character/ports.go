package character

import "github.com/Whealinghold/srd5e-go/dice"

// Roller is the dice port the character use cases need: one method. Any
// dice.Roller satisfies it, as does dicetest.FixedRoller. A roller shared across
// goroutines MUST be safe for concurrent use.
type Roller interface {
	Roll(dice.Expression) (dice.Result, error)
}
