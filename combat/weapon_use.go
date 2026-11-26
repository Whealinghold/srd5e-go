package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// WeaponUseManner says how an object is used in an attack. The zero value is
// invalid.
type WeaponUseManner int

// The three manners.
const (
	// WieldedAsDesigned: used as its design intends.
	WieldedAsDesigned WeaponUseManner = iota + 1
	// ThrownAttack: thrown.
	ThrownAttack
	// MeleeAttackWithRanged: a Ranged weapon used in melee.
	MeleeAttackWithRanged
)

// String returns "wielded_as_designed", "thrown_attack", "melee_attack_with_ranged"
// or "unknown".
func (m WeaponUseManner) String() string {
	return enumName([]string{"wielded_as_designed", "thrown_attack", "melee_attack_with_ranged"}, int(m))
}

// WeaponUse is how an object is used in a weapon attack, from content vocabulary:
// whether it is an improvised weapon decides whether the Proficiency Bonus may be
// added (SRD 5.2.1, "Improvised Weapons", p. 183). Build it with NewWeaponUse.
// The zero value is no weapon use supplied. Safe for concurrent use.
type WeaponUse struct {
	category   content.WeaponCategory
	weaponType content.WeaponType
	properties []content.WeaponProperty
	manner     WeaponUseManner
}

// NewWeaponUse returns the use of an object. A zero category is an object that is
// not a Simple or Martial weapon: always improvised, and weaponType and
// properties are not read. properties is copied.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid manner, for a
// category, weapon type or property that is not valid, and for
// MeleeAttackWithRanged on a weapon type that is not Ranged. It never panics.
func NewWeaponUse(category content.WeaponCategory, weaponType content.WeaponType, properties []content.WeaponProperty,
	manner WeaponUseManner) (WeaponUse, error) {
	if manner < WieldedAsDesigned || manner > MeleeAttackWithRanged {
		return WeaponUse{}, invalidInput("weapon use manner")
	}
	if category == 0 {
		return WeaponUse{manner: manner}, nil
	}
	if !category.Valid() || !weaponType.Valid() || slices.ContainsFunc(properties, func(p content.WeaponProperty) bool { return !p.Valid() }) {
		return WeaponUse{}, invalidInput("weapon use vocabulary")
	}
	if manner == MeleeAttackWithRanged && weaponType != content.WeaponRanged {
		return WeaponUse{}, invalidInput("a melee attack with a weapon that is not Ranged")
	}
	return WeaponUse{category, weaponType, slices.Clone(properties), manner}, nil
}

// Supplied reports whether w is a weapon use at all (not the zero value).
func (w WeaponUse) Supplied() bool { return w.manner != 0 }

// Category returns the weapon category, or 0 for an object that is not a weapon.
func (w WeaponUse) Category() content.WeaponCategory { return w.category }

// Properties returns a copy of the properties.
func (w WeaponUse) Properties() []content.WeaponProperty { return slices.Clone(w.properties) }

// Manner returns the manner of use.
func (w WeaponUse) Manner() WeaponUseManner { return w.manner }

// IsImprovised reports whether the use is an improvised weapon (SRD 5.2.1,
// "Improvised Weapons", p. 183): an object with no weapon category, a Ranged
// weapon used in melee, or a Melee weapon without Thrown that is thrown. It is
// false for the zero value.
func (w WeaponUse) IsImprovised() bool {
	switch {
	case !w.Supplied():
		return false
	case w.category == 0, w.manner == MeleeAttackWithRanged:
		return true
	}
	return w.manner == ThrownAttack && w.weaponType == content.WeaponMelee &&
		!slices.Contains(w.properties, content.WeaponPropertyThrown)
}

// ImprovisedDamage returns the default damage of an improvised weapon: 1d4 of the
// Damage Type the Game Master chooses, adding the ability modifier (SRD 5.2.1,
// "Improvised Weapons", p. 183). It returns an error matching
// srderr.ErrInvalidInput for an invalid Damage Type.
func ImprovisedDamage(t content.DamageType) (DiceDamage, error) {
	return NewDiceDamage(1, dice.D4(), t, 0, true)
}

// ImprovisedThrownRange returns the default range of a thrown improvised weapon:
// normal 20 feet, long 60 feet (SRD 5.2.1, "Improvised Weapons", p. 183).
func ImprovisedThrownRange() AttackRange { return AttackRange{20, 60} }
