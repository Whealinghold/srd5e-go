package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/heroic-inspiration.md FR-020, FR-021, FR-033, table E11, E12
// - NewRerollTarget accepts 1..dice.MaxCount dice and a position in 0..diceRolled-1,
// and otherwise returns a *RerollTargetError (matching ErrInvalidInput) with the
// reason, argument and value; diceRolled is checked before position (SRD 5.2.1, p.
// 183 "any die immediately after rolling it").
// Path: per case.
func TestNewRerollTarget(t *testing.T) {
	t.Parallel()
	const dr, pos = character.RerollTargetDiceRolledOutOfRange, character.RerollTargetPositionOutOfRange
	cases := []struct {
		name, requirement, path string
		diceRolled, position    int
		want                    character.RerollTargetReason // 0 accepts
		argument                string
		value                   int
	}{
		{"one die, position 0", "FR-020", "happy", 1, 0, 0, "", 0},
		{"two dice, last index", "FR-020", "happy", 2, 1, 0, "", 0},
		{"maximum dice, last index", "FR-020", "unhappy (boundary)", dice.MaxCount, dice.MaxCount - 1, 0, "", 0},
		{"E12 no dice rolled", "FR-020", "unhappy (boundary)", 0, 0, dr, "diceRolled", 0},
		{"negative dice rolled", "FR-020", "unhappy (boundary)", -1, 0, dr, "diceRolled", -1},
		{"above the maximum", "FR-020", "unhappy (boundary)", dice.MaxCount + 1, 0, dr, "diceRolled", dice.MaxCount + 1},
		{"E11 position at diceRolled", "FR-020", "unhappy (boundary)", 2, 2, pos, "position", 2},
		{"position below zero", "FR-020", "unhappy (boundary)", 2, -1, pos, "position", -1},
		{"one die, position 1", "FR-020", "unhappy (boundary)", 1, 1, pos, "position", 1},
		{"both wrong: diceRolled first", "FR-021", "unhappy (rule interaction)", 0, -5, dr, "diceRolled", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewRerollTarget(dice.D20(), c.diceRolled, c.position)
			if c.want == 0 {
				if err != nil || got.Die() != dice.D20() || got.DiceRolled() != c.diceRolled || got.Position() != c.position || got.IsZero() {
					t.Errorf("target = %#v, %v [%s]", got, err, c.requirement)
				}
				if character.MustRerollTarget(dice.D20(), c.diceRolled, c.position) != got {
					t.Error("MustRerollTarget differs from NewRerollTarget")
				}
				return
			}
			var te *character.RerollTargetError
			if !errors.As(err, &te) || !errors.Is(err, srderr.ErrInvalidInput) || te.Reason != c.want || te.Argument != c.argument || te.Value != c.value || !got.IsZero() || err.Error() == "" {
				t.Fatalf("err = %#v [%s]", err, c.requirement)
			}
			defer func() {
				p, _ := recover().(*character.RerollTargetError)
				if p == nil || p.Reason != c.want {
					t.Errorf("MustRerollTarget panic = %v, want the error", p)
				}
			}()
			character.MustRerollTarget(dice.D20(), c.diceRolled, c.position)
		})
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-021 - NewRerollTarget does not
// panic for any ints or the zero Die and errors if and only if FR-020 rejects; a
// zero Die is accepted but the target reports IsZero.
// Path: unhappy (invalid input).
func TestNewRerollTargetNeverPanics(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-1 << 62, -2, -1, 0, 1, 2, 7, dice.MaxCount, dice.MaxCount + 1, 1 << 62} {
		for _, p := range []int{-1 << 62, -1, 0, 1, n - 1, n, n + 1, 1 << 62} {
			for _, d := range []dice.Die{{}, dice.D4(), dice.D100()} {
				_, err := character.NewRerollTarget(d, n, p)
				if reject := n < 1 || n > dice.MaxCount || p < 0 || p >= n; (err != nil) != reject {
					t.Errorf("NewRerollTarget(%v, %d, %d) error = %v, rejects %v", d, n, p, err, reject)
				}
			}
		}
	}
	zero, err := character.NewRerollTarget(dice.Die{}, 1, 0)
	if err != nil || !zero.IsZero() {
		t.Errorf("zero Die target = %#v, %v", zero, err)
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-033 - RerollTargetReason has a
// String for exactly its two members, and "unknown" otherwise.
// Path: happy.
func TestRerollTargetReasonStrings(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.RerollTargetReason]string{
		character.RerollTargetDiceRolledOutOfRange: "dice_rolled_out_of_range", character.RerollTargetPositionOutOfRange: "position_out_of_range", 0: "unknown", 3: "unknown",
	} {
		if r.String() != want {
			t.Errorf("%d.String() = %q, want %q", r, r.String(), want)
		}
	}
}
