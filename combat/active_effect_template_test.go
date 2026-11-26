package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func hasProblem(err error, p combat.EffectModelProblem) bool {
	return errors.Is(err, &combat.EffectModelError{Problem: p})
}

// Requirement: doc/specs/active-effects.md FR-009 - the actions, triggers and saves are built by validating constructors.
// Path: unhappy (invalid input).
func TestEffectActionConstructors(t *testing.T) {
	t.Parallel()
	d6 := dice.MustParseExpression("1d6")
	if _, err := combat.NewRepeatSave(0, 10); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("repeat save without an ability err = %v", err)
	}
	if s := aeRepeatSave(t, ability.Wisdom, 14); s.Ability() != ability.Wisdom || s.DifficultyClass() != 14 {
		t.Errorf("repeat save = %v", s)
	}
	if _, err := combat.NewEffectDamageSave(0, 10, combat.SaveSuccessHalfDamage); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("save without an ability err = %v", err)
	}
	if _, err := combat.NewEffectDamageSave(ability.Dexterity, 10, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("save without an outcome err = %v", err)
	}
	save := aeSaveOf(t, ability.Dexterity, 13, combat.SaveSuccessNoDamage)
	if save.IsZero() || save.Ability() != ability.Dexterity || save.DifficultyClass() != 13 || save.OnSuccess() != combat.SaveSuccessNoDamage {
		t.Errorf("save = %v", save)
	}
	for name, args := range map[string]struct {
		e dice.Expression
		t content.DamageType
		a combat.EffectDamageAudience
	}{"zero amount": {dice.Expression{}, content.DamageFire, combat.EffectAudienceSubject},
		"no type": {d6, 0, combat.EffectAudienceSubject}, "no audience": {d6, content.DamageFire, 0}} {
		if _, err := combat.NewDealDamage(args.e, args.t, args.a); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewDealDamage %s err = %v", name, err)
		}
	}
	deal := aeDeal(t, "1d6", content.DamageFire, combat.EffectAudienceTurnTaker)
	if _, has := deal.Save(); has || deal.Amount() != d6 || deal.DamageType() != content.DamageFire || deal.Audience() != combat.EffectAudienceTurnTaker {
		t.Errorf("deal = %v", deal)
	}
	if s, has := deal.WithSave(save).Save(); !has || s != save {
		t.Error("WithSave did not set the save")
	}
	if _, has := deal.Save(); has {
		t.Error("WithSave changed the original")
	}
	if _, err := combat.NewGrowStoredDamage(dice.Expression{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero growth err = %v", err)
	}
	if g := aeMust(combat.NewGrowStoredDamage(d6)); g.Extra() != d6 {
		t.Errorf("growth = %v", g)
	}
	if _, err := combat.NewDealStoredDamage(0, combat.EffectAudienceSubject); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("stored damage without a type err = %v", err)
	}
	if _, err := combat.NewDealStoredDamage(content.DamageFire, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("stored damage without an audience err = %v", err)
	}
	stored := aeMust(combat.NewDealStoredDamage(content.DamageFire, combat.EffectAudienceAreaMembers))
	if s, has := stored.WithSave(save).Save(); !has || s != save || stored.DamageType() != content.DamageFire || stored.Audience() != combat.EffectAudienceAreaMembers {
		t.Errorf("stored = %v", stored)
	}
	if _, has := stored.Save(); has {
		t.Error("a stored damage action has a save it was not given")
	}
}

