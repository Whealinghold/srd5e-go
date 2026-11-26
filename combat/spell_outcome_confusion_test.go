package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-030, table C - each face of the Confusion "Behavior for the Turn" table gives its row: 1 moves randomly, 2 to 6 does nothing (both edges), 7 and 8 attack a random creature, 9 and 10 the target chooses (SRD 5.2.1, "Confusion", p. 117); a face outside 1 to 10 is ErrFaceOutOfRange.
// Path: per case.
func TestConfusionRowFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		face                    int
		want                    combat.ConfusionRow
		wantName                string
	}{
		{"C1 face 1", "FR-030", "happy", 1, combat.ConfusionRowMovesRandomly, "moves_randomly"},
		{"C2 face 2 lower edge", "FR-030", "unhappy (boundary)", 2, combat.ConfusionRowDoesNothing, "does_nothing"},
		{"C3 face 6 upper edge", "FR-030", "unhappy (boundary)", 6, combat.ConfusionRowDoesNothing, "does_nothing"},
		{"C4 face 7", "FR-030", "unhappy (boundary)", 7, combat.ConfusionRowAttacksRandomCreature, "attacks_random_creature"},
		{"C5 face 8", "FR-030", "unhappy (boundary)", 8, combat.ConfusionRowAttacksRandomCreature, "attacks_random_creature"},
		{"C7 face 9 lower edge", "FR-030", "unhappy (boundary)", 9, combat.ConfusionRowChoosesBehavior, "chooses_behavior"},
		{"C8 face 10", "FR-030", "unhappy (boundary)", 10, combat.ConfusionRowChoosesBehavior, "chooses_behavior"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.ConfusionRowFor(c.face)
			if err != nil || got != c.want || got.String() != c.wantName {
				t.Errorf("ConfusionRowFor(%d) = %v, %v, want %v [%s%s]", c.face, got, err, c.want, otSpec, c.requirement)
			}
		})
	}
	for _, face := range []int{0, -1, 11} {
		if _, err := combat.ConfusionRowFor(face); !errors.Is(err, combat.ErrFaceOutOfRange) {
			t.Errorf("ConfusionRowFor(%d) = %v, want ErrFaceOutOfRange", face, err)
		}
	}
	if combat.ConfusionRow(0).String() != "unknown" {
		t.Error("the zero row is unknown")
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-031, FR-032, FR-035, FR-036 - a random move is one of the four cardinal directions and uses all the movement; north-east, south-west and an invalid direction are rejected with an invalid input; the attack is one melee attack without movement; every turn says no Bonus Actions or Reactions (SRD 5.2.1, "Confusion", p. 117).
// Path: per case.
func TestConfusionBehaviours(t *testing.T) {
	t.Parallel()
	for _, d := range []combat.CompassDirection{combat.North, combat.East, combat.South, combat.West} {
		b, err := combat.NewConfusionMovesRandomly(d)
		if err != nil || b.Direction() != d || !b.TargetMoves() {
			t.Errorf("NewConfusionMovesRandomly(%v) = %v, %v", d, b, err)
		}
	}
	for _, d := range []combat.CompassDirection{0, combat.Northeast, combat.Southwest, combat.Southeast, combat.Northwest, 9} {
		if _, err := combat.NewConfusionMovesRandomly(d); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewConfusionMovesRandomly(%v) = %v, want ErrInvalidInput", d, err)
		}
	}
	if (combat.ConfusionDoesNothing{}).TargetMoves() || (combat.ConfusionNoCreatureInReach{}).TargetMoves() ||
		(combat.ConfusionAttacksRandomCreature{}).TargetMoves() || (combat.ConfusionAttacksRandomCreature{}).AttackCount() != 1 {
		t.Error("an attack is one attack, and a target that does nothing or attacks does not move (FR-033, FR-035)")
	}
	if !(combat.ConfusionTurn{}).CannotTakeBonusActionsOrReactions() {
		t.Error("every turn reports that no Bonus Action or Reaction can be taken (FR-036)")
	}
}
