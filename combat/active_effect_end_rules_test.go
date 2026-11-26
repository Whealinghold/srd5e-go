package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-042 (SRD 5.2.1, "Duration", p. 106) - a caster dismisses Time Span effects; Incapacitated is refused alone before any other check; each named effect is then checked for unknownEffect, notTheCaster, usesConcentration and notTimeSpan and every problem is listed with nothing ended.
// Path: per case.
func TestDismissed(t *testing.T) {
	t.Parallel()
	timed := aeSpec{id: "timed", spell: "bless", caster: "cleric", subject: aeCreature2("t"), template: aeSimpleNoT(10, 0, combat.CombiningOnTarget)}
	until := untilSpec("until", "t", content.BoundaryEnd, "")
	until.caster = "cleric"
	reg := aeSeq(t, timed, holdSpec("held", "orc"), until)
	dismiss := func(incapacitated bool, caster string, ids ...string) combat.EndActiveEffectsResult {
		var effectIDs []combat.ActiveEffectID
		for _, id := range ids {
			effectIDs = append(effectIDs, aeEffectID(id))
		}
		return aeEnd(t, reg, aeMust(combat.NewDismissed(combat.MustCombatantID(caster), incapacitated, effectIDs...)))
	}
	out := endedOf(t, dismiss(false, "cleric", "timed"))
	if !slices.Equal(aeIDs(out.Effects()), []string{"held", "until"}) || endReasons(out.Events())[0] != combat.EffectDismissed {
		t.Errorf("dismissal left %v with %v", aeIDs(out.Effects()), endReasons(out.Events()))
	}
	reasons := func(res combat.EndActiveEffectsResult) []combat.ActiveEffectRejectionKind {
		rej, ok := res.(combat.EndActiveEffectsRejected)
		if !ok {
			t.Fatalf("result = %#v, want EndActiveEffectsRejected", res)
		}
		var out []combat.ActiveEffectRejectionKind
		for _, p := range rej.Rejection().Problems() {
			out = append(out, p.Reason)
		}
		return out
	}
	cases := []struct {
		name string
		got  combat.EndActiveEffectsResult
		want []combat.ActiveEffectRejectionKind
	}{
		{"incapacitated caster, before anything else", dismiss(true, "cleric", "timed", "zzz"), []combat.ActiveEffectRejectionKind{combat.EffectRejectCasterIncapacitated}},
		{"unknown id", dismiss(false, "cleric", "zzz"), []combat.ActiveEffectRejectionKind{combat.EffectRejectUnknownEffect}},
		{"another caster's effect", dismiss(false, "wiz", "timed"), []combat.ActiveEffectRejectionKind{combat.EffectRejectNotTheCaster}},
		{"an effect that uses Concentration", dismiss(false, "wiz", "held"), []combat.ActiveEffectRejectionKind{combat.EffectRejectUsesConcentration}},
		{"an effect that is not a Time Span", dismiss(false, "cleric", "until"), []combat.ActiveEffectRejectionKind{combat.EffectRejectNotTimeSpan}},
		{"every problem is listed", dismiss(false, "wiz", "timed", "zzz", "until"), []combat.ActiveEffectRejectionKind{
			combat.EffectRejectNotTheCaster, combat.EffectRejectUnknownEffect, combat.EffectRejectNotTheCaster, combat.EffectRejectNotTimeSpan}},
	}
	for _, c := range cases {
		if got := reasons(c.got); !slices.Equal(got, c.want) {
			t.Errorf("%s: problems %v, want %v [FR-042]", c.name, got, c.want)
		}
	}
	if rej := dismiss(false, "cleric", "timed", "zzz").(combat.EndActiveEffectsRejected).Rejection(); rej.Equal(combat.ActiveEffectRejection{}) {
		t.Error("a rejection with an unknown id equals the empty one")
	}
	if none, ok := dismiss(false, "cleric").(combat.EndActiveEffectsNothingEnded); !ok || !none.Effects().Equal(reg) {
		t.Error("a dismissal naming nothing ended something")
	}
}

