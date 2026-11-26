package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/active-effects.md FR-054 - each event of the family has its snake_case Kind and lists the creature the effect is on, then the caster, without a repeat; for an effect on a place the caster only; ActiveEffectSubjectMoved lists the new subject, the old, then the caster.
// Path: per case.
func TestActiveEffectEventFamily(t *testing.T) {
	t.Parallel()
	blast := aeSeq(t, aeSpec{id: "bf", spell: "delayed_blast_fireball", caster: "wiz", subject: aePlace(t, "bead"), template: aeDelayedBlast(t), active: "wiz"})
	grown := aeAdvance(t, blast, aeEndTick(t, "wiz"), "wiz").Events()[0]
	sleep := aeSeq(t, aeSpec{id: "sl", spell: "sleep", caster: "wiz", subject: aeCreature2("orc"), template: aeSleep(t, "orc", 13), active: "wiz"})
	adv := aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: sleep, Tick: aeEndTick(t, "orc"), Saves: aeSaves("orc", ability.Wisdom, d20.ChooseToFail{})}).(combat.AdvanceActiveEffectsApplied).Events()
	started := aeStartCmd2(t, aeSpec{id: "s", spell: "bless", caster: "wiz", subject: aeCreature2("wiz"), template: aeSimpleNoT(3, 0, combat.CombiningOnTarget)}).Events()[0]
	removed := aeEnd(t, sleep, combat.NewRemovedByEffect("sl")).(combat.EndActiveEffectsEnded).Events()[0]
	reg := aeSeq(t, mark("m", "ranger", "x"))
	moved := aeEnd(t, reg, aeMust(combat.NewMoveEffectSubject("m", aeCreature2("ranger")))).(combat.EndActiveEffectsMoved).Events()[0]
	cases := []struct {
		name  string
		e     combat.Event
		kind  string
		wants []combat.CombatantID
	}{
		{"started on the caster", started, "active_effect_started", []combat.CombatantID{"wiz"}},
		{"stored damage on a place", grown, "active_effect_stored_damage_grew", []combat.CombatantID{"wiz"}},
		{"a repeat save", adv[0], "active_effect_repeat_save_made", []combat.CombatantID{"orc", "wiz"}},
		{"a stage advanced", adv[1], "active_effect_stage_advanced", []combat.CombatantID{"orc", "wiz"}},
		{"ended", removed, "active_effect_ended", []combat.CombatantID{"orc", "wiz"}},
		{"moved onto the caster", moved, "active_effect_subject_moved", []combat.CombatantID{"ranger", "x"}},
	}
	var all []combat.Event
	for _, c := range cases {
		combattest.CheckEvent(t, c.e, c.kind, c.wants...)
		all = append(all, c.e)
	}
	combattest.CheckEventKinds(t, all)
	if got := moved.(combat.ActiveEffectSubjectMoved).Combatants(); !slices.Equal(got, []combat.CombatantID{"ranger", "x"}) {
		t.Errorf("moved onto the caster lists %v, want the caster once", got)
	}
	if adv[0].(combat.ActiveEffectRepeatSaveMade).SaveKind().String() != "stage_gate" || combat.EffectSaveRepeat.String() != "repeat" ||
		combat.EffectSaveDamage.String() != "damage" || combat.EffectSaveKind(0).String() != "unknown" {
		t.Error("EffectSaveKind names")
	}
}

func aeStartCmd2(t *testing.T, s aeSpec) combat.StartActiveEffectStarted {
	t.Helper()
	res, err := combat.NewStartActiveEffect().Run(t.Context(), s.command(t, combat.ActiveEffects{}))
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.StartActiveEffectStarted)
}

// aeEnd runs EndActiveEffects with a cause.
func aeEnd(t *testing.T, reg combat.ActiveEffects, cause combat.EndingCause) combat.EndActiveEffectsResult {
	t.Helper()
	res, err := combat.NewEndActiveEffects().Run(t.Context(), combat.EndActiveEffectsCommand{Effects: reg, Cause: cause})
	if err != nil {
		t.Fatal(err)
	}
	return res
}
