package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-009 - a combining subject is valid only as one of the two named values: a negative value is not (zero is the target subject).
// Path: unhappy (invalid input).
func TestCombiningSubjectRejectsOutOfRangeValues(t *testing.T) {
	t.Parallel()
	for _, s := range []combat.EffectCombiningSubject{-1, 2, 3} {
		if s.Valid() {
			t.Errorf("EffectCombiningSubject(%d).Valid() = true, want false", s)
		}
	}
	if !combat.CombiningOnTarget.Valid() || !combat.CombiningOnCaster.Valid() {
		t.Error("a named combining subject is not valid")
	}
}

// Requirement: doc/specs/active-effects.md FR-009 - every EffectModelProblem is a positive value, so the zero value is not a problem, and each is distinct.
// Path: happy.
func TestEffectModelProblemsArePositiveAndDistinct(t *testing.T) {
	t.Parallel()
	all := []combat.EffectModelProblem{combat.EffectModelNoStages, combat.EffectModelStageDurationInvalid, combat.EffectModelOpenEndedStageNotLast,
		combat.EffectModelGatedLastStage, combat.EffectModelRemainderWithoutOverall, combat.EffectModelOverallInvalid, combat.EffectModelConditionInvalid,
		combat.EffectModelGateNotRepeatSave, combat.EffectModelActionInvalid, combat.EffectModelAudienceInvalid, combat.EffectModelStoredDamageMissing,
		combat.EffectModelStoredDamageDieMismatch, combat.EffectModelCombiningSubjectInvalid}
	seen := map[combat.EffectModelProblem]bool{}
	for _, p := range all {
		if p <= 0 || seen[p] {
			t.Errorf("problem %d is not positive or repeats", p)
		}
		seen[p] = true
	}
	if combat.EffectModelNoStages != 1 {
		t.Errorf("EffectModelNoStages = %d, want 1", combat.EffectModelNoStages)
	}
}

// Requirement: doc/specs/active-effects.md FR-009 - an *EffectModelError matches only another *EffectModelError with the same Problem: not a different problem, a nil one or another error type.
// Path: unhappy (invalid input).
func TestEffectModelErrorIsOnlyTheSameProblem(t *testing.T) {
	t.Parallel()
	e := &combat.EffectModelError{Problem: combat.EffectModelNoStages}
	if !errors.Is(e, &combat.EffectModelError{Problem: combat.EffectModelNoStages}) || !errors.Is(e, srderr.ErrInvalidInput) {
		t.Error("an effect model error does not match the same problem or ErrInvalidInput")
	}
	for name, target := range map[string]error{
		"another problem": &combat.EffectModelError{Problem: combat.EffectModelOverallInvalid},
		"a nil error":     (*combat.EffectModelError)(nil),
		"another type":    errors.New("other"),
	} {
		if e.Is(target) {
			t.Errorf("Is(%s) = true, want false", name)
		}
	}
	if (*combat.EffectModelError)(nil).Is(&combat.EffectModelError{Problem: combat.EffectModelNoStages}) {
		t.Error("a nil receiver matched a problem")
	}
}

// Requirement: doc/specs/active-effects.md FR-009 - a stage may hold every condition except Exhaustion, including those that sort before it.
// Path: happy.
func TestStageAcceptsEveryConditionButExhaustion(t *testing.T) {
	t.Parallel()
	for c := content.ConditionBlinded; c.Valid(); c++ {
		_, err := combat.NewEffectStage(combat.EffectStageParams{Duration: aeRounds(t, 1), Conditions: []content.ConditionKind{c}})
		if (c == content.ConditionExhaustion) != (err != nil) {
			t.Errorf("condition %v: err = %v, want an error only for Exhaustion", c, err)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-009 - triggers, stages and templates are equal only when every part is equal: a trigger differing by owner or boundary in either order, a stage differing in one part and a template differing by potency in either order are not equal.
// Path: unhappy (boundary).
func TestEffectModelEquality(t *testing.T) {
	t.Parallel()
	save := aeRepeatSave(t, ability.Wisdom, 10)
	trig := func(w combat.EffectTurnOwner, b content.TurnBoundary) combat.EffectTrigger {
		return aeTrigger(t, w, b, save)
	}
	pairs := [][2]combat.EffectTrigger{
		{trig(combat.EffectTurnOfCaster, content.BoundaryEnd), trig(combat.EffectTurnOfSubject, content.BoundaryEnd)},
		{trig(combat.EffectTurnOfSubject, content.BoundaryStart), trig(combat.EffectTurnOfSubject, content.BoundaryEnd)},
	}
	for _, p := range pairs {
		if p[0].Equal(p[1]) || p[1].Equal(p[0]) {
			t.Errorf("triggers %v and %v are equal", p[0], p[1])
		}
	}
	base := combat.EffectStageParams{Duration: aeRounds(t, 2), Conditions: []content.ConditionKind{content.ConditionProne}, Gate: save}
	stage := func(edit func(*combat.EffectStageParams)) combat.EffectStage {
		p := base
		edit(&p)
		return aeStage(t, p)
	}
	ref := stage(func(*combat.EffectStageParams) {})
	if !ref.Equal(stage(func(*combat.EffectStageParams) {})) {
		t.Error("identical stages are not equal")
	}
	for name, edit := range map[string]func(*combat.EffectStageParams){
		"duration":       func(p *combat.EffectStageParams) { p.Duration = aeRounds(t, 3) },
		"ends on damage": func(p *combat.EffectStageParams) { p.EndsOnDamage = true },
		"gate":           func(p *combat.EffectStageParams) { p.Gate = aeRepeatSave(t, ability.Wisdom, 11) },
		"conditions":     func(p *combat.EffectStageParams) { p.Conditions = []content.ConditionKind{content.ConditionBlinded} },
		"triggers": func(p *combat.EffectStageParams) {
			p.Triggers = []combat.EffectTrigger{trig(combat.EffectTurnOfCaster, content.BoundaryEnd)}
		},
		"on end": func(p *combat.EffectStageParams) {
			p.OnEnd = []combat.EffectAction{aeDeal(t, "1d4", content.DamageFire, combat.EffectAudienceSubject)}
		},
	} {
		if ref.Equal(stage(edit)) {
			t.Errorf("stages differing in %s are equal", name)
		}
	}
	tpl := func(potency int) combat.EffectTemplate {
		return aeTemplate(t, combat.EffectTemplateParams{Stages: []combat.EffectStage{ref, aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 1)})},
			Potency: combat.EffectPotency(potency), CombiningSubject: combat.CombiningOnTarget})
	}
	if tpl(1).Equal(tpl(2)) || tpl(2).Equal(tpl(1)) || !tpl(2).Equal(tpl(2)) {
		t.Error("templates are not compared by potency")
	}
}
