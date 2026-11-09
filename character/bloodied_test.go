package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/derived-stats.md FR-071 - Bloodied is twice the
// current Hit Points at most the maximum (SRD 5.2.1, Rules Glossary, "Bloodied",
// p. 177).
// Path: per case.
func TestIsBloodied(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path   string
		current, max int
		want         bool
		wantErr      bool
	}{
		{"exactly half", "unhappy (boundary)", 10, 20, true, false},
		{"one above half", "unhappy (boundary)", 11, 20, false, false},
		{"odd maximum, below half", "unhappy (boundary)", 7, 15, true, false},
		{"odd maximum, above half", "unhappy (boundary)", 8, 15, false, false},
		{"above the maximum", "unhappy (boundary)", 25, 20, false, false},
		{"zero current", "unhappy (boundary)", 0, 20, true, false},
		{"negative current", "unhappy (invalid input)", -1, 20, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.IsBloodied(c.current, character.MustHitPointMaximum(c.max))
			if got != c.want || (err != nil) != c.wantErr {
				t.Errorf("IsBloodied(%d, %d) = %v, %v", c.current, c.max, got, err)
			}
			if c.wantErr {
				wantInvalid(t, err, "IsBloodied")
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-071 - the zero maximum is invalid.
// Path: unhappy (invalid input).
func TestIsBloodiedZeroMaximum(t *testing.T) {
	t.Parallel()
	got, err := character.IsBloodied(3, character.HitPointMaximum{})
	wantInvalid(t, err, "IsBloodied zero maximum")
	if got {
		t.Error("IsBloodied with an error returned true")
	}
}
