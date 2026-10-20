package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// WeaponCategory is Simple or Martial (SRD 5.2.1, "Weapons", p. 89). The zero
// value is invalid. Comparable and safe for concurrent use.
type WeaponCategory int

// The weapon categories.
const (
	WeaponSimple WeaponCategory = iota + 1
	WeaponMartial
)

func weaponCategoryNames() []string { return []string{"Simple", "Martial"} }

// Valid reports whether c is a weapon category.
func (c WeaponCategory) Valid() bool { return enumValid(weaponCategoryNames(), int(c)) }

// String returns the printed name of c, or "unknown".
func (c WeaponCategory) String() string { return enumString(weaponCategoryNames(), int(c)) }

func parseWeaponCategory(s string) (WeaponCategory, error) {
	v, err := parseEnum("WeaponCategory", weaponCategoryNames(), s)
	return WeaponCategory(v), err
}

// WeaponType is Melee or Ranged, following the table group a weapon is in,
// not its properties (SRD 5.2.1, "Weapons", p. 89). The zero value is
// invalid. Comparable and safe for concurrent use.
type WeaponType int

// The weapon types.
const (
	WeaponMelee WeaponType = iota + 1
	WeaponRanged
)

func weaponTypeNames() []string { return []string{"Melee", "Ranged"} }

// Valid reports whether t is a weapon type.
func (t WeaponType) Valid() bool { return enumValid(weaponTypeNames(), int(t)) }

// String returns the printed name of t, or "unknown".
func (t WeaponType) String() string { return enumString(weaponTypeNames(), int(t)) }

func parseWeaponType(s string) (WeaponType, error) {
	v, err := parseEnum("WeaponType", weaponTypeNames(), s)
	return WeaponType(v), err
}

// strictDice reports whether e is a dice expression of the weapon tables: no
// modifier and a count of at least 1 (so "d8" is not one, content-equipment
// FR-110).
func strictDice(e dice.Expression) bool { return e.Modifier() == 0 && e.Count() >= 1 }

// parseStrictDice parses text as a dice expression with an explicit count and
// no modifier ("1d8"). It rejects what dice.ParseExpression reads with a
// default or a modifier: the text must equal the notation of the result.
func parseStrictDice(text string) (dice.Expression, bool) {
	e, err := dice.ParseExpression(text)
	if err != nil || !strictDice(e) || e.Notation() != text {
		return dice.Expression{}, false
	}
	return e, true
}

// WeaponDamage is the damage of a weapon: a closed set with the variants
// [WeaponDiceDamage] and [WeaponFixedDamage].
type WeaponDamage interface {
	// DamageType returns the type of the damage.
	DamageType() DamageType
	isWeaponDamage()
}

// WeaponDiceDamage is damage rolled with dice, such as 1d8 Slashing. The zero
// value is invalid; build it with [NewWeaponDiceDamage].
type WeaponDiceDamage struct {
	dice dice.Expression
	typ  DamageType
}

func (WeaponDiceDamage) isWeaponDamage() {}

// NewWeaponDiceDamage returns damage of expression e and type t.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockOutOfRange for an expression with a modifier or no dice, then
// BlockInvalidMember for an invalid damage type. It never panics.
func NewWeaponDiceDamage(e dice.Expression, t DamageType) (WeaponDiceDamage, error) {
	switch {
	case !strictDice(e):
		return WeaponDiceDamage{}, blockError("WeaponDiceDamage", BlockOutOfRange)
	case !t.Valid():
		return WeaponDiceDamage{}, blockError("WeaponDiceDamage", BlockInvalidMember)
	}
	return WeaponDiceDamage{dice: e, typ: t}, nil
}

// Dice returns the damage dice.
func (d WeaponDiceDamage) Dice() dice.Expression { return d.dice }

// DamageType returns the type of the damage.
func (d WeaponDiceDamage) DamageType() DamageType { return d.typ }

// WeaponFixedDamage is damage of a fixed amount, such as the 1 Piercing of a
// weapon printed without dice. The zero value is invalid; build it with
// [NewWeaponFixedDamage]. Comparable.
type WeaponFixedDamage struct {
	amount int
	typ    DamageType
}

func (WeaponFixedDamage) isWeaponDamage() {}

