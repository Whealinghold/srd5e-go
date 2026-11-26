package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewExhaustionLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		n                       int
		ok                      bool
	}{
		{"level 1 is the lowest", `conditions.md FR-008; SRD 5.2.1, "Exhaustion", p. 181`, "happy", 1, true},
		{"level 3", `conditions.md FR-008; SRD 5.2.1, "Exhaustion", p. 181`, "happy", 3, true},
		{"level 6 is the highest", `conditions.md FR-008, FR-010; SRD 5.2.1, "Exhaustion", p. 181`, "unhappy (boundary)", 6, true},
		{"level 0 is not the condition", `conditions.md FR-009`, "unhappy (boundary)", 0, false},
		{"negative level", `conditions.md FR-009`, "unhappy (invalid input)", -1, false},
		{"level 7 is above the maximum", `conditions.md FR-010`, "unhappy (boundary)", 7, false},
		{"level 100", `conditions.md FR-010`, "unhappy (invalid input)", 100, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			l, err := combat.NewExhaustionLevel(c.n)
			switch {
			case c.ok && (err != nil || l.Value() != c.n || l.IsZero()):
				t.Errorf("NewExhaustionLevel(%d) = %v, %v, want level %d [%s]", c.n, l, err, c.n, c.requirement)
			case !c.ok && (!errors.Is(err, srderr.ErrInvalidInput) || !l.IsZero() || l.Value() != 0):
				t.Errorf("NewExhaustionLevel(%d) = %v, %v, want the zero level and ErrInvalidInput [%s]", c.n, l, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-008, FR-009 - MustExhaustionLevel returns the level NewExhaustionLevel
// accepts and panics with the error it returns for 0 and 7.
// Path: unhappy (invalid input).
func TestMustExhaustionLevel(t *testing.T) {
	t.Parallel()
	if got := combat.MustExhaustionLevel(4); got.Value() != 4 {
		t.Errorf("MustExhaustionLevel(4) = %v, want level 4", got)
	}
	for _, n := range []int{0, 7} {
		func() {
			defer func() {
				err, ok := recover().(error)
				if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("MustExhaustionLevel(%d) panicked with %v, want an error matching ErrInvalidInput", n, err)
				}
			}()
			combat.MustExhaustionLevel(n)
		}()
	}
}

// Requirement: doc/specs/conditions.md FR-008 to FR-010, testing.md section 10 - NewExhaustionLevel never panics and
// accepts exactly 1 to 6.
// Path: unhappy (invalid input).
func FuzzNewExhaustionLevel(f *testing.F) {
	for _, n := range []int{math.MinInt, -1, 0, 1, 6, 7, math.MaxInt} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		l, err := combat.NewExhaustionLevel(n)
		if (err == nil) != (n >= 1 && n <= 6) || (err == nil && l.Value() != n) {
			t.Fatalf("NewExhaustionLevel(%d) = %v, %v", n, l, err)
		}
	})
}
