package combat

import "github.com/Whealinghold/srd5e-go/content"

// PossessionPrevention is a spell that prevents Possession. The state of a
// possession belongs to the toolbox, not to this package. The zero value is not a
// prevention.
type PossessionPrevention int

// The spells that prevent Possession (SRD 5.2.1, "Possession", p. 186).
const (
	// ProtectionFromEvilAndGood: the Protection from Evil and Good spell.
	ProtectionFromEvilAndGood PossessionPrevention = iota + 1
)

// String returns "protection_from_evil_and_good" or "unknown".
func (p PossessionPrevention) String() string {
	return enumName([]string{"protection_from_evil_and_good"}, int(p))
}

// PossessionEnding is what ends Possession. The zero value is not an ending.
type PossessionEnding int

// The ways Possession ends (SRD 5.2.1, "Possession", p. 186).
const (
	// DispelEvilAndGood: the Dispel Evil and Good spell.
	DispelEvilAndGood PossessionEnding = iota + 1
	// PossessingEffect: an end the possessing effect itself defines.
	PossessingEffect
)

// String returns "dispel_evil_and_good", "possessing_effect" or "unknown".
func (e PossessionEnding) String() string {
	return enumName([]string{"dispel_evil_and_good", "possessing_effect"}, int(e))
}

// PossessionPreventionFromSpell returns ProtectionFromEvilAndGood for
// protection_from_evil_and_good and the zero value with false for any other spell.
// It never panics.
func PossessionPreventionFromSpell(id content.SpellID) (PossessionPrevention, bool) {
	if id == "protection_from_evil_and_good" {
		return ProtectionFromEvilAndGood, true
	}
	return 0, false
}

// PossessionEndingFromSpell returns DispelEvilAndGood for dispel_evil_and_good and
// the zero value with false for any other spell. It never panics.
func PossessionEndingFromSpell(id content.SpellID) (PossessionEnding, bool) {
	if id == "dispel_evil_and_good" {
		return DispelEvilAndGood, true
	}
	return 0, false
}
