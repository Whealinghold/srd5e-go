package combat

import "github.com/Whealinghold/srd5e-go/srderr"

// GearRejection is why the gear rules refuse an action that is well formed: legal input
// the rules do not allow in the current state. The zero value is no rejection. String
// returns the snake_case reason, the value of srd5e.combat.result.
type GearRejection int

// The reasons, in the order deployed-and-restraining-gear.md section 5 lists them.
const (
	GearRejectionOutOfReach GearRejection = iota + 1
	GearRejectionGroundNotLevel
	GearRejectionIDInUse
	GearRejectionUnknownGear
	GearRejectionUnknownHold
	GearRejectionNotRecoverable
	GearRejectionNotPouredOil
	GearRejectionAlreadyLit
	GearRejectionNotSprung
	GearRejectionTrapOccupied
	GearRejectionAlreadyTriggeredThisTurn
	GearRejectionOilNotBurning
	GearRejectionTrapAlreadySprung
	GearRejectionTriggerNotApplicable
	GearRejectionTargetNotSeen
	GearRejectionTargetAlreadyHeld
	GearRejectionTargetWilling
	GearRejectionSizeNotAllowed
	GearRejectionTargetNotHeldDown
	GearRejectionMethodNotAvailable
	GearRejectionHelperNotAllowed
	GearRejectionHelperOutOfReach
	GearRejectionNoAction
	GearRejectionNotDestructible
	GearRejectionToolsMissing
)

func gearRejectionNames() []string {
	return []string{"out_of_reach", "ground_not_level", "id_in_use", "unknown_gear", "unknown_hold",
		"not_recoverable", "not_poured_oil", "already_lit", "not_sprung", "trap_occupied", "already_triggered_this_turn",
		"oil_not_burning", "trap_already_sprung", "trigger_not_applicable", "target_not_seen", "target_already_held",
		"target_willing", "size_not_allowed", "target_not_held_down", "method_not_available", "helper_not_allowed",
		"helper_out_of_reach", "no_action", "not_destructible", "tools_missing"}
}

// IsZero reports whether r is the zero value, no rejection.
func (r GearRejection) IsZero() bool { return r == 0 }

// String returns the snake_case reason, "none" for the zero value or "unknown".
func (r GearRejection) String() string {
	if r == 0 {
		return "none"
	}
	return enumName(gearRejectionNames(), int(r))
}

// GearProblem says why a value handed to the gear rules is malformed: untrusted input,
// not a rule outcome. String returns the snake_case problem, the value of srd5e.combat.result
// for an invalid result. The zero value is not a problem.
type GearProblem int

// The problems. GearProblemInvalidValue extends the list of deployed-and-restraining-gear.md
// FR-071 for an enum value that is not one of its members and for a zero value.
const (
	GearProblemNegativeDistance GearProblem = iota + 1
	GearProblemNegativeRound
	GearProblemBlankID
	GearProblemEndTurnStampMismatch
	GearProblemHoldKindMismatch
	GearProblemNegativeMovementLimit
	GearProblemGearIDMismatch
	GearProblemInvalidValue
	// GearProblemLimitExceeded: a set that already holds 256 gears or 256 holds (plan P10).
	GearProblemLimitExceeded
)

// String returns the snake_case problem or "unknown".
func (p GearProblem) String() string {
	return enumName([]string{"negative_distance", "negative_round", "blank_id", "end_turn_stamp_mismatch",
		"hold_kind_mismatch", "negative_movement_limit", "gear_id_mismatch", "invalid_value", valueResultLimitExceeded}, int(p))
}

// GearError reports a malformed value handed to a gear constructor or rule. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type GearError struct{ Problem GearProblem }

// Error renders the error. The text is not a contract.
func (e *GearError) Error() string { return "combat: invalid gear value: " + e.Problem.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *GearError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *GearError with the same Problem.
func (e *GearError) Is(target error) bool {
	t, ok := target.(*GearError)
	return ok && t != nil && e != nil && *t == *e
}

// gearInvalid returns the *GearError of p.
func gearInvalid(p GearProblem) error { return &GearError{Problem: p} }
