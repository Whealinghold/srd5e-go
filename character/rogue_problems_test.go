package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-rogue.md FR-022, FR-067, FR-103 - a problems error matches srderr.ErrInvalidInput, lists every
// problem once in check order, and Problems returns a copy, so changing it leaves the error unchanged.
// Path: unhappy (immutability) and unhappy (invalid input).
func TestRogueProblemsError(t *testing.T) {
	t.Parallel()
	_, err := character.NewRogueLanguageChoice(content.LanguageThievesCant, nil)
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
	got := rogProblems[character.RogueLanguageChoiceProblem](t, err)
	want := []character.RogueLanguageChoiceProblem{character.RogueLanguageIsThievesCant, character.RogueLanguageNotPermitted}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("problems = %v, want %v", got, want)
	}
	got[0] = 0
	if again := rogProblems[character.RogueLanguageChoiceProblem](t, err); again[0] != want[0] {
		t.Errorf("Problems returned shared storage: %v", again)
	}
	if err.Error() == "" {
		t.Error("empty error text")
	}
}
