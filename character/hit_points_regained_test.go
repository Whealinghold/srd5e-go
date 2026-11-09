package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func conMod(t *testing.T, v int) ability.Modifier {
	t.Helper()
	// A score of 10 + 2v has the modifier v; score 1 is the modifier -5.
	score := max(1, 10+2*v)
	s, err := ability.NewScore(score)
	if err != nil {
		t.Fatal(err)
	}
	return s.Modifier()
}

// Requirement: doc/specs/rests-and-hit-dice.md table 4.8c (HD-1 to HD-8), FR-043,
// FR-056 - the Hit Points regained are the roll plus the Constitution modifier, at
// least 1 (SRD 5.2.1, "Short Rest", "Spend Hit Point Dice", p. 187).
// Path: per case.
func TestHitPointsRegained(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		die                     content.HitPointDie
		roll, mod, want         int
	}{
		{"HD-1", "FR-043", "happy", content.HitDieD8, 4, 2, 6},
		{"HD-2", "FR-043", "happy", content.HitDieD8, 4, 0, 4},
		{"HD-3 total of exactly 1", "FR-043", "unhappy (boundary)", content.HitDieD8, 4, -3, 1},
		{"HD-4 total of -2 raised to 1", "FR-043", "unhappy (boundary)", content.HitDieD8, 1, -3, 1},
		{"HD-5 total of 2", "FR-043", "unhappy (boundary)", content.HitDieD8, 5, -3, 2},
		{"HD-6 total of -4 raised to 1", "FR-043", "unhappy (boundary)", content.HitDieD10, 1, -5, 1},
		{"HD-7 maximum roll", "FR-056", "unhappy (boundary: maximum roll)", content.HitDieD12, 12, 5, 17},
		{"HD-8 minimum roll", "FR-056", "unhappy (boundary: minimum roll)", content.HitDieD6, 1, 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.HitPointsRegained(c.die, c.roll, conMod(t, c.mod))
			if err != nil || got != c.want {
				t.Errorf("HitPointsRegained(%v, %d, %+d) = %d, %v, want %d [%s]", c.die, c.roll, c.mod, got, err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md table 4.8c (HD-9, HD-10), FR-044 - a
// roll outside 1 to the die's faces is a *RollRangeError matching ErrInvalidInput.
// Path: per case.
func TestHitPointsRegainedRejectsARollOutsideTheDie(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		die  content.HitPointDie
		roll int
		max  int
	}{"HD-9 zero": {content.HitDieD8, 0, 8}, "HD-10 nine on a d8": {content.HitDieD8, 9, 8}, "invalid die": {0, 1, 0}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := character.HitPointsRegained(c.die, c.roll, conMod(t, 2))
			var re *character.RollRangeError
			if !errors.As(err, &re) || !errors.Is(err, srderr.ErrInvalidInput) || got != 0 {
				t.Fatalf("HitPointsRegained = %d, %v", got, err)
			}
			if re.Name != "roll" || re.Value != c.roll || re.Min != 1 || re.Max != c.max || re.Error() == "" {
				t.Errorf("error = %+v", *re)
			}
		})
	}
}
