package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (doc/specs/character-creation.md FR-023 to FR-027 and the table "Background increase cases"; Soldier lists
// Strength, Dexterity and Constitution, SRD 5.2.1, p. 83).
// Path: per case.
func TestBackgroundAbilityIncreaseApplyTo(t *testing.T) {
	t.Parallel()
	soldier := []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution}
	fighter := scoresOf(t, 15, 14, 13, 8, 10, 12)
	for _, c := range []struct {
		name, requirement, path string
		increase                character.BackgroundAbilityIncrease
		scores                  ability.Scores
		want                    ability.Scores
		notListed, exceeds      []ability.Ability
	}{
		{"plus two Strength, plus one Constitution", "p. 21", "happy", character.MustPlusTwoPlusOne(ability.Strength, ability.Constitution), fighter, scoresOf(t, 17, 14, 14, 8, 10, 12), nil, nil},
		{"plus two Constitution, plus one Dexterity", "p. 21", "happy", character.MustPlusTwoPlusOne(ability.Constitution, ability.Dexterity), fighter, scoresOf(t, 15, 15, 15, 8, 10, 12), nil, nil},
		{"plus one to all", "p. 83", "happy", character.PlusOneToAll{}, fighter, scoresOf(t, 16, 15, 14, 8, 10, 12), nil, nil},
		{"plus two to an unlisted ability", "p. 83", "unhappy (invalid input)", character.MustPlusTwoPlusOne(ability.Intelligence, ability.Strength), fighter, fighter, []ability.Ability{ability.Intelligence}, nil},
		{"plus one to an unlisted ability", "p. 83", "unhappy (invalid input)", character.MustPlusTwoPlusOne(ability.Strength, ability.Charisma), fighter, fighter, []ability.Ability{ability.Charisma}, nil},
		{"both unlisted", "p. 83", "unhappy (invalid input)", character.MustPlusTwoPlusOne(ability.Charisma, ability.Wisdom), fighter, fighter, []ability.Ability{ability.Wisdom, ability.Charisma}, nil},
		{"plus two lands on 20", "p. 21", "unhappy (boundary)", character.MustPlusTwoPlusOne(ability.Strength, ability.Dexterity), scoresOf(t, 18, 14, 13, 8, 10, 12), scoresOf(t, 20, 15, 13, 8, 10, 12), nil, nil},
		{"plus one lands on 20", "p. 21", "unhappy (boundary)", character.MustPlusTwoPlusOne(ability.Constitution, ability.Strength), scoresOf(t, 19, 14, 13, 8, 10, 12), scoresOf(t, 20, 14, 15, 8, 10, 12), nil, nil},
		{"plus two over 20", "p. 21", "unhappy (boundary)", character.MustPlusTwoPlusOne(ability.Strength, ability.Dexterity), scoresOf(t, 19, 14, 13, 8, 10, 12), scoresOf(t, 19, 14, 13, 8, 10, 12), nil, []ability.Ability{ability.Strength}},
		{"plus one over 20", "p. 21", "unhappy (boundary)", character.MustPlusTwoPlusOne(ability.Dexterity, ability.Strength), scoresOf(t, 20, 14, 13, 8, 10, 12), scoresOf(t, 20, 14, 13, 8, 10, 12), nil, []ability.Ability{ability.Strength}},
		{"plus one to all with Dexterity 20", "p. 21", "unhappy (boundary)", character.PlusOneToAll{}, scoresOf(t, 15, 20, 13, 8, 10, 12), scoresOf(t, 15, 20, 13, 8, 10, 12), nil, []ability.Ability{ability.Dexterity}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			switch r := c.increase.ApplyTo(c.scores, soldier).(type) {
			case character.BackgroundIncreaseApplied:
				if len(c.notListed)+len(c.exceeds) != 0 || r.Scores() != c.want {
					t.Errorf("applied %v, want %v [%s]", r.Scores(), c.want, c.requirement)
				}
			case character.BackgroundIncreaseRefused:
				if !slices.Equal(r.NotListed(), c.notListed) || !slices.Equal(r.ExceedsMaximum(), c.exceeds) {
					t.Errorf("refused not listed %v, exceeds %v, want %v and %v", r.NotListed(), r.ExceedsMaximum(), c.notListed, c.exceeds)
				}
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-024 - the same ability twice, or an invalid one, is a validation error and
// MustPlusTwoPlusOne panics with it; a valid pair keeps its two abilities.
// Path: unhappy (invalid input).
func TestNewPlusTwoPlusOne(t *testing.T) {
	t.Parallel()
	for _, pair := range [][2]ability.Ability{{ability.Strength, ability.Strength}, {0, ability.Strength}, {ability.Strength, 0}} {
		if _, err := character.NewPlusTwoPlusOne(pair[0], pair[1]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewPlusTwoPlusOne(%v, %v) error = %v", pair[0], pair[1], err)
		}
	}
	p := character.MustPlusTwoPlusOne(ability.Strength, ability.Wisdom)
	if p.PlusTwo() != ability.Strength || p.PlusOne() != ability.Wisdom {
		t.Errorf("= %+v", p)
	}
	defer func() {
		if err, _ := recover().(error); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic = %v", err)
		}
	}()
	character.MustPlusTwoPlusOne(ability.Strength, ability.Strength)
}
