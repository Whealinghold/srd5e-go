package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/deployed-and-restraining-gear.md section 5 - GearRejection has the twenty-five snake_case reasons the spec lists, in order, and the zero value is no rejection.
// Path: unhappy (invalid input made unrepresentable).
func TestGearRejectionNames(t *testing.T) {
	t.Parallel()
	want := []string{"out_of_reach", "ground_not_level", "id_in_use", "unknown_gear", "unknown_hold", "not_recoverable", "not_poured_oil",
		"already_lit", "not_sprung", "trap_occupied", "already_triggered_this_turn", "oil_not_burning", "trap_already_sprung",
		"trigger_not_applicable", "target_not_seen", "target_already_held", "target_willing", "size_not_allowed", "target_not_held_down",
		"method_not_available", "helper_not_allowed", "helper_out_of_reach", "no_action", "not_destructible", "tools_missing"}
	for i, w := range want {
		r := combat.GearRejection(i + 1)
		if r.String() != w || r.IsZero() {
			t.Errorf("GearRejection(%d) = %q (zero %v), want %q", i+1, r, r.IsZero(), w)
		}
	}
	if !combat.GearRejection(0).IsZero() || combat.GearRejection(0).String() != "none" || combat.GearRejection(len(want)+1).String() != "unknown" {
		t.Error("the zero rejection is none and a value past the list is unknown")
	}
	if combat.GearRejectionToolsMissing != combat.GearRejection(len(want)) || combat.GearRejectionIDInUse != 3 {
		t.Error("constants must follow the list")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-071 - GearProblem names the seven problems of the spec (and the extra invalid_value) and a GearError matches srderr.ErrInvalidInput and another GearError with the same Problem only.
// Path: unhappy (invalid input).
func TestGearProblemAndError(t *testing.T) {
	t.Parallel()
	want := []string{"negative_distance", "negative_round", "blank_id", "end_turn_stamp_mismatch", "hold_kind_mismatch", "negative_movement_limit", "gear_id_mismatch", "invalid_value"}
	for i, w := range want {
		if p := combat.GearProblem(i + 1); p.String() != w {
			t.Errorf("GearProblem(%d) = %q, want %q", i+1, p, w)
		}
	}
	if combat.GearProblem(0).String() != "unknown" || combat.GearProblemInvalidValue != combat.GearProblem(len(want)) {
		t.Error("the zero problem is unknown")
	}
	_, err := combat.NewDeployedGearID(" ")
	var ge *combat.GearError
	if !errors.As(err, &ge) || ge.Problem != combat.GearProblemBlankID || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("err = %v, want a blank_id GearError matching ErrInvalidInput", err)
	}
	if !errors.Is(err, &combat.GearError{Problem: combat.GearProblemBlankID}) || errors.Is(err, &combat.GearError{Problem: combat.GearProblemNegativeRound}) ||
		errors.Is(err, errors.New("other")) || ge.Is((*combat.GearError)(nil)) || (*combat.GearError)(nil).Is(ge) || ge.Error() == "" {
		t.Error("GearError.Is must match by Problem only")
	}
}
