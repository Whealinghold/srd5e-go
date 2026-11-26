package combat

import (
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// counterspellRange is how far away a creature can be and still be countered (SRD 5.2.1,
// "Counterspell" (Range), p. 120).
const counterspellRange = 60

// UsedComponents is the components a cast actually uses. It is not the spell's
// printed components: a cast without components (for example by Subtle Spell,
// SRD 5.2.1, p. 67) uses none. Comparable and safe for concurrent use.
type UsedComponents struct {
	verbal, somatic, material bool
}

// NewUsedComponents returns the components a cast uses. It never fails and never panics.
func NewUsedComponents(verbal, somatic, material bool) UsedComponents {
	return UsedComponents{verbal: verbal, somatic: somatic, material: material}
}

// Verbal reports whether the cast uses a Verbal component.
func (u UsedComponents) Verbal() bool { return u.verbal }

// Somatic reports whether the cast uses a Somatic component.
func (u UsedComponents) Somatic() bool { return u.somatic }

// Material reports whether the cast uses a Material component.
func (u UsedComponents) Material() bool { return u.material }

// Any reports whether the cast uses at least one component.
func (u UsedComponents) Any() bool { return u.verbal || u.somatic || u.material }

// CastObservation is a spell cast that a creature may counter: the spell, the
// casting creature, where the cast comes from, the components it uses, whether it
// requires Concentration, and its casting time (which sets the action it wastes).
// Build it with NewCastObservation; the zero value is invalid. Comparable and safe
// for concurrent use.
type CastObservation struct {
	spell                 content.SpellID
	caster                CombatantID
	source                character.CastSource
	used                  UsedComponents
	requiresConcentration bool
	castingTime           content.CastingTime
}

// NewCastObservation returns the observation of a cast. It returns an error
// matching srderr.ErrInvalidInput for a spell id that is not valid, a zero
// caster, a cast source that is not one of the defined sources, or a nil casting
// time. It never panics.
func NewCastObservation(spell content.SpellID, caster CombatantID, source character.CastSource, used UsedComponents, requiresConcentration bool, castingTime content.CastingTime) (CastObservation, error) {
	switch {
	case !spell.Valid():
		return CastObservation{}, invalidInput("a cast needs a valid spell id")
	case caster.IsZero():
		return CastObservation{}, invalidInput("a cast needs a caster")
	case !source.Valid():
		return CastObservation{}, invalidInput("a cast needs a defined cast source")
	case castingTime == nil:
		return CastObservation{}, invalidInput("a cast needs a casting time")
	}
	return CastObservation{spell, caster, source, used, requiresConcentration, castingTime}, nil
}

// Spell returns the spell cast.
func (o CastObservation) Spell() content.SpellID { return o.spell }

// Source returns where the cast comes from.
func (o CastObservation) Source() character.CastSource { return o.source }

// Used returns the components the cast uses.
func (o CastObservation) Used() UsedComponents { return o.used }

// RequiresConcentration reports whether the spell requires Concentration.
func (o CastObservation) RequiresConcentration() bool { return o.requiresConcentration }

// CastingTime returns the casting time of the spell.
func (o CastObservation) CastingTime() content.CastingTime { return o.castingTime }

// Counterspell is the Counterspell spell (SRD 5.2.1, "Counterspell", p. 120). It
// is stateless: use the zero value. Assess decides whether a cast can be
// countered; Conclude applies the caster's save.
type Counterspell struct{}

// CounterspellRefusal says why a cast cannot be countered. The zero value is not a reason.
type CounterspellRefusal int

const (
	// CounterspellCasterNotSeen: the counterspeller does not see the casting creature.
	CounterspellCasterNotSeen CounterspellRefusal = iota + 1
	// CounterspellOutOfRange: the casting creature is more than 60 feet away.
	CounterspellOutOfRange
	// CounterspellNoComponents: the cast uses no Verbal, Somatic or Material component.
	CounterspellNoComponents
)

// String returns "caster_not_seen", "out_of_range", "no_components" or "unknown".
func (r CounterspellRefusal) String() string {
	return enumName([]string{"caster_not_seen", "out_of_range", "no_components"}, int(r))
}

// Assess decides whether Counterspell can counter the cast (dispel-and-counterspell.md
// FR-031, table T-C). counterspeller is the creature casting Counterspell,
// distanceFeet the distance to the casting creature, seesCaster whether it sees
// the caster, and dc the spell save DC of the counterspeller. It returns
// CounterspellEligible when all three hold, and otherwise CounterspellIneligible
// with the first refusal in the order caster not seen, out of range, no
// components. It returns an error matching srderr.ErrInvalidInput for a zero
// counterspeller or a negative distance. It never panics.
func (Counterspell) Assess(obs CastObservation, counterspeller CombatantID, distanceFeet int, seesCaster bool, dc character.SpellSaveDC) (CounterspellAssessment, error) {
	if counterspeller.IsZero() {
		return nil, invalidInput("counterspell needs a counterspeller")
	}
	if distanceFeet < 0 {
		return nil, invalidInput("distanceFeet must not be negative")
	}
	switch {
	case !seesCaster:
		return CounterspellIneligible{refusal: CounterspellCasterNotSeen}, nil
	case distanceFeet > counterspellRange:
		return CounterspellIneligible{refusal: CounterspellOutOfRange}, nil
	case !obs.used.Any():
		return CounterspellIneligible{refusal: CounterspellNoComponents}, nil
	}
	return CounterspellEligible{counterspeller: counterspeller, obs: obs, dc: dc}, nil
}
