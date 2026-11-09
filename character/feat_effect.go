package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// FeatTriggerAction is an action a feat effect names (SRD 5.2.1, "Boon of
// Dimensional Travel", "Boon of the Night Spirit", p. 88). The zero value is
// invalid. Comparable and safe for concurrent use.
type FeatTriggerAction int

// The actions.
const (
	TriggerAttackAction FeatTriggerAction = iota + 1
	TriggerMagicAction
	TriggerBonusAction
	TriggerReaction
	TriggerAction
)

// String returns "attackAction", "magicAction", "bonusAction", "reaction",
// "action" or "unknown".
func (a FeatTriggerAction) String() string {
	return featNameOf(int(a), "attackAction", "magicAction", "bonusAction", "reaction", "action")
}

// HandsUsed is how many hands hold a weapon. The zero value is invalid.
// Comparable and safe for concurrent use.
type HandsUsed int

// The hands.
const (
	// TwoHands: held with two hands.
	TwoHands HandsUsed = iota + 1
)

// String returns "twoHands" or "unknown".
func (h HandsUsed) String() string { return featNameOf(int(h), "twoHands") }

// FeatInitiativeTiming is when the Alert Initiative Swap happens. The zero value
// is invalid. Comparable and safe for concurrent use.
type FeatInitiativeTiming int

// The timings.
const (
	// AfterRollingInitiative: right after Initiative is rolled.
	AfterRollingInitiative FeatInitiativeTiming = iota + 1
)

// String returns "afterRollingInitiative" or "unknown".
func (t FeatInitiativeTiming) String() string { return featNameOf(int(t), "afterRollingInitiative") }

// FeatSwapPartner is who Alert swaps Initiative with. The zero value is invalid.
// Comparable and safe for concurrent use.
type FeatSwapPartner int

// The partners.
const (
	// WillingAllyInSameCombat: a willing ally in the same combat.
	WillingAllyInSameCombat FeatSwapPartner = iota + 1
)

// String returns "willingAllyInSameCombat" or "unknown".
func (p FeatSwapPartner) String() string { return featNameOf(int(p), "willingAllyInSameCombat") }

// FeatTrigger is what sets off a feat effect. The zero value is invalid.
// Comparable and safe for concurrent use.
type FeatTrigger int

// The triggers.
const (
	// TriggerWeaponHit: a hit with a weapon.
	TriggerWeaponHit FeatTrigger = iota + 1
	// TriggerNatural20OnAttackRoll: a 20 on the d20 of an attack roll.
	TriggerNatural20OnAttackRoll
)

// String returns "weaponHit", "natural20OnAttackRoll" or "unknown".
func (t FeatTrigger) String() string { return featNameOf(int(t), "weaponHit", "natural20OnAttackRoll") }

// FeatSizeRelation is the size of a grappled target relative to you. The zero
// value is invalid. Comparable and safe for concurrent use.
type FeatSizeRelation int

// The relations.
const (
	// SameSizeOrSmaller: the same size as you or smaller.
	SameSizeOrSmaller FeatSizeRelation = iota + 1
)

// String returns "sameSizeOrSmaller" or "unknown".
func (r FeatSizeRelation) String() string { return featNameOf(int(r), "sameSizeOrSmaller") }

// FeatDestination is where Blink Steps ends. The zero value is invalid.
// Comparable and safe for concurrent use.
type FeatDestination int

// The destinations.
const (
	// UnoccupiedSpaceYouCanSee: an unoccupied space you can see.
	UnoccupiedSpaceYouCanSee FeatDestination = iota + 1
)

// String returns "unoccupiedSpaceYouCanSee" or "unknown".
func (d FeatDestination) String() string { return featNameOf(int(d), "unoccupiedSpaceYouCanSee") }

// FeatDamageTypeRule says which damage type extra damage has, a closed set.
// Variants: SameAsAttackDamageType, AttackDamageTypeIs.
type FeatDamageTypeRule interface{ isFeatDamageTypeRule() }

// SameAsAttackDamageType: the damage type of the attack.
type SameAsAttackDamageType struct{}

// AttackDamageTypeIs: the given damage type. Construct it with a keyed literal.
type AttackDamageTypeIs struct{ Type content.DamageType }

func (SameAsAttackDamageType) isFeatDamageTypeRule() {}
func (AttackDamageTypeIs) isFeatDamageTypeRule()     {}

