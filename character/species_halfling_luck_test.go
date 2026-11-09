package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-042, FR-044, Table H (SRD 5.2.1, "Luck", p. 86; "Interactions with Rerolls", p. 8) - the eligible dice are those showing 1 that no effect rerolled, whether or not the natural value is 1, so a 1 under Advantage qualifies and two 1s give two indexes.
// Path: per case.
func TestHalflingLuckEligibleDice(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		roll     d20.Roll
		rerolled []int
		want     []int
	}{
		{"single 1", d20.MustRoll(modifier.Normal, 1), nil, []int{0}},
		{"single 2", d20.MustRoll(modifier.Normal, 2), nil, []int{}},
		{"advantage, one 1 (natural 14 kept)", d20.MustRoll(modifier.WithAdvantage, 1, 14), nil, []int{0}},
		{"advantage, the 1 is the second die", d20.MustRoll(modifier.WithAdvantage, 14, 1), nil, []int{1}},
		{"two 1s under Advantage", d20.MustRoll(modifier.WithAdvantage, 1, 1), nil, []int{0, 1}},
		{"disadvantage, two 1s", d20.MustRoll(modifier.WithDisadvantage, 1, 1), nil, []int{0, 1}},
		{"a 1 already rerolled by an effect", d20.MustRoll(modifier.Normal, 1), []int{0}, []int{}},
		{"two 1s, one rerolled", d20.MustRoll(modifier.WithAdvantage, 1, 1), []int{0}, []int{1}},
		{"a 20 is not eligible", d20.MustRoll(modifier.Normal, 20), nil, []int{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.HalflingLuck{}.EligibleDice(c.roll, c.rerolled)
			if !slices.Equal(got, c.want) || got == nil {
				t.Errorf("EligibleDice = %v, want %v", got, c.want)
			}
		})
	}
}
