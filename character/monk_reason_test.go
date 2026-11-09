package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-monk.md FR-135 and observability.md section 5 - every reason has its snake_case name, which is the
// srd5e.character.rejection value, in the order the checks run; the zero and an out-of-range value are unknown and not valid.
// Path: per case.
func TestMonkReasonNames(t *testing.T) {
	t.Parallel()
	want := []struct {
		r    character.MonkReason
		name string
	}{
		{character.MonkNotAMonk, "not_a_monk"}, {character.MonkLevelTooLow, "level_too_low"}, {character.MonkNotEligible, "not_eligible"},
		{character.MonkAlreadyUsed, "already_used"}, {character.MonkNoUsesRemaining, "no_uses_remaining"},
		{character.MonkNotEnoughFocusPoints, "not_enough_focus_points"},
	}
	for i, c := range want {
		if int(c.r) != i+1 || c.r.String() != c.name || !c.r.Valid() {
			t.Errorf("reason %d = %q valid %v, want %q valid in order %d", int(c.r), c.r.String(), c.r.Valid(), c.name, i+1)
		}
	}
	for _, r := range []character.MonkReason{0, 7, -1} {
		if r.String() != "unknown" || r.Valid() {
			t.Errorf("reason %d = %q valid %v, want unknown and not valid", int(r), r.String(), r.Valid())
		}
	}
}