// NewWeaponFixedDamage returns damage of amount points and type t.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockOutOfRange for an amount below 1 or above 2^31 - 1, then
// BlockInvalidMember for an invalid damage type. It never panics.
func NewWeaponFixedDamage(amount int, t DamageType) (WeaponFixedDamage, error) {
	switch {
	case amount < 1 || amount > equipmentMaxInt:
		return WeaponFixedDamage{}, blockError("WeaponFixedDamage", BlockOutOfRange)
	case !t.Valid():
		return WeaponFixedDamage{}, blockError("WeaponFixedDamage", BlockInvalidMember)
	}
	return WeaponFixedDamage{amount: amount, typ: t}, nil
}

// Amount returns the fixed damage.
func (d WeaponFixedDamage) Amount() int { return d.amount }

// DamageType returns the type of the damage.
func (d WeaponFixedDamage) DamageType() DamageType { return d.typ }

func validWeaponDamage(d WeaponDamage) bool {
	var err error
	switch v := d.(type) {
	case WeaponDiceDamage:
		_, err = NewWeaponDiceDamage(v.dice, v.typ)
	case WeaponFixedDamage:
		_, err = NewWeaponFixedDamage(v.amount, v.typ)
	default:
		return false
	}
	return err == nil
}

// RangePair is the normal and long range of a weapon in feet (SRD 5.2.1,
// "Properties", pp. 89-90). The zero value is invalid; build it with
// [NewRangePair]. Comparable.
type RangePair struct{ normal, long int }

// NewRangePair returns the range normal/long. It returns a *[BlockError]
// (BlockOutOfRange) wrapping srderr.ErrInvalidInput for a normal range below
// 1 or a long range that is not greater than it, or above 2^31 - 1.
func NewRangePair(normal, long int) (RangePair, error) {
	if normal < 1 || long <= normal || long > equipmentMaxInt {
		return RangePair{}, blockError("RangePair", BlockOutOfRange)
	}
	return RangePair{normal: normal, long: long}, nil
}

// Normal returns the normal range.
func (r RangePair) Normal() int { return r.normal }

// Long returns the long range.
func (r RangePair) Long() int { return r.long }

// AmmunitionType is the kind of ammunition a weapon fires. The zero value is
// invalid. Comparable and safe for concurrent use.
type AmmunitionType int

// The ammunition types.
const (
	AmmunitionArrow AmmunitionType = iota + 1
	AmmunitionBolt
	AmmunitionBullet
	AmmunitionNeedle
)

func ammunitionTypeNames() []string { return []string{"Arrow", "Bolt", "Bullet", "Needle"} }

// Valid reports whether t is an ammunition type.
func (t AmmunitionType) Valid() bool { return enumValid(ammunitionTypeNames(), int(t)) }

// String returns the printed name of t, or "unknown".
func (t AmmunitionType) String() string { return enumString(ammunitionTypeNames(), int(t)) }

func parseAmmunitionType(s string) (AmmunitionType, error) {
	v, err := parseEnum("AmmunitionType", ammunitionTypeNames(), s)
	return AmmunitionType(v), err
}

// TwoHandedCondition says when a Two-Handed weapon needs two hands. The zero
// value is invalid. Comparable and safe for concurrent use.
type TwoHandedCondition int

// The Two-Handed conditions.
const (
	TwoHandedAlways TwoHandedCondition = iota + 1
	TwoHandedUnlessMounted
)

func twoHandedConditionNames() []string { return []string{"always", "unlessMounted"} }

// Valid reports whether c is a condition.
func (c TwoHandedCondition) Valid() bool { return enumValid(twoHandedConditionNames(), int(c)) }

// String returns "always" or "unlessMounted", or "unknown".
func (c TwoHandedCondition) String() string { return enumString(twoHandedConditionNames(), int(c)) }

// WeaponPropertyUse is one property of one weapon with the data the table
// prints for it: a closed set with the variants [PlainUse], [AmmunitionUse],
// [ThrownUse], [VersatileUse] and [TwoHandedUse].
type WeaponPropertyUse interface {
	// Property returns the property.
	Property() WeaponProperty
	isWeaponPropertyUse()
}

// PlainUse is a property printed without data: Finesse, Heavy, Light, Loading
// or Reach. The zero value is invalid; build it with [NewPlainUse]. Comparable.
type PlainUse struct{ property WeaponProperty }

