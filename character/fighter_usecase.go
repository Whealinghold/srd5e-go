package character

import (
	"github.com/Whealinghold/srd5e-go/dice"
)

// fighterLevelOf returns the Fighter level of c and false when it has no Fighter
// levels. It reads ClassLevels alone, so a stored FighterState never makes a
// non-Fighter a Fighter.
func fighterLevelOf(c Character) (FighterLevel, bool) {
	n := c.ClassLevels().LevelIn(FighterClassID)
	if n < 1 {
		return FighterLevel{}, false
	}
	return FighterLevel{n: n}, true
}

// readFighterState returns the FighterState c stores, or the initial state when
// none is stored. It returns an error matching srderr.ErrInvalidInput when another
// type of state sits under the Fighter key.
func readFighterState(c Character) (FighterState, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), FighterStateKey())
	if err != nil || !ok {
		return InitialFighterState(), err
	}
	return s, nil
}

// rollOne rolls one die of the given size through roller and returns the face. A
// roller failure, panic or broken contract comes back as the *srderr.PortError
// dice.Roll builds.
func rollOne(roller Roller, die dice.Die) (int, error) {
	res, err := dice.Roll(roller, dice.MustExpression(1, die, 0))
	if err != nil {
		return 0, err
	}
	return res.Total(), nil
}

// fighterCommandProblem is the validation shared by the Fighter use cases: it
// returns an error matching srderr.ErrInvalidInput for the zero use case or a zero
// Character.
func fighterCommandProblem(set bool, c Character) error {
	if !set || c.IsZero() {
		return invalidInput("Fighter use case: zero use case or character")
	}
	return nil
}

// fighterContext is what every Fighter use case derives from its command.
type fighterContext struct {
	stats FighterLevelStats
	state FighterState
}

// contextOf derives the Fighter context of c. It returns a non-zero reason, and no
// error, when c has no Fighter levels (FighterNotAFighter), decided before any roll
// or lookup. It returns an error matching srderr.ErrInvalidInput when another type
// of state sits under the Fighter key.
func (p FighterProgression) contextOf(c Character) (fighterContext, FighterReason, error) {
	level, ok := fighterLevelOf(c)
	if !ok {
		return fighterContext{}, FighterNotAFighter, nil
	}
	state, err := readFighterState(c)
	if err != nil {
		return fighterContext{}, 0, err
	}
	return fighterContext{stats: p.StatsAt(level), state: state}, 0, nil
}

// FighterActionCost is what a feature costs to activate. The zero value is not a cost.
type FighterActionCost int

// The action costs.
const (
	// FighterBonusAction: the feature is activated as a Bonus Action. The use case does not track the action economy.
	FighterBonusAction FighterActionCost = iota + 1
)

// String returns "bonus_action" or "unknown".
func (c FighterActionCost) String() string {
	if c == FighterBonusAction {
		return "bonus_action"
	}
	return "unknown"
}

// FighterAdditionalAction is the action Action Surge gives. The zero value is not an action.
type FighterAdditionalAction int

// The additional actions.
const (
	// FighterAnyActionButMagic: one additional action, except the Magic action.
	FighterAnyActionButMagic FighterAdditionalAction = iota + 1
)

// String returns "any_action_but_magic" or "unknown".
func (a FighterAdditionalAction) String() string {
	if a == FighterAnyActionButMagic {
		return "any_action_but_magic"
	}
	return "unknown"
}
