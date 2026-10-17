package randroller

import (
	"math/rand/v2"

	"github.com/Whealinghold/srd5e-go/dice"
)

var _ dice.Roller = Roller{}

// Roller is an unseeded dice.Roller. The zero value is ready to use. Safe for
// concurrent use: it keeps no state and calls only the goroutine-safe
// top-level functions of math/rand/v2.
type Roller struct{}

// New returns a Roller. It is equivalent to the zero value.
func New() Roller { return Roller{} }

// Roll rolls each die of e uniformly from 1..Sides().
//
// A zero e returns the zero Result and the *dice.InvalidError that dice.NewResult
// reports (dice.Roll wraps it as a *srderr.PortError). Any valid
// expression succeeds.
func (Roller) Roll(e dice.Expression) (dice.Result, error) {
	die := e.Die()
	rolls := make([]dice.DieRoll, e.Count())
	for i := range rolls {
		rolls[i] = dice.MustDieRoll(die, rand.IntN(die.Sides())+1) //nolint:gosec // game dice need no cryptographic randomness
	}
	return dice.NewResult(e, rolls)
}
