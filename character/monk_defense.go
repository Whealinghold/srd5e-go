package character

// MonkFocusSpendResolution is the outcome of a feature that only expends Focus Points:
// the state with them expended. Only DisciplinedSurvivor.Resolve and
// SuperiorDefense.Resolve build one.
type MonkFocusSpendResolution struct {
	spent int
	state MonkState
}

// FocusPointsSpent returns the Focus Points expended.
func (r MonkFocusSpendResolution) FocusPointsSpent() int { return r.spent }

// State returns the new state.
func (r MonkFocusSpendResolution) State() MonkState { return r.state }

const (
	disciplinedSurvivorCost = 1
	superiorDefenseCost     = 3
)

// DisciplinedSurvivor decides and resolves the reroll of Disciplined Survivor (SRD
// 5.2.1, "Level 14: Disciplined Survivor", p. 52). It only expends the Focus Point; the
// host rerolls through d20.Roll.ApplyReroll, which keeps the new value even when lower.
// It is stateless and pure; the zero value is ready to use. Safe for concurrent use.
type DisciplinedSurvivor struct{}

// Availability reports whether the Monk can reroll a saving throw: MonkNotAMonk for zero
// stats, MonkLevelTooLow below level 14, MonkNotEligible when the saving throw did not
// fail, MonkNotEnoughFocusPoints with no Focus Point.
func (DisciplinedSurvivor) Availability(stats MonkLevelStats, state MonkState, saveFailed bool) MonkAvailability {
	return monkAvailability(
		monkGate(stats, monkSurvivorLevel),
		monkIf(!saveFailed, MonkNotEligible),
		monkIf(state.FocusPoints(stats).Available() < disciplinedSurvivorCost, MonkNotEnoughFocusPoints),
	)
}

// Resolve expends the Focus Point. It returns an error matching srderr.ErrInvalidInput
// when Availability says unavailable. It never panics.
func (d DisciplinedSurvivor) Resolve(stats MonkLevelStats, state MonkState, saveFailed bool) (MonkFocusSpendResolution, error) {
	if _, ok := d.Availability(stats, state, saveFailed).(MonkAvailable); !ok {
		return MonkFocusSpendResolution{}, invalidInput("Disciplined Survivor resolved while unavailable")
	}
	state.focusExpended += disciplinedSurvivorCost
	return MonkFocusSpendResolution{spent: disciplinedSurvivorCost, state: state}, nil
}

// Event returns the DisciplinedSurvivorUsed event for the character id.
func (DisciplinedSurvivor) Event(id Identifier) MonkEvent { return DisciplinedSurvivorUsed{id: id} }

// SuperiorDefense decides and resolves Superior Defense (SRD 5.2.1, "Level 18: Superior
// Defense", p. 52). It is stateless and pure; the zero value is ready to use. Safe for
// concurrent use.
type SuperiorDefense struct{}

// Availability reports whether the Monk can bolster itself at the start of its turn
// (atStartOfTurn): MonkNotAMonk for zero stats, MonkLevelTooLow below level 18,
// MonkNotEligible outside the start of the Monk's turn, MonkNotEnoughFocusPoints with
// fewer than 3 Focus Points.
func (SuperiorDefense) Availability(stats MonkLevelStats, state MonkState, atStartOfTurn bool) MonkAvailability {
	return monkAvailability(
		monkGate(stats, monkSuperiorDefenseLevel),
		monkIf(!atStartOfTurn, MonkNotEligible),
		monkIf(state.FocusPoints(stats).Available() < superiorDefenseCost, MonkNotEnoughFocusPoints),
	)
}

// Resolve expends the 3 Focus Points. It returns an error matching srderr.ErrInvalidInput
// when Availability says unavailable. It never panics.
func (s SuperiorDefense) Resolve(stats MonkLevelStats, state MonkState, atStartOfTurn bool) (MonkFocusSpendResolution, error) {
	if _, ok := s.Availability(stats, state, atStartOfTurn).(MonkAvailable); !ok {
		return MonkFocusSpendResolution{}, invalidInput("Superior Defense resolved while unavailable")
	}
	state.focusExpended += superiorDefenseCost
	return MonkFocusSpendResolution{spent: superiorDefenseCost, state: state}, nil
}

// Effect returns the Resistance the activation gives: all damage except Force, for one
// minute or until the Monk is Incapacitated.
func (SuperiorDefense) Effect() SuperiorDefenseEffect { return SuperiorDefenseEffect{} }

// Event returns the SuperiorDefenseActivated event for the character id.
func (SuperiorDefense) Event(id Identifier) MonkEvent { return SuperiorDefenseActivated{id: id} }
