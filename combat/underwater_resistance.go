package combat

import "github.com/Whealinghold/srd5e-go/content"

// UnderwaterStatus says whether a creature or object is underwater (SRD 5.2.1, "Fire
// Resistance", p. 16: "Anything underwater"). The host reports it. The zero value is invalid.
type UnderwaterStatus int

// The two states.
const (
	// UnderwaterStatusUnderwater: the creature or object is underwater.
	UnderwaterStatusUnderwater UnderwaterStatus = iota + 1
	// UnderwaterStatusNotUnderwater: it is not.
	UnderwaterStatusNotUnderwater
)

// UnderwaterResistanceGrant says whether the damage being taken may be resisted because of
// the water; some actions state that being underwater does not grant Resistance to their
// Fire damage (SRD 5.2.1, "Monsters A-Z", pp. 281, 308). The zero value is invalid.
type UnderwaterResistanceGrant int

// The two answers.
const (
	// UnderwaterResistanceGranted: the water grants Resistance to this damage.
	UnderwaterResistanceGranted UnderwaterResistanceGrant = iota + 1
	// UnderwaterResistanceWithheldByEffect: the effect says the water does not.
	UnderwaterResistanceWithheldByEffect
)

// UnderwaterFireResistanceAffinities returns the affinities for damage taken now: base, plus
// Resistance to Fire when status is UnderwaterStatusUnderwater and grant is
// UnderwaterResistanceGranted (SRD 5.2.1, "Fire Resistance", p. 16). It changes nothing
// else: Resistance still does not stack and the damage pipeline halves, rounds down and
// orders it as for any Resistance. The result is derived on each call and never stored.
// A status or grant that is not one of the constants adds nothing. It is pure, silent and
// never panics.
func UnderwaterFireResistanceAffinities(base DamageAffinities, status UnderwaterStatus, grant UnderwaterResistanceGrant) DamageAffinities {
	if status == UnderwaterStatusUnderwater && grant == UnderwaterResistanceGranted && !base.resists(content.DamageFire) {
		base.resist |= typeBit(content.DamageFire)
	}
	return base
}
