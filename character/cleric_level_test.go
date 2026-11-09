package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-cleric.md FR-117 (SRD 5.2.1, "Cleric Features", p. 36) - a Cleric level is 1 to 20; 0 and 21 are rejected as
// invalid input, and the zero value is invalid.
// Path: per case (unhappy: boundary).
func TestNewClericLevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n    int
		want bool
	}{{0, false}, {1, true}, {20, true}, {21, false}, {-1, false}} {
		l, err := character.NewClericLevel(c.n)
		if (err == nil) != c.want || (err == nil && (l.Value() != c.n || l.IsZero())) {
			t.Errorf("NewClericLevel(%d) = %v, %v; want ok %v", c.n, l, err, c.want)
		}
		if err != nil {
			wantInvalid(t, err, "level")
		}
	}
	if !(character.ClericLevel{}).IsZero() || character.MustClericLevel(7).Value() != 7 {
		t.Error("zero value or MustClericLevel wrong")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustClericLevel(0) did not panic")
		}
	}()
	character.MustClericLevel(0)
}
