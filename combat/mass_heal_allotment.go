package combat

import "maps"

const (
	// maxMassHealTotal is the Hit Points Mass Heal restores in all ("up to 700").
	maxMassHealTotal = 700
	// maxMassHealCreatures is the cap on the creatures of one allotment: combatants per encounter (plan P10).
	maxMassHealCreatures = 256
)

// MassHealAllotment is how a caster divides the Hit Points of Mass Heal among
// creatures (SRD 5.2.1, "Mass Heal", p. 147). Build it with NewMassHealAllotment.
// The zero value is the empty allotment, which heals no one. Immutable and safe for
// concurrent use; it holds a map, so it is not comparable with ==.
type MassHealAllotment struct{ hitPoints map[CombatantID]int }

// NewMassHealAllotment returns the allotment of hitPoints, copied. An allotment of 0
// means the creature is not healed.
//
// It returns a *RestorationRangeError (matching srderr.ErrInvalidInput) for an
// allotment below 0, a total above 700 or more than 256 creatures, and an error
// matching srderr.ErrInvalidInput for a blank combatant id. A nil or empty map is
// the empty allotment. It never panics.
func NewMassHealAllotment(hitPoints map[CombatantID]int) (MassHealAllotment, error) {
	if len(hitPoints) > maxMassHealCreatures {
		return MassHealAllotment{}, rangeError("mass_heal_creatures", int64(len(hitPoints)))
	}
	total := 0
	for id, n := range hitPoints {
		if id.IsZero() {
			return MassHealAllotment{}, invalidInput("a Mass Heal allotment needs a combatant id")
		}
		if n < 0 {
			return MassHealAllotment{}, rangeError("mass_heal_allotment", int64(n))
		}
		total = satAdd(total, n)
	}
	if total > maxMassHealTotal {
		return MassHealAllotment{}, rangeError("mass_heal_total", int64(total))
	}
	return MassHealAllotment{maps.Clone(hitPoints)}, nil
}

// For returns the Hit Points allotted to id, 0 when it is not in the allotment.
func (a MassHealAllotment) For(id CombatantID) int { return a.hitPoints[id] }

// Total returns the Hit Points allotted in all, at most 700.
func (a MassHealAllotment) Total() int {
	total := 0
	for _, n := range a.hitPoints {
		total += n
	}
	return total
}

// Allotments returns a copy of the map of Hit Points by creature.
func (a MassHealAllotment) Allotments() map[CombatantID]int { return maps.Clone(a.hitPoints) }
