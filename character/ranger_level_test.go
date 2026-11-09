package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-ranger.md FR-098 (SRD 5.2.1, "Ranger Features", p. 58) - a Ranger level is 1 to 20; 0, 21 and -1 are rejected as invalid input, the zero value is
// invalid and MustRangerLevel panics with the error only for an invalid level.
// Path: per case (unhappy: boundary, invalid input).
func TestNewRangerLevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n    int
		want bool
	}{{0, false}, {1, true}, {20, true}, {21, false}, {-1, false}} {
		l, err := character.NewRangerLevel(c.n)
		if (err == nil) != c.want || (err == nil && (l.Value() != c.n || l.IsZero())) {
			t.Errorf("NewRangerLevel(%d) = %v, %v; want ok %v", c.n, l, err, c.want)
		}
		if err != nil {
			wantInvalid(t, err, "level")
		}
	}
	if !(character.RangerLevel{}).IsZero() || character.MustRangerLevel(7).Value() != 7 {
		t.Error("zero value or MustRangerLevel wrong")
	}
	for _, n := range []int{0, 21} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("MustRangerLevel(%d) did not panic", n)
				}
			}()
			character.MustRangerLevel(n)
		}()
	}
}