// FeatUsageLimit limits how often an effect applies, a closed set. Variants:
// NoUsageLimit, OncePerTurn, OnceUntilStartOfYourNextTurn, UseRenewedBy. This
// context holds no per-turn state; the combat rules or the host record the uses.
type FeatUsageLimit interface{ isFeatUsageLimit() }

// NoUsageLimit: the effect has no limit.
type NoUsageLimit struct{}

// OncePerTurn: once on each of your turns.
type OncePerTurn struct{}

// OnceUntilStartOfYourNextTurn: once until the start of your next turn.
type OnceUntilStartOfYourNextTurn struct{}

func (NoUsageLimit) isFeatUsageLimit()                 {}
func (OncePerTurn) isFeatUsageLimit()                  {}
func (OnceUntilStartOfYourNextTurn) isFeatUsageLimit() {}

// FeatEffect is a benefit a held feat declares for the combat rules or the host to
// interpret; this context never applies one. It is a closed set of immutable
// value objects of structured facts, comparable with ==, built with keyed
// literals (the array fields are copied by value). Variants: InitiativeProficiency,
// InitiativeSwap, RollDamageTwice, UnarmedStrikeDamageAndGrapple,
// AdvantageAgainstGrappledByYou, GrappleMovementDiscount, AttackRollBonus,
// ArmoredArmorClassBonus, DamageDieMinimum, ExtraAttackAbilityModifierToDamage,
// MissBecomesHit, BlinkSteps, ImproveFate, IgnoreResistance, OverwhelmingStrike,
// FreeCasting, MergeWithShadows, ShadowyForm, TruesightSense.
type FeatEffect interface{ isFeatEffect() }

// InitiativeProficiency adds the Proficiency Bonus to Initiative (Alert).
type InitiativeProficiency struct{}

// InitiativeSwap swaps Initiative with a partner (Alert).
type InitiativeSwap struct {
	Timing             FeatInitiativeTiming
	Partner            FeatSwapPartner
	BlockedByCondition content.ConditionKind
}

// RollDamageTwice rolls a weapon's damage dice twice and uses either (Savage
// Attacker).
type RollDamageTwice struct {
	Trigger FeatTrigger
	Limit   FeatUsageLimit
}

// UnarmedStrikeDamageAndGrapple is Punch and Grab (Grappler).
type UnarmedStrikeDamageAndGrapple struct{ Limit FeatUsageLimit }

// AdvantageAgainstGrappledByYou gives Advantage on attacks against a creature you
// grapple (Grappler).
type AdvantageAgainstGrappledByYou struct{}

// GrappleMovementDiscount is Fast Wrestler (Grappler).
type GrappleMovementDiscount struct{ TargetSize FeatSizeRelation }

// AttackRollBonus is a bonus to attack rolls with a type of weapon (Archery).
type AttackRollBonus struct {
	Bonus      modifier.Modifier
	WeaponType content.WeaponType
}

// ArmoredArmorClassBonus is a bonus to Armor Class while wearing armor (Defense).
type ArmoredArmorClassBonus struct {
	Bonus        FlatBonus
	WhileWearing [3]content.ArmorCategory
}

// DamageDieMinimum raises a low damage die roll (Great Weapon Fighting).
type DamageDieMinimum struct {
	Minimum               int
	AppliesToRolledAtMost int
	WeaponType            content.WeaponType
	Hands                 HandsUsed
	AnyOfProperties       [2]content.WeaponProperty
}

// ExtraAttackAbilityModifierToDamage adds the ability modifier to the damage of
// the extra attack (Two-Weapon Fighting).
type ExtraAttackAbilityModifierToDamage struct{ RequiredProperty content.WeaponProperty }

// MissBecomesHit turns a missed attack into a hit (Boon of Combat Prowess).
type MissBecomesHit struct{ Limit FeatUsageLimit }

// BlinkSteps is Blink Steps (Boon of Dimensional Travel).
type BlinkSteps struct {
	DistanceFeet int
	After        [2]FeatTriggerAction
	Destination  FeatDestination
}

// ImproveFate is Improve Fate (Boon of Fate).
type ImproveFate struct {
	RangeFeet int
	Dice      dice.Expression
	Use       UseRenewedBy
}

