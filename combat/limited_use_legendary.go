package combat

// LegendaryTrack is the Legendary Action uses of a monster combatant (SRD 5.2.1,
// "Legendary Actions", p. 257). It is a closed set with two variants:
// LegendaryNone and LegendaryTracked. Every variant is an immutable comparable
// value.
type LegendaryTrack interface{ isLegendaryTrack() }

// LegendaryNone: the monster has no Legendary Actions. The zero value is that.
type LegendaryNone struct{}

func (LegendaryNone) isLegendaryTrack() {}

// LegendaryTracked: the monster has Legendary Actions and this many uses left.
// 0 <= Remaining <= Maximum. Only a state builds one.
type LegendaryTracked struct{ maximum, remaining, uses, usesInLair int }

func (LegendaryTracked) isLegendaryTrack() {}

// Maximum returns the uses the monster regains at the start of each of its turns,
// for the lair status of the state that holds this value.
func (t LegendaryTracked) Maximum() int { return t.maximum }

// Remaining returns the uses left.
func (t LegendaryTracked) Remaining() int { return t.remaining }

// legendaryMaximum is the number of uses while the monster has status lair.
func legendaryMaximum(uses, usesInLair int, lair LairStatus) int {
	if lair == InLair && usesInLair != 0 {
		return usesInLair
	}
	return uses
}

// LegendaryActionWindow is the gap a Legendary Action may be taken in: right
// after the turn of another creature ends, in a round. It is built from a
// TrackerTurnEnded event. Comparable. The zero value is invalid; build one with
// NewLegendaryActionWindow. Safe for concurrent use.
type LegendaryActionWindow struct {
	endedTurnOf CombatantID
	round       int
}

// NewLegendaryActionWindow returns the window after the turn of endedTurnOf in
// round.
//
// It returns an error matching srderr.ErrInvalidInput for a zero endedTurnOf or
// a round below 1. It never panics.
func NewLegendaryActionWindow(endedTurnOf CombatantID, round int) (LegendaryActionWindow, error) {
	if endedTurnOf.IsZero() || round < 1 {
		return LegendaryActionWindow{}, invalidInput("a Legendary Action window needs a creature and a round of at least 1")
	}
	return LegendaryActionWindow{endedTurnOf, round}, nil
}

// Combatants returns a new slice holding the creature whose turn just ended.
func (w LegendaryActionWindow) Combatants() []CombatantID { return []CombatantID{w.endedTurnOf} }

// Round returns the round, 1 or more.
func (w LegendaryActionWindow) Round() int { return w.round }

// IsZero reports whether w is the invalid zero window.
func (w LegendaryActionWindow) IsZero() bool { return w == LegendaryActionWindow{} }

// LegendaryWindowRecord is the last window a Legendary Action was taken in. It
// is a closed set with two variants: LegendaryWindowNoneTaken and
// LegendaryWindowTaken. Both are immutable comparable values.
type LegendaryWindowRecord interface{ isLegendaryWindowRecord() }

// LegendaryWindowNoneTaken: no Legendary Action was taken yet. The zero value is
// that.
type LegendaryWindowNoneTaken struct{}

func (LegendaryWindowNoneTaken) isLegendaryWindowRecord() {}

// LegendaryWindowTaken: a Legendary Action was taken in Window.
type LegendaryWindowTaken struct{ Window LegendaryActionWindow }

func (LegendaryWindowTaken) isLegendaryWindowRecord() {}

// LegendaryActionRefusal is why a Legendary Action was refused. The zero value is
// invalid. When several apply the first of this list is reported: no Legendary
// Actions, own turn, incapacitated, window already used, no uses remaining.
// Comparable and safe for concurrent use.
type LegendaryActionRefusal int

// The refusals of a Legendary Action, in the order they are judged.
const (
	// LegendaryRefusedNoActions: the monster has no Legendary Actions.
	LegendaryRefusedNoActions LegendaryActionRefusal = iota + 1
	// LegendaryRefusedOwnTurn: the turn that ended is the monster's own.
	LegendaryRefusedOwnTurn
	// LegendaryRefusedIncapacitated: the monster cannot take actions.
	LegendaryRefusedIncapacitated
	// LegendaryRefusedWindowUsed: a Legendary Action was already taken in this window.
	LegendaryRefusedWindowUsed
	// LegendaryRefusedNoUses: no use remains.
	LegendaryRefusedNoUses
)

// String returns "no_legendary_actions", "own_turn", "incapacitated",
// "window_already_used", "no_uses_remaining" or "unknown".
func (r LegendaryActionRefusal) String() string {
	return enumName([]string{"no_legendary_actions", "own_turn", "incapacitated", "window_already_used", "no_uses_remaining"}, int(r))
}

// LimitedUseTransition is a new state and the events that produced it, in the
// order they happened. The zero value is the zero state and no events.
type LimitedUseTransition struct {
	state  LimitedUseState
	events []MonsterLimitedUseEvent
}

// State returns the new state.
func (t LimitedUseTransition) State() LimitedUseState { return t.state }

// Events returns a copy of the events.
func (t LimitedUseTransition) Events() []MonsterLimitedUseEvent {
	return append([]MonsterLimitedUseEvent(nil), t.events...)
}

// LimitedUseSpendOutcome is the closed set of outcomes of LimitedUseState.Spend
// and SpendSpell. Variants: LimitedUseSpent, LimitedUseSpendRefused.
type LimitedUseSpendOutcome interface{ isLimitedUseSpendOutcome() }

// LimitedUseSpent: a use was spent.
type LimitedUseSpent struct{ Transition LimitedUseTransition }

func (LimitedUseSpent) isLimitedUseSpendOutcome() {}

// LimitedUseSpendRefused: the spend was refused and the state is unchanged.
type LimitedUseSpendRefused struct{ Reason LimitedUseRefusal }

func (LimitedUseSpendRefused) isLimitedUseSpendOutcome() {}

// LegendaryActionOutcome is the closed set of outcomes of
// LimitedUseState.TakeLegendaryAction. Variants: LegendaryActionAllowed,
// LegendaryActionRefused.
type LegendaryActionOutcome interface{ isLegendaryActionOutcome() }

// LegendaryActionAllowed: a Legendary Action may be taken and one use was spent.
type LegendaryActionAllowed struct{ Transition LimitedUseTransition }

func (LegendaryActionAllowed) isLegendaryActionOutcome() {}

// LegendaryActionRefused: the Legendary Action was refused and the state is
// unchanged.
type LegendaryActionRefused struct{ Reason LegendaryActionRefusal }

func (LegendaryActionRefused) isLegendaryActionOutcome() {}
