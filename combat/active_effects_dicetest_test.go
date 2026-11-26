package combat_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// dicetestFixed is a FixedRoller of faces, in order.
func dicetestFixed(faces ...int) *dicetest.FixedRoller { return dicetest.NewFixedRoller(faces...) }

// aeBothModes is one source of Advantage and one of Disadvantage on the roll.
func aeBothModes() []modifier.RollCircumstance {
	return []modifier.RollCircumstance{modifier.Advantage{}, modifier.Disadvantage{}}
}

func sprint(v any) string { return fmt.Sprintf("%T", v) }
