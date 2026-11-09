package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/derived-stats.md FR-062 - the gain is the value plus
// the Constitution modifier, at least 1 (SRD 5.2.1, "Gaining a Level", step 2, p.
// 23).
// Path: per case.
func TestHitPointGainAmount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		gain       character.HitPointGain
		con        int
		want       int
	}{
		{"roll plus modifier", "happy", character.MustRolledHitPointGain(5, content.HitDieD8), 2, 7},
		{"negative modifier floors at 1", "unhappy (boundary)", character.MustRolledHitPointGain(1, content.HitDieD8), -3, 1},
		{"fixed plus modifier", "happy", must(character.NewFixedHitPointGain(6)), 1, 7},
		{"fixed negative modifier floors at 1", "unhappy (boundary)", must(character.NewFixedHitPointGain(1)), -4, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.gain.Amount(modifierOf(t, c.con)); got != c.want {
				t.Errorf("Amount = %d, want %d", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-062 - a roll is 1 to the die's
// faces; the Must form panics exactly when the constructor errors.
// Path: per case.
func TestNewRolledHitPointGain(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		value      int
		die        content.HitPointDie
		want       character.RolledHitPointGainRejection
	}{
		{"roll of the top face", "unhappy (boundary)", 8, content.HitDieD8, 0},
		{"roll of 1", "unhappy (boundary)", 1, content.HitDieD6, 0},
		{"roll 9 on d8", "unhappy (boundary)", 9, content.HitDieD8, character.RolledHitPointGainAboveDieFaces},
		{"roll 7 on d6", "unhappy (boundary)", 7, content.HitDieD6, character.RolledHitPointGainAboveDieFaces},
		{"roll 6 on d6", "unhappy (boundary)", 6, content.HitDieD6, 0},
		{"roll 0", "unhappy (boundary)", 0, content.HitDieD8, character.RolledHitPointGainBelowOne},
		{"invalid die", "unhappy (invalid input)", 1, 0, character.RolledHitPointGainAboveDieFaces},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			g, err := character.NewRolledHitPointGain(c.value, c.die)
			if c.want == 0 {
				if err != nil || g.Value() != c.value || g.Die() != c.die {
					t.Fatalf("gain = %v, %v", g, err)
				}
				return
			}
			var ge *character.RolledHitPointGainError
			if !errors.As(err, &ge) || ge.Reason != c.want || ge.Value != c.value || ge.Die != c.die || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error = %v, want reason %v", err, c.want)
			}
			if p := recovered(t, func() { character.MustRolledHitPointGain(c.value, c.die) }); !errors.Is(p, ge) {
				t.Errorf("Must panic = %v", p)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-062 - a fixed value of at least 1.
// Path: unhappy (invalid input).
func TestNewFixedHitPointGain(t *testing.T) {
	t.Parallel()
	if g, err := character.NewFixedHitPointGain(1); err != nil || g.Value() != 1 {
		t.Errorf("fixed 1 = %v, %v", g, err)
	}
	if _, err := character.NewFixedHitPointGain(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("fixed 0 error = %v", err)
	}
	for r, want := range map[character.RolledHitPointGainRejection]string{
		character.RolledHitPointGainBelowOne: "belowOne", character.RolledHitPointGainAboveDieFaces: "aboveDieFaces", 0: "unknown",
	} {
		if r.String() != want {
			t.Errorf("%d.String() = %q", r, r.String())
		}
	}
	e := &character.RolledHitPointGainError{Value: 9, Die: content.HitDieD8, Reason: character.RolledHitPointGainAboveDieFaces}
	if e.Error() == "" || errors.Is(e, &character.RolledHitPointGainError{Value: 7, Die: content.HitDieD8, Reason: character.RolledHitPointGainAboveDieFaces}) {
		t.Error("error text or Is wrong")
	}
}
