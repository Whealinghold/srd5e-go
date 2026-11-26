package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/active-effects.md FR-012, FR-014, table D row D8 - a start is refused with duplicateID, bindingMismatch or concentrationConflict, registry unchanged and no events; when several apply the order is duplicateID, bindingMismatch, concentrationConflict.
// Path: per case.
func TestStartRejections(t *testing.T) {
	t.Parallel()
	base := aeSeq(t, holdPersonSpec())
	hp := holdPersonSpec()
	hp.id = "hp2"
	good := hp.command(t, base)
	cases := []struct {
		name string
		edit func(c *combat.StartActiveEffectCommand)
		want combat.ActiveEffectRejectionKind
	}{
		{"duplicate id", func(c *combat.StartActiveEffectCommand) { c.ID = "hp" }, combat.EffectRejectDuplicateID},
		{"a Concentration spell with no binding", func(c *combat.StartActiveEffectCommand) { c.Binding = combat.ConcentrationBinding{} }, combat.EffectRejectBindingMismatch},
		{"a binding for a spell that needs none", func(c *combat.StartActiveEffectCommand) { c.RequiresConcentration = false }, combat.EffectRejectBindingMismatch},
		{"D8 a binding on an other origin", func(c *combat.StartActiveEffectCommand) { c.Origin = aeMust(combat.NewOtherEffectOrigin("ring")) }, combat.EffectRejectBindingMismatch},
		{"a binding on another spell", func(c *combat.StartActiveEffectCommand) { c.Binding = aeBinding(t, "wiz", "bless") }, combat.EffectRejectBindingMismatch},
		{"a binding of another caster", func(c *combat.StartActiveEffectCommand) { c.Binding = aeBinding(t, "cleric", "hold_person") }, combat.EffectRejectBindingMismatch},
		{"a binding on a feature", func(c *combat.StartActiveEffectCommand) {
			c.Binding = aeMust(combat.NewConcentrationBinding("wiz", character.NewConcentrationSubjectFeature("x")))
		}, combat.EffectRejectBindingMismatch},
		{"the caster is bound to another subject", func(c *combat.StartActiveEffectCommand) {
			c.Origin, c.Binding = aeSpellOrigin(t, "bane", 1), aeBinding(t, "wiz", "bane")
		}, combat.EffectRejectConcentrationConflict},
		{"duplicate id before a binding mismatch", func(c *combat.StartActiveEffectCommand) { c.ID, c.Binding = "hp", combat.ConcentrationBinding{} }, combat.EffectRejectDuplicateID},
		{"a binding mismatch before a conflict", func(c *combat.StartActiveEffectCommand) {
			c.Origin, c.Binding = aeSpellOrigin(t, "bane", 1), aeBinding(t, "wiz", "hold_person")
		}, combat.EffectRejectBindingMismatch},
	}
	for _, c := range cases {
		cmd := good
		c.edit(&cmd)
		res, err := runStart(t, cmd)
		rej, ok := res.(combat.StartActiveEffectRejected)
		if err != nil || !ok || !slices.Equal(rej.Rejection().Reasons(), []combat.ActiveEffectRejectionKind{c.want}) {
			t.Errorf("%s: = %#v, %v, want rejected %v [FR-014]", c.name, res, err, c.want)
		}
	}
	if base.Len() != 1 {
		t.Error("a rejected start changed the input registry")
	}
}

// Requirement: doc/specs/active-effects.md FR-014 - the same caster, the same Concentration subject and another target is allowed (Hold Person at a higher level).
// Path: happy.
func TestStartSecondTargetOfOneConcentration(t *testing.T) {
	t.Parallel()
	second := holdPersonSpec()
	second.id, second.subject = "hp2", aeCreature2("goblin")
	reg := aeSeq(t, holdPersonSpec(), second)
	if got := reg.BoundTo(aeBinding(t, "wiz", "hold_person")); len(got) != 2 {
		t.Errorf("bound effects = %d, want 2", len(got))
	}
}
