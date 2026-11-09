package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-fighter.md Table E, FR-071, FR-072, FR-073, FR-075, FR-076 - ActionSurge.Evaluate checks in the order
// feature not gained (below level 2), not own turn, already used this turn, no uses remaining and reports the first failing one; a
// grant expends one use. Rows AS1 to AS7 (AS8, no Fighter levels, is decided by the use case).
// Path: per case (unhappy: boundary, rule interaction, state error).
func TestActionSurgeEvaluateTableE(t *testing.T) {
	t.Parallel()
	own, other := character.FighterOwnTurn, character.FighterOtherTurn
	unused, used := character.FighterActionSurgeUnused, character.FighterActionSurgeUsed
	for _, c := range []struct {
		name     string
		level    int
		turn     character.FighterTurnOwnership
		perTurn  character.FighterPerTurnUse
		expended int
		refused  character.FighterReason
		newCount int
	}{
		{"AS1", 1, own, unused, 0, character.FighterFeatureNotGained, 0},
		{"AS2", 2, other, used, 1, character.FighterNotOwnTurn, 0},
		{"AS3", 17, own, used, 0, character.FighterAlreadyUsedThisTurn, 0},
		{"AS4", 17, own, used, 2, character.FighterAlreadyUsedThisTurn, 0},
		{"AS5", 2, own, unused, 1, character.FighterNoUsesRemaining, 0},
		{"AS6", 2, own, unused, 0, 0, 1},
		{"AS7", 17, own, unused, 1, 0, 2},
		{"level 17 with both uses left", 17, own, unused, 0, 0, 1},
		{"level 16 has one use only", 16, own, unused, 1, character.FighterNoUsesRemaining, 0},
		{"an invalid turn is not an own turn", 2, 0, unused, 0, character.FighterNotOwnTurn, 0},
		{"an invalid per-turn state is a used one", 2, own, 0, 0, character.FighterAlreadyUsedThisTurn, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := fgtState(1, c.expended, 1)
			got := character.ActionSurge{}.Evaluate(fgtStats(t, c.level), in, c.turn, c.perTurn)
			switch r := got.(type) {
			case character.ActionSurgeRefused:
				if r.Reason != c.refused {
					t.Errorf("refused with %v, want %v", r.Reason, c.refused)
				}
			case character.ActionSurgeGranted:
				if c.refused != 0 || r.State() != fgtState(1, c.newCount, 1) {
					t.Errorf("granted with state %v, want refused %v or %d expended", r.State(), c.refused, c.newCount)
				}
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-073, FR-154 - the turn ownership and per-turn state are enums with names, valid for their
// two values only.
// Path: per case.
func TestActionSurgeInputEnums(t *testing.T) {
	t.Parallel()
	for _, v := range []int{-1, 3, 99} {
		if character.FighterTurnOwnership(v).Valid() || character.FighterPerTurnUse(v).Valid() || character.FighterTurnOwnership(v).String() != "unknown" || character.FighterPerTurnUse(v).String() != "unknown" {
			t.Errorf("value %d must be unknown and not valid", v)
		}
	}
	for v, name := range map[character.FighterTurnOwnership]string{character.FighterOwnTurn: "own_turn", character.FighterOtherTurn: "other_turn", 0: "unknown"} {
		if v.String() != name || v.Valid() != (name != "unknown") {
			t.Errorf("turn %d = %q valid %v", int(v), v.String(), v.Valid())
		}
	}
	for v, name := range map[character.FighterPerTurnUse]string{character.FighterActionSurgeUnused: "unused", character.FighterActionSurgeUsed: "used", 0: "unknown"} {
		if v.String() != name || v.Valid() != (name != "unknown") {
			t.Errorf("per-turn %d = %q valid %v", int(v), v.String(), v.Valid())
		}
	}
}
