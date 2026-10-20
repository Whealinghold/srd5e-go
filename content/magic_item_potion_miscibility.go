package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// MixtureChoice is a decision the SRD leaves to "you" when potions are mixed
// (SRD 5.2.1, "Potion Miscibility", p. 204), a closed set of five members.
// The package never makes the choice. The zero MixtureChoice is invalid.
// Comparable and safe for concurrent use.
type MixtureChoice int

// The mixture choices.
const (
	MixtureChoicePoison MixtureChoice = iota + 1
	MixtureChoiceLosingPotion
	MixtureChoiceDoubledPotion
	MixtureChoiceSurvivingPotion
	MixtureChoicePermanentEffect
)

func mixtureChoiceNames() []string {
	return []string{"poison", "losingPotion", "doubledPotion", "survivingPotion", "permanentEffect"}
}

// Valid reports whether c is one of the five choices.
func (c MixtureChoice) Valid() bool { return enumValid(mixtureChoiceNames(), int(c)) }

// String returns the name of c, or "unknown" for an undefined value.
func (c MixtureChoice) String() string { return enumString(mixtureChoiceNames(), int(c)) }

// ExplosionOrigin says where an exploding mixture is centred. The only member
// is [ExplosionOriginMixture]: the mixture itself, not the drinker, because
// potions can be poured into a container. The zero ExplosionOrigin is
// invalid. Comparable and safe for concurrent use.
type ExplosionOrigin int

// ExplosionOriginMixture is a point of origin at the mixture itself.
const ExplosionOriginMixture ExplosionOrigin = 1

// Valid reports whether o is [ExplosionOriginMixture].
func (o ExplosionOrigin) Valid() bool { return o == ExplosionOriginMixture }

// String returns "mixture" for [ExplosionOriginMixture] and "unknown" for an
// undefined value.
func (o ExplosionOrigin) String() string { return enumString([]string{"mixture"}, int(o)) }

// ExplosionShape is the shape of an exploding mixture's area. The only member
// is [ExplosionSphere]. The zero ExplosionShape is invalid. Comparable and safe
// for concurrent use.
type ExplosionShape int

// ExplosionSphere is a Sphere.
const ExplosionSphere ExplosionShape = 1

// Valid reports whether s is [ExplosionSphere].
func (s ExplosionShape) Valid() bool { return s == ExplosionSphere }

// String returns "Sphere" for [ExplosionSphere] and "unknown" for an
// undefined value.
func (s ExplosionShape) String() string { return enumString([]string{"Sphere"}, int(s)) }

// ExplosionFacts are the facts of the "Potions Explode" row: a Sphere of a
// radius in feet around an origin, a damage roll and a damage type, affecting
// each creature in the area. The zero value is the zero ExplosionFacts that
// only [PotionsExplode] fills. Comparable and safe for concurrent use.
type ExplosionFacts struct {
	radiusFeet int
	shape      ExplosionShape
	origin     ExplosionOrigin
	damage     dice.Expression
	damageType DamageType
}

// RadiusFeet returns the radius of the area in feet.
func (e ExplosionFacts) RadiusFeet() int { return e.radiusFeet }

// Shape returns the shape of the area.
func (e ExplosionFacts) Shape() ExplosionShape { return e.shape }

// Origin returns the point of origin.
func (e ExplosionFacts) Origin() ExplosionOrigin { return e.origin }

// Damage returns the damage roll.
func (e ExplosionFacts) Damage() dice.Expression { return e.damage }

// DamageType returns the damage type.
func (e ExplosionFacts) DamageType() DamageType { return e.damageType }

// PotionMiscibilityEffect is the outcome of one row of the Potion Miscibility
// table, a closed family with eight variants: [PotionsExplode],
// [PotionsBecomeIngestedPoison], [BothPotionsLoseEffects],
// [OnePotionLosesEffect], [PotionEffectsHalved], [PotionsWorkNormally],
// [OnePotionEffectsDoubled] and [OnePotionPermanent]. Each exposes the facts of
// its row as typed fields, never text. Every variant is an immutable,
// comparable value whose zero value is valid, and safe for concurrent use.
type PotionMiscibilityEffect interface {
	// BothPotionsLoseEffects reports whether both potions lose their effects.
	BothPotionsLoseEffects() bool
	// HostChoices returns the decisions the SRD leaves to the host, as a new
	// slice, nil when there are none.
	HostChoices() []MixtureChoice
	isPotionMiscibilityEffect()
}

// PotionsExplode is a roll of 1: the mixture explodes. Both potions lose their
// effects.
type PotionsExplode struct{}

