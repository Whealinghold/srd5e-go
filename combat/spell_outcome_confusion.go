package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// ConfusionRow is a row of the Confusion "Behavior for the Turn" table (SRD 5.2.1, "Confusion", p. 117). The zero
// value is invalid.
type ConfusionRow int

// The rows of the table.
const (
	// ConfusionRowMovesRandomly is face 1.
	ConfusionRowMovesRandomly ConfusionRow = iota + 1
	// ConfusionRowDoesNothing is faces 2 to 6.
	ConfusionRowDoesNothing
	// ConfusionRowAttacksRandomCreature is faces 7 and 8.
	ConfusionRowAttacksRandomCreature
	// ConfusionRowChoosesBehavior is faces 9 and 10.
	ConfusionRowChoosesBehavior
)

// String returns the value of srd5e.combat.table_outcome for the row: moves_randomly, does_nothing,
// attacks_random_creature or chooses_behavior, or "unknown".
func (r ConfusionRow) String() string {
	return enumName([]string{"moves_randomly", "does_nothing", "attacks_random_creature", "chooses_behavior"}, int(r))
}

// confusionTable returns the 1d10 table C of spell-outcome-tables.md.
func confusionTable() outcomeTable[ConfusionRow] {
	return newOutcomeTable(dice.D10(), []outcomeBand[ConfusionRow]{
		{1, 1, ConfusionRowMovesRandomly}, {2, 6, ConfusionRowDoesNothing},
		{7, 8, ConfusionRowAttacksRandomCreature}, {9, 10, ConfusionRowChoosesBehavior}})
}

// ConfusionRowFor returns the row of the 1d10 face (SRD 5.2.1, "Confusion", p. 117). It returns an error matching
// ErrFaceOutOfRange for a face outside 1 to 10. It never panics.
func ConfusionRowFor(face int) (ConfusionRow, error) { return confusionTable().lookup(face) }

// ConfusionBehavior is the closed set of behaviours a Confusion roll gives for the turn. Variants:
// ConfusionMovesRandomly, ConfusionDoesNothing, ConfusionAttacksRandomCreature, ConfusionNoCreatureInReach,
// ConfusionTargetChoosesBehavior.
type ConfusionBehavior interface{ isConfusionBehavior() }

// ConfusionMovesRandomly: the target takes no action and uses all its movement to move in a cardinal direction.
// Build it with NewConfusionMovesRandomly; the zero value is invalid.
type ConfusionMovesRandomly struct{ direction CompassDirection }

// NewConfusionMovesRandomly returns the behaviour for a cardinal direction. It returns an error matching
// srderr.ErrInvalidInput for any other direction. It never panics.
func NewConfusionMovesRandomly(d CompassDirection) (ConfusionMovesRandomly, error) {
	if !d.IsCardinal() {
		return ConfusionMovesRandomly{}, invalidInput("confusion moves north, east, south or west")
	}
	return ConfusionMovesRandomly{d}, nil
}

// Direction returns the direction of the movement.
func (b ConfusionMovesRandomly) Direction() CompassDirection { return b.direction }

// TargetMoves returns true: the target uses all its movement.
func (ConfusionMovesRandomly) TargetMoves() bool { return true }

// ConfusionDoesNothing: the target does not move and takes no action.
type ConfusionDoesNothing struct{}

// TargetMoves returns false.
func (ConfusionDoesNothing) TargetMoves() bool { return false }

// ConfusionAttacksRandomCreature: the target does not move and takes the Attack action to make one melee attack
// against the creature (SRD 5.2.1, "Confusion", p. 117).
type ConfusionAttacksRandomCreature struct {
	// Target is the creature the attack is aimed at.
	Target CombatantID
}

// AttackCount returns 1: one melee attack, to which no Extra Attack or other source adds.
func (ConfusionAttacksRandomCreature) AttackCount() int { return 1 }

// TargetMoves returns false.
func (ConfusionAttacksRandomCreature) TargetMoves() bool { return false }

// ConfusionNoCreatureInReach: no creature is within reach, so the target takes no action and does not move.
type ConfusionNoCreatureInReach struct{}

// TargetMoves returns false.
func (ConfusionNoCreatureInReach) TargetMoves() bool { return false }

// ConfusionTargetChoosesBehavior: the target chooses its behavior for the turn; the host or player decides.
type ConfusionTargetChoosesBehavior struct{}

func (ConfusionMovesRandomly) isConfusionBehavior()         {}
func (ConfusionDoesNothing) isConfusionBehavior()           {}
func (ConfusionAttacksRandomCreature) isConfusionBehavior() {}
func (ConfusionNoCreatureInReach) isConfusionBehavior()     {}
func (ConfusionTargetChoosesBehavior) isConfusionBehavior() {}

// ConfusionTurn is the row rolled and the behaviour it gives for one turn. It rolls no end-of-turn save. Immutable.
type ConfusionTurn struct {
	row      ConfusionRow
	behavior ConfusionBehavior
}

// Row returns the row of the 1d10.
func (t ConfusionTurn) Row() ConfusionRow { return t.row }

// Behavior returns the behaviour for the turn.
func (t ConfusionTurn) Behavior() ConfusionBehavior { return t.behavior }

// CannotTakeBonusActionsOrReactions returns true: a confused target can't take Bonus Actions or Reactions, whatever
// the row (SRD 5.2.1, "Confusion", p. 117).
func (ConfusionTurn) CannotTakeBonusActionsOrReactions() bool { return true }

// validateConfusionCandidates checks the creatures within reach: at most dice.MaxSides (an error matching
// ErrSpellTableLimit), none zero and none repeated (an error matching srderr.ErrInvalidInput).
func validateConfusionCandidates(ids []CombatantID) error {
	if len(ids) > dice.MaxSides {
		return ErrSpellTableLimit
	}
	if slices.ContainsFunc(ids, CombatantID.IsZero) || len(slices.Compact(slices.Sorted(slices.Values(ids)))) != len(ids) {
		return invalidInput("confusion candidates must be distinct ids")
	}
	return nil
}
