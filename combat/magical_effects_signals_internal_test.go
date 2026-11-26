package combat

import "testing"

// Requirement: doc/specs/magical-effects.md section 7, FR-148 - each of the five operations is a valid operation with its span, and the nine result values and the operation names are pinned at their catalogue values (a rename breaks consumers' dashboards).
// Path: per case.
func TestMagicalEffectsSignalsAreValidAndPinned(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ value, span, wantValue string }{
		{valueOperationRemoveCombatantCurse, spanRemoveCombatantCurse, "remove_combatant_curse"},
		{valueOperationShiftCombatantShape, spanShiftCombatantShape, "shift_combatant_shape"},
		{valueOperationTeleportCombatant, spanTeleportCombatant, "teleport_combatant"},
		{valueOperationResolveEffectOrder, spanResolveEffectOrder, "resolve_effect_order"},
		{valueOperationAssessAreaLocation, spanAssessAreaLocation, "assess_area_location"},
	} {
		if _, err := newCombatOperation(c.value, c.span); err != nil {
			t.Errorf("operation %s with span %s is not valid: %v", c.value, c.span, err)
		}
		if c.value != c.wantValue || c.span != "srd5e.combat."+c.wantValue {
			t.Errorf("operation %q span %q, want %q", c.value, c.span, c.wantValue)
		}
	}
	for got, want := range map[string]string{
		valueResultCurseNotPresent: "curse_not_present", valueResultCurseNotACurse: "curse_not_a_curse",
		valueResultShiftExceptionNotHeld: "shift_exception_not_held", valueResultRevertNotDead: "revert_not_dead",
		valueResultTeleportDestinationNotSeen: "teleport_destination_not_seen", valueResultTeleportInvalidTouchedSet: "teleport_invalid_touched_set",
		valueResultOrderRejected: "order_rejected", valueResultAreaLocationExcluded: "area_location_excluded",
		valueResultAreaNoLinesConsidered: "area_no_lines_considered",
	} {
		if got != want {
			t.Errorf("result value %q, want %q", got, want)
		}
	}
}
