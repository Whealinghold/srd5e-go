package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/derived-stats.md FR-057 - a level is 1 to 20; outside
// that NewLevel returns a *LevelError with the reason (SRD 5.2.1, "Level
// Advancement", p. 23).
// Path: per case.
func TestNewLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		n          int
		want       character.LevelRejection
	}{
		{"level 0", "unhappy (boundary)", 0, character.LevelBelowMinimum},
		{"level -1", "unhappy (boundary)", -1, character.LevelBelowMinimum},
		{"level 1", "unhappy (boundary)", 1, 0},
		{"level 20", "unhappy (boundary)", 20, 0},
		{"level 21", "unhappy (boundary)", 21, character.LevelAboveMaximum},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			l, err := character.NewLevel(c.n)
			if c.want == 0 {
				if err != nil || l.Value() != c.n || l.IsZero() {
					t.Fatalf("NewLevel(%d) = %v, %v", c.n, l, err)
				}
				return
			}
			var le *character.LevelError
			if !errors.As(err, &le) || le.Reason != c.want || le.Value != c.n || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewLevel(%d) error = %v, want reason %v", c.n, err, c.want)
			}
			if !l.IsZero() {
				t.Errorf("NewLevel(%d) = %v with an error, want the zero Level", c.n, l)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-057 - MustLevel panics with the
// NewLevel error if and only if NewLevel returns one; the zero Level is invalid.
// Path: unhappy (invalid input).
func TestMustLevel(t *testing.T) {
	t.Parallel()
	if got := recovered(t, func() { character.MustLevel(5) }); got != nil {
		t.Errorf("MustLevel(5) panicked: %v", got)
	}
	_, want := character.NewLevel(21)
	got := recovered(t, func() { character.MustLevel(21) })
	if !errors.Is(got, want) {
		t.Errorf("MustLevel(21) panic = %v, want %v", got, want)
	}
	if !(character.Level{}).IsZero() {
		t.Error("zero Level is not IsZero")
	}
}

// Requirement: doc/specs/derived-stats.md FR-057 - rejection names are stable.
// Path: happy.
func TestLevelRejectionString(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.LevelRejection]string{
		character.LevelBelowMinimum: "belowMinimum", character.LevelAboveMaximum: "aboveMaximum", 0: "unknown",
	} {
		if got := r.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", r, got, want)
		}
	}
	if (&character.LevelError{Value: 3, Reason: character.LevelBelowMinimum}).Error() == "" {
		t.Error("empty error text")
	}
	if errors.Is(&character.LevelError{Value: 1, Reason: character.LevelBelowMinimum}, &character.LevelError{Value: 2, Reason: character.LevelBelowMinimum}) {
		t.Error("errors with different values match")
	}
}
