package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// SpeciesTraitEffect is the effect a species trait declares when used: plain facts
// the host or combat interprets, a closed set and published language. Nothing is
// rolled or applied here. Variants: BreathWeaponEffect, DraconicFlightEffect,
// StonecunningEffect, CloudsJauntEffect, FiresBurnEffect, FrostsChillEffect,
// HillsTumbleEffect, StormsThunderEffect, LargeFormEffect, AdrenalineRushEffect,
// SpeciesFreeCastEffect. All are comparable.
type SpeciesTraitEffect interface{ isSpeciesTraitEffect() }

// GiantAncestryEffect is the effect of a Giant Ancestry benefit other than Stone's
// Endurance, which RollStonesEndurance resolves. It is a SpeciesTraitEffect.
// Variants: CloudsJauntEffect, FiresBurnEffect, FrostsChillEffect,
// HillsTumbleEffect, StormsThunderEffect.
type GiantAncestryEffect interface {
	SpeciesTraitEffect
	isGiantAncestryEffect()
}

// BreathWeaponEffect declares a Breath Weapon (SRD 5.2.1, p. 84): the shape, a
// Dexterity saving throw against SaveDC, Damage of DamageType (half on a success),
// and that it replaces one attack of the Attack action. Comparable.
type BreathWeaponEffect struct {
	Shape             BreathWeaponShape
	SaveAbility       ability.Ability
	SaveDC            int
	Damage            dice.Expression
	DamageType        content.DamageType
	HalfOnSave        bool
	ReplacesOneAttack bool
}

// DraconicFlightEffect declares Draconic Flight (SRD 5.2.1, p. 84). Comparable.
type DraconicFlightEffect struct {
	Cost                  SpeciesTraitAction
	DurationMinutes       int
	FlySpeedEqualsSpeed   bool
	EndsWhenIncapacitated bool
}

// StonecunningEffect declares Stonecunning (SRD 5.2.1, p. 84). Using it on or
// touching a stone surface is the host's fact. Comparable.
type StonecunningEffect struct {
	Cost            SpeciesTraitAction
	TremorsenseFeet int
	DurationMinutes int
}

// CloudsJauntEffect declares Cloud's Jaunt (SRD 5.2.1, p. 85): teleport to an
// unoccupied space the Goliath can see. Comparable.
type CloudsJauntEffect struct {
	Cost         SpeciesTraitAction
	TeleportFeet int
}

// FiresBurnEffect declares Fire's Burn (SRD 5.2.1, p. 85): extra damage on a hit
// that deals damage. The dice are declared, not rolled. Comparable.
type FiresBurnEffect struct {
	Damage     dice.Expression
	DamageType content.DamageType
}

// FrostsChillEffect declares Frost's Chill (SRD 5.2.1, p. 85): extra damage on a
// hit that deals damage and the target's Speed reduced until the start of the
// Goliath's next turn. Comparable.
type FrostsChillEffect struct {
	Damage             dice.Expression
	DamageType         content.DamageType
	SpeedReductionFeet int
}

// HillsTumbleEffect declares Hill's Tumble (SRD 5.2.1, p. 85): on a hit that deals
// damage to a target of MaxTargetSize or smaller, the target gains Condition.
// Comparable.
type HillsTumbleEffect struct {
	MaxTargetSize content.CreatureSize
	Condition     content.ConditionKind
}

// StormsThunderEffect declares Storm's Thunder (SRD 5.2.1, p. 86): a Reaction to
// damage from a creature within RangeFeet, dealing Damage to it. Comparable.
type StormsThunderEffect struct {
	Cost       SpeciesTraitAction
	RangeFeet  int
	Damage     dice.Expression
	DamageType content.DamageType
}

// LargeFormEffect declares Large Form (SRD 5.2.1, p. 86). Comparable.
type LargeFormEffect struct {
	Cost                   SpeciesTraitAction
	Size                   content.CreatureSize
	DurationMinutes        int
	SpeedBonusFeet         int
	StrengthCheckAdvantage SpeciesAdvantage
}

