package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/active-effects.md FR-058, FR-063 - each reason has its snake_case name (the value of srd5e.combat.active_effects.rejection); a rejection lists its distinct reasons in the order met, compares by value, and is invalid when any problem is.
// Path: per case.
func TestActiveEffectRejection(t *testing.T) {
	t.Parallel()
	names := map[combat.ActiveEffectRejectionKind]string{
		combat.EffectRejectDuplicateID: "duplicate_id", combat.EffectRejectBindingMismatch: "binding_mismatch", combat.EffectRejectConcentrationConflict: "concentration_conflict",
		combat.EffectRejectInvalidOrder: "invalid_order", combat.EffectRejectMissingSaveAttempt: "missing_save_attempt", combat.EffectRejectCasterIncapacitated: "caster_incapacitated",
		combat.EffectRejectUnknownEffect: "unknown_effect", combat.EffectRejectNotTheCaster: "not_the_caster", combat.EffectRejectUsesConcentration: "uses_concentration",
		combat.EffectRejectNotTimeSpan: "not_time_span", combat.EffectRejectNoTargets: "no_targets", combat.EffectRejectDuplicateTarget: "duplicate_target",
		combat.EffectRejectNotMovable: "not_movable", combat.EffectRejectNewSubjectNotCreature: "new_subject_not_creature", combat.EffectRejectSameSubject: "same_subject",
		combat.EffectRejectLimitExceeded: "limit_exceeded", 0: "unknown", 17: "unknown",
	}
	for k, want := range names {
		if got := k.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", k, got, want)
		}
	}
	// A mixed rejection is invalid when any of its problems is (FR-063): a dismissal with an
	// unknown id (invalid) and another caster's effect (refused).
	reg := aeSeq(t, bane2("a", "t"))
	res := aeEnd(t, reg, aeMust(combat.NewDismissed("wiz", false, "a", "zzz"))).(combat.EndActiveEffectsRejected)
	if got := res.Rejection().Reasons(); !slices.Equal(got, []combat.ActiveEffectRejectionKind{combat.EffectRejectNotTheCaster, combat.EffectRejectUnknownEffect}) {
		t.Errorf("reasons = %v", got)
	}
	rec, opts := recording()
	if _, err := combat.NewEndActiveEffects(opts...).Run(t.Context(), combat.EndActiveEffectsCommand{Effects: reg, Cause: aeMust(combat.NewDismissed("wiz", false, "a", "zzz"))}); err != nil {
		t.Fatal(err)
	}
	if got := measured(rec, "srd5e.combat.operations"); got[0]["srd5e.combat.result_class"] != "invalid" || got[0]["srd5e.combat.result"] != "rejected_invalid" {
		t.Errorf("mixed rejection counted as %v, want invalid [FR-063]", got)
	}
	same := res.Rejection()
	if !same.Equal(res.Rejection()) || same.Equal(combat.ActiveEffectRejection{}) || len(combat.ActiveEffectRejection{}.Reasons()) != 0 {
		t.Error("Equal or Reasons")
	}
	first := combat.ActiveEffectRejection{}
	if first.Problems() != nil && len(first.Problems()) != 0 {
		t.Error("the zero rejection lists problems")
	}
	res.Rejection().Problems()[0] = combat.ActiveEffectProblem{}
	if res.Rejection().Problems()[0].Reason != combat.EffectRejectNotTheCaster {
		t.Error("a caller's slice changed the rejection")
	}
}
