package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-druid.md FR-023, FR-024 (SRD 5.2.1, "Druid Features", p. 41) - a Druid level is 1 to 20; 0, 21 and a negative
// number are rejected as invalid input, the zero value is invalid and MustDruidLevel panics only for an invalid level.
// Path: per case (unhappy: boundary).
func TestNewDruidLevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n    int
		want bool
	}{{0, false}, {1, true}, {20, true}, {21, false}, {-1, false}} {
		l, err := character.NewDruidLevel(c.n)
		if (err == nil) != c.want || (err == nil && (l.Value() != c.n || l.IsZero())) {
			t.Errorf("NewDruidLevel(%d) = %v, %v; want ok %v", c.n, l, err, c.want)
		}
		if err != nil {
			wantInvalid(t, err, "level")
		}
	}
	if !(character.DruidLevel{}).IsZero() || character.MustDruidLevel(7).Value() != 7 {
		t.Error("zero value or MustDruidLevel wrong")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustDruidLevel(0) did not panic")
		}
	}()
	character.MustDruidLevel(0)
}