// TemporaryHitPointsGrant is an amount of Temporary Hit Points to give. Whether it
// replaces the Temporary Hit Points already held is the rule of combat ("They Don't
// Stack", SRD 5.2.1, p. 18). Comparable.
type TemporaryHitPointsGrant struct{ Amount int }

// AdrenalineRushEffect declares Adrenaline Rush (SRD 5.2.1, p. 86): the Dash action
// as a Bonus Action and Temporary Hit Points. Comparable.
type AdrenalineRushEffect struct {
	DashAsBonusAction  bool
	TemporaryHitPoints TemporaryHitPointsGrant
}

// SpeciesFreeCastEffect declares that the lineage or legacy spell, or Speak with
// Animals, may be cast once without a spell slot, using Ability. Casting it is the
// casting specs'. Comparable.
type SpeciesFreeCastEffect struct {
	Spell   content.SpellID
	Ability ability.Ability
}

func (BreathWeaponEffect) isSpeciesTraitEffect()    {}
func (DraconicFlightEffect) isSpeciesTraitEffect()  {}
func (StonecunningEffect) isSpeciesTraitEffect()    {}
func (CloudsJauntEffect) isSpeciesTraitEffect()     {}
func (FiresBurnEffect) isSpeciesTraitEffect()       {}
func (FrostsChillEffect) isSpeciesTraitEffect()     {}
func (HillsTumbleEffect) isSpeciesTraitEffect()     {}
func (StormsThunderEffect) isSpeciesTraitEffect()   {}
func (LargeFormEffect) isSpeciesTraitEffect()       {}
func (AdrenalineRushEffect) isSpeciesTraitEffect()  {}
func (SpeciesFreeCastEffect) isSpeciesTraitEffect() {}

func (CloudsJauntEffect) isGiantAncestryEffect()   {}
func (FiresBurnEffect) isGiantAncestryEffect()     {}
func (FrostsChillEffect) isGiantAncestryEffect()   {}
func (HillsTumbleEffect) isGiantAncestryEffect()   {}
func (StormsThunderEffect) isGiantAncestryEffect() {}

// SpeciesEffects declares the effects of the traits that have one (SRD 5.2.1, pp.
// 84-86). The zero value is ready to use. It is stateless and safe for concurrent use.
type SpeciesEffects struct{}

// GiantAncestry returns the effect of the benefit a, and false for Stone's
// Endurance (resolved by RollStonesEndurance) and an invalid value.
func (SpeciesEffects) GiantAncestry(a GiantAncestry) (GiantAncestryEffect, bool) {
	switch a {
	case AncestryCloud:
		return CloudsJauntEffect{Cost: TraitBonusAction, TeleportFeet: 30}, true
	case AncestryFire:
		return FiresBurnEffect{Damage: dice.MustExpression(1, dice.D10(), 0), DamageType: content.DamageFire}, true
	case AncestryFrost:
		return FrostsChillEffect{Damage: dice.MustExpression(1, dice.D6(), 0), DamageType: content.DamageCold, SpeedReductionFeet: 10}, true
	case AncestryHill:
		return HillsTumbleEffect{MaxTargetSize: content.SizeLarge, Condition: content.ConditionProne}, true
	case AncestryStone:
		return nil, false // Stone's Endurance is resolved by RollStonesEndurance
	case AncestryStorm:
		return StormsThunderEffect{Cost: TraitReaction, RangeFeet: 60, Damage: dice.MustExpression(1, dice.D8(), 0), DamageType: content.DamageThunder}, true
	}
	return nil, false
}

// breathWeaponDice is the number of d10 of the Breath Weapon at a total level: 1 at
// levels 1 to 4, 2 at 5 to 10, 3 at 11 to 16 and 4 at 17 to 20.
func breathWeaponDice(level int) int {
	switch {
	case level >= 17:
		return 4
	case level >= 11:
		return 3
	case level >= 5:
		return 2
	}
	return 1
}