func (PlainUse) isWeaponPropertyUse() {}

// NewPlainUse returns the use of property p. It returns a *[BlockError]
// (BlockInvalidMember) wrapping srderr.ErrInvalidInput for any other
// property.
func NewPlainUse(p WeaponProperty) (PlainUse, error) {
	switch p { //nolint:exhaustive // the other four properties carry data and are rejected
	case WeaponPropertyFinesse, WeaponPropertyHeavy, WeaponPropertyLight, WeaponPropertyLoading, WeaponPropertyReach:
		return PlainUse{property: p}, nil
	}
	return PlainUse{}, blockError("PlainUse", BlockInvalidMember)
}

// Property returns the property.
func (u PlainUse) Property() WeaponProperty { return u.property }

// AmmunitionUse is the Ammunition property with its range and ammunition type.
// The zero value is invalid; build it with [NewAmmunitionUse]. Comparable.
type AmmunitionUse struct {
	rng  RangePair
	kind AmmunitionType
}

func (AmmunitionUse) isWeaponPropertyUse() {}

// NewAmmunitionUse returns the use with range r and ammunition t. It returns
// a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange for an
// invalid range, then BlockInvalidMember for an invalid type.
func NewAmmunitionUse(r RangePair, t AmmunitionType) (AmmunitionUse, error) {
	switch {
	case !validRange(r):
		return AmmunitionUse{}, blockError("AmmunitionUse", BlockOutOfRange)
	case !t.Valid():
		return AmmunitionUse{}, blockError("AmmunitionUse", BlockInvalidMember)
	}
	return AmmunitionUse{rng: r, kind: t}, nil
}

// Property returns [WeaponPropertyAmmunition].
func (AmmunitionUse) Property() WeaponProperty { return WeaponPropertyAmmunition }

// Range returns the normal and long range.
func (u AmmunitionUse) Range() RangePair { return u.rng }

// Ammunition returns the type of ammunition.
func (u AmmunitionUse) Ammunition() AmmunitionType { return u.kind }

func validRange(r RangePair) bool {
	_, err := NewRangePair(r.normal, r.long)
	return err == nil
}

// ThrownUse is the Thrown property with its range. The zero value is invalid;
// build it with [NewThrownUse]. Comparable.
type ThrownUse struct{ rng RangePair }

func (ThrownUse) isWeaponPropertyUse() {}

// NewThrownUse returns the use with range r. It returns a *[BlockError]
// (BlockOutOfRange) wrapping srderr.ErrInvalidInput for an invalid range.
func NewThrownUse(r RangePair) (ThrownUse, error) {
	if !validRange(r) {
		return ThrownUse{}, blockError("ThrownUse", BlockOutOfRange)
	}
	return ThrownUse{rng: r}, nil
}

// Property returns [WeaponPropertyThrown].
func (ThrownUse) Property() WeaponProperty { return WeaponPropertyThrown }

// Range returns the normal and long range.
func (u ThrownUse) Range() RangePair { return u.rng }

// VersatileUse is the Versatile property with the two-handed damage. The zero
// value is invalid; build it with [NewVersatileUse]. Comparable.
type VersatileUse struct{ damage dice.Expression }

func (VersatileUse) isWeaponPropertyUse() {}

// NewVersatileUse returns the use with two-handed damage e. It returns a
// *[BlockError] (BlockOutOfRange) wrapping srderr.ErrInvalidInput for an
// expression with a modifier or no dice.
func NewVersatileUse(e dice.Expression) (VersatileUse, error) {
	if !strictDice(e) {
		return VersatileUse{}, blockError("VersatileUse", BlockOutOfRange)
	}
	return VersatileUse{damage: e}, nil
}

// Property returns [WeaponPropertyVersatile].
func (VersatileUse) Property() WeaponProperty { return WeaponPropertyVersatile }

// Damage returns the damage when the weapon is used with two hands.
func (u VersatileUse) Damage() dice.Expression { return u.damage }

// TwoHandedUse is the Two-Handed property with its condition. The zero value
// is invalid; build it with [NewTwoHandedUse]. Comparable.
type TwoHandedUse struct{ condition TwoHandedCondition }

func (TwoHandedUse) isWeaponPropertyUse() {}

