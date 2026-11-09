package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/heroic-inspiration.md FR-040 - NewRerolledDie accepts
// newValue 1..die.Sides() and a position of 0 or more, and otherwise returns a
// *RerolledDieError (matching ErrInvalidInput) naming the argument and value.
// Path: per case.
func TestNewRerolledDie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		die                     dice.Die
		position, newValue      int
		argument                string // "" accepts
		value                   int
	}{
		{"lowest face", "FR-040", "unhappy (boundary)", dice.D20(), 0, 1, "", 0},
		{"highest face", "FR-040", "unhappy (boundary)", dice.D20(), 1, 20, "", 0},
		{"new value 0", "FR-040", "unhappy (boundary)", dice.D20(), 0, 0, "newValue", 0},
		{"new value above the die", "FR-040", "unhappy (boundary)", dice.D20(), 0, 21, "newValue", 21},
		{"negative position", "FR-040", "unhappy (boundary)", dice.D6(), -1, 3, "position", -1},
		{"zero die", "FR-040", "unhappy (invalid input)", dice.Die{}, 0, 1, "newValue", 1},
		{"both wrong: newValue first", "FR-040", "unhappy (rule interaction)", dice.D6(), -1, 7, "newValue", 7},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewRerolledDie(c.die, c.position, c.newValue)
			if c.argument == "" {
				if err != nil || got.Die() != c.die || got.Position() != c.position || got.NewValue() != c.newValue || character.MustRerolledDie(c.die, c.position, c.newValue) != got {
					t.Errorf("rerolled = %#v, %v [%s]", got, err, c.requirement)
				}
				return
			}
			var re *character.RerolledDieError
			if !errors.As(err, &re) || !errors.Is(err, srderr.ErrInvalidInput) || re.Argument != c.argument || re.Value != c.value || got != (character.RerolledDie{}) || err.Error() == "" {
				t.Fatalf("err = %#v [%s]", err, c.requirement)
			}
			defer func() {
				if p, _ := recover().(*character.RerolledDieError); p == nil {
					t.Error("MustRerolledDie did not panic with the error")
				}
			}()
			character.MustRerolledDie(c.die, c.position, c.newValue)
		})
	}
}
