package combat

import "testing"

// Requirement: doc/specs/returning-to-life.md section 7 and combat-action-resolution.md FR-170 - the operation and span of ReviveCombatant are a valid
// pair, and its four refusal names are pinned to their catalogue literals (a rename breaks consumers' dashboards).
// Path: unhappy (drift between code and catalogue).
func TestReviveCombatantSignalsArePinned(t *testing.T) {
	t.Parallel()
	if _, err := newCombatOperation(valueOperationReviveCombatant, spanReviveCombatant); err != nil {
		t.Errorf("operation %s with span %s is not valid: %v", valueOperationReviveCombatant, spanReviveCombatant, err)
	}
	for got, want := range map[string]string{
		valueOperationReviveCombatant:      "revive_combatant",
		spanReviveCombatant:                "srd5e.combat.revive_combatant",
		valueResultReviveNotDead:           "revive_not_dead",
		valueResultReviveRefused:           "revive_refused",
		valueResultReviveHitPointsBelowOne: "revive_hit_points_below_one",
		valueResultReviveMaximumBelowOne:   "revive_maximum_below_one",
	} {
		if got != want {
			t.Errorf("constant = %q, want %q", got, want)
		}
	}
}