// NewTwoHandedUse returns the use with condition c. It returns a *[BlockError]
// (BlockInvalidMember) wrapping srderr.ErrInvalidInput for an invalid
// condition.
func NewTwoHandedUse(c TwoHandedCondition) (TwoHandedUse, error) {
	if !c.Valid() {
		return TwoHandedUse{}, blockError("TwoHandedUse", BlockInvalidMember)
	}
	return TwoHandedUse{condition: c}, nil
}

// Property returns [WeaponPropertyTwoHanded].
func (TwoHandedUse) Property() WeaponProperty { return WeaponPropertyTwoHanded }

// Condition returns when two hands are needed.
func (u TwoHandedUse) Condition() TwoHandedCondition { return u.condition }

func validPropertyUse(p WeaponPropertyUse) bool {
	var err error
	switch v := p.(type) {
	case PlainUse:
		_, err = NewPlainUse(v.property)
	case AmmunitionUse:
		_, err = NewAmmunitionUse(v.rng, v.kind)
	case ThrownUse:
		_, err = NewThrownUse(v.rng)
	case VersatileUse:
		_, err = NewVersatileUse(v.damage)
	case TwoHandedUse:
		_, err = NewTwoHandedUse(v.condition)
	default:
		return false
	}
	return err == nil
}

// Weapon is a row of the Weapons table (SRD 5.2.1, "Weapons", pp. 89-91). It
// is an [EquipmentEntry] of kind [KindEquipment]. Immutable, with its
// property list copied on construction and on read; the zero Weapon is
// invalid. It holds a slice, so it is not comparable with ==. Safe for
// concurrent use.
type Weapon struct {
	equipmentHeader
	category   WeaponCategory
	weaponType WeaponType
	damage     WeaponDamage
	properties []WeaponPropertyUse
	mastery    MasteryProperty
	weight     WeightSpec
	price      FixedCost
}

var _ EquipmentEntry = Weapon{}

// NewWeapon returns the weapon id called name. The properties are in printed
// order, none repeated, and empty for a weapon printed "—". The weight is a
// [Weight] or [NoWeight], never [WeightVaries].
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule broken:
// BlockInvalidName; BlockInvalidMember for a category, type, damage, property
// use, mastery, weight or price that is invalid; BlockDuplicate for a repeated
// property. It never panics.
func NewWeapon(id ItemID, name string, source SourceCitation, category WeaponCategory, weaponType WeaponType, damage WeaponDamage, properties []WeaponPropertyUse, mastery MasteryProperty, weight WeightSpec, price FixedCost) (Weapon, error) {
	const block = "Weapon"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return Weapon{}, err
	}
	if !category.Valid() || !weaponType.Valid() || !validWeaponDamage(damage) || !mastery.Valid() || !validFixedWeight(weight) || !price.cost.Valid() ||
		slices.ContainsFunc(properties, func(p WeaponPropertyUse) bool { return !validPropertyUse(p) }) {
		return Weapon{}, blockError(block, BlockInvalidMember)
	}
	seen := map[WeaponProperty]struct{}{}
	for _, p := range properties {
		if _, repeated := seen[p.Property()]; repeated {
			return Weapon{}, blockError(block, BlockDuplicate)
		}
		seen[p.Property()] = struct{}{}
	}
	return Weapon{
		equipmentHeader: h, category: category, weaponType: weaponType, damage: damage,
		properties: slices.Clone(properties), mastery: mastery, weight: weight, price: price,
	}, nil
}

// EquipmentCategory returns [EquipmentWeapon].
func (Weapon) EquipmentCategory() EquipmentCategory { return EquipmentWeapon }

// Category returns Simple or Martial.
func (w Weapon) Category() WeaponCategory { return w.category }

// Type returns Melee or Ranged.
func (w Weapon) Type() WeaponType { return w.weaponType }

// Damage returns the damage.
func (w Weapon) Damage() WeaponDamage { return w.damage }

// Properties returns the properties in printed order, as a new slice; nil for
// a weapon with none.
func (w Weapon) Properties() []WeaponPropertyUse { return slices.Clone(w.properties) }

// Mastery returns the mastery property.
func (w Weapon) Mastery() MasteryProperty { return w.mastery }

// Weight returns the weight: a [Weight] or [NoWeight].
func (w Weapon) Weight() WeightSpec { return w.weight }

// Price returns the price.
func (w Weapon) Price() FixedCost { return w.price }
