package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// SwarmRules are the rules the Swarm trait gives its creature (SRD 5.2.1, "Swarm of Crawling
// Claws", p. 278; p. 361). It is a comparable value built by SwarmTrait.Rules.
type SwarmRules struct{ size content.CreatureSize }

// Rules returns the rules of the swarm.
func (SwarmTrait) Rules() SwarmRules { return SwarmRules{content.SizeTiny} }

// CannotRegainHitPoints reports that the swarm cannot regain Hit Points: true.
func (SwarmRules) CannotRegainHitPoints() bool { return true }

// CanGainTemporaryHitPoints reports that the swarm cannot gain Temporary Hit Points: false.
func (SwarmRules) CanGainTemporaryHitPoints() bool { return false }

// SharesSpaces reports that the swarm can occupy another creature's space and vice versa:
// true.
func (SwarmRules) SharesSpaces() bool { return true }

// SmallestOpening returns the size of the smallest opening the swarm moves through: Tiny.
func (r SwarmRules) SmallestOpening() content.CreatureSize { return r.size }

// SwarmChange is a change to a swarm's Hit Points that CheckSwarmChange judges. Variants:
// SwarmHealing, SwarmTemporaryHitPoints, SwarmMaximumReduction.
type SwarmChange interface{ isSwarmChange() }

// SwarmHealing is Healing, a call to HitPointStatus.Restore.
type SwarmHealing struct{}

// SwarmTemporaryHitPoints is a gain of Temporary Hit Points, a call to
// HitPointStatus.GainTemporary.
type SwarmTemporaryHitPoints struct{}

// SwarmMaximumReduction is a reduction of the Hit Point maximum, which a swarm allows.
type SwarmMaximumReduction struct{}

func (SwarmHealing) isSwarmChange()            {}
func (SwarmTemporaryHitPoints) isSwarmChange() {}
func (SwarmMaximumReduction) isSwarmChange()   {}

// SwarmRefusal says why a swarm refuses a change. The zero value is not a refusal.
// Comparable and safe for concurrent use.
type SwarmRefusal int

const (
	// SwarmCannotRegainHitPoints: a swarm cannot regain Hit Points.
	SwarmCannotRegainHitPoints SwarmRefusal = iota + 1
	// SwarmCannotGainTemporaryHitPoints: a swarm cannot gain Temporary Hit Points.
	SwarmCannotGainTemporaryHitPoints
)

// String returns "cannot_regain_hit_points", "cannot_gain_temporary_hit_points" or
// "unknown".
func (r SwarmRefusal) String() string {
	return enumName([]string{"cannot_regain_hit_points", "cannot_gain_temporary_hit_points"}, int(r))
}

// CheckSwarmChange says whether the swarm t refuses change (SRD 5.2.1, p. 278; "Temporary Hit
// Points", p. 18). It returns the refusal and true for SwarmHealing and
// SwarmTemporaryHitPoints, and (0, false), permitted, for SwarmMaximumReduction, a nil
// change and a zero trait. The host calls it before Restore or GainTemporary. It never fails.
func CheckSwarmChange(t SwarmTrait, change SwarmChange) (SwarmRefusal, bool) {
	if t.IsZero() {
		return 0, false
	}
	switch change.(type) {
	case SwarmHealing:
		return SwarmCannotRegainHitPoints, true
	case SwarmTemporaryHitPoints:
		return SwarmCannotGainTemporaryHitPoints, true
	}
	return 0, false
}

// FlybyExemptsFromOpportunityAttack reports whether the creature provokes no Opportunity
// Attack when it moves out of an enemy's reach in mode (SRD 5.2.1, "Hippogriff", p. 298;
// "Opportunity Attacks", p. 185): true only for MovementModeFly, and false for every other
// mode and for a zero trait. It never changes how a creature departs, and Disengage is
// unaffected.
func FlybyExemptsFromOpportunityAttack(t FlybyTrait, mode MovementMode) bool {
	return !t.IsZero() && mode == MovementModeFly
}

// NarrowestSpaceInches returns the narrowest space the creature moves through without extra
// movement: 1 inch (SRD 5.2.1, "Black Pudding", p. 265).
func (AmorphousTrait) NarrowestSpaceInches() int { return 1 }

// NarrowOpeningCostsExtraMovement reports whether a narrow opening costs the creature extra
// movement: false. Terrain is host-supplied (the grid package types), so this is a fact the
// host honours when it builds terrain: it must not mark narrow-opening squares as Difficult
// Terrain for this mover (SRD 5.2.1, "Difficult Terrain", p. 181).
func (AmorphousTrait) NarrowOpeningCostsExtraMovement() bool { return false }

// DamageRecipient says what damage is dealt to. The zero value is invalid. Comparable and
// safe for concurrent use.
type DamageRecipient int

const (
	// DamageRecipientCreature: a creature.
	DamageRecipientCreature DamageRecipient = iota + 1
	// DamageRecipientObjectOrStructure: an object or a structure (a CompositeObject).
	DamageRecipientObjectOrStructure
)

// String returns "creature", "object_or_structure" or "unknown".
func (r DamageRecipient) String() string {
	return enumName([]string{"creature", "object_or_structure"}, int(r))
}

// ApplySiegeMonster returns damages as Siege Monster changes them for recipient (SRD 5.2.1,
// "Treant", p. 333): new Damage values with each amount doubled and the same Damage Type for
// DamageRecipientObjectOrStructure (saturating at the largest int), and equal values for
// DamageRecipientCreature. The doubling is a multiplier, so it comes before the target's
// adjustments, Resistance and Vulnerability (SRD 5.2.1, "Order of Application", p. 17;
// interpretation). Pass the result to ApplyDamage or ApplyDamageToObject. damages is not
// changed.
//
// It returns an error matching srderr.ErrInvalidInput for a zero trait, an empty damages, an
// invalid Damage or a recipient that is not valid, and a LimitExceededError (plan P10) for more than 64 damages. It
// never panics.
func ApplySiegeMonster(t SiegeMonsterTrait, damages []Damage, r DamageRecipient) ([]Damage, error) {
	if t.IsZero() || len(damages) == 0 || (r != DamageRecipientCreature && r != DamageRecipientObjectOrStructure) {
		return nil, invalidInput("siege monster needs a trait, damage and a recipient")
	}
	if err := overCap("siege monster damages", len(damages), maxSiegeDamages); err != nil {
		return nil, err
	}
	out := make([]Damage, len(damages))
	for i, d := range damages {
		if d.amount < 0 || !d.typ.Valid() {
			return nil, invalidInput("siege monster got an invalid Damage")
		}
		out[i] = d
		if r == DamageRecipientObjectOrStructure && d.amount > 0 {
			out[i].amount = satMulPositive(d.amount, 2)
		}
	}
	return out, nil
}
