package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// Damage is one instance of damage: an amount of one Damage Type (SRD 5.2.1,
// "Damage Rolls", p. 16). Resistance and Vulnerability apply once per instance.
// It is immutable and comparable, and one value may be applied to several
// creatures. The zero value has no Damage Type and is invalid.
type Damage struct {
	amount int
	typ    content.DamageType
}

// NewDamage returns the instance of amount damage of type t.
//
// It returns an error matching srderr.ErrInvalidInput for a negative amount or
// a Damage Type that is not valid. An amount of 0 is valid. It never panics.
func NewDamage(amount int, t content.DamageType) (Damage, error) {
	if amount < 0 || !t.Valid() {
		return Damage{}, invalidInput("damage needs an amount of 0 or more and a valid Damage Type")
	}
	return Damage{amount, t}, nil
}

// MustDamage is NewDamage for values known to be valid. It panics with the
// error NewDamage returns, and only then.
func MustDamage(amount int, t content.DamageType) Damage {
	d, err := NewDamage(amount, t)
	if err != nil {
		panic(err)
	}
	return d
}

// Amount returns the amount of damage.
func (d Damage) Amount() int { return d.amount }

// Type returns the Damage Type.
func (d Damage) Type() content.DamageType { return d.typ }

// DamageScope is the damage a Resistance or a flat adjustment covers: all
// damage or one Damage Type. Variants: AllDamage, DamageOfType.
type DamageScope interface{ damageScope() }

// AllDamage covers damage of every Damage Type.
type AllDamage struct{}

// DamageOfType covers damage of one Damage Type. The zero value covers nothing.
// Construct with a keyed literal.
type DamageOfType struct{ Type content.DamageType }

func (AllDamage) damageScope()    {}
func (DamageOfType) damageScope() {}

// covers reports whether s covers damage of type t. A nil scope covers all.
func covers(s DamageScope, t content.DamageType) bool {
	if o, ok := s.(DamageOfType); ok {
		return o.Type == t
	}
	return true
}

// DamageAdjustment changes damage before Resistance and Vulnerability (SRD
// 5.2.1, "Order of Application", p. 17). Variants: FlatAdjustment, HalfDamage.
type DamageAdjustment interface{ damageAdjustment() }

// FlatAdjustment is a bonus (positive) or penalty (negative) to damage covered
// by its scope. Flat adjustments apply first and the result is floored at 0. A
// zero FlatAdjustment adds nothing to all damage. Build it with
// NewFlatAdjustment.
type FlatAdjustment struct {
	amount int
	scope  DamageScope
}

// NewFlatAdjustment returns a flat adjustment of amount to damage within scope.
// A nil scope is AllDamage.
func NewFlatAdjustment(amount int, scope DamageScope) FlatAdjustment {
	if scope == nil {
		scope = AllDamage{}
	}
	return FlatAdjustment{amount, scope}
}

// Amount returns the signed amount.
func (a FlatAdjustment) Amount() int { return a.amount }

// Scope returns the damage the adjustment covers.
func (a FlatAdjustment) Scope() DamageScope {
	if a.scope == nil {
		return AllDamage{}
	}
	return a.scope
}

// HalfDamage halves damage, rounded down: a saving throw the effect halves
// (SRD 5.2.1, "Half Damage", p. 16). It applies after the flat adjustments.
type HalfDamage struct{}

func (FlatAdjustment) damageAdjustment() {}
func (HalfDamage) damageAdjustment()     {}

// HealingCeiling caps the Hit Points Healing can reach. Variants:
// NoHealingCeiling, HealingCeilingAt.
type HealingCeiling interface{ healingCeiling() }

// NoHealingCeiling is the default: Healing reaches the Hit Point maximum.
type NoHealingCeiling struct{}

// HealingCeilingAt caps current Hit Points at a value (and never above the
// maximum). Build it with NewHealingCeilingAt; the zero value caps at 0.
type HealingCeilingAt struct{ hitPoints int }

// NewHealingCeilingAt returns a ceiling at hitPoints. It returns an error
// matching srderr.ErrInvalidInput for a negative value.
func NewHealingCeilingAt(hitPoints int) (HealingCeilingAt, error) {
	if hitPoints < 0 {
		return HealingCeilingAt{}, invalidInput("negative healing ceiling")
	}
	return HealingCeilingAt{hitPoints}, nil
}

// HitPoints returns the cap.
func (c HealingCeilingAt) HitPoints() int { return c.hitPoints }

func (NoHealingCeiling) healingCeiling() {}
func (HealingCeilingAt) healingCeiling() {}
