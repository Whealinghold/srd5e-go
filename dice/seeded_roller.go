package dice

import (
	"fmt"
	"math/rand/v2"

	"github.com/Whealinghold/srd5e-go/srderr"
)

var _ Roller = (*SeededRoller)(nil)

// SeededRoller is a Roller backed by a caller-supplied pseudo-random
// generator, so a fixed seed gives a reproducible game.
//
// It is NOT safe for concurrent use: *rand.Rand is not. Give each goroutine
// its own SeededRoller or guard calls with a mutex.
type SeededRoller struct{ rng *rand.Rand }

// NewSeededRoller returns a roller drawing from rng. The roller never creates
// a generator itself. A nil rng is not rejected here (the constructor stays infallible);
// Roll then fails with a *srderr.PortError wrapping srderr.ErrInvalidInput.
func NewSeededRoller(rng *rand.Rand) *SeededRoller { return &SeededRoller{rng: rng} }

// Roll rolls each die of e uniformly from 1..Sides(), drawing in order.
//
// A nil receiver or nil generator returns a *srderr.PortError (Port)
// wrapping srderr.ErrInvalidInput. A panicking generator is recovered into a
// *srderr.PortError matching srderr.ErrPortPanic. The cause is reachable via
// errors.Is; the result is then the zero Result. A zero e fails the same way
// with srderr.ErrInvalidInput as the cause.
func (s *SeededRoller) Roll(e Expression) (Result, error) {
	if s == nil || s.rng == nil {
		return Result{}, srderr.WrapPort(Port, "roll", fmt.Errorf("seeded roller has no generator: %w", srderr.ErrInvalidInput))
	}
	return srderr.GuardValue(Port, "roll", func() (Result, error) {
		if e.IsZero() {
			return Result{}, invalid(ReasonCountOutOfRange, "zero expression")
		}
		rolls := make([]DieRoll, e.count)
		for i := range rolls {
			rolls[i] = DieRoll{e.die, s.rng.IntN(e.die.sides) + 1}
		}
		return NewResult(e, rolls)
	})
}
