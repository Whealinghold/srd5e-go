package character

import "slices"

// RestEffect is one thing a finished rest confers. The effects are data the host
// applies in the context that owns each (Hit Points, Temporary Hit Points, Hit
// Point maximum, ability scores, Exhaustion, features); this package holds none
// of that state. Variants: SpendHitPointDiceAvailable, AllLostHitPointsRegained,
// HitPointDiceRegained, HitPointMaximumReturnedToNormal,
// AbilityScoresReturnedToNormal, ExhaustionLevelReduced,
// TemporaryHitPointsEndedByRest, FeaturesRecharged.
type RestEffect interface{ isRestEffect() }

// SpendHitPointDiceAvailable: the creature may spend Hit Point Dice (Short Rest
// benefit).
type SpendHitPointDiceAvailable struct{}

// AllLostHitPointsRegained: the creature regains all lost Hit Points.
type AllLostHitPointsRegained struct{}

// HitPointDiceRegained: the creature regains every spent Hit Point Die; Count is
// how many were spent, at least 1.
type HitPointDiceRegained struct{ count int }

// Count returns the number of spent dice regained.
func (e HitPointDiceRegained) Count() int { return e.count }

// HitPointMaximumReturnedToNormal: a reduced Hit Point maximum returns to normal.
type HitPointMaximumReturnedToNormal struct{}

// AbilityScoresReturnedToNormal: reduced ability scores return to normal.
type AbilityScoresReturnedToNormal struct{}

// ExhaustionLevelReduced: the Exhaustion level drops by 1, From to To. The effect
// states the Long Rest rule alone; a host that has Exhaustion locked by
// dehydration or malnutrition applies it through the owner of that lock.
type ExhaustionLevelReduced struct{ from, to int }

// From returns the level before the rest, 1 to 6.
func (e ExhaustionLevelReduced) From() int { return e.from }

// To returns the level after the rest, From minus 1.
func (e ExhaustionLevelReduced) To() int { return e.to }

// TemporaryHitPointsEndedByRest: Temporary Hit Points end with the rest (SRD
// 5.2.1, "Temporary Hit Points", "Duration", p. 18). It is data the host
// applies, not an event.
type TemporaryHitPointsEndedByRest struct{}

// FeaturesRecharged: the features whose trigger the rest satisfies recharge; use
// RestKind.Recharges to test a trigger. How much a feature regains is the
// feature's own rule.
type FeaturesRecharged struct{ kind RestKind }

// Kind returns the kind of rest whose recharge applies.
func (e FeaturesRecharged) Kind() RestKind { return e.kind }

func (SpendHitPointDiceAvailable) isRestEffect()      {}
func (AllLostHitPointsRegained) isRestEffect()        {}
func (HitPointDiceRegained) isRestEffect()            {}
func (HitPointMaximumReturnedToNormal) isRestEffect() {}
func (AbilityScoresReturnedToNormal) isRestEffect()   {}
func (ExhaustionLevelReduced) isRestEffect()          {}
func (TemporaryHitPointsEndedByRest) isRestEffect()   {}
func (FeaturesRecharged) isRestEffect()               {}

// RestBenefits is what a rest of one kind confers: the ordered effects and the
// spent Hit Point Dice afterwards. It is immutable; it holds a slice, so compare
// the parts, not the value with ==. The zero RestBenefits confers nothing.
type RestBenefits struct {
	effects []RestEffect
	spent   SpentHitDice
}

// NewRestBenefits returns the benefits of a rest of kind benefitsOf (SRD 5.2.1,
// "Short Rest", "Benefits of the Rest", p. 187; "Long Rest", p. 185) for a
// creature at exhaustionLevel with the given spent dice.
//
// A ShortRest gives SpendHitPointDiceAvailable and FeaturesRecharged(ShortRest)
// and leaves the spent dice unchanged. A LongRest gives, in this order and
// skipping the omitted: AllLostHitPointsRegained, HitPointDiceRegained (only
// when some dice are spent; the spent dice become none),
// HitPointMaximumReturnedToNormal, AbilityScoresReturnedToNormal,
// ExhaustionLevelReduced (only at level 1 or more), TemporaryHitPointsEndedByRest
// and FeaturesRecharged(LongRest). It can be called without a RestReport, to
// grant the benefits of a Short Rest to a creature that did not rest, and raises
// no event.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid benefitsOf
// or an exhaustionLevel outside 0 to 6. It never panics.
func NewRestBenefits(benefitsOf RestKind, exhaustionLevel int, spent SpentHitDice) (RestBenefits, error) {
	if !benefitsOf.Valid() || exhaustionLevel < 0 || exhaustionLevel > maxExhaustionLevel {
		return RestBenefits{}, invalidInput("rest benefits of an invalid kind or Exhaustion level")
	}
	return restBenefits(benefitsOf, exhaustionLevel, spent), nil
}

// restBenefits builds the benefits of a kind and level already known to be valid.
func restBenefits(benefitsOf RestKind, exhaustionLevel int, spent SpentHitDice) RestBenefits {
	if benefitsOf == ShortRest {
		return RestBenefits{effects: []RestEffect{SpendHitPointDiceAvailable{}, FeaturesRecharged{kind: ShortRest}}, spent: spent}
	}
	effects := []RestEffect{AllLostHitPointsRegained{}}
	if n := spent.Total(); n > 0 {
		effects = append(effects, HitPointDiceRegained{count: n})
	}
	effects = append(effects, HitPointMaximumReturnedToNormal{}, AbilityScoresReturnedToNormal{})
	if exhaustionLevel > 0 {
		effects = append(effects, ExhaustionLevelReduced{from: exhaustionLevel, to: exhaustionLevel - 1})
	}
	effects = append(effects, TemporaryHitPointsEndedByRest{}, FeaturesRecharged{kind: LongRest})
	return RestBenefits{effects: effects}
}

// Effects returns a new slice of the effects, in the order of NewRestBenefits.
func (b RestBenefits) Effects() []RestEffect { return slices.Clone(b.effects) }

// SpentHitDice returns the spent Hit Point Dice after the rest.
func (b RestBenefits) SpentHitDice() SpentHitDice { return b.spent }
