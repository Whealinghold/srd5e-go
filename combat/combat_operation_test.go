package combat

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-foundations.md FR-028, FR-040 - an operation is
// snake_case with a span name srd5e.combat.<value>; anything else is
// ErrInvalidInput.
// Path: per case.
func TestNewCombatOperation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path, value, span string
		ok                      bool
	}{
		{"valid", "happy", "apply_damage", "srd5e.combat.apply_damage", true},
		{"starts with z", "unhappy (boundary)", "z_9", "srd5e.combat.z_9", true},
		{"middle letters and last characters", "unhappy (boundary)", "mz09_a", "srd5e.combat.mz09_a", true},
		{"starts just after z", "unhappy (boundary)", "{x", "srd5e.combat.{x", false},
		{"starts just before a", "unhappy (boundary)", "`x", "srd5e.combat.`x", false},
		{"contains a character just after z", "unhappy (boundary)", "a{", "srd5e.combat.a{", false},
		{"contains a character just before a", "unhappy (boundary)", "a`", "srd5e.combat.a`", false},
		{"contains a character just after 9", "unhappy (boundary)", "a:", "srd5e.combat.a:", false},
		{"contains a character just before 0", "unhappy (boundary)", "a/", "srd5e.combat.a/", false},
		{"contains a hyphen", "unhappy (invalid input)", "a-b", "srd5e.combat.a-b", false},
		{"contains a tilde", "unhappy (invalid input)", "a~", "srd5e.combat.a~", false},
		{"digit after the first", "unhappy (boundary)", "a1", "srd5e.combat.a1", true},
		{"empty", "unhappy (invalid input)", "", "srd5e.combat.", false},
		{"upper case", "unhappy (invalid input)", "Apply", "srd5e.combat.Apply", false},
		{"space", "unhappy (invalid input)", "apply damage", "srd5e.combat.apply damage", false},
		{"leading digit", "unhappy (invalid input)", "1x", "srd5e.combat.1x", false},
		{"span ends in another operation", "unhappy (invalid input)", "apply_damage", "srd5e.combat.apply", false},
		{"span has no dot before the value", "unhappy (invalid input)", "damage", "srd5e.combat.apply_damage", false},
		{"span outside the combat prefix", "unhappy (invalid input)", "apply_damage", "srd5e.d20_test.apply_damage", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			op, err := newCombatOperation(c.value, c.span)
			if c.ok {
				if err != nil || op.Value() != c.value || op.SpanName() != c.span {
					t.Fatalf("newCombatOperation = %+v, %v, want the pair", op, err)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || op != (combatOperation{}) {
				t.Fatalf("newCombatOperation(%q, %q) = %+v, %v, want zero and ErrInvalidInput", c.value, c.span, op, err)
			}
		})
	}
}
