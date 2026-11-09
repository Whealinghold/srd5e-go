package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/magic-item-effects.md FR-044 (SRD 5.2.1, as cited per row, pp. 223-251) - every row of table "Last charge rules"; an id not in
// the table has NoLastChargeEffect (Q7).
// Path: per case.
func TestLastChargeRulesTable(t *testing.T) {
	t.Parallel()
	d8 := character.RegainsDice(dice.MustExpression(1, dice.D8(), 2))
	d12 := character.RegainsDice(dice.MustExpression(1, dice.D12(), 1))
	rolled := func(c character.LastChargeConsequence, r character.LastChargeRegain) character.LastChargeRule {
		return must(character.NewLastChargeRolled(c, r))
	}
	destroyed := rolled(character.LastChargeDestroyed, character.NoRegain())
	rows := map[string]character.LastChargeRule{
		"wand_of_binding": destroyed, "wand_of_enemy_detection": destroyed, "wand_of_fear": destroyed, "wand_of_fireballs": destroyed,
		"wand_of_lightning_bolts": destroyed, "wand_of_magic_missiles": destroyed, "wand_of_paralysis": destroyed, "wand_of_polymorph": destroyed,
		"wand_of_web": destroyed, "wand_of_wonder": destroyed, "rod_of_resurrection": destroyed, "staff_of_charming": destroyed, "staff_of_frost": destroyed,
		"staff_of_fire": destroyed, "staff_of_swarming_insects": destroyed, "staff_of_healing": destroyed,
		"staff_of_striking":       rolled(character.LastChargeBecomesNonmagical, character.NoRegain()),
		"staff_of_the_woodlands":  rolled(character.LastChargeBecomesNonmagical, character.NoRegain()),
		"staff_of_power":          rolled(character.LastChargeRetainsBonusLosesOtherProperties, d8),
		"staff_of_the_magi":       rolled(character.LastChargeNone, d12),
		"ring_of_three_wishes":    must(character.NewLastChargeAutomatic(character.LastChargeBecomesNonmagical)),
		"gem_of_brightness":       must(character.NewLastChargeAutomatic(character.LastChargeBecomesNonmagical)),
		"scarab_of_protection":    must(character.NewLastChargeAutomatic(character.LastChargeDestroyed)),
		"wand_of_magic_detection": character.NoLastChargeEffect{}, "wand_of_secrets": character.NoLastChargeEffect{}, "wand_of_the_war_mage_plus": character.NoLastChargeEffect{},
		"flame_tongue": character.NoLastChargeEffect{},
	}
	for id, want := range rows {
		if got := (character.LastChargeRules{}).For(mid(id)); got != want {
			t.Errorf("%s = %+v, want %+v", id, got, want)
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-037, FR-039, FR-040 - a rolled rule gives onOne on face 1, regain dice on face 20 and nothing on any
// other face; none and NoRegain mean no effect.
// Path: unhappy (boundary).
func TestLastChargeRolledForFace(t *testing.T) {
	t.Parallel()
	d8 := character.RegainsDice(dice.MustExpression(1, dice.D8(), 2))
	r := must(character.NewLastChargeRolled(character.LastChargeDestroyed, d8))
	for face, want := range map[int]struct {
		c      character.LastChargeConsequence
		regain bool
	}{0: {character.LastChargeNone, false}, 1: {character.LastChargeDestroyed, false}, 2: {character.LastChargeNone, false}, 19: {character.LastChargeNone, false},
		20: {character.LastChargeNone, true}, 21: {character.LastChargeNone, false}} {
		c, regain := r.ForFace(face)
		_, has := regain.Dice()
		if c != want.c || has != want.regain {
			t.Errorf("face %d = %v %v, want %v %v", face, c, has, want.c, want.regain)
		}
	}
	if r.OnOne() != character.LastChargeDestroyed || r.OnTwenty() != d8 {
		t.Error("accessors")
	}
	if e, ok := character.NoRegain().Dice(); ok || !e.IsZero() {
		t.Error("NoRegain has dice")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-037, FR-041 - the rule constructors reject an invalid consequence; the consequence names are the
// attribute values.
// Path: unhappy (invalid input).
func TestLastChargeConstructors(t *testing.T) {
	t.Parallel()
	_, e1 := character.NewLastChargeAutomatic(0)
	_, e2 := character.NewLastChargeAutomatic(character.LastChargeRetainsBonusLosesOtherProperties + 1)
	_, e3 := character.NewLastChargeRolled(0, character.NoRegain())
	wantInvalid(t, e1, "automatic 0")
	wantInvalid(t, e2, "automatic past the set")
	wantInvalid(t, e3, "rolled 0")
	a := must(character.NewLastChargeAutomatic(character.LastChargeNone))
	if a.Consequence() != character.LastChargeNone {
		t.Error("consequence")
	}
	names := map[character.LastChargeConsequence]string{character.LastChargeNone: "none", character.LastChargeDestroyed: "destroyed",
		character.LastChargeBecomesNonmagical: "becomes_nonmagical", character.LastChargeRetainsBonusLosesOtherProperties: "retains_bonus", 0: "unknown", 5: "unknown"}
	for c, n := range names {
		if c.String() != n || c.Valid() != (n != "unknown") {
			t.Errorf("%d = %s", c, c)
		}
	}
}
