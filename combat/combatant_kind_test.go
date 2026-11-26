package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func kindName(k combat.CombatantKind) string {
	switch k.(type) {
	case combat.PlayerCharacterCombatant:
		return "player_character"
	case combat.MonsterCombatant:
		return "monster"
	}
	return "other"
}

// Requirement: doc/specs/combat-foundations.md FR-010, FR-011 - CombatantKind is
// the closed set of two value variants, equal by value, not equal across
// variants even when the underlying strings match.
// Path: per case.
func TestCombatantKind(t *testing.T) {
	t.Parallel()
	pc := combat.PlayerCharacterCombatant{ID: character.MustIdentifier("x")}
	mon := combat.MonsterCombatant{ID: content.MonsterID("x")}
	cases := []struct {
		name, path string
		a, b       combat.CombatantKind
		equal      bool
	}{
		{"same character", "happy", pc, combat.PlayerCharacterCombatant{ID: character.MustIdentifier("x")}, true},
		{"same monster", "happy", mon, combat.MonsterCombatant{ID: content.MonsterID("x")}, true},
		{"other character", "unhappy (boundary)", pc, combat.PlayerCharacterCombatant{ID: character.MustIdentifier("y")}, false},
		{"other monster", "unhappy (boundary)", mon, combat.MonsterCombatant{ID: content.MonsterID("y")}, false},
		{"same string, other variant", "unhappy (boundary)", pc, mon, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.a == c.b; got != c.equal {
				t.Errorf("%v == %v is %v, want %v", c.a, c.b, got, c.equal)
			}
		})
	}
	if kindName(pc) != "player_character" || kindName(mon) != "monster" {
		t.Error("the type switch does not tell the variants apart")
	}
}
