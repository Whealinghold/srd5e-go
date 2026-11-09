package character

// barbarianLevelOf returns the Barbarian level of c and false when it has no Barbarian
// levels. It reads ClassLevels alone, so a stored BarbarianState never makes a non-Barbarian
// a Barbarian.
func barbarianLevelOf(c Character) (BarbarianLevel, bool) {
	n := c.ClassLevels().LevelIn(BarbarianClassID)
	if n < 1 {
		return BarbarianLevel{}, false
	}
	return BarbarianLevel{n: n}, true
}

// readBarbarianState returns the BarbarianState c stores, or the initial state when none is
// stored. It returns an error matching srderr.ErrInvalidInput when another type of state sits
// under the Barbarian key.
func readBarbarianState(c Character) (BarbarianState, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), BarbarianStateKey())
	if err != nil || !ok {
		return InitialBarbarianState(), err
	}
	return s, nil
}

// barbarianCommandProblem is the validation shared by the Barbarian use cases: it returns an
// error matching srderr.ErrInvalidInput for the zero use case or a zero Character.
func barbarianCommandProblem(set bool, c Character) error {
	if !set || c.IsZero() {
		return invalidInput("Barbarian use case: zero use case or character")
	}
	return nil
}

// barbarianContext is what every Barbarian use case derives from its command.
type barbarianContext struct {
	stats BarbarianLevelStats
	state BarbarianState
}

// contextOf derives the Barbarian context of c. It returns a non-zero reason, and no error,
// when c has no Barbarian levels (BarbarianNotABarbarian), decided before any roll or lookup.
// It returns an error matching srderr.ErrInvalidInput when another type of state sits under
// the Barbarian key.
func (p BarbarianProgression) contextOf(c Character) (barbarianContext, BarbarianReason, error) {
	level, ok := barbarianLevelOf(c)
	if !ok {
		return barbarianContext{}, BarbarianNotABarbarian, nil
	}
	state, err := readBarbarianState(c)
	if err != nil {
		return barbarianContext{}, 0, err
	}
	return barbarianContext{stats: p.StatsAt(level), state: state}, 0, nil
}
