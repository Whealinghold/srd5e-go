package character

import "github.com/Whealinghold/srd5e-go/dice"

// monkLevelOf returns the Monk level of c and false when it has no Monk levels. It
// reads ClassLevels alone, so a stored MonkState never makes a non-Monk a Monk.
func monkLevelOf(c Character) (MonkLevel, bool) {
	n := c.ClassLevels().LevelIn(MonkClassID)
	if n < 1 {
		return MonkLevel{}, false
	}
	return MonkLevel{n: n}, true
}

// readMonkState returns the MonkState c stores, or the initial state when none is
// stored. It returns an error matching srderr.ErrInvalidInput when another type of
// state sits under the Monk key.
func readMonkState(c Character) (MonkState, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), MonkStateKey())
	if err != nil || !ok {
		return InitialMonkState(), err
	}
	return s, nil
}

// monkCommandProblem is the validation shared by the Monk use cases: it returns an
// error matching srderr.ErrInvalidInput for the zero use case or a zero Character.
func monkCommandProblem(set bool, c Character) error {
	if !set || c.IsZero() {
		return invalidInput("Monk use case: zero use case or character")
	}
	return nil
}

// monkContext is what every Monk use case derives from its command.
type monkContext struct {
	stats MonkLevelStats
	state MonkState
}

// contextOf derives the Monk context of c. A character with no Monk levels has the zero stats, which
// every Availability answers with MonkNotAMonk before any roll. It returns an error matching
// srderr.ErrInvalidInput when another type of state sits under the Monk key.
func (p MonkProgression) contextOf(c Character) (monkContext, error) {
	level, ok := monkLevelOf(c)
	if !ok {
		return monkContext{}, nil
	}
	state, err := readMonkState(c)
	if err != nil {
		return monkContext{}, err
	}
	return monkContext{stats: p.StatsAt(level), state: state}, nil
}

// monkRolledWithin reports whether total is a possible sum of count dice of the given sides.
func monkRolledWithin(total, count int, die dice.Die) bool {
	return total >= count && total <= count*die.Sides()
}

// rollTotal rolls the expression through roller and returns the total. A roller
// failure, panic or broken contract comes back as the *srderr.PortError dice.Roll builds.
func rollTotal(roller Roller, expr dice.Expression) (int, error) {
	res, err := dice.Roll(roller, expr)
	if err != nil {
		return 0, err
	}
	return res.Total(), nil
}

// rollMartialArts rolls count Martial Arts dice of stats' level through roller and returns their sum.
func rollMartialArts(roller Roller, stats MonkLevelStats, count int) (int, error) {
	return rollTotal(roller, dice.MustExpression(count, stats.martialArts, 0))
}

// deflectDie is the 1d10 Deflect Attacks rolls.
func deflectDie() dice.Expression { return dice.MustExpression(1, dice.D10(), 0) }