// IgnoreResistance ignores the Resistance to damage types (Boon of Irresistible
// Offense).
type IgnoreResistance struct{ DamageTypes [3]content.DamageType }

// OverwhelmingStrike is the extra damage of a 20 on an attack roll (Boon of
// Irresistible Offense); ExtraDamage gives the amount.
type OverwhelmingStrike struct {
	Ability    ability.Ability
	Trigger    FeatTrigger
	DamageType FeatDamageTypeRule
}

// FreeCasting is the Free Casting of Boon of Spell Recall; FreeCastingRule decides it.
type FreeCasting struct {
	SlotLevelsFrom SlotLevel
	SlotLevelsTo   SlotLevel
	Die            dice.Expression
}

// MergeWithShadows is Merge with Shadows (Boon of the Night Spirit).
type MergeWithShadows struct {
	Requires  [2]content.LightLevel
	Grants    content.ConditionKind
	Cost      FeatTriggerAction
	EndsAfter [3]FeatTriggerAction
}

// ShadowyForm is Shadowy Form (Boon of the Night Spirit).
type ShadowyForm struct {
	Requires              [2]content.LightLevel
	ResistanceToAllExcept [2]content.DamageType
}

// TruesightSense is Truesight (Boon of Truesight).
type TruesightSense struct{ RangeFeet int }

func (InitiativeProficiency) isFeatEffect()              {}
func (InitiativeSwap) isFeatEffect()                     {}
func (RollDamageTwice) isFeatEffect()                    {}
func (UnarmedStrikeDamageAndGrapple) isFeatEffect()      {}
func (AdvantageAgainstGrappledByYou) isFeatEffect()      {}
func (GrappleMovementDiscount) isFeatEffect()            {}
func (AttackRollBonus) isFeatEffect()                    {}
func (ArmoredArmorClassBonus) isFeatEffect()             {}
func (DamageDieMinimum) isFeatEffect()                   {}
func (ExtraAttackAbilityModifierToDamage) isFeatEffect() {}
func (MissBecomesHit) isFeatEffect()                     {}
func (BlinkSteps) isFeatEffect()                         {}
func (ImproveFate) isFeatEffect()                        {}
func (IgnoreResistance) isFeatEffect()                   {}
func (OverwhelmingStrike) isFeatEffect()                 {}
func (FreeCasting) isFeatEffect()                        {}
func (MergeWithShadows) isFeatEffect()                   {}
func (ShadowyForm) isFeatEffect()                        {}
func (TruesightSense) isFeatEffect()                     {}

// ExtraDamage returns the extra damage of the Overwhelming Strike: the current
// score of the ability the feat increased.
func (o OverwhelmingStrike) ExtraDamage(scores ability.Scores) int {
	return scores.Score(o.Ability).Value()
}

// FeatEffects reads the declared effects of held feats (SRD 5.2.1, "Parts of a
// Feat", Benefit, p. 87). The zero value is ready to use. It is stateless and
// safe for concurrent use.
type FeatEffects struct{}

// Effects returns a new slice of the declared effects of the held feats, in the
// order held (table E); a character with no feats has none. Magic Initiate and
// Skilled declare no effect: their spells and proficiencies are
// TakenFeats.GrantedSpells and TakenFeats.ProficiencyGrants, their free
// cast is UseStateOf. Prerequisites are not re-checked, so a feat whose
// prerequisite later fails keeps its effects.
func (FeatEffects) Effects(feats TakenFeats) []FeatEffect {
	var out []FeatEffect
	for _, t := range feats.taken {
		out = append(out, effectsOf(t)...)
	}
	return out
}

