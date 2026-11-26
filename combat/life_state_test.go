package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-088 - NewDying accepts tallies 0 to 2 and returns
// ErrInvalidInput for any other (SRD 5.2.1, "Death Saving Throws", p. 17: the third success or failure ends it).
// Path: per case.
func TestNewDying(t *testing.T) {
	t.Parallel()
	for s := 0; s <= 2; s++ {
		for f := 0; f <= 2; f++ {
			if d, err := combat.NewDying(s, f); err != nil || d.Successes() != s || d.Failures() != f {
				t.Errorf("NewDying(%d, %d) = %+v, %v", s, f, d, err)
			}
		}
	}
	for _, c := range [][2]int{{3, 0}, {0, 3}, {-1, 0}, {0, -1}} {
		if _, err := combat.NewDying(c[0], c[1]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewDying(%d, %d) error = %v, want ErrInvalidInput", c[0], c[1], err)
		}
	}
	if (combat.Dying{}) != dying(0, 0) {
		t.Errorf("the zero Dying is not (0, 0)")
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-087, FR-106 - NewDead needs a valid cause.
// Path: unhappy (invalid input).
func TestNewDead(t *testing.T) {
	t.Parallel()
	if d, err := combat.NewDead(combat.Exhaustion); err != nil || d.Cause() != combat.Exhaustion {
		t.Errorf("= %+v, %v", d, err)
	}
	if _, err := combat.NewDead(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewDead(0) error = %v, want ErrInvalidInput", err)
	}
}
