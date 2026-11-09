package character

// MonkReason is why a Monk feature use was refused. The values are in the order the
// checks run (SRD-independent engine decision, class-monk.md FR-135): the first that
// applies is the one reported. The zero value is not a reason.
type MonkReason int

// The reasons a Monk feature use is refused.
const (
	// MonkNotAMonk: the character has no Monk levels; decided before any roll or lookup.
	MonkNotAMonk MonkReason = iota + 1
	// MonkLevelTooLow: the Monk level does not yet give the feature.
	MonkLevelTooLow
	// MonkNotEligible: the trigger, the target, the subclass or the weapon does not qualify.
	MonkNotEligible
	// MonkAlreadyUsed: the once-only use (per turn, per Long Rest, one creature at a time) is spent.
	MonkAlreadyUsed
	// MonkNoUsesRemaining: every use of a counted feature is expended.
	MonkNoUsesRemaining
	// MonkNotEnoughFocusPoints: fewer Focus Points are available than the feature costs.
	MonkNotEnoughFocusPoints
)

func monkReasonNames() []string {
	return []string{"not_a_monk", "level_too_low", "not_eligible", "already_used", "no_uses_remaining", "not_enough_focus_points"}
}

// String returns the snake_case name ("not_a_monk", ...), the
// srd5e.character.rejection value, or "unknown".
func (r MonkReason) String() string {
	if r < 1 || int(r) > len(monkReasonNames()) {
		return "unknown"
	}
	return monkReasonNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r MonkReason) Valid() bool { return r >= 1 && int(r) <= len(monkReasonNames()) }
