package character

// FighterTurnOwnership says whose turn it is when Action Surge is used. The zero
// value is not a value.
type FighterTurnOwnership int

// The turn ownerships.
const (
	// FighterOwnTurn: it is the Fighter's turn.
	FighterOwnTurn FighterTurnOwnership = iota + 1
	// FighterOtherTurn: it is another creature's turn.
	FighterOtherTurn
)

// String returns "own_turn", "other_turn" or "unknown".
func (t FighterTurnOwnership) String() string {
	switch t {
	case FighterOwnTurn:
		return "own_turn"
	case FighterOtherTurn:
		return "other_turn"
	}
	return "unknown"
}

// Valid reports whether t is one of the values.
func (t FighterTurnOwnership) Valid() bool { return t == FighterOwnTurn || t == FighterOtherTurn }

// FighterPerTurnUse says whether Action Surge was already used this turn. The
// zero value is not a value.
type FighterPerTurnUse int

// The per-turn states.
const (
	// FighterActionSurgeUnused: Action Surge was not used this turn.
	FighterActionSurgeUnused FighterPerTurnUse = iota + 1
	// FighterActionSurgeUsed: Action Surge was used this turn.
	FighterActionSurgeUsed
)

// String returns "unused", "used" or "unknown".
func (u FighterPerTurnUse) String() string {
	switch u {
	case FighterActionSurgeUnused:
		return "unused"
	case FighterActionSurgeUsed:
		return "used"
	}
	return "unknown"
}

// Valid reports whether u is one of the values.
func (u FighterPerTurnUse) Valid() bool {
	return u == FighterActionSurgeUnused || u == FighterActionSurgeUsed
}

// ActionSurgeEvaluation is the closed outcome of ActionSurge.Evaluate. Variants:
// ActionSurgeGranted, ActionSurgeRefused.
type ActionSurgeEvaluation interface{ isActionSurgeEvaluation() }

// ActionSurgeGranted: the Fighter takes one additional action, except the Magic
// action. Only ActionSurge.Evaluate builds one.
type ActionSurgeGranted struct{ state FighterState }

// State returns the new state: one Action Surge use expended.
func (g ActionSurgeGranted) State() FighterState { return g.state }

// ActionSurgeRefused: Action Surge cannot be used. Construct it with a keyed literal.
type ActionSurgeRefused struct{ Reason FighterReason }

func (ActionSurgeGranted) isActionSurgeEvaluation() {}
func (ActionSurgeRefused) isActionSurgeEvaluation() {}

// ActionSurge decides a use of Action Surge (SRD 5.2.1, "Level 2: Action Surge",
// p. 48). It is stateless and pure; the zero value is ready to use. Safe for
// concurrent use.
type ActionSurge struct{}

// Evaluate checks the use in this order and reports the first reason that holds:
// FighterFeatureNotGained below Fighter level 2, FighterNotOwnTurn,
// FighterAlreadyUsedThisTurn, FighterNoUsesRemaining. Otherwise it grants the use
// with one more use expended. An invalid turn or perTurn is read as not own turn
// and already used, the safe refusals.
func (ActionSurge) Evaluate(stats FighterLevelStats, state FighterState, turn FighterTurnOwnership, perTurn FighterPerTurnUse) ActionSurgeEvaluation {
	switch {
	case stats.actionSurge < 1:
		return ActionSurgeRefused{Reason: FighterFeatureNotGained}
	case turn != FighterOwnTurn:
		return ActionSurgeRefused{Reason: FighterNotOwnTurn}
	case perTurn != FighterActionSurgeUnused:
		return ActionSurgeRefused{Reason: FighterAlreadyUsedThisTurn}
	case state.ActionSurgeRemaining(stats) < 1:
		return ActionSurgeRefused{Reason: FighterNoUsesRemaining}
	}
	state.actionSurgeExpended++
	return ActionSurgeGranted{state: state}
}
