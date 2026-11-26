package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attacks.md FR-049, FR-100, FR-126 - IsImprovised is derived: true for an object with no weapon
// category, a Ranged weapon used in melee, or a Melee weapon without Thrown that is thrown (SRD 5.2.1, "Improvised
// Weapons", p. 183; "Thrown", p. 90).
// Path: per case.
func TestWeaponUseIsImprovised(t *testing.T) {
	t.Parallel()
	thrown := []content.WeaponProperty{content.WeaponPropertyThrown, content.WeaponPropertyLight}
	light := []content.WeaponProperty{content.WeaponPropertyLight}
	cases := []struct {
		name, requirement, path string
		category                content.WeaponCategory
		typ                     content.WeaponType
		props                   []content.WeaponProperty
		manner                  combat.WeaponUseManner
		improvised              bool
	}{
		{"object that is not a weapon", "FR-126", "unhappy (boundary)", 0, 0, nil, combat.WieldedAsDesigned, true},
		{"Simple melee weapon as designed", "FR-100", "happy", content.WeaponSimple, content.WeaponMelee, nil, combat.WieldedAsDesigned, false},
		{"Martial ranged weapon as designed", "FR-100", "happy", content.WeaponMartial, content.WeaponRanged, nil, combat.WieldedAsDesigned, false},
		{"Ranged weapon in melee", "FR-049", "unhappy (boundary)", content.WeaponSimple, content.WeaponRanged, nil, combat.MeleeAttackWithRanged, true},
		{"Melee weapon thrown without Thrown", "FR-049", "unhappy (boundary)", content.WeaponMartial, content.WeaponMelee, light, combat.ThrownAttack, true},
		{"Melee weapon thrown with no properties", "FR-049", "unhappy (empty and minimal case)", content.WeaponMartial, content.WeaponMelee, nil, combat.ThrownAttack, true},
		{"Melee weapon thrown with Thrown", "FR-049", "unhappy (boundary)", content.WeaponSimple, content.WeaponMelee, thrown, combat.ThrownAttack, false},
		{"object thrown", "FR-126", "unhappy (boundary)", 0, 0, nil, combat.ThrownAttack, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			w, err := combat.NewWeaponUse(c.category, c.typ, c.props, c.manner)
			if err != nil || !w.Supplied() || w.IsImprovised() != c.improvised || w.Manner() != c.manner || w.Category() != c.category {
				t.Errorf("NewWeaponUse = %+v, %v improvised %v, want improvised %v [%s]", w, err, w.IsImprovised(), c.improvised, c.requirement)
			}
		})
	}
	if (combat.WeaponUse{}).Supplied() || (combat.WeaponUse{}).IsImprovised() {
		t.Errorf("the zero WeaponUse is a supplied or improvised weapon")
	}
}

// Requirement: doc/specs/attacks.md FR-049, FR-126 - MeleeAttackWithRanged on a Melee weapon, a bad manner, category, type
// or property is a validation error; the properties are copied in and out.
// Path: per case.
func TestNewWeaponUseRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		category content.WeaponCategory
		typ      content.WeaponType
		props    []content.WeaponProperty
		manner   combat.WeaponUseManner
	}{
		{"melee attack with a Melee weapon", content.WeaponSimple, content.WeaponMelee, nil, combat.MeleeAttackWithRanged},
		{"zero manner", content.WeaponSimple, content.WeaponMelee, nil, 0},
		{"manner past the last", content.WeaponSimple, content.WeaponMelee, nil, combat.MeleeAttackWithRanged + 1},
		{"zero manner without a category", 0, 0, nil, 0},
		{"bad category", content.WeaponMartial + 1, content.WeaponMelee, nil, combat.WieldedAsDesigned},
		{"zero weapon type", content.WeaponSimple, 0, nil, combat.WieldedAsDesigned},
		{"bad property", content.WeaponSimple, content.WeaponMelee, []content.WeaponProperty{0}, combat.WieldedAsDesigned},
	}
	for _, c := range cases {
		if w, err := combat.NewWeaponUse(c.category, c.typ, c.props, c.manner); !errors.Is(err, srderr.ErrInvalidInput) || w.Supplied() {
			t.Errorf("%s: NewWeaponUse = %+v, %v, want ErrInvalidInput [FR-049, FR-126]", c.name, w, err)
		}
	}
	props := []content.WeaponProperty{content.WeaponPropertyThrown}
	w, _ := combat.NewWeaponUse(content.WeaponSimple, content.WeaponMelee, props, combat.ThrownAttack)
	props[0] = content.WeaponPropertyHeavy
	got := w.Properties()
	got[0] = content.WeaponPropertyLight
	if !slices.Equal(w.Properties(), []content.WeaponProperty{content.WeaponPropertyThrown}) || w.IsImprovised() {
		t.Errorf("properties = %v, want them copied on the way in and out [FR-126]", w.Properties())
	}
}

// Requirement: doc/specs/attacks.md FR-047, FR-048 - an improvised weapon's default damage is 1d4 of a Damage Type the
// caller chooses, and a thrown one's default range is 20 and 60 feet (SRD 5.2.1, "Improvised Weapons", p. 183).
// Path: happy.
func TestImprovisedDefaults(t *testing.T) {
	t.Parallel()
	d, err := combat.ImprovisedDamage(content.DamageBludgeoning)
	if err != nil || d.Count() != 1 || d.Die() != dice.D4() || d.Type() != content.DamageBludgeoning || !d.AddsAbilityModifier() || d.Modifier() != 0 {
		t.Errorf("ImprovisedDamage = %+v, %v, want 1d4 Bludgeoning adding the ability modifier", d, err)
	}
	if _, err := combat.ImprovisedDamage(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ImprovisedDamage(no type) error = %v, want ErrInvalidInput", err)
	}
	if r := combat.ImprovisedThrownRange(); r.Normal() != 20 || r.Long() != 60 {
		t.Errorf("ImprovisedThrownRange = %d/%d, want 20/60", r.Normal(), r.Long())
	}
	if combat.WieldedAsDesigned.String() != "wielded_as_designed" || combat.ThrownAttack.String() != "thrown_attack" ||
		combat.MeleeAttackWithRanged.String() != "melee_attack_with_ranged" || combat.WeaponUseManner(0).String() != "unknown" {
		t.Errorf("manner names are wrong")
	}
}
