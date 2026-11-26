package combat

import "github.com/Whealinghold/srd5e-go/srderr"

// GrappleProblem says why host input to the grapple rules is unusable (SRD
// 5.2.1, "Grappling", p. 182). It is the reason held by a GrappleProblemError and
// by every ...Invalid result of the grapple use cases. The zero value is not a
// problem. Comparable and safe for concurrent use.
type GrappleProblem int

// The problems. The first two belong to the part constructors, SameGrapplerAndTarget
// and NegativeRange to NewGrapple, SameGrapplerAndTarget and NegativeDistance to
// NewGrappleObservation, UnknownGrapple and DuplicateObservation to Grapples.Review,
// MissingValue to a blank CombatantID or an absent part, and LimitExceeded (the shared srd5e.combat.result
// value, plan P10) to Grapples.Begin on a full set of 256 and to Grapples.Review given more than 256 observations.
const (
	GrappleProblemNegativeHandIndex GrappleProblem = iota + 1
	GrappleProblemBlankPartName
	GrappleProblemSameGrapplerAndTarget
	GrappleProblemNegativeRange
	GrappleProblemNegativeDistance
	GrappleProblemUnknownGrapple
	GrappleProblemDuplicateObservation
	GrappleProblemMissingValue
	GrappleProblemLimitExceeded
)

// String returns the snake_case name, the srd5e.combat.result of an invalid
// grapple result, or "unknown".
func (p GrappleProblem) String() string {
	return enumName([]string{"negative_hand_index", "blank_part_name", "same_grappler_and_target", "negative_range",
		"negative_distance", "unknown_grapple", "duplicate_observation", "missing_value", "limit_exceeded"}, int(p))
}

// GrappleProblemError reports unusable grapple input. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type GrappleProblemError struct{ Problem GrappleProblem }

// Error renders the error. The text is not a contract.
func (e *GrappleProblemError) Error() string {
	return "combat: invalid grapple input: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *GrappleProblemError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *GrappleProblemError with the same Problem.
func (e *GrappleProblemError) Is(target error) bool {
	t, ok := target.(*GrappleProblemError)
	return ok && t != nil && e != nil && *t == *e
}

func problemError(p GrappleProblem) error { return &GrappleProblemError{Problem: p} }

// GrappleRejection says why the rules refuse a grapple operation in the current
// state (SRD 5.2.1, "Grappling", p. 182). It is the reason held by every
// ...Rejected result of the grapple use cases. The zero value is not a
// rejection. Comparable and safe for concurrent use.
type GrappleRejection int

// The rejections. NoAction is reported by the escape use case only.
const (
	GrappleRejectionPartInUse GrappleRejection = iota + 1
	GrappleRejectionTargetAlreadyGrappled
	GrappleRejectionNotHeld
	GrappleRejectionNotGrappled
	GrappleRejectionNoAction
)

// String returns the snake_case name, the srd5e.combat.result of a refused
// grapple result, or "unknown".
func (r GrappleRejection) String() string {
	return enumName([]string{"part_in_use", "target_already_grappled", "not_held", "not_grappled", "no_action"}, int(r))
}
