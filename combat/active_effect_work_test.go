package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/active-effects.md FR-045 - ending an effect plans the removal of the conditions it held unless another effect remaining in the registry on the same creature holds the kind; ending the last holder plans it; ending two holders in one command plans it once.
// Path: unhappy (rule interaction).
func TestConditionRemovalIsOmittedWhileAnotherEffectHoldsTheKind(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, holdSpec("h1", "orc"), aeSpec{id: "h2", spell: "other_hold", caster: "cleric", subject: aeCreature2("orc"), template: aeSimpleNoT(10, 0, combat.CombiningOnTarget),
		conditions: []content.ConditionKind{content.ConditionParalyzed, content.ConditionIncapacitated}})
	first := endedOf(t, aeEnd(t, reg, combat.NewRemovedByEffect("h1")))
	if len(first.ConditionsToRemove()) != 0 {
		t.Errorf("removal %v planned while h2 still holds Paralyzed [FR-045]", first.ConditionsToRemove())
	}
	last := endedOf(t, aeEnd(t, first.Effects(), combat.NewRemovedByEffect("h2")))
	want := []combat.EffectConditionPlan{{Effect: "h2", Target: "orc", Kind: content.ConditionParalyzed}, {Effect: "h2", Target: "orc", Kind: content.ConditionIncapacitated}}
	if !slices.Equal(last.ConditionsToRemove(), want) {
		t.Errorf("last holder removal = %v, want %v", last.ConditionsToRemove(), want)
	}
	both := endedOf(t, aeEnd(t, reg, combat.NewRemovedByEffect("h1", "h2")))
	if got := len(both.ConditionsToRemove()); got != 2 {
		t.Errorf("two holders ended together planned %d removals, want one per kind (2): %v", got, both.ConditionsToRemove())
	}
}

// Requirement: doc/specs/active-effects.md FR-045 - advancing a stage plans no removal for a condition another effect on the creature still holds, nor a removal and an application of a condition both stages hold.
// Path: unhappy (rule interaction).
func TestStageAdvanceKeepsSharedConditions(t *testing.T) {
	t.Parallel()
	holder := aeSpec{id: "hold", spell: "hold_person", caster: "c", subject: aeCreature2("orc"), template: aeHoldPerson(t, 10), conditions: []content.ConditionKind{content.ConditionIncapacitated}}
	sleep := aeSpec{id: "sl", spell: "sleep", caster: "wiz", subject: aeCreature2("orc"), template: aeSleep(t, "orc", 13), active: "wiz", conditions: []content.ConditionKind{content.ConditionIncapacitated}}
	reg := aeSeq(t, holder, sleep)
	out := aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Saves: map[combat.CombatantID]map[ability.Ability]d20.SaveAttempt{
		"orc": {ability.Wisdom: d20.ChooseToFail{}}}}).(combat.AdvanceActiveEffectsApplied)
	if len(out.ConditionsToRemove()) != 0 {
		t.Errorf("removal %v planned for a condition the other effect still holds", out.ConditionsToRemove())
	}
	same := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{
		aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 1), Conditions: []content.ConditionKind{content.ConditionPoisoned}}),
		aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 1), Conditions: []content.ConditionKind{content.ConditionPoisoned}})}}))
	reg = aeSeq(t, aeSpec{id: "pp", spell: "bless", caster: "c", subject: aeCreature2("orc"), template: same, conditions: []content.ConditionKind{content.ConditionPoisoned}})
	adv := aeAdvance(t, reg, aeElapsed(t, 1), "")
	if len(adv.ConditionsToRemove())+len(adv.ConditionsToApply()) != 0 || aeEffect(t, adv.Effects(), "pp").Stage() != 1 {
		t.Errorf("a condition both stages hold was removed or applied: %v %v", adv.ConditionsToRemove(), adv.ConditionsToApply())
	}
}
