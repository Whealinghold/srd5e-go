package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// StaffSpellAbsorptionResult is the closed set of outcomes of
// StaffSpellAbsorption.Absorb. Variants: StaffSpellAbsorptionAbsorbed,
// StaffSpellAbsorptionOverflowed.
type StaffSpellAbsorptionResult interface{ isStaffSpellAbsorptionResult() }

// StaffSpellAbsorptionAbsorbed: the staff took the spell's level as charges.
type StaffSpellAbsorptionAbsorbed struct {
	charges ItemCharges
	event   StaffSpellAbsorbed
}

// Charges returns the new charges.
func (r StaffSpellAbsorptionAbsorbed) Charges() ItemCharges { return r.charges }

// Event returns the StaffSpellAbsorbed event.
func (r StaffSpellAbsorptionAbsorbed) Event() StaffSpellAbsorbed { return r.event }

// StaffSpellAbsorptionOverflowed: the charges would pass the maximum, so the
// staff explodes as if its Retributive Strike were activated; total is the charge
// count after adding the spell level.
type StaffSpellAbsorptionOverflowed struct{ total int }

// Total returns the charges the staff held when it exploded, the count
// RetributiveStrike.ForCharges takes.
func (r StaffSpellAbsorptionOverflowed) Total() int { return r.total }

func (StaffSpellAbsorptionAbsorbed) isStaffSpellAbsorptionResult()   {}
func (StaffSpellAbsorptionOverflowed) isStaffSpellAbsorptionResult() {}

// StaffSpellAbsorption is the Staff of the Magi's Spell Absorption (SRD 5.2.1,
// "Staff of the Magi", p. 247). The zero value is ready to use.
type StaffSpellAbsorption struct{}

// Absorb adds spellLevel to the charges while the total stays within the
// maximum; more is StaffSpellAbsorptionOverflowed (49 + 3 gives 52). It assumes
// the spell targets only the wearer, which is the host's Reaction to check. It
// returns an error matching srderr.ErrInvalidInput for a spell level below 1, a
// zero holder, blank instance or zero charges.
func (StaffSpellAbsorption) Absorb(holder Identifier, instance MagicItemInstanceID, charges ItemCharges, spellLevel int) (StaffSpellAbsorptionResult, error) {
	if spellLevel < 1 || holder.IsZero() || !instance.Valid() || charges.IsZero() {
		return nil, invalidInput("absorb spell: level below 1, zero holder or charges, or blank instance")
	}
	total := charges.current + spellLevel
	if total > charges.maximum {
		return StaffSpellAbsorptionOverflowed{total: total}, nil
	}
	return StaffSpellAbsorptionAbsorbed{charges: ItemCharges{current: total, maximum: charges.maximum},
		event: StaffSpellAbsorbed{id: holder, instance: instance, level: spellLevel}}, nil
}

// RetributiveStrikeStaff is a staff with a Retributive Strike. The zero value is invalid.
type RetributiveStrikeStaff int

// The staffs.
const (
	StrikeStaffOfPower RetributiveStrikeStaff = iota + 1
	StrikeStaffOfTheMagi
)

// RetributiveStrikeSaveDC is the Dexterity saving throw DC of the explosion.
const RetributiveStrikeSaveDC = 17

// RetributiveStrikeDamage is the Force damage of a Retributive Strike: to the
// breaker, to creatures in the area that fail the save, and to those that succeed.
type RetributiveStrikeDamage struct{ self, areaFailed, areaSucceeded int }

// Self returns the damage to the one who broke the staff.
func (d RetributiveStrikeDamage) Self() int { return d.self }

// AreaFailed returns the damage on a failed Dexterity save.
func (d RetributiveStrikeDamage) AreaFailed() int { return d.areaFailed }

// AreaSucceeded returns the damage on a successful save, half rounded down.
func (d RetributiveStrikeDamage) AreaSucceeded() int { return d.areaSucceeded }

// DamageType returns Force.
func (RetributiveStrikeDamage) DamageType() content.DamageType { return content.DamageForce }

// RetributiveStrike computes the damage of a Retributive Strike (SRD 5.2.1, "Staff
// of Power", p. 246; "Staff of the Magi", p. 247). It rolls no travel chance and
// applies no damage. The zero value is ready to use.
type RetributiveStrike struct{}

// ForCharges returns self damage 16 per charge and area damage 4 (Staff of
// Power) or 6 (Staff of the Magi) per charge, half rounded down on a success. It
// returns an error matching srderr.ErrInvalidInput for a negative count or an
// invalid staff.
func (RetributiveStrike) ForCharges(staff RetributiveStrikeStaff, charges int) (RetributiveStrikeDamage, error) {
	if charges < 0 || staff < StrikeStaffOfPower || staff > StrikeStaffOfTheMagi {
		return RetributiveStrikeDamage{}, invalidInput("retributive strike: negative charges or invalid staff")
	}
	per := 4
	if staff == StrikeStaffOfTheMagi {
		per = 6
	}
	return RetributiveStrikeDamage{self: 16 * charges, areaFailed: per * charges, areaSucceeded: per * charges / 2}, nil
}
