package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-ranger.md section 7, FR-105 - every RangerReason has a distinct snake_case String, the first is not_a_ranger and the last no_reaction, an out-of-range
// value is unknown and not Valid.
// Path: unhappy (boundary, invalid input).
func TestRangerReasonStrings(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	n := 0
	for r := character.RangerNotARanger; r.Valid(); r++ {
		n++
		if s := r.String(); s == "unknown" || s == "" || seen[s] {
			t.Errorf("reason %d = %q (duplicate or unknown)", r, s)
		} else {
			seen[s] = true
		}
	}
	if n != 14 || character.RangerNotARanger.String() != "not_a_ranger" || character.RangerNoReaction.String() != "no_reaction" {
		t.Errorf("%d reasons, first %q, last %q", n, character.RangerNotARanger, character.RangerNoReaction)
	}
	for _, r := range []character.RangerReason{0, -1, character.RangerNoReaction + 1} {
		if r.Valid() || r.String() != "unknown" {
			t.Errorf("reason %d is valid or named %q", r, r.String())
		}
	}
}

// Requirement: doc/specs/class-ranger.md T9 (SRD 5.2.1, "Level 3: Hunter’s Prey", "Level 11: Superior Hunter’s Prey", p. 61) - every RangerRiderReason has a distinct snake_case String and an
// out-of-range value is unknown.
// Path: unhappy (boundary, invalid input).
func TestRangerRiderReasonStrings(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for r := character.RangerRiderFeatureNotAvailable; r <= character.RangerRiderNotVisible; r++ {
		if s := r.String(); s == "unknown" || seen[s] {
			t.Errorf("rider reason %d = %q (duplicate or unknown)", r, s)
		} else {
			seen[s] = true
		}
	}
	if len(seen) != 10 || character.RangerRiderTooFar.String() != "too_far" || character.RangerRiderReason(0).String() != "unknown" || (character.RangerRiderNotVisible+1).String() != "unknown" {
		t.Errorf("%d rider reasons", len(seen))
	}
}