// Explosion returns the facts of the explosion: a 5-foot-radius Sphere centred
// on the mixture, 4d10 Force damage to each creature in the area.
func (PotionsExplode) Explosion() ExplosionFacts {
	return ExplosionFacts{
		radiusFeet: 5, shape: ExplosionSphere, origin: ExplosionOriginMixture,
		damage: dice.MustParseExpression("4d10"), damageType: DamageForce,
	}
}

// BothPotionsLoseEffects returns true.
func (PotionsExplode) BothPotionsLoseEffects() bool { return true }

// HostChoices returns nil.
func (PotionsExplode) HostChoices() []MixtureChoice { return nil }
func (PotionsExplode) isPotionMiscibilityEffect()   {}

// PotionsBecomeIngestedPoison is a roll of 2 to 8: the mixture becomes a
// poison of the host's choice. Both potions lose their effects.
type PotionsBecomeIngestedPoison struct{}

// BothPotionsLoseEffects returns true.
func (PotionsBecomeIngestedPoison) BothPotionsLoseEffects() bool { return true }

// HostChoices returns {poison}.
func (PotionsBecomeIngestedPoison) HostChoices() []MixtureChoice {
	return []MixtureChoice{MixtureChoicePoison}
}
func (PotionsBecomeIngestedPoison) isPotionMiscibilityEffect() {}

// BothPotionsLoseEffects is a roll of 9 to 15: both potions lose their
// effects.
type BothPotionsLoseEffects struct{}

// BothPotionsLoseEffects returns true.
func (BothPotionsLoseEffects) BothPotionsLoseEffects() bool { return true }

// HostChoices returns nil.
func (BothPotionsLoseEffects) HostChoices() []MixtureChoice { return nil }
func (BothPotionsLoseEffects) isPotionMiscibilityEffect()   {}

// OnePotionLosesEffect is a roll of 16 to 25: one potion, of the host's
// choice, loses its effect.
type OnePotionLosesEffect struct{}

// BothPotionsLoseEffects returns false.
func (OnePotionLosesEffect) BothPotionsLoseEffects() bool { return false }

// HostChoices returns {losingPotion}.
func (OnePotionLosesEffect) HostChoices() []MixtureChoice {
	return []MixtureChoice{MixtureChoiceLosingPotion}
}
func (OnePotionLosesEffect) isPotionMiscibilityEffect() {}

// PotionEffectsHalved is a roll of 26 to 35: the numerical effects and
// durations of both potions are halved.
type PotionEffectsHalved struct{}

// PotionWithNothingToHalveLosesEffect returns true: a potion with no numerical
// effect or duration to halve loses its effect.
func (PotionEffectsHalved) PotionWithNothingToHalveLosesEffect() bool { return true }

// BothPotionsLoseEffects returns false.
func (PotionEffectsHalved) BothPotionsLoseEffects() bool { return false }

// HostChoices returns nil.
func (PotionEffectsHalved) HostChoices() []MixtureChoice { return nil }
func (PotionEffectsHalved) isPotionMiscibilityEffect()   {}

// PotionsWorkNormally is a roll of 36 to 90: the potions work normally.
type PotionsWorkNormally struct{}

// BothPotionsLoseEffects returns false.
func (PotionsWorkNormally) BothPotionsLoseEffects() bool { return false }

// HostChoices returns nil.
func (PotionsWorkNormally) HostChoices() []MixtureChoice { return nil }
func (PotionsWorkNormally) isPotionMiscibilityEffect()   {}

// OnePotionEffectsDoubled is a roll of 91 to 99: the numerical effects and
// duration of one potion, of the host's choice, are doubled.
type OnePotionEffectsDoubled struct{}

// WorkNormallyIfNothingToDouble returns true: the potions work normally when
// neither has anything to double.
func (OnePotionEffectsDoubled) WorkNormallyIfNothingToDouble() bool { return true }

// BothPotionsLoseEffects returns false.
func (OnePotionEffectsDoubled) BothPotionsLoseEffects() bool { return false }

// HostChoices returns {doubledPotion}.
func (OnePotionEffectsDoubled) HostChoices() []MixtureChoice {
	return []MixtureChoice{MixtureChoiceDoubledPotion}
}
func (OnePotionEffectsDoubled) isPotionMiscibilityEffect() {}

// OnePotionPermanent is a roll of 100 (the table's "00"): only one potion
// works, with permanent effects.
type OnePotionPermanent struct{}

// BothPotionsLoseEffects returns false.
func (OnePotionPermanent) BothPotionsLoseEffects() bool { return false }

// HostChoices returns {survivingPotion, permanentEffect}.
func (OnePotionPermanent) HostChoices() []MixtureChoice {
	return []MixtureChoice{MixtureChoiceSurvivingPotion, MixtureChoicePermanentEffect}
}
func (OnePotionPermanent) isPotionMiscibilityEffect() {}

