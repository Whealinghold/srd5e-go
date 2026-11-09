package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: equipSpec FR-051, FR-054 - a TableCoverageError matches
// srderr.ErrInvalidInput, and another TableCoverageError only when all four
// lists are equal.
// Path: unhappy (invalid input).
func TestTableCoverageErrorIs(t *testing.T) {
	t.Parallel()
	l := func(n ...int) []content.SpellLevel {
		var out []content.SpellLevel
		for _, v := range n {
			out = append(out, levelOf(t, v))
		}
		return out
	}
	base := &character.TableCoverageError{Missing: l(1), Repeated: l(2), Uncovered: l(3), Overlapping: l(4)}
	same := &character.TableCoverageError{Missing: l(1), Repeated: l(2), Uncovered: l(3), Overlapping: l(4)}
	wantInvalid(t, base, "base")
	if base.Error() == "" || !errors.Is(base, same) {
		t.Error("equal lists should match")
	}
	for name, other := range map[string]*character.TableCoverageError{
		"missing":     {Missing: l(0), Repeated: l(2), Uncovered: l(3), Overlapping: l(4)},
		"repeated":    {Missing: l(1), Repeated: l(0), Uncovered: l(3), Overlapping: l(4)},
		"uncovered":   {Missing: l(1), Repeated: l(2), Uncovered: l(0), Overlapping: l(4)},
		"overlapping": {Missing: l(1), Repeated: l(2), Uncovered: l(3), Overlapping: l(0)},
	} {
		if errors.Is(base, other) {
			t.Errorf("differing %s list should not match", name)
		}
	}
	var nilErr *character.TableCoverageError
	if base.Is(nilErr) || base.Is(errors.New("x")) {
		t.Error("a nil or foreign target should not match")
	}
}
