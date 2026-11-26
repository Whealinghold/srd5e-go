package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// SpellModifierRider is what a spell of table C adds to a roll or to damage on top of its numeric modifiers, as the
// catalogue fixes it (spell-granted-modifiers.md section 4.11): the signed die of Bless and Bane on attack rolls and
// saving throws, the Disadvantage of Hex on ability checks with the ability its caster chose, and the extra damage of
// Hex and Hunter's Mark. The queries read it from the entry; no function here rolls or applies anything. The zero
// value is no rider. Not comparable with ==; immutable.
type SpellModifierRider struct {
	rollDie           BonusDie
	extraDamage       DamageComponent
	checkDisadvantage bool
}

// RollDie returns the die the spell adds to its subject's attack rolls and saving throws, a Death Saving Throw
// included (a penalty die for Bane), and whether the spell has one.
func (r SpellModifierRider) RollDie() (BonusDie, bool) { return r.rollDie, !r.rollDie.IsZero() }

// ExtraDamage returns the damage the caster adds whenever it hits the subject with an attack roll, and whether the
// spell has any.
func (r SpellModifierRider) ExtraDamage() (DamageComponent, bool) {
	return r.extraDamage, r.extraDamage != nil
}

// CheckDisadvantage reports whether the spell gives its subject Disadvantage on ability checks made with the ability
// the caster chose when casting (Hex).
func (r SpellModifierRider) CheckDisadvantage() bool { return r.checkDisadvantage }

// IsZero reports whether the rider adds nothing.
func (r SpellModifierRider) IsZero() bool {
	return r.rollDie.IsZero() && r.extraDamage == nil && !r.checkDisadvantage
}

// riderOf returns the rider of the spell, the zero rider for a spell that is not in table C.
func riderOf(spell content.SpellID) SpellModifierRider {
	e, _ := catalogueEntry(spell)
	return e.rider
}

// d4Die returns the 1d4 of Bless and Bane, a penalty when negative.
func d4Die(negative bool) BonusDie {
	return MustBonusDie(dice.MustExpression(1, dice.D4(), 0), negative)
}
