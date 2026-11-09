package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// HalflingLuck is the Halfling trait that rerolls a 1 on the d20 of a D20 Test (SRD
// 5.2.1, "Luck", p. 86). The zero value is ready to use. It is stateless and safe for
// concurrent use.
type HalflingLuck struct{}

// EligibleDice returns a new slice of the indexes of the dice of roll that show a 1,
// minus those already rerolled by an effect. The natural value need not be 1: a 1
// under Advantage qualifies even when the other die is kept. The new roll must be
// used, even when lower, and each die is rerolled at most once ("Interactions with
// Rerolls", SRD 5.2.1, p. 8). The per-effect limit of one die is the caller's.
func (HalflingLuck) EligibleDice(roll d20.Roll, rerolled []int) []int {
	out := []int{}
	for i, face := range roll.Dice() {
		if face == 1 && !slices.Contains(rerolled, i) {
			out = append(out, i)
		}
	}
	return out
}
