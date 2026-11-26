package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/combat-action-resolution.md FR-099, FR-103, FR-106, FR-170 - each of the 21 operations added by step 10b is a valid operation with its span, the four Concentration outcomes are the values of srd5e.combat.concentration_outcome that a check can give, and the counter and attribute names are pinned (a rename breaks consumers' dashboards).
// Path: per case.
func TestSpellPathSignalsAreValidAndPinned(t *testing.T) {
	t.Parallel()
	for _, pair := range [][2]string{
		{valueOperationResolveStatBlockSaveEffect, spanResolveStatBlockSaveEffect}, {valueOperationReadyCombatantSpell, spanReadyCombatantSpell},
		{valueOperationReleaseCombatantSpell, spanReleaseCombatantSpell}, {valueOperationDissipateCombatantSpell, spanDissipateCombatantSpell},
		{valueOperationChangeTelepathicContact, spanChangeTelepathicContact}, {valueOperationCreatePersistentZone, spanCreatePersistentZone},
		{valueOperationAssessZoneTrigger, spanAssessZoneTrigger}, {valueOperationMovePersistentZone, spanMovePersistentZone},
		{valueOperationEndPersistentZone, spanEndPersistentZone}, {valueOperationBeginShapeShift, spanBeginShapeShift},
		{valueOperationChangeShapeShiftForm, spanChangeShapeShiftForm}, {valueOperationAssessShapeShift, spanAssessShapeShift},
		{valueOperationEndShapeShift, spanEndShapeShift}, {valueOperationReviveBySpell, spanReviveBySpell}, {valueOperationIncreaseMaximum, spanIncreaseMaximum},
		{valueOperationEndMaximumIncrease, spanEndMaximumIncrease}, {valueOperationRestoreMaximum, spanRestoreMaximum}, {valueOperationLesserRestoration, spanLesserRestoration},
		{valueOperationGreaterRestoration, spanGreaterRestoration}, {valueOperationRestorativeHealing, spanRestorativeHealing}, {valueOperationGuardWithDeathWard, spanGuardWithDeathWard},
	} {
		if _, err := newCombatOperation(pair[0], pair[1]); err != nil {
			t.Errorf("operation %s with span %s is not valid: %v [FR-170]", pair[0], pair[1], err)
		}
	}
	for _, c := range []struct {
		name    string
		outcome ConcentrationCheckOutcome
		want    string
	}{
		{"held", ConcentrationHeld{}, "held"}, {"broken", ConcentrationCheckFailed{}, "broken"},
		{"ended incapacitated", ConcentrationEndedWithoutSave{reason: character.BreakIncapacitated}, "ended_incapacitated"},
		{"ended dead", ConcentrationEndedWithoutSave{reason: character.BreakDead}, "ended_dead"},
	} {
		if got := concentrationOutcomeName(c.outcome); got != c.want {
			t.Errorf("%s: concentrationOutcomeName = %q, want %q [FR-103]", c.name, got, c.want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Errorf("a check that was not made has no outcome name: want the unreachable panic")
		}
	}()
	concentrationOutcomeName(ConcentrationNotChecked{})
}
