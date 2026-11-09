package character

import "github.com/Whealinghold/srd5e-go/dice"

// rollDiceValues rolls count dice of the given size through roller and returns the faces in order. A
// roller failure, panic or broken contract comes back as the *srderr.PortError dice.Roll builds.
func rollDiceValues(roller Roller, count int, die dice.Die) ([]int, error) {
	res, err := dice.Roll(roller, dice.MustExpression(count, die, 0))
	if err != nil {
		return nil, err
	}
	faces := make([]int, 0, count)
	for _, r := range res.Rolls() {
		faces = append(faces, r.Value())
	}
	return faces, nil
}
