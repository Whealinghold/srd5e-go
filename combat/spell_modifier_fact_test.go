package combat_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/spell-granted-modifiers.md FR-070, FR-071 - the facts are descriptive: Haste's extra action allows Attack (one attack), Dash, Disengage, Hide and Utilize; its lethargy is Incapacitated and a Speed of 0; Slow's chance of failure is 25 percent; an entry's facts are a copy (SRD 5.2.1, "Haste", p. 139; "Slow", p. 163).
// Path: unhappy (immutability).
func TestFactsCarryTheirFigures(t *testing.T) {
	t.Parallel()
	a := combat.ExtraActionGranted{}
	want := []combat.StandardAction{combat.StandardActionAttack, combat.StandardActionDash, combat.StandardActionDisengage, combat.StandardActionHide, combat.StandardActionUtilize}
	if !reflect.DeepEqual(a.Allowed(), want) || a.AttackLimit() != 1 {
		t.Errorf("ExtraActionGranted = %v, %d, want %v, 1", a.Allowed(), a.AttackLimit(), want)
	}
	if l := (combat.LethargyOnEnd{}); l.Condition() != content.ConditionIncapacitated || l.SpeedFeet() != 0 {
		t.Errorf("LethargyOnEnd = %v, %d", l.Condition(), l.SpeedFeet())
	}
	if p := (combat.SpellFailureChancePercent{}).Percent(); p != 25 {
		t.Errorf("SpellFailureChancePercent = %d, want 25", p)
	}
	e := combat.LookupSpellModifiers("haste").(combat.SpellModifiersKnown).Entry()
	f := e.Facts()
	f[0] = nil
	if e.Facts()[0] == nil {
		t.Error("Facts returned the entry's own slice")
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-071 - a fact changes no numeric result: the Speed and Armor Class of a hasted creature are the same whatever the host does with the facts of the entry.
// Path: unhappy (rule interaction).
func TestFactsChangeNoNumber(t *testing.T) {
	t.Parallel()
	src := []combat.SpellModifierSource{smPlain(t, "haste", 0)}
	speeds := func() combat.MovementSpeeds { return luMust(combat.QuerySpeeds(src, cid("a"), smSpeeds(t, 30, -1))) }
	before := speeds()
	facts := combat.LookupSpellModifiers("haste").(combat.SpellModifiersKnown).Entry().Facts()
	if len(facts) != 2 {
		t.Fatalf("Haste has %d facts, want 2", len(facts))
	}
	if after := speeds(); after != before || after.Walk() != 60 || finalAC(12, smAC(t, src...)) != 14 {
		t.Errorf("hasted Speed %d and Armor Class %d changed with the facts", after.Walk(), finalAC(12, smAC(t, src...)))
	}
}