// Requirement: doc/specs/active-effects.md FR-027 - a trigger needs an owner, a boundary and at least one non-nil action, copies its actions and compares by value.
// Path: unhappy (empty and minimal).
func TestEffectTrigger(t *testing.T) {
	t.Parallel()
	rs := aeRepeatSave(t, ability.Wisdom, 12)
	cases := map[string]struct {
		whose combat.EffectTurnOwner
		b     content.TurnBoundary
		acts  []combat.EffectAction
	}{
		"no owner": {0, content.BoundaryEnd, []combat.EffectAction{rs}}, "no boundary": {combat.EffectTurnOfSubject, 0, []combat.EffectAction{rs}},
		"no action": {combat.EffectTurnOfSubject, content.BoundaryEnd, nil}, "nil action": {combat.EffectTurnOfSubject, content.BoundaryEnd, []combat.EffectAction{nil}},
	}
	for name, c := range cases {
		if _, err := combat.NewEffectTrigger(c.whose, c.b, c.acts...); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	acts := []combat.EffectAction{rs}
	tr := aeTrigger(t, combat.EffectTurnOfCaster, content.BoundaryStart, acts...)
	acts[0] = nil
	got := tr.Actions()
	got[0] = nil
	if tr.Actions()[0] != combat.EffectAction(rs) || tr.Whose() != combat.EffectTurnOfCaster || tr.Boundary() != content.BoundaryStart {
		t.Error("a caller's slice changed the trigger")
	}
	if !tr.Equal(aeTrigger(t, combat.EffectTurnOfCaster, content.BoundaryStart, rs)) || tr.Equal(aeTrigger(t, combat.EffectTurnOfSubject, content.BoundaryStart, rs)) {
		t.Error("Equal")
	}
}

// Requirement: doc/specs/active-effects.md FR-009, FR-038 - a stage reports every problem: no duration, a bad condition (Exhaustion included), a gate that is not a RepeatSave, an action that cannot run where it is, an audience that is not valid there.
// Path: unhappy (invalid input).
func TestNewEffectStageProblems(t *testing.T) {
	t.Parallel()
	d4 := dice.MustParseExpression("1d4")
	grow := aeMust(combat.NewGrowStoredDamage(d4))
	areaDeal := aeDeal(t, "1d4", content.DamageFire, combat.EffectAudienceAreaMembers)
	takerDeal := aeDeal(t, "1d4", content.DamageFire, combat.EffectAudienceTurnTaker)
	areaTrigger := aeTrigger(t, combat.EffectTurnOfCaster, content.BoundaryEnd, areaDeal)
	cases := []struct {
		name string
		p    combat.EffectStageParams
		want combat.EffectModelProblem
	}{
		{"no duration", combat.EffectStageParams{}, combat.EffectModelStageDurationInvalid},
		{"zero rounds", combat.EffectStageParams{Duration: combat.EffectLastsRounds{}}, combat.EffectModelStageDurationInvalid},
		{"zero condition", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, Conditions: []content.ConditionKind{0}}, combat.EffectModelConditionInvalid},
		{"exhaustion", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, Conditions: []content.ConditionKind{content.ConditionExhaustion}}, combat.EffectModelConditionInvalid},
		{"gate not a save", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, Gate: takerDeal}, combat.EffectModelGateNotRepeatSave},
		{"repeat save on end", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, OnEnd: []combat.EffectAction{aeRepeatSave(t, ability.Wisdom, 10)}}, combat.EffectModelActionInvalid},
		{"nil on end", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, OnEnd: []combat.EffectAction{nil}}, combat.EffectModelActionInvalid},
		{"grow on end", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, OnEnd: []combat.EffectAction{grow}}, combat.EffectModelActionInvalid},
		{"turn taker on end", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, OnEnd: []combat.EffectAction{takerDeal}}, combat.EffectModelAudienceInvalid},
		{"area members in a trigger", combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}, Triggers: []combat.EffectTrigger{areaTrigger}}, combat.EffectModelAudienceInvalid},
	}
	for _, c := range cases {
		if _, err := combat.NewEffectStage(c.p); !hasProblem(err, c.want) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want %v [FR-009]", c.name, err, c.want)
		}
	}
	_, err := combat.NewEffectStage(combat.EffectStageParams{Conditions: []content.ConditionKind{0}, Gate: takerDeal})
	for _, p := range []combat.EffectModelProblem{combat.EffectModelStageDurationInvalid, combat.EffectModelConditionInvalid, combat.EffectModelGateNotRepeatSave} {
		if !hasProblem(err, p) {
			t.Errorf("a stage with three problems did not report %v: %v", p, err)
		}
	}
	if (&combat.EffectModelError{Problem: combat.EffectModelNoStages}).Error() == "" {
		t.Error("the error has no text")
	}
}