// Requirement: doc/specs/active-effects.md FR-060 (SRD 5.2.1, "Hunter's Mark", p. 141; "Hex", p. 140) - MoveEffectSubject re-points an effect keeping its rounds, binding, potency, stage and stored damage with one ActiveEffectSubjectMoved and no onEnd or condition plan; a template that survives its subject's death can be moved after SubjectDied; the reasons are checked in the order unknownEffect, notMovable, newSubjectNotCreature, sameSubject.
// Path: per case.
func TestMoveEffectSubject(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, mark("m", "ranger", "x"), bless("b", "c", "x", 0))
	reg = aeAdvance(t, reg, aeElapsed(t, 4), "").Effects()
	move := func(id string, to combat.EffectSubject) combat.EndActiveEffectsResult {
		return aeEnd(t, reg, aeMust(combat.NewMoveEffectSubject(aeEffectID(id), to)))
	}
	moved, ok := move("m", aeCreature2("y")).(combat.EndActiveEffectsMoved)
	if !ok {
		t.Fatalf("move = %#v", move("m", aeCreature2("y")))
	}
	e, before := aeEffect(t, moved.Effects(), "m"), aeEffect(t, reg, "m")
	if !e.OnCreature("y") || e.OnCreature("x") || e.RemainingRounds() != before.RemainingRounds() || e.Stage() != before.Stage() || e.Potency() != before.Potency() ||
		e.Anchored() != before.Anchored() || !slices.Equal(aeIDs(moved.Effects()), aeIDs(reg)) {
		t.Errorf("moved effect = %#v, want everything but the subject kept", e)
	}
	if b1, _ := e.Binding(); b1 != func() combat.ConcentrationBinding { b, _ := before.Binding(); return b }() {
		t.Error("the binding changed")
	}
	if len(moved.Events()) != 1 || len(moved.ConditionsToApply())+len(moved.ConditionsToRemove())+len(moved.DamagePlans()) != 0 {
		t.Errorf("events %v, want one and no plan", aeKinds(moved.Events()))
	}
	combattest.CheckEvent(t, moved.Events()[0], "active_effect_subject_moved", "y", "x", "ranger")
	ev := moved.Events()[0].(combat.ActiveEffectSubjectMoved)
	if !ev.Previous().(combat.CreatureEffectSubject).Is("x") || !ev.Subject().(combat.CreatureEffectSubject).Is("y") {
		t.Error("the event names the wrong subjects")
	}
	rejected := map[string]struct {
		res  combat.EndActiveEffectsResult
		want combat.ActiveEffectRejectionKind
	}{
		"unknown effect":     {move("zzz", aeCreature2("y")), combat.EffectRejectUnknownEffect},
		"not movable":        {move("b", aeCreature2("y")), combat.EffectRejectNotMovable},
		"a place":            {move("m", aePlace(t, "p")), combat.EffectRejectNewSubjectNotCreature},
		"the same subject":   {move("m", aeCreature2("x")), combat.EffectRejectSameSubject},
		"not movable first":  {move("b", aePlace(t, "p")), combat.EffectRejectNotMovable},
		"the place is first": {move("m", aePlace(t, "p")), combat.EffectRejectNewSubjectNotCreature},
	}
	for name, c := range rejected {
		rej, isRej := c.res.(combat.EndActiveEffectsRejected)
		if !isRej || !slices.Equal(rej.Rejection().Reasons(), []combat.ActiveEffectRejectionKind{c.want}) {
			t.Errorf("%s: = %#v, want %v [FR-060]", name, c.res, c.want)
		}
	}
	afterDeath := endedOf(t, aeEnd(t, reg, aeMust(combat.NewSubjectDied("x"))))
	if !slices.Equal(aeIDs(afterDeath.Effects()), []string{"m"}) {
		t.Fatalf("after the death %v, want the mark to survive", aeIDs(afterDeath.Effects()))
	}
	if _, ok := aeEnd(t, afterDeath.Effects(), aeMust(combat.NewMoveEffectSubject("m", aeCreature2("y")))).(combat.EndActiveEffectsMoved); !ok {
		t.Error("a mark that survived its subject's death could not be moved")
	}
	if _, err := combat.NewMoveEffectSubject("", aeCreature2("y")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank id err = %v", err)
	}
	if _, err := combat.NewMoveEffectSubject("m", nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil subject err = %v", err)
	}
}

// Requirement: doc/specs/active-effects.md FR-043, FR-038, FR-060 - SubjectDied ends the effects on that creature (reason subjectDied) without running their onEnd actions, except those whose template survives the subject's death; it never ends an effect the dead creature cast (a non-Concentration spell persists) and leaves a place alone.
// Path: unhappy (rule interaction).
func TestSubjectDied(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t,
		aeSpec{id: "aa", spell: "acid_arrow", caster: "wiz", subject: aeCreature2("orc"), template: aeAcidArrow(t, "orc")},
		mark("m", "ranger", "orc"),
		aeSpec{id: "cast", spell: "bless", caster: "orc", subject: aeCreature2("goblin"), template: aeSimpleNoT(10, 0, combat.CombiningOnTarget)},
		aeSpec{id: "sp", spell: "flaming_sphere", caster: "orc", subject: aePlace(t, "orc"), template: aeFlamingSphere(t, 13)})
	out := endedOf(t, aeEnd(t, reg, aeMust(combat.NewSubjectDied("orc"))))
	if !slices.Equal(aeIDs(out.Effects()), []string{"m", "cast", "sp"}) || !slices.Equal(endReasons(out.Events()), []combat.EffectEndReason{combat.EffectSubjectDied}) {
		t.Errorf("registry %v reasons %v, want the mark, the dead caster's effect and the place effect left", aeIDs(out.Effects()), endReasons(out.Events()))
	}
	if len(out.DamagePlans()) != 0 {
		t.Errorf("a dead subject ran onEnd: %v [FR-038]", out.DamagePlans())
	}
	if _, ok := aeEnd(t, reg, aeMust(combat.NewSubjectDied("nobody"))).(combat.EndActiveEffectsNothingEnded); !ok {
		t.Error("the death of a creature with no effect ended something")
	}
	if _, err := combat.NewSubjectDied(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero creature err = %v", err)
	}
}
