package combat

// EffectClockTick is one tick of the clock a host passes to AdvanceActiveEffects. Variants:
// TurnStartReached, TurnEndReached, RoundStartReached, TimeElapsed, EndOfCombatReached, CombatantLeft. No
// tick carries a round number. The host chooses one clock per span of time (FR-021).
type EffectClockTick interface{ isEffectClockTick() }

// TurnStartReached is the start of the turn of a combatant. Build it with NewTurnStartReached.
type TurnStartReached struct{ combatant CombatantID }

// NewTurnStartReached returns the tick for the start of the turn of id. It returns an error
// matching srderr.ErrInvalidInput for a zero id. It never panics.
func NewTurnStartReached(id CombatantID) (TurnStartReached, error) {
	if id.IsZero() {
		return TurnStartReached{}, invalidInput("turn start tick has a zero combatant")
	}
	return TurnStartReached{id}, nil
}

// TurnEndReached is the end of the turn of a combatant. Build it with NewTurnEndReached.
type TurnEndReached struct{ combatant CombatantID }

// NewTurnEndReached returns the tick for the end of the turn of id. It returns an error matching
// srderr.ErrInvalidInput for a zero id. It never panics.
func NewTurnEndReached(id CombatantID) (TurnEndReached, error) {
	if id.IsZero() {
		return TurnEndReached{}, invalidInput("turn end tick has a zero combatant")
	}
	return TurnEndReached{id}, nil
}

// RoundStartReached is the start of a round. The zero value is valid.
type RoundStartReached struct{}

// TimeElapsed is time that passed in or out of combat, in rounds (at least 1). Build it with
// NewTimeElapsed.
type TimeElapsed struct{ rounds int }

// NewTimeElapsed returns the tick for rounds rounds. It returns an error matching
// srderr.ErrInvalidInput for rounds below 1. It never panics.
func NewTimeElapsed(rounds int) (TimeElapsed, error) {
	if rounds < 1 {
		return TimeElapsed{}, invalidInput("time elapsed is at least 1 round")
	}
	return TimeElapsed{rounds}, nil
}

// Rounds returns the rounds that passed.
func (t TimeElapsed) Rounds() int { return t.rounds }

// EndOfCombatReached is the end of the combat (FR-022). The zero value is valid.
type EndOfCombatReached struct{}

// CombatantLeft is a combatant leaving the combat; the host sends it when it removes one (FR-062).
// Build it with NewCombatantLeft.
type CombatantLeft struct{ combatant CombatantID }

// NewCombatantLeft returns the tick for the departure of id. It returns an error matching
// srderr.ErrInvalidInput for a zero id. It never panics.
func NewCombatantLeft(id CombatantID) (CombatantLeft, error) {
	if id.IsZero() {
		return CombatantLeft{}, invalidInput("combatant left tick has a zero combatant")
	}
	return CombatantLeft{id}, nil
}

func (TurnStartReached) isEffectClockTick()   {}
func (TurnEndReached) isEffectClockTick()     {}
func (RoundStartReached) isEffectClockTick()  {}
func (TimeElapsed) isEffectClockTick()        {}
func (EndOfCombatReached) isEffectClockTick() {}
func (CombatantLeft) isEffectClockTick()      {}

// ClockTickFromTurnEvent maps a tracker event to the tick it stands for: TrackerTurnStarted to
// TurnStartReached, TrackerTurnEnded to TurnEndReached, TrackerRoundStarted to RoundStartReached.
// Any other event, or a tracker event about no combatant, returns false.
func ClockTickFromTurnEvent(e TurnEvent) (EffectClockTick, bool) {
	switch v := e.(type) {
	case TrackerTurnStarted:
		if !v.combatant.IsZero() {
			return TurnStartReached{v.combatant}, true
		}
	case TrackerTurnEnded:
		if !v.combatant.IsZero() {
			return TurnEndReached{v.combatant}, true
		}
	case TrackerRoundStarted:
		return RoundStartReached{}, true
	}
	return nil, false
}

// validEffectClockTick reports whether t is a variant its constructor would accept.
func validEffectClockTick(t EffectClockTick) bool {
	switch v := t.(type) {
	case TurnStartReached:
		return !v.combatant.IsZero()
	case TurnEndReached:
		return !v.combatant.IsZero()
	case CombatantLeft:
		return !v.combatant.IsZero()
	case TimeElapsed:
		return v.rounds >= 1
	case RoundStartReached, EndOfCombatReached:
		return true
	}
	return false
}