// Requirement: doc/specs/active-effects.md FR-009 - a valid stage copies its slices, exposes copies and compares by value.
// Path: unhappy (immutability).
func TestEffectStageIsImmutable(t *testing.T) {
	t.Parallel()
	conds := []content.ConditionKind{content.ConditionParalyzed}
	save := aeRepeatSave(t, ability.Wisdom, 10)
	trig := aeTrigger(t, combat.EffectTurnOfSubject, content.BoundaryEnd, save)
	onEnd := aeDeal(t, "1d4", content.DamageAcid, combat.EffectAudienceSubject)
	s := aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 3), Conditions: conds, Triggers: []combat.EffectTrigger{trig},
		EndsOnDamage: true, Gate: save, OnEnd: []combat.EffectAction{onEnd}})
	conds[0] = content.ConditionBlinded
	s.Conditions()[0] = content.ConditionBlinded
	s.Triggers()[0] = combat.EffectTrigger{}
	s.OnEnd()[0] = nil
	g, hasGate := s.Gate()
	if s.Conditions()[0] != content.ConditionParalyzed || !s.Triggers()[0].Equal(trig) || s.OnEnd()[0] != combat.EffectAction(onEnd) ||
		!hasGate || g != save || !s.EndsOnDamage() || s.Duration() != aeRounds(t, 3) || s.IsZero() {
		t.Error("a caller's slice changed the stage")
	}
	if !s.Equal(s) || s.Equal(combat.EffectStage{}) || !(combat.EffectStage{}).IsZero() {
		t.Error("Equal or IsZero")
	}
	if _, has := aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 1)}).Gate(); has {
		t.Error("a stage without a gate reports one")
	}
}

// Requirement: doc/specs/active-effects.md FR-009 (SRD 5.2.1, "Sleep", p. 163) - a template reports every problem: no stage, a zero stage, an open-ended stage before the last, a gated last stage, a remainder without an overall clock, an overall clock of another kind, stored damage that is missing or of another die, an invalid combining subject.
// Path: unhappy (invalid input).
func TestNewEffectTemplateProblems(t *testing.T) {
	t.Parallel()
	timedStage := aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 2)})
	open := aeStage(t, combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}})
	remainder := aeStage(t, combat.EffectStageParams{Duration: combat.EffectLastsRemainder{}})
	gated := aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 2), Gate: aeRepeatSave(t, ability.Wisdom, 10)})
	grow := aeMust(combat.NewGrowStoredDamage(dice.MustParseExpression("1d4")))
	boom := aeMust(combat.NewDealStoredDamage(content.DamageFire, combat.EffectAudienceSubject))
	growStage := aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 2), Triggers: []combat.EffectTrigger{aeTrigger(t, combat.EffectTurnOfCaster, content.BoundaryEnd, grow)}})
	boomStage := aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 2), OnEnd: []combat.EffectAction{boom}})
	st := func(s ...combat.EffectStage) []combat.EffectStage { return s }
	cases := []struct {
		name string
		p    combat.EffectTemplateParams
		want combat.EffectModelProblem
	}{
		{"no stage", combat.EffectTemplateParams{}, combat.EffectModelNoStages},
		{"zero stage", combat.EffectTemplateParams{Stages: st(combat.EffectStage{})}, combat.EffectModelStageDurationInvalid},
		{"indefinite stage first", combat.EffectTemplateParams{Stages: st(open, timedStage)}, combat.EffectModelOpenEndedStageNotLast},
		{"remainder stage first", combat.EffectTemplateParams{Stages: st(remainder, timedStage), Overall: aeRounds(t, 4)}, combat.EffectModelOpenEndedStageNotLast},
		{"gated last stage", combat.EffectTemplateParams{Stages: st(gated)}, combat.EffectModelGatedLastStage},
		{"remainder without overall", combat.EffectTemplateParams{Stages: st(timedStage, remainder)}, combat.EffectModelRemainderWithoutOverall},
		{"overall until a boundary", combat.EffectTemplateParams{Stages: st(timedStage), Overall: aeUntil(t, "a", content.BoundaryEnd)}, combat.EffectModelOverallInvalid},
		{"overall remainder", combat.EffectTemplateParams{Stages: st(timedStage), Overall: combat.EffectLastsRemainder{}}, combat.EffectModelOverallInvalid},
		{"overall of zero rounds", combat.EffectTemplateParams{Stages: st(timedStage), Overall: combat.EffectLastsRounds{}}, combat.EffectModelOverallInvalid},
		{"growth without stored damage", combat.EffectTemplateParams{Stages: st(growStage)}, combat.EffectModelStoredDamageMissing},
		{"stored damage action without stored damage", combat.EffectTemplateParams{Stages: st(boomStage)}, combat.EffectModelStoredDamageMissing},
		{"growth of another die", combat.EffectTemplateParams{Stages: st(growStage), StoredDamage: dice.MustParseExpression("2d6")}, combat.EffectModelStoredDamageDieMismatch},
		{"bad combining subject", combat.EffectTemplateParams{Stages: st(timedStage), CombiningSubject: 7}, combat.EffectModelCombiningSubjectInvalid},
	}
	for _, c := range cases {
		if _, err := combat.NewEffectTemplate(c.p); !hasProblem(err, c.want) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want %v [FR-009]", c.name, err, c.want)
		}
	}
	_, err := combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: st(open, gated, remainder), CombiningSubject: 7})
	for _, p := range []combat.EffectModelProblem{combat.EffectModelOpenEndedStageNotLast, combat.EffectModelRemainderWithoutOverall, combat.EffectModelCombiningSubjectInvalid} {
		if !hasProblem(err, p) {
			t.Errorf("a template with several problems did not report %v: %v", p, err)
		}
	}
	if hasProblem(err, combat.EffectModelGatedLastStage) {
		t.Error("a gated middle stage is reported as a gated last stage")
	}
}

