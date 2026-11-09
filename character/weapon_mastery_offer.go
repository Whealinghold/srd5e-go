package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// MasteryAttackOutcome is what the attack did, for the mastery effects: the
// closed set MasteryAttackRolled, MasteryAttackForcedMiss and
// MasteryAttackUnrolledMiss. It wraps the published d20.AttackRollResult and
// adds the two misses combat decides. It is a hit exactly when it is
// MasteryAttackRolled with a hit; every other outcome is a miss.
//
//sumtype:decl
type MasteryAttackOutcome interface{ isMasteryAttackOutcome() }

// MasteryAttackRolled is a roll that stands: a hit (a Critical Hit included) or
// a miss as d20 reported it.
type MasteryAttackRolled struct{ Roll d20.AttackRollResult }

// MasteryAttackForcedMiss is a roll a rule turned into a miss whatever it showed
// (the target was not at the targeted location), a natural 20 included.
type MasteryAttackForcedMiss struct{ Roll d20.AttackRollResult }

// MasteryAttackUnrolledMiss is a miss decided without a roll (the underwater
// automatic miss).
type MasteryAttackUnrolledMiss struct{}

func (MasteryAttackRolled) isMasteryAttackOutcome()       {}
func (MasteryAttackForcedMiss) isMasteryAttackOutcome()   {}
func (MasteryAttackUnrolledMiss) isMasteryAttackOutcome() {}

// masteryHit reports whether the outcome is a hit, and whether it is a miss with
// a roll that stands.
func masteryHit(o MasteryAttackOutcome) (hit, rolledMiss bool) {
	r, ok := o.(MasteryAttackRolled)
	return ok && r.Roll.Hit(), ok && !r.Roll.Hit()
}

// MasteryNoneReason says why no mastery effect is offered. When several apply,
// the first in this order is reported. The zero value is not a reason.
type MasteryNoneReason int

const (
	// MasteryNoneFeatureNotAvailable: the character has no feature that lets them use it.
	MasteryNoneFeatureNotAvailable MasteryNoneReason = iota + 1
	// MasteryNoneAttackDidNotQualify: the attack did not hit (or miss with a roll) as the property needs.
	MasteryNoneAttackDidNotQualify
	// MasteryNoneNotMeleeAttack: Cleave needs a melee attack.
	MasteryNoneNotMeleeAttack
	// MasteryNoneAlreadyUsedThisTurn: Cleave was already used this turn.
	MasteryNoneAlreadyUsedThisTurn
	// MasteryNoneTargetTooLarge: Push needs a target Large or smaller.
	MasteryNoneTargetTooLarge
	// MasteryNoneNoDamageDealt: Slow and Vex need damage dealt.
	MasteryNoneNoDamageDealt
)

