package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/magical-effects.md FR-030 to FR-033, FR-047 to FR-055 (SRD 5.2.1, "Polymorph", "Remove Curse") - the effects a hero carries into a Polymorph form are the input of Remove Curse: the curse ends while shifted, the host carries the reduced effects into the revert, and the curse does not come back; a second Remove Curse then finds nothing.
// Path: happy (several use cases chained), then unhappy (state error).
func TestMagicalEffectsCurseRemovedWhileShiftedStaysGone(t *testing.T) {
	t.Parallel()
	ctx, hero := context.Background(), combat.MustCombatantID("hero")
	polymorph := grpAMust(combat.SpellSource(content.MustSpellID("polymorph")))
	conditions := combat.MustConditionSet(nil, nil)
	shifted, err := combat.NewShiftCombatantShape().Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.ShiftRequest{Command: combat.ShiftShapeCommand{
		Creature: hero, State: combat.TrueForm(), Conditions: conditions, Effects: grpAEffects(), Effect: polymorph}}})
	if err != nil {
		t.Fatal(err)
	}
	applied, ok := shifted.(combat.ShiftShapeApplied)
	if !ok || applied.Effects().Len() != 2 {
		t.Fatalf("shift = %#v, want applied with both effects carried over", shifted)
	}
	remove := func(effects combat.OngoingEffects) combat.RemoveCombatantCurseResult {
		t.Helper()
		res, err := combat.NewRemoveCombatantCurse().Run(ctx, combat.RemoveCombatantCurseCommand{Creature: hero, Effects: effects, Curse: grpAMust(combat.NewOngoingEffectID("c1")), Means: combat.RemoveCurseSpell})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	freed, ok := remove(applied.Effects()).(combat.RemoveCurseApplied)
	if !ok || !slices.Equal(grpAEffectIDs(freed.Effects()), []string{"bless"}) {
		t.Fatalf("remove curse while shifted = %#v, want only Bless left", freed)
	}
	reverted, err := combat.NewShiftCombatantShape().Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.RevertRequest{Command: combat.RevertToTrueFormCommand{
		Creature: hero, State: applied.State(), Conditions: applied.Conditions(), Effects: freed.Effects()}}})
	if err != nil {
		t.Fatal(err)
	}
	back, ok := reverted.(combat.RevertToTrueFormApplied)
	if !ok || back.State().IsShifted() || !slices.Equal(grpAEffectIDs(back.Effects()), []string{"bless"}) {
		t.Fatalf("revert = %#v, want the true form with only Bless", reverted)
	}
	if _, ok := remove(back.Effects()).(combat.RemoveCurseNotPresent); !ok {
		t.Error("a second Remove Curse after the revert must find nothing")
	}
}
