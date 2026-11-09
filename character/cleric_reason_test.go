package character_test

import (
	"regexp"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-cleric.md FR-128, section 4.14.1 - every reason has a distinct snake_case name, the first is not_a_cleric, the
// zero value and values past the last are unknown and invalid.
// Path: per case (unhappy: boundary).
func TestClericReasonNames(t *testing.T) {
	t.Parallel()
	snake := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	seen := map[string]bool{}
	n := 0
	for r := character.ClericNotACleric; r.Valid(); r++ {
		n++
		if s := r.String(); !snake.MatchString(s) || seen[s] || s == "unknown" {
			t.Errorf("reason %d name %q is not unique snake_case", r, s)
		} else {
			seen[s] = true
		}
	}
	if n != int(character.ClericRecipientOutOfRange) || character.ClericNotACleric.String() != "not_a_cleric" || character.ClericBelowLevel2.String() != "below_level_2" {
		t.Errorf("%d reasons, first %q", n, character.ClericNotACleric)
	}
	for _, r := range []character.ClericReason{0, character.ClericRecipientOutOfRange + 1, -1} {
		if r.Valid() || r.String() != "unknown" {
			t.Errorf("reason %d = %q, valid %v; want unknown", r, r, r.Valid())
		}
	}
}