// Requirement: doc/specs/active-effects.md FR-009, FR-037, FR-061 - a valid template exposes its parts, copies its stages and compares by value; the pilot templates build.
// Path: happy.
func TestEffectTemplateAccessors(t *testing.T) {
	t.Parallel()
	sleep := aeSleep(t, "target", 15)
	if sleep.IsZero() || len(sleep.Stages()) != 2 || sleep.Potency() != 0 || sleep.CombiningSubject() != combat.CombiningOnTarget || sleep.SurvivesSubjectDeath() {
		t.Errorf("sleep = %#v", sleep)
	}
	if o, has := sleep.Overall(); !has || o != aeRounds(t, 10) {
		t.Errorf("overall = %v, %v", o, has)
	}
	if _, has := sleep.StoredDamage(); has {
		t.Error("sleep stores damage")
	}
	sleep.Stages()[0] = combat.EffectStage{}
	if sleep.Stages()[0].IsZero() {
		t.Error("a caller's slice changed the template")
	}
	blast := aeDelayedBlast(t)
	if base, has := blast.StoredDamage(); !has || base != dice.MustParseExpression("12d6") {
		t.Errorf("stored = %v, %v", base, has)
	}
	mark := aeMark(t, 600)
	if !mark.SurvivesSubjectDeath() || mark.CombiningSubject() != combat.CombiningOnCaster {
		t.Error("the mark template does not survive death or combine per caster")
	}
	if !sleep.Equal(aeSleep(t, "target", 15)) || sleep.Equal(aeSleep(t, "target", 16)) || sleep.Equal(blast) || !(combat.EffectTemplate{}).IsZero() {
		t.Error("Equal or IsZero")
	}
	if !blast.Equal(blast) {
		t.Error("a template does not equal itself")
	}
	for _, tpl := range []combat.EffectTemplate{aeHoldPerson(t, 14), aeAcidArrow(t, "b"), aeFlamingSphere(t, 13)} {
		if tpl.IsZero() {
			t.Error("a pilot template is zero")
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-009 - the enums have names and an invalid value is not valid.
// Path: unhappy (invalid input).
func TestEffectEnumsNames(t *testing.T) {
	t.Parallel()
	names := []string{
		combat.CombiningOnTarget.String(), combat.CombiningOnCaster.String(), combat.EffectCombiningSubject(9).String(),
		combat.EffectTurnOfCaster.String(), combat.EffectTurnOfSubject.String(), combat.EffectTurnOfCreatureInArea.String(), combat.EffectTurnOwner(0).String(),
		combat.EffectAudienceSubject.String(), combat.EffectAudienceTurnTaker.String(), combat.EffectAudienceAreaMembers.String(), combat.EffectDamageAudience(0).String(),
		combat.SaveSuccessNoDamage.String(), combat.SaveSuccessHalfDamage.String(), combat.EffectSaveSuccess(0).String(),
	}
	want := []string{"target", "caster", "unknown", "caster", "subject", "creature_in_area", "unknown", "subject", "turn_taker", "area_members", "unknown", "none", "half", "unknown"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("name %d = %q, want %q", i, names[i], want[i])
		}
	}
	if combat.EffectCombiningSubject(9).Valid() || combat.EffectTurnOwner(0).Valid() || combat.EffectDamageAudience(4).Valid() || combat.EffectSaveSuccess(3).Valid() {
		t.Error("an undefined value is valid")
	}
}
