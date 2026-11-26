package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// NetThrowRangeFeet is the farthest a Net can be thrown: a creature you can see within 15
// feet (SRD 5.2.1, "Net", p. 98).
const NetThrowRangeFeet = 15

// NetProfile holds the statistics of a Net as an object: Armor Class 10, 5 Hit Points and
// Immunity to Bludgeoning, Poison and Psychic damage (SRD 5.2.1, "Net", p. 98). The zero value
// is ready to use. Its size is Medium, a placeholder: no Net rule reads it (Q-8).
type NetProfile struct{}

// Status returns the Net as an undamaged ObjectStatus: Armor Class 10, current and maximum Hit
// Points 5, no Temporary Hit Points, no Damage Threshold, and the standard object affinities
// (Poison and Psychic) with the ruling Immunity to Bludgeoning.
func (NetProfile) Status() ObjectStatus {
	affinities := ObjectAffinities{}.StandardWith(DamageAffinities{immune: typeBit(content.DamageBludgeoning)})
	return ObjectStatus{content.SizeMedium, d20.ArmorClass(10), HitPoints{5, 5, 0}, DamageThreshold{}, affinities, NoAbilityScores{}}
}

// NetRules is the rules of throwing a Net (SRD 5.2.1, "Net", p. 98). The zero value is ready
// to use. Pure; safe for concurrent use.
type NetRules struct{}

// Possibility says whether a Net can be thrown at a creature distanceFeet away: it returns
// targetNotSeen when the thrower cannot see the target, then outOfReach beyond 15 feet (15 is
// accepted), else the zero GearRejection. The attack the throw replaces is recorded by
// AttackActionProgress; "you can see" is the host's.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemNegativeDistance
// for a negative distance or GearProblemInvalidValue for a sight that is not valid. It never
// panics.
func (NetRules) Possibility(distanceFeet int, sight ThrowerSight) (GearRejection, error) {
	switch {
	case distanceFeet < 0:
		return 0, gearInvalid(GearProblemNegativeDistance)
	case !sight.Valid():
		return 0, gearInvalid(GearProblemInvalidValue)
	}
	return netPossibility(distanceFeet, sight), nil
}

// netPossibility is Possibility for values known to be valid.
func netPossibility(distanceFeet int, sight ThrowerSight) GearRejection {
	switch {
	case sight == ThrowerSightCannotSeeTarget:
		return GearRejectionTargetNotSeen
	case distanceFeet > NetThrowRangeFeet:
		return GearRejectionOutOfReach
	}
	return 0
}

// SaveDC returns the Difficulty Class of the target's Dexterity saving throw: 8 plus the
// thrower's Dexterity modifier plus Proficiency Bonus, with no cap or floor. The sum
// saturates at the limits of int.
func (NetRules) SaveDC(abilityModifier, proficiencyBonus int) d20.DifficultyClass {
	return d20.DifficultyClass(satAdd(satAdd(8, abilityModifier), proficiencyBonus))
}

// AutoSucceeds reports whether a target of size succeeds against a Net without a roll: Huge
// or Gargantuan (SRD 5.2.1, "Net", p. 98; "Creature Size", p. 14).
func (NetRules) AutoSucceeds(size content.CreatureSize) bool {
	return size == content.SizeHuge || size == content.SizeGargantuan
}