// Potion Miscibility rolls run from 1 to 100, the table's "00" being 100.
const (
	potionMiscibilityMin = 1
	potionMiscibilityMax = 100
)

// PotionMiscibilityBand is one row of the Potion Miscibility table: the rolls
// minRoll to maxRoll (inclusive) and their effect. The zero value is invalid;
// build it with [NewPotionMiscibilityBand]. Comparable.
type PotionMiscibilityBand struct {
	minRoll, maxRoll int
	effect           PotionMiscibilityEffect
}

// NewPotionMiscibilityBand returns the band of rolls min to max with effect.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// when min is above max or a roll is outside 1 to 100, BlockInvalidMember for a
// pointer to a variant (nil or not), BlockMissing for a nil
// effect. It never panics.
func NewPotionMiscibilityBand(minRoll, maxRoll int, effect PotionMiscibilityEffect) (PotionMiscibilityBand, error) {
	const block = "PotionMiscibilityBand"
	if minRoll > maxRoll || minRoll < potionMiscibilityMin || maxRoll > potionMiscibilityMax {
		return PotionMiscibilityBand{}, blockError(block, BlockOutOfRange)
	}
	if effect == nil {
		return PotionMiscibilityBand{}, blockError(block, BlockMissing)
	}
	switch effect.(type) {
	case PotionsExplode, PotionsBecomeIngestedPoison, BothPotionsLoseEffects, OnePotionLosesEffect,
		PotionEffectsHalved, PotionsWorkNormally, OnePotionEffectsDoubled, OnePotionPermanent:
	default: // a pointer to a variant (nil or not) is not a member of the closed set
		return PotionMiscibilityBand{}, blockError(block, BlockInvalidMember)
	}
	return PotionMiscibilityBand{minRoll: minRoll, maxRoll: maxRoll, effect: effect}, nil
}

// MinRoll returns the lowest roll of the band.
func (b PotionMiscibilityBand) MinRoll() int { return b.minRoll }

// MaxRoll returns the highest roll of the band.
func (b PotionMiscibilityBand) MaxRoll() int { return b.maxRoll }

// Effect returns the effect of the band.
func (b PotionMiscibilityBand) Effect() PotionMiscibilityEffect { return b.effect }

// PotionMiscibilityTable is the Potion Miscibility table (SRD 5.2.1, "Potion
// Miscibility", p. 204): eight bands that together cover every roll from 1 to
// 100 exactly once. It holds no state; its zero value is the table. The dice
// are rolled by the caller, never here. Safe for concurrent use.
type PotionMiscibilityTable struct{}

// potionMiscibilityBands returns the eight rows in table order.
func potionMiscibilityBands() []PotionMiscibilityBand {
	row := func(lo, hi int, e PotionMiscibilityEffect) PotionMiscibilityBand {
		return PotionMiscibilityBand{minRoll: lo, maxRoll: hi, effect: e}
	}
	return []PotionMiscibilityBand{
		row(1, 1, PotionsExplode{}),
		row(2, 8, PotionsBecomeIngestedPoison{}),
		row(9, 15, BothPotionsLoseEffects{}),
		row(16, 25, OnePotionLosesEffect{}),
		row(26, 35, PotionEffectsHalved{}),
		row(36, 90, PotionsWorkNormally{}),
		row(91, 99, OnePotionEffectsDoubled{}),
		row(100, 100, OnePotionPermanent{}),
	}
}

// Bands returns the eight bands in table order, as a new slice.
func (PotionMiscibilityTable) Bands() []PotionMiscibilityBand { return potionMiscibilityBands() }

// ForRoll returns the effect of the band holding roll, a d100 with 100 as the
// table's "00".
//
// It returns a *[RollRangeError] wrapping srderr.ErrInvalidInput for a roll
// below 1 or above 100. It never panics.
func (PotionMiscibilityTable) ForRoll(roll int) (PotionMiscibilityEffect, error) {
	if err := checkRoll(roll, potionMiscibilityMax); err != nil {
		return nil, err
	}
	i := slices.IndexFunc(potionMiscibilityBands(), func(b PotionMiscibilityBand) bool { return roll <= b.maxRoll })
	return potionMiscibilityBands()[i].effect, nil
}

// MustForRoll is like [PotionMiscibilityTable.ForRoll] but panics with its
// error when the roll is out of range.
func (t PotionMiscibilityTable) MustForRoll(roll int) PotionMiscibilityEffect {
	e, err := t.ForRoll(roll)
	if err != nil {
		panic(err)
	}
	return e
}
