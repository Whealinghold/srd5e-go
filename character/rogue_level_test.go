package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-rogue.md FR-013 - a Rogue level below 1 or above 20 is refused by the constructor with an
// error matching srderr.ErrInvalidInput, the panicking helper panics with that error, and 1 and 20 are accepted.
// Path: per case (levels 0, -1, 1, 20 and 21).
func TestRogueLevelBoundaries(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, path string
		n          int
		ok         bool
	}{
		{"level 0", "unhappy (boundary)", 0, false},
		{"level -1", "unhappy (boundary)", -1, false},
		{"level 1", "happy", 1, true},
		{"level 20", "unhappy (boundary)", 20, true},
		{"level 21", "unhappy (boundary)", 21, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			l, err := character.NewRogueLevel(c.n)
			if c.ok {
				if err != nil || l.Value() != c.n || l.IsZero() {
					t.Errorf("NewRogueLevel(%d) = %v, %v", c.n, l, err)
				}
				return
			}
			wantInvalid(t, err, c.name)
			if !l.IsZero() || l.Value() != 0 {
				t.Errorf("NewRogueLevel(%d) = %v, want the zero level", c.n, l)
			}
			defer func() {
				e, ok := recover().(error)
				if !ok {
					t.Fatalf("MustRogueLevel(%d) did not panic with an error", c.n)
				}
				wantInvalid(t, e, "panic value")
			}()
			character.MustRogueLevel(c.n)
		})
	}
}
