package character

// MonkAvailability is the closed outcome of the Availability method of the Monk
// services: the use is allowed or it is refused for the first MonkReason that
// applies (class-monk.md FR-135). Variants: MonkAvailable, MonkUnavailable.
type MonkAvailability interface{ isMonkAvailability() }

// MonkAvailable: the feature can be used.
type MonkAvailable struct{}

// MonkUnavailable: the feature cannot be used. Construct it with a keyed literal.
type MonkUnavailable struct{ Reason MonkReason }

func (MonkAvailable) isMonkAvailability()   {}
func (MonkUnavailable) isMonkAvailability() {}

// monkAvailability returns MonkUnavailable with the first non-zero reason of
// reasons, which the caller lists in the order of class-monk.md FR-135, or
// MonkAvailable when there is none.
func monkAvailability(reasons ...MonkReason) MonkAvailability {
	for _, r := range reasons {
		if r != 0 {
			return MonkUnavailable{Reason: r}
		}
	}
	return MonkAvailable{}
}

// monkIf returns reason when cond holds and 0 otherwise.
func monkIf(cond bool, reason MonkReason) MonkReason {
	if cond {
		return reason
	}
	return 0
}

// monkGate returns MonkNotAMonk for zero stats, MonkLevelTooLow below minLevel and 0 otherwise.
func monkGate(stats MonkLevelStats, minLevel int) MonkReason {
	switch {
	case stats.IsZero():
		return MonkNotAMonk
	case stats.level.n < minLevel:
		return MonkLevelTooLow
	}
	return 0
}

// MonkAction is an action a Monk feature lets the Monk take. The zero value is not an action.
type MonkAction int

// The actions Flurry of Blows, Patient Defense and Step of the Wind grant.
const (
	// MonkDisengage: the Disengage action.
	MonkDisengage MonkAction = iota + 1
	// MonkDodge: the Dodge action.
	MonkDodge
	// MonkDash: the Dash action.
	MonkDash
)

// String returns "disengage", "dodge", "dash" or "unknown".
func (a MonkAction) String() string {
	switch a {
	case MonkDisengage:
		return "disengage"
	case MonkDodge:
		return "dodge"
	case MonkDash:
		return "dash"
	}
	return "unknown"
}

// MonkEffectEnd is when an effect a Monk feature imposes ends. The zero value is not an end.
type MonkEffectEnd int

// The ends.
const (
	// MonkEffectEndsAtStartOfMonkNextTurn: the effect lasts until the start of the Monk's next turn.
	MonkEffectEndsAtStartOfMonkNextTurn MonkEffectEnd = iota + 1
)

// String returns "start_of_monk_next_turn" or "unknown".
func (e MonkEffectEnd) String() string {
	if e == MonkEffectEndsAtStartOfMonkNextTurn {
		return "start_of_monk_next_turn"
	}
	return "unknown"
}

// monkReasonOf returns the reason of a MonkUnavailable and 0 for MonkAvailable.
func monkReasonOf(a MonkAvailability) MonkReason {
	if un, ok := a.(MonkUnavailable); ok {
		return un.Reason
	}
	return 0
}
