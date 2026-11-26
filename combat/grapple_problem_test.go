package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/grappling.md FR-083, FR-084, section 7 - the problem and rejection names are the closed
// snake_case srd5e.combat.result values, and the zero value is not one.
// Path: per case.
func TestGrappleProblemAndRejectionNames(t *testing.T) {
	t.Parallel()
	problems := map[combat.GrappleProblem]string{
		combat.GrappleProblemNegativeHandIndex: "negative_hand_index", combat.GrappleProblemBlankPartName: "blank_part_name",
		combat.GrappleProblemSameGrapplerAndTarget: "same_grappler_and_target", combat.GrappleProblemNegativeRange: "negative_range",
		combat.GrappleProblemNegativeDistance: "negative_distance", combat.GrappleProblemUnknownGrapple: "unknown_grapple",
		combat.GrappleProblemDuplicateObservation: "duplicate_observation", combat.GrappleProblemMissingValue: "missing_value",
		combat.GrappleProblemLimitExceeded: "limit_exceeded",
		0:                                  "unknown", 99: "unknown",
	}
	for p, want := range problems {
		if got := p.String(); got != want {
			t.Errorf("GrappleProblem(%d).String() = %q, want %q", int(p), got, want)
		}
	}
	rejections := map[combat.GrappleRejection]string{
		combat.GrappleRejectionPartInUse: "part_in_use", combat.GrappleRejectionTargetAlreadyGrappled: "target_already_grappled",
		combat.GrappleRejectionNotHeld: "not_held", combat.GrappleRejectionNotGrappled: "not_grappled",
		combat.GrappleRejectionNoAction: "no_action", 0: "unknown", 99: "unknown",
	}
	for r, want := range rejections {
		if got := r.String(); got != want {
			t.Errorf("GrappleRejection(%d).String() = %q, want %q", int(r), got, want)
		}
	}
}

// Requirement: doc/specs/grappling.md FR-083 - a GrappleProblemError matches ErrInvalidInput and another error with the
// same problem, and no other.
// Path: unhappy (invalid input).
func TestGrappleProblemErrorMatching(t *testing.T) {
	t.Parallel()
	err := error(&combat.GrappleProblemError{Problem: combat.GrappleProblemNegativeRange})
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &combat.GrappleProblemError{Problem: combat.GrappleProblemNegativeRange}) {
		t.Error("error does not match ErrInvalidInput or its own problem")
	}
	if errors.Is(err, &combat.GrappleProblemError{Problem: combat.GrappleProblemNegativeDistance}) || errors.Is(err, srderr.ErrContractViolation) {
		t.Error("error matches another problem or sentinel")
	}
	if err.Error() == "" {
		t.Error("empty error text")
	}
	var nilErr *combat.GrappleProblemError
	if nilErr.Is(&combat.GrappleProblemError{Problem: combat.GrappleProblemNegativeRange}) || (&combat.GrappleProblemError{Problem: combat.GrappleProblemNegativeRange}).Is(nilErr) {
		t.Error("a nil error matches")
	}
	if nilErr.Is(nilErr) {
		t.Error("a nil error matches")
	}
}