// Of returns the effect of req for the character c with traits t, and false when
// req declares none: a nil or invalid request, a request of a trait t does not
// have, or one that has its own use case (Stone's Endurance, Relentless
// Endurance). It does not check levels, uses left or space; SpeciesUses.Expend does.
// The Breath Weapon DC is 8 plus the Constitution modifier and Proficiency Bonus.
// It never panics.
func (e SpeciesEffects) Of(req SpeciesUseRequest, c Character, t SpeciesTraits) (SpeciesTraitEffect, bool) {
	use, ok := requestedUse(req)
	if !ok || !t.offersRequest(req, use) || hasOwnUseCase(req) {
		return nil, false
	}
	pb := c.ProficiencyBonus().Value()
	switch v := req.(type) {
	case BreathWeaponRequest:
		d := t.choices.(DragonbornChoices).Ancestry
		return BreathWeaponEffect{Shape: v.Shape, SaveAbility: ability.Dexterity,
			SaveDC: 8 + c.AbilityScores().Modifier(ability.Constitution).Value() + pb,
			Damage: dice.MustExpression(breathWeaponDice(c.TotalLevel().Value()), dice.D10(), 0), DamageType: d.DamageType(),
			HalfOnSave: true, ReplacesOneAttack: true}, true
	case DraconicFlightRequest:
		return DraconicFlightEffect{Cost: TraitBonusAction, DurationMinutes: 10, FlySpeedEqualsSpeed: true, EndsWhenIncapacitated: true}, true
	case StonecunningRequest:
		return StonecunningEffect{Cost: TraitBonusAction, TremorsenseFeet: 60, DurationMinutes: 10}, true
	case GiantAncestryRequest:
		return e.GiantAncestry(v.Benefit)
	case LargeFormRequest:
		return LargeFormEffect{Cost: TraitBonusAction, Size: content.SizeLarge, DurationMinutes: 10, SpeedBonusFeet: 10,
			StrengthCheckAdvantage: SpeciesAdvantage{Source: TraitLargeForm, Circumstance: modifier.Advantage{}}}, true
	case AdrenalineRushRequest:
		return AdrenalineRushEffect{DashAsBonusAction: true, TemporaryHitPoints: TemporaryHitPointsGrant{Amount: pb}}, true
	}
	return freeCastEffect(use, c, t)
}

// freeCastEffect is the effect of the free cast uses: the granted spell that carries
// use, with its ability.
func freeCastEffect(use SpeciesUse, c Character, t SpeciesTraits) (SpeciesTraitEffect, bool) {
	for _, s := range (SpeciesGrantedSpells{}).Of(t, c.TotalLevel()) {
		if u, has := s.Use(); has && u == use {
			return SpeciesFreeCastEffect{Spell: s.Spell(), Ability: s.Ability()}, true
		}
	}
	return nil, false
}

// chosenGiant returns the Giant Ancestry of a Goliath, or 0.
func chosenGiant(t SpeciesTraits) GiantAncestry {
	g, _ := t.choices.(GoliathChoices)
	return g.Ancestry
}

// offersRequest reports whether t offers the use of req: for Giant Ancestry and
// Stone's Endurance only the benefit the Goliath chose.
func (t SpeciesTraits) offersRequest(req SpeciesUseRequest, use SpeciesUse) bool {
	if !t.Offers(use) {
		return false
	}
	switch v := req.(type) {
	case GiantAncestryRequest:
		return chosenGiant(t) == v.Benefit
	case StonesEnduranceRequest:
		return chosenGiant(t) == AncestryStone
	}
	return true
}

// hasOwnUseCase reports whether req belongs to RollStonesEndurance or
// UseRelentlessEndurance.
func hasOwnUseCase(req SpeciesUseRequest) bool {
	switch v := req.(type) {
	case StonesEnduranceRequest, RelentlessEnduranceRequest:
		return true
	case GiantAncestryRequest:
		return v.Benefit == AncestryStone
	}
	return false
}
