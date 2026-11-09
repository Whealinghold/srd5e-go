package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-monk.md FR-016 - a Monk level below 1 or above 20 is refused by the constructor with an
// error matching srderr.ErrInvalidInput, the panicking helper panics with that error, and 1 and 20 are accepted.
// Path: per case (levels 0, 1, 20, 21 and a negative one).
func TestMonkLevelBoundaries(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		n                       int
		ok                      bool
	}{
		{"level 0", `SRD 5.2.1, "Monk Features", p. 50`, "unhappy (boundary)", 0, false},
		{"level -1", `SRD 5.2.1, "Monk Features", p. 50`, "unhappy (boundary)", -1, false},
		{"level 1", `SRD 5.2.1, "Monk Features", p. 50`, "happy", 1, true},
		{"level 20", `SRD 5.2.1, "Monk Features", p. 50`, "unhappy (boundary)", 20, true},
		{"level 21", `SRD 5.2.1, "Monk Features", p. 50`, "unhappy (boundary)", 21, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			l, err := character.NewMonkLevel(c.n)
			if c.ok {
				if err != nil || l.Value() != c.n || l.IsZero() {
					t.Errorf("NewMonkLevel(%d) = %v, %v", c.n, l, err)
				}
				return
			}
			wantInvalid(t, err, c.name)
			if !l.IsZero() || l.Value() != 0 {
				t.Errorf("NewMonkLevel(%d) = %v, want the zero level [%s]", c.n, l, c.requirement)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("MustMonkLevel(%d) did not panic", c.n)
				} else if e, ok := r.(error); !ok {
					t.Errorf("MustMonkLevel(%d) panicked with %T, want an error", c.n, r)
				} else {
					wantInvalid(t, e, "panic value")
				}
			}()
			character.MustMonkLevel(c.n)
		})
	}
}
