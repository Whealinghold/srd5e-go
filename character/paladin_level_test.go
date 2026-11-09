package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-paladin.md FR-043, Table H (SRD 5.2.1, "Paladin Features", p. 53) - a Paladin level is 1 to 20; 0, 21 and -1 are rejected as invalid
// input, the zero value is invalid and MustPaladinLevel panics with the error only for an invalid level.
// Path: per case (unhappy: boundary, invalid input).
func TestNewPaladinLevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n    int
		want bool
	}{{0, false}, {1, true}, {20, true}, {21, false}, {-1, false}} {
		l, err := character.NewPaladinLevel(c.n)
		if (err == nil) != c.want || (err == nil && (l.Value() != c.n || l.IsZero())) {
			t.Errorf("NewPaladinLevel(%d) = %v, %v; want ok %v", c.n, l, err, c.want)
		}
		if err != nil {
			wantInvalid(t, err, "level")
		}
	}
	if !(character.PaladinLevel{}).IsZero() || character.MustPaladinLevel(7).Value() != 7 {
		t.Error("zero value or MustPaladinLevel wrong")
	}
	for _, n := range []int{0, 21} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("MustPaladinLevel(%d) did not panic", n)
				}
			}()
			character.MustPaladinLevel(n)
		}()
	}
}

// Requirement: doc/specs/class-paladin.md FR-043, Table H (SRD 5.2.1, "Level 1: Lay On Hands", p. 53) - the Lay On Hands maximum is five times the Paladin level at the six
// rows of Table H.
// Path: per case (Table H): happy / unhappy (boundary: levels 1 and 20).
func TestLayOnHandsMaximum(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label, path string
		level, want int
	}{
		{"lowest level", "unhappy (boundary)", 1, 5}, {"level 2", "happy", 2, 10}, {"level 5", "happy", 5, 25},
		{"level 10", "happy", 10, 50}, {"level 11", "happy", 11, 55}, {"level 20", "unhappy (boundary)", 20, 100},
	} {
		if got := character.LayOnHandsMaximum(character.MustPaladinLevel(c.level)); got != c.want {
			t.Errorf("%s: LayOnHandsMaximum(%d) = %d, want %d", c.label, c.level, got, c.want)
		}
	}
}
