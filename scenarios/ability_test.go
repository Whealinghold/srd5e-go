package scenarios_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Untrusted integers are parsed, a complete set is built and a modifier is
// converted for a roll (SRD 5.2.1, "Ability Scores", p. 5; "Ability Modifiers",
// p. 6).
func TestScoresBuildAndConvertToRollModifier(t *testing.T) {
	t.Parallel()
	values := []int{10, 14, 12, 8, 16, 20}
	byAbility := map[ability.Ability]ability.Score{}
	for i, a := range ability.All() {
		s, err := ability.NewScore(values[i])
		if err != nil {
			t.Fatalf("NewScore(%d): %v", values[i], err)
		}
		byAbility[a] = s
	}
	scores, err := ability.NewScores(byAbility)
	if err != nil {
		t.Fatalf("NewScores: %v", err)
	}

	strength := modifier.New(scores.Modifier(ability.Strength).Value())
	charisma := modifier.New(scores.Modifier(ability.Charisma).Value())

	if strength != modifier.New(0) || charisma != modifier.New(5) {
		t.Fatalf("modifiers = %v, %v; want +0, +5", strength, charisma)
	}
	if got := modifier.Sum(strength, charisma).ApplyTo(11); got != 16 {
		t.Errorf("ApplyTo(11) = %d, want 16", got)
	}
}

// An out-of-range score is a typed error a caller can branch on.
func TestOutOfRangeScoreIsTypedError(t *testing.T) {
	t.Parallel()
	_, err := ability.NewScore(31)

	var ie *ability.InvalidError
	if !errors.As(err, &ie) || ie.Reason != ability.ReasonAboveMaximum {
		t.Fatalf("err = %v, want InvalidError with ReasonAboveMaximum", err)
	}
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Error("err does not match srderr.ErrInvalidInput")
	}
}

// Every score from 1 to 30 yields the SRD modifier: (score - 10) / 2 rounded
// down.
func TestAbilityModifiersTable(t *testing.T) {
	t.Parallel()
	for v, want := range map[int]int{1: -5, 8: -1, 9: -1, 10: 0, 11: 0, 12: 1, 15: 2, 20: 5, 29: 9, 30: 10} {
		if got := ability.MustScore(v).Modifier().Value(); got != want {
			t.Errorf("score %d: modifier %d, want %d", v, got, want)
		}
	}
}
