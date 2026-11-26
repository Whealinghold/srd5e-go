package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-076 - a recorded variant returns its events as a copy, one event; a refusal returns an empty list that is also a copy.
// Path: unhappy (immutability).
func TestAttackOutcomeEventsAreCopies(t *testing.T) {
	t.Parallel()
	p := progressOf(t, 2, 2, "")
	rec := p.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRecorded)
	clobbered := rec.Events()
	clobbered[0] = nil
	if rec.Events()[0] == nil {
		t.Error("Events() is not a copy")
	}
	chg := p.HandleWeapon(combat.WeaponChangeDrop, combat.WeaponChangeBeforeAttack).(combat.AttackActionWeaponChangeRecorded)
	clobbered = chg.Events()
	clobbered[0] = nil
	if chg.Events()[0] == nil {
		t.Error("Events() is not a copy")
	}
	refused := progressOf(t, 1, 1, "O").RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRefused)
	if got := refused.Events(); got == nil || len(got) != 0 {
		t.Errorf("refusal events = %v, want an empty non-nil slice", got)
	}
}
