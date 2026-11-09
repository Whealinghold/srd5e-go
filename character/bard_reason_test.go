package character_test

import (
	"regexp"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-bard.md section 7 - every BardReason has a distinct snake_case String (the rejection value of the Bard use cases), the zero value and values past the last are not reasons and render "unknown".
// Path: per case (happy / unhappy: invalid input).
func TestBardReasonStrings(t *testing.T) {
	t.Parallel()
	snake := regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	seen := map[string]bool{}
	n := 0
	for r := character.BardNotABard; r.Valid(); r++ {
		n++
		if !snake.MatchString(r.String()) || seen[r.String()] {
			t.Errorf("reason %d renders %q (repeated or not snake_case)", int(r), r.String())
		}
		seen[r.String()] = true
	}
	if n != 13 || character.BardSaveNotFailed.String() != "save_not_failed" || character.BardNotABard.String() != "not_a_bard" {
		t.Errorf("%d reasons, last %q, first %q", n, character.BardSaveNotFailed, character.BardNotABard)
	}
	for _, r := range []character.BardReason{0, -1, 14} {
		if r.Valid() || r.String() != "unknown" {
			t.Errorf("BardReason(%d) = valid %v, %q", int(r), r.Valid(), r.String())
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-005, FR-062, FR-071, FR-077 - every BardChoiceProblem has a distinct snake_case String; the zero value and values past the last render "unknown".
// Path: per case (happy / unhappy: invalid input).
func TestBardChoiceProblemStrings(t *testing.T) {
	t.Parallel()
	snake := regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	seen := map[string]bool{}
	n := 0
	for p := character.BardExpertiseCountWrong; p.Valid(); p++ {
		n++
		if !snake.MatchString(p.String()) || seen[p.String()] {
			t.Errorf("problem %d renders %q (repeated or not snake_case)", int(p), p.String())
		}
		seen[p.String()] = true
	}
	if n != 21 || character.BardReplacedNotHeld.String() != "replaced_not_held" {
		t.Errorf("%d problems, last %q", n, character.BardReplacedNotHeld)
	}
	for _, p := range []character.BardChoiceProblem{0, -1, 22} {
		if p.Valid() || p.String() != "unknown" {
			t.Errorf("BardChoiceProblem(%d) = valid %v, %q", int(p), p.Valid(), p.String())
		}
	}
}
