package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-foundations.md FR-013 - NewCombatantRef rejects
// a zero id or a nil kind with ErrInvalidInput and accepts a valid pair.
// Path: per case.
func TestNewCombatantRef(t *testing.T) {
	t.Parallel()
	pc := combat.PlayerCharacterCombatant{ID: character.MustIdentifier("x")}
	cases := []struct {
		name, path string
		id         combat.CombatantID
		kind       combat.CombatantKind
		ok         bool
	}{
		{"valid", "happy", "a", pc, true},
		{"zero id", "unhappy (invalid input)", "", pc, false},
		{"nil kind", "unhappy (invalid input)", "a", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ref, err := combat.NewCombatantRef(c.id, c.kind)
			if c.ok {
				if err != nil || ref.ID != c.id || ref.Kind != c.kind {
					t.Fatalf("NewCombatantRef = %v, %v, want the pair and nil", ref, err)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || ref != (combat.CombatantRef{}) {
				t.Fatalf("NewCombatantRef = %v, %v, want the zero ref and ErrInvalidInput", ref, err)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-013 - two refs with the same
// id and different kinds are not equal; equal id and kind are equal.
// Path: unhappy (rule interaction).
func TestCombatantRefEquality(t *testing.T) {
	t.Parallel()
	pc := combat.PlayerCharacterCombatant{ID: character.MustIdentifier("x")}
	mon := combat.MonsterCombatant{ID: content.MonsterID("x")}
	a := combat.CombatantRef{ID: "g", Kind: pc}
	if a != (combat.CombatantRef{ID: "g", Kind: pc}) {
		t.Error("equal refs are not equal")
	}
	if a == (combat.CombatantRef{ID: "g", Kind: mon}) {
		t.Error("refs with the same id and different kinds are equal")
	}
	if a == (combat.CombatantRef{ID: "h", Kind: pc}) {
		t.Error("refs with different ids are equal")
	}
}
