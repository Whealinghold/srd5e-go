package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-paladin.md FR-055, FR-106, FR-115 - every PaladinReason has a distinct snake_case String, the first is not_a_paladin, an out-of-range value is
// unknown and not Valid.
// Path: unhappy (boundary, invalid input).
func TestPaladinReasonStrings(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	n := 0
	for r := character.PaladinNotAPaladin; r.Valid(); r++ {
		n++
		if s := r.String(); s == "unknown" || s == "" || seen[s] {
			t.Errorf("reason %d = %q (duplicate or unknown)", r, s)
		} else {
			seen[s] = true
		}
	}
	if n != 27 || character.PaladinNotAPaladin.String() != "not_a_paladin" || character.PaladinNotACandidate.String() != "not_a_candidate" {
		t.Errorf("%d reasons, first %q, last %q", n, character.PaladinNotAPaladin, character.PaladinNotACandidate)
	}
	for _, r := range []character.PaladinReason{0, -1, character.PaladinNotACandidate + 1} {
		if r.Valid() || r.String() != "unknown" {
			t.Errorf("reason %d is valid or named %q", r, r.String())
		}
	}
}