// String returns the short name ("featureNotAvailable", ...) or "unknown".
func (r MasteryNoneReason) String() string {
	names := []string{"featureNotAvailable", "attackDidNotQualify", "notMeleeAttack", "alreadyUsedThisTurn", "targetTooLarge", "noDamageDealt"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// MasteryEffectExpiry is when a timed mastery effect ends. The zero value is not
// an expiry.
type MasteryEffectExpiry int

const (
	// ExpiryStartOfAttackersNextTurn ends the effect at the start of the attacker's next turn.
	ExpiryStartOfAttackersNextTurn MasteryEffectExpiry = iota + 1
	// ExpiryEndOfAttackersNextTurn ends the effect at the end of the attacker's next turn.
	ExpiryEndOfAttackersNextTurn
	// ExpiryWhenNextAttackRollMade ends the effect when the one attack roll it modifies is made.
	ExpiryWhenNextAttackRollMade
)

// String returns "startOfAttackersNextTurn", "endOfAttackersNextTurn",
// "whenNextAttackRollMade" or "unknown".
func (e MasteryEffectExpiry) String() string {
	switch e {
	case ExpiryStartOfAttackersNextTurn:
		return "startOfAttackersNextTurn"
	case ExpiryEndOfAttackersNextTurn:
		return "endOfAttackersNextTurn"
	case ExpiryWhenNextAttackRollMade:
		return "whenNextAttackRollMade"
	}
	return "unknown"
}

// maxSpeedReductionFeet is the most Slow reduces Speed (SRD 5.2.1, "Slow", p. 90).
const maxSpeedReductionFeet = 10

// SpeedReduction is a reduction of Speed in feet, never above 10. The zero value
// is no reduction. Comparable.
type SpeedReduction struct{ feet int }

// Feet returns the reduction in feet.
func (r SpeedReduction) Feet() int { return r.feet }

// Combine returns the larger of r and o, never above 10 feet: two Slow hits
// still reduce Speed by 10 feet.
func (r SpeedReduction) Combine(o SpeedReduction) SpeedReduction {
	return SpeedReduction{feet: min(max(r.feet, o.feet), maxSpeedReductionFeet)}
}

// ApplyTo returns speed reduced by r, at least 0. It returns an error matching
// srderr.ErrInvalidInput, naming speed, for a negative speed.
func (r SpeedReduction) ApplyTo(speed int) (int, error) {
	if speed < 0 {
		return 0, invalidInput("speed: negative")
	}
	return max(speed-r.feet, 0), nil
}

// PushMovement is the number of feet a target is moved by Push, 0 to 10.
type PushMovement int

// PushDistanceRejection says why a Push distance was refused. The zero value is
// not a rejection.
type PushDistanceRejection int

const (
	// PushDistanceNegative: the distance is below 0.
	PushDistanceNegative PushDistanceRejection = iota + 1
	// PushDistanceAboveMaximum: the distance is above the offer's maximum.
	PushDistanceAboveMaximum
)

// PushDistanceResult is the closed outcome of PushOffer.ChooseDistance:
// PushDistanceAccepted or PushDistanceRejected.
//
//sumtype:decl
type PushDistanceResult interface{ isPushDistanceResult() }

// PushDistanceAccepted carries the movement chosen.
type PushDistanceAccepted struct{ Movement PushMovement }

// PushDistanceRejected carries why the distance was refused.
type PushDistanceRejected struct{ Reason PushDistanceRejection }

func (PushDistanceAccepted) isPushDistanceResult() {}
func (PushDistanceRejected) isPushDistanceResult() {}

// CleaveTargetPlacement is where a second creature is for Cleave: its distance in
// feet from the first creature and from the attacker. Build it with
// NewCleaveTargetPlacement.
type CleaveTargetPlacement struct{ fromFirst, fromAttacker int }

// NewCleaveTargetPlacement returns the placement. It returns an error matching
// srderr.ErrInvalidInput, naming the distance, for a negative distance.
func NewCleaveTargetPlacement(distanceFromFirst, distanceFromAttacker int) (CleaveTargetPlacement, error) {
	switch {
	case distanceFromFirst < 0:
		return CleaveTargetPlacement{}, invalidInput("distanceFromFirst: negative")
	case distanceFromAttacker < 0:
		return CleaveTargetPlacement{}, invalidInput("distanceFromAttacker: negative")
	}
	return CleaveTargetPlacement{fromFirst: distanceFromFirst, fromAttacker: distanceFromAttacker}, nil
}

// MustCleaveTargetPlacement is NewCleaveTargetPlacement for values known to be
// valid. It panics with the error that function returns, and only then.
func MustCleaveTargetPlacement(distanceFromFirst, distanceFromAttacker int) CleaveTargetPlacement {
	p, err := NewCleaveTargetPlacement(distanceFromFirst, distanceFromAttacker)
	if err != nil {
		panic(err)
	}
	return p
}

// CleaveTargetFailure says which Cleave placement rule a second creature broke.
// The zero value is not a failure.
type CleaveTargetFailure int

const (
	// CleaveNotWithinFiveFeetOfFirst: more than 5 feet from the first creature.
	CleaveNotWithinFiveFeetOfFirst CleaveTargetFailure = iota + 1
	// CleaveNotWithinReach: beyond the attacker's reach.
	CleaveNotWithinReach
	// CleaveBothRulesBroken: both of the above.
	CleaveBothRulesBroken
)

// CleaveTargetResult is the closed outcome of CleaveOffer.ValidateTarget:
// CleaveTargetAccepted or CleaveTargetRejected.
//
//sumtype:decl
type CleaveTargetResult interface{ isCleaveTargetResult() }

// CleaveTargetAccepted: the second creature may be attacked.
type CleaveTargetAccepted struct{}

// CleaveTargetRejected carries which rule the placement broke.
type CleaveTargetRejected struct{ Failure CleaveTargetFailure }

func (CleaveTargetAccepted) isCleaveTargetResult() {}
func (CleaveTargetRejected) isCleaveTargetResult() {}

// MasteryOffer is a mastery effect the host may take or leave, as data: the
// closed set CleaveOffer, GrazeOffer, PushOffer, SapOffer, SlowOffer,
// ToppleOffer, VexOffer and MasteryNone. The engine never applies damage,
// movement, a condition or a Speed change.
//
//sumtype:decl
type MasteryOffer interface{ isMasteryOffer() }

// CleaveOffer: a melee weapon attack with the same weapon against a second
// creature within 5 feet of the first and within reach, once per turn, whose
// damage omits the ability modifier unless it is negative (SRD 5.2.1, "Cleave",
// p. 90).
type CleaveOffer struct {
	WeaponID       content.ItemID
	Kind           WeaponAttackKind
	Damage         content.WeaponDamage
	DamageModifier AbilityModifierToDamage
}

// ValidateTarget checks the placement of the second creature against the 5-foot
// rule and the attacker's reach (the reach with the weapon, see
// OpportunityAttackReach).
func (CleaveOffer) ValidateTarget(p CleaveTargetPlacement, reach ReachInFeet) CleaveTargetResult {
	farFromFirst := p.fromFirst > 5
	outOfReach := p.fromAttacker > reach.orDefault().feet
	switch {
	case farFromFirst && outOfReach:
		return CleaveTargetRejected{Failure: CleaveBothRulesBroken}
	case farFromFirst:
		return CleaveTargetRejected{Failure: CleaveNotWithinFiveFeetOfFirst}
	case outOfReach:
		return CleaveTargetRejected{Failure: CleaveNotWithinReach}
	}
	return CleaveTargetAccepted{}
}

// GrazeOffer: damage of Amount (the ability modifier, at least 0, nothing else)
// of DamageType to the creature missed (SRD 5.2.1, "Graze", p. 90).
type GrazeOffer struct {
	Amount     int
	DamageType content.DamageType
}

// PushOffer: move the target up to 10 feet straight away from the attacker (SRD
// 5.2.1, "Push", p. 90).
type PushOffer struct{ maximumFeet int }

// MaximumFeet returns the most the target may be moved: 10.
func (o PushOffer) MaximumFeet() int { return o.maximumFeet }

// ChooseDistance accepts 0 to the maximum feet and rejects any other distance as
// a result, never an error.
func (o PushOffer) ChooseDistance(feet int) PushDistanceResult {
	switch {
	case feet < 0:
		return PushDistanceRejected{Reason: PushDistanceNegative}
	case feet > o.maximumFeet:
		return PushDistanceRejected{Reason: PushDistanceAboveMaximum}
	}
	return PushDistanceAccepted{Movement: PushMovement(feet)}
}

// SapOffer: Disadvantage on the target's next attack roll (SRD 5.2.1, "Sap", p. 90).
type SapOffer struct{}

// Circumstance returns the Disadvantage circumstance.
func (SapOffer) Circumstance() modifier.RollCircumstance { return modifier.Disadvantage{} }

// Expiries returns when it ends, as a new slice.
func (SapOffer) Expiries() []MasteryEffectExpiry {
	return []MasteryEffectExpiry{ExpiryStartOfAttackersNextTurn, ExpiryWhenNextAttackRollMade}
}

// SlowOffer: reduce the target's Speed by 10 feet until the start of the
// attacker's next turn (SRD 5.2.1, "Slow", p. 90).
type SlowOffer struct{}

// Reduction returns the 10-foot reduction.
func (SlowOffer) Reduction() SpeedReduction { return SpeedReduction{feet: maxSpeedReductionFeet} }

// Expiries returns when it ends, as a new slice.
func (SlowOffer) Expiries() []MasteryEffectExpiry {
	return []MasteryEffectExpiry{ExpiryStartOfAttackersNextTurn}
}

// ToppleOffer: the target makes a Constitution saving throw against DC, and has
// the Prone condition on a failure (SRD 5.2.1, "Topple", p. 90). Resolve it
// with ToppleSaveResolver.ResolveToppleSave. The zero value is not an offer.
type ToppleOffer struct {
	ability ability.Ability
	dc      d20.DifficultyClass
}

// Ability returns the saving throw ability, Constitution.
func (o ToppleOffer) Ability() ability.Ability { return o.ability }

// DC returns 8 plus the ability modifier of the attack roll plus the Proficiency Bonus.
func (o ToppleOffer) DC() d20.DifficultyClass { return o.dc }

// VexOffer: Advantage on the attacker's next attack roll against the creature
// (SRD 5.2.1, "Vex", p. 90).
type VexOffer struct{}

// Circumstance returns the Advantage circumstance.
func (VexOffer) Circumstance() modifier.RollCircumstance { return modifier.Advantage{} }

// Expiries returns when it ends, as a new slice.
func (VexOffer) Expiries() []MasteryEffectExpiry {
	return []MasteryEffectExpiry{ExpiryEndOfAttackersNextTurn, ExpiryWhenNextAttackRollMade}
}

// MasteryNone: no effect is offered, for Reason.
type MasteryNone struct{ Reason MasteryNoneReason }

func (CleaveOffer) isMasteryOffer() {}
func (GrazeOffer) isMasteryOffer()  {}
func (PushOffer) isMasteryOffer()   {}
func (SapOffer) isMasteryOffer()    {}
func (SlowOffer) isMasteryOffer()   {}
func (ToppleOffer) isMasteryOffer() {}
func (VexOffer) isMasteryOffer()    {}
func (MasteryNone) isMasteryOffer() {}

// MasteryAttackFacts is the input of OfferMastery. Construct it with a keyed
// literal. A nil Access reads as NoMasteryFeature, a nil InPlay as
// WeaponsOwnMastery. DamageDealt is after Resistance, Vulnerability and Immunity.
type MasteryAttackFacts struct {
	WeaponID         content.ItemID
	Weapon           content.Weapon
	Access           WeaponMasteryAccess
	InPlay           MasteryInPlay
	Outcome          MasteryAttackOutcome
	Mode             WeaponAttackMode
	AbilityModifier  ability.Modifier
	ProficiencyBonus ProficiencyBonus
	TargetSize       content.CreatureSize
	DamageDealt      int
	TurnState        WeaponTurnState
}

func none(r MasteryNoneReason) (MasteryOffer, error) { return MasteryNone{Reason: r}, nil }

// OfferMastery reports the mastery effect of the attack as an offer (SRD 5.2.1,
// "Mastery Properties", p. 90). It is pure and applies nothing. When the
// character has no feature for the weapon the offer is MasteryNone(featureNotAvailable)
// whatever the attack did; otherwise the reasons are checked in the order of
// MasteryNoneReason. It returns an error matching srderr.ErrInvalidInput, naming
// damageDealt, only for a negative DamageDealt.
func OfferMastery(f MasteryAttackFacts) (MasteryOffer, error) {
	if f.DamageDealt < 0 {
		return nil, invalidInput("damageDealt: negative")
	}
	if f.Access == nil || !f.Access.Permits(f.WeaponID) {
		return none(MasteryNoneFeatureNotAvailable)
	}
	hit, rolledMiss := masteryHit(f.Outcome)
	switch masteryInPlay(f.Weapon, f.InPlay) {
	case content.MasteryCleave:
		_, melee := f.Mode.(MeleeAttackMode)
		switch {
		case !hit:
			return none(MasteryNoneAttackDidNotQualify)
		case !melee:
			return none(MasteryNoneNotMeleeAttack)
		case f.TurnState.cleave == markDone:
			return none(MasteryNoneAlreadyUsedThisTurn)
		}
		return CleaveOffer{WeaponID: f.WeaponID, Kind: MeleeWeaponAttackKind, Damage: f.Weapon.Damage(), DamageModifier: AddModifierOmitUnlessNegative}, nil
	case content.MasteryGraze:
		if !rolledMiss {
			return none(MasteryNoneAttackDidNotQualify)
		}
		return GrazeOffer{Amount: max(f.AbilityModifier.Value(), 0), DamageType: f.Weapon.Damage().DamageType()}, nil
	case content.MasteryNick:
		return none(MasteryNoneAttackDidNotQualify)
	case content.MasteryPush:
		switch {
		case !hit:
			return none(MasteryNoneAttackDidNotQualify)
		case f.TargetSize >= content.SizeHuge:
			return none(MasteryNoneTargetTooLarge)
		}
		return PushOffer{maximumFeet: 10}, nil
	case content.MasterySap:
		if !hit {
			return none(MasteryNoneAttackDidNotQualify)
		}
		return SapOffer{}, nil
	case content.MasterySlow:
		switch {
		case !hit:
			return none(MasteryNoneAttackDidNotQualify)
		case f.DamageDealt == 0:
			return none(MasteryNoneNoDamageDealt)
		}
		return SlowOffer{}, nil
	case content.MasteryTopple:
		if !hit {
			return none(MasteryNoneAttackDidNotQualify)
		}
		return ToppleOffer{ability: ability.Constitution, dc: d20.DifficultyClass(8 + f.AbilityModifier.Value() + f.ProficiencyBonus.Value())}, nil
	case content.MasteryVex:
		switch {
		case !hit:
			return none(MasteryNoneAttackDidNotQualify)
		case f.DamageDealt == 0:
			return none(MasteryNoneNoDamageDealt)
		}
		return VexOffer{}, nil
	}
	return none(MasteryNoneAttackDidNotQualify) // a weapon with no mastery property: the zero Weapon
}