func effectsOf(t TakenFeat) []FeatEffect {
	switch t.feat {
	case featAlert:
		return []FeatEffect{InitiativeProficiency{}, InitiativeSwap{Timing: AfterRollingInitiative, Partner: WillingAllyInSameCombat, BlockedByCondition: content.ConditionIncapacitated}}
	case featSavageAttacker:
		return []FeatEffect{RollDamageTwice{Trigger: TriggerWeaponHit, Limit: OncePerTurn{}}}
	case featGrappler:
		return []FeatEffect{UnarmedStrikeDamageAndGrapple{Limit: OncePerTurn{}}, AdvantageAgainstGrappledByYou{}, GrappleMovementDiscount{TargetSize: SameSizeOrSmaller}}
	case featArchery:
		return []FeatEffect{AttackRollBonus{Bonus: modifier.New(2), WeaponType: content.WeaponRanged}}
	case featDefense:
		return []FeatEffect{ArmoredArmorClassBonus{Bonus: defenseBonus(), WhileWearing: [3]content.ArmorCategory{content.ArmorLight, content.ArmorMedium, content.ArmorHeavy}}}
	case featGreatWeaponFighting:
		return []FeatEffect{DamageDieMinimum{Minimum: 3, AppliesToRolledAtMost: 2, WeaponType: content.WeaponMelee, Hands: TwoHands,
			AnyOfProperties: [2]content.WeaponProperty{content.WeaponPropertyTwoHanded, content.WeaponPropertyVersatile}}}
	case featTwoWeaponFighting:
		return []FeatEffect{ExtraAttackAbilityModifierToDamage{RequiredProperty: content.WeaponPropertyLight}}
	case featBoonOfCombatProwess:
		return []FeatEffect{MissBecomesHit{Limit: OnceUntilStartOfYourNextTurn{}}}
	case featBoonOfDimensionalTravel:
		return []FeatEffect{BlinkSteps{DistanceFeet: 30, After: [2]FeatTriggerAction{TriggerAttackAction, TriggerMagicAction}, Destination: UnoccupiedSpaceYouCanSee}}
	case featBoonOfFate:
		shape, _ := shapeOf(featBoonOfFate)
		return []FeatEffect{ImproveFate{RangeFeet: 60, Dice: dice.MustExpression(2, dice.D4(), 0), Use: shape.renewedBy}}
	case featBoonOfIrresistibleOffense:
		return []FeatEffect{
			IgnoreResistance{DamageTypes: [3]content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing}},
			OverwhelmingStrike{Ability: t.choices.(SingleAbilityIncrease).ability, Trigger: TriggerNatural20OnAttackRoll, DamageType: SameAsAttackDamageType{}},
		}
	case featBoonOfSpellRecall:
		return []FeatEffect{FreeCasting{SlotLevelsFrom: MustSlotLevel(1), SlotLevelsTo: MustSlotLevel(4), Die: dice.MustExpression(1, dice.D4(), 0)}}
	case featBoonOfNightSpirit:
		dim := [2]content.LightLevel{content.LightDim, content.LightDarkness}
		return []FeatEffect{
			MergeWithShadows{Requires: dim, Grants: content.ConditionInvisible, Cost: TriggerBonusAction, EndsAfter: [3]FeatTriggerAction{TriggerAction, TriggerBonusAction, TriggerReaction}},
			ShadowyForm{Requires: dim, ResistanceToAllExcept: [2]content.DamageType{content.DamagePsychic, content.DamageRadiant}},
		}
	case featBoonOfTruesight:
		return []FeatEffect{TruesightSense{RangeFeet: 60}}
	}
	return nil
}

// defenseBonus is the +1 Armor Class of Defense, whose source is the feature
// named for the feat.
func defenseBonus() FlatBonus {
	return FlatBonus{modifier: modifier.New(1), source: FeatureFlatBonusSource{Feature: content.FeatureID(featDefense)}}
}

// InitiativeModifier returns the Initiative bonus the held feats give: the
// Proficiency Bonus with Alert (SRD 5.2.1, "Alert", p. 87), and 0 without it.
func (FeatEffects) InitiativeModifier(feats TakenFeats, pb ProficiencyBonus) modifier.Modifier {
	if feats.Holds(featAlert) {
		return modifier.New(pb.Value())
	}
	return modifier.New(0)
}

// ArmorClassBonuses returns a new slice of the Armor Class bonuses the held feats
// give for the armor categories worn (SRD 5.2.1, "Defense", p. 88): one +1 for
// each held Defense when worn includes Light, Medium or Heavy armor, and none for
// a Shield alone or nothing worn.
func (FeatEffects) ArmorClassBonuses(feats TakenFeats, worn []content.ArmorCategory) []FlatBonus {
	armored := slices.ContainsFunc(worn, func(c content.ArmorCategory) bool {
		return slices.Contains([]content.ArmorCategory{content.ArmorLight, content.ArmorMedium, content.ArmorHeavy}, c)
	})
	var out []FlatBonus
	for _, t := range feats.taken {
		if t.feat == featDefense && armored {
			out = append(out, defenseBonus())
		}
	}
	return out
}
