package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ThrownItemKind is a thrown item that replaces one attack (SRD 5.2.1, pp. 94-98). The values
// equal the first five ItemUseKind values. The zero value is invalid.
type ThrownItemKind int

// The five thrown items.
const (
	ThrownAcid ThrownItemKind = iota + 1
	ThrownAlchemistsFire
	ThrownHolyWater
	ThrownOil
	ThrownNet
)

// Valid reports whether k is one of the five.
func (k ThrownItemKind) Valid() bool { return k >= ThrownAcid && k <= ThrownNet }

// String returns "acid", "alchemists_fire", "holy_water", "oil", "net" or "unknown".
func (k ThrownItemKind) String() string { return enumName(itemUseNames()[:5], int(k)) }

// ItemUseKind returns the same item as an ItemUseKind; the zero kind for an invalid k.
func (k ThrownItemKind) ItemUseKind() ItemUseKind {
	if !k.Valid() {
		return 0
	}
	return ItemUseKind(k)
}

// ItemID returns the content id of the item; the zero ItemID for an invalid kind.
func (k ThrownItemKind) ItemID() content.ItemID { return k.ItemUseKind().ItemID() }

// ThrownItemFacts holds the facts of one thrown item (table in item-use-effects.md FR-003).
// Immutable and comparable. The zero value is the facts of no kind.
type ThrownItemFacts struct {
	creaturesOnly   bool
	rangeFeet       int
	sightRequired   bool
	damageCount     int
	damageDie       dice.Die
	damageType      content.DamageType
	fiendOrUndead   bool
	startsBurning   bool
	coatsInOil      bool
	restrainsTarget bool
}

// CreaturesOnly reports whether the item can be thrown only at a creature, not an object.
func (f ThrownItemFacts) CreaturesOnly() bool { return f.creaturesOnly }

// RangeFeet returns the farthest distance the item can be thrown.
func (f ThrownItemFacts) RangeFeet() int { return f.rangeFeet }

// SightRequired reports whether the thrower must see the target.
func (f ThrownItemFacts) SightRequired() bool { return f.sightRequired }

// SaveAbility returns Dexterity: every thrown item forces a Dexterity saving throw.
func (ThrownItemFacts) SaveAbility() ability.Ability { return ability.Dexterity }

// Damage returns the damage dice of a failed save and true, or the zero expression and false
// for an item that deals none (Oil, Net). The Damage Type is DamageType.
func (f ThrownItemFacts) Damage() (dice.Expression, bool) {
	if f.damageCount == 0 {
		return dice.Expression{}, false
	}
	return itemDice(f.damageCount, f.damageDie), true
}

// DamageType returns the Damage Type of the damage, or the zero type for an item that deals none.
func (f ThrownItemFacts) DamageType() content.DamageType { return f.damageType }

// OnlyFiendsAndUndead reports whether the damage applies only to a Fiend or an Undead.
func (f ThrownItemFacts) OnlyFiendsAndUndead() bool { return f.fiendOrUndead }

// StartsBurning reports whether a failed save sets the target burning.
func (f ThrownItemFacts) StartsBurning() bool { return f.startsBurning }

// CoatsInOil reports whether a failed save covers the target in oil.
func (f ThrownItemFacts) CoatsInOil() bool { return f.coatsInOil }

// RestrainsTarget reports whether a failed save restrains the target.
func (f ThrownItemFacts) RestrainsTarget() bool { return f.restrainsTarget }

// ThrownItemFactsOf returns the facts of kind (SRD 5.2.1, pp. 94, 97, 98): Acid (creature or
// object, 20 feet, sight, 2d6 Acid), Alchemist's Fire (20, sight, 1d4 Fire and burning), Holy
// Water (creature only, 20, sight, 2d8 Radiant to a Fiend or an Undead), Oil (20, no sight,
// covered in oil) and Net (creature only, 15, sight, Restrained). It returns the zero facts
// for an invalid kind.
func ThrownItemFactsOf(kind ThrownItemKind) ThrownItemFacts {
	switch kind {
	case ThrownAcid:
		return ThrownItemFacts{rangeFeet: 20, sightRequired: true, damageCount: 2, damageDie: dice.D6(), damageType: content.DamageAcid}
	case ThrownAlchemistsFire:
		return ThrownItemFacts{rangeFeet: 20, sightRequired: true, damageCount: 1, damageDie: dice.D4(), damageType: content.DamageFire, startsBurning: true}
	case ThrownHolyWater:
		return ThrownItemFacts{creaturesOnly: true, rangeFeet: 20, sightRequired: true, damageCount: 2, damageDie: dice.D8(),
			damageType: content.DamageRadiant, fiendOrUndead: true}
	case ThrownOil:
		return ThrownItemFacts{rangeFeet: 20, coatsInOil: true}
	case ThrownNet:
		return ThrownItemFacts{creaturesOnly: true, rangeFeet: NetThrowRangeFeet, sightRequired: true, restrainsTarget: true}
	}
	return ThrownItemFacts{}
}

// ThrownTarget is what an item is thrown at. Variants: ThrownAtCreature, ThrownAtObject.
type ThrownTarget interface{ isThrownTarget() }

// ThrownAtCreature: a creature, with the facts the rules read. Construct with a keyed
// literal; the zero value is invalid.
type ThrownAtCreature struct {
	ID   CombatantID
	Size content.CreatureSize
	Type content.CreatureType
}

// ThrownAtObject: an object.
type ThrownAtObject struct{}

func (ThrownAtCreature) isThrownTarget() {}
func (ThrownAtObject) isThrownTarget()   {}

// valid reports whether t is an object or a creature with an id, a size and a type.
func validThrownTarget(t ThrownTarget) bool {
	switch v := t.(type) {
	case ThrownAtObject:
		return true
	case ThrownAtCreature:
		return !v.ID.IsZero() && v.Size.Valid() && v.Type.Valid()
	}
	return false
}

// ThrownSight says whether the thrower can see the target. The zero value is invalid.
type ThrownSight int

// The two states.
const (
	ThrownSightSeen ThrownSight = iota + 1
	ThrownSightUnseen
)

// Valid reports whether s is one of the two.
func (s ThrownSight) Valid() bool { return s == ThrownSightSeen || s == ThrownSightUnseen }

// String returns "seen", "unseen" or "unknown".
func (s ThrownSight) String() string { return enumName([]string{"seen", "unseen"}, int(s)) }

// ThrownCover says whether Cover lets the target be targeted. The zero value is invalid.
type ThrownCover int

// The two states.
const (
	ThrownCoverTargetable ThrownCover = iota + 1
	ThrownCoverTotal
)

// Valid reports whether c is one of the two.
func (c ThrownCover) Valid() bool { return c == ThrownCoverTargetable || c == ThrownCoverTotal }

// String returns "targetable", "total" or "unknown".
func (c ThrownCover) String() string { return enumName([]string{"targetable", "total"}, int(c)) }

// ThrownNotPossibleReason is why a thrown item cannot be thrown. The zero value is no reason.
// String returns the snake_case reason, the value of srd5e.combat.result.
type ThrownNotPossibleReason int

// The reasons, in the order they are checked (FR-006).
const (
	ThrownReasonTargetNotACreature ThrownNotPossibleReason = iota + 1
	ThrownReasonTotalCover
	ThrownReasonBeyondRange
	ThrownReasonTargetNotSeen
)

// String returns "target_not_a_creature", "total_cover", "beyond_range", "target_not_seen",
// "none" for the zero value or "unknown".
func (r ThrownNotPossibleReason) String() string {
	if r == 0 {
		return "none"
	}
	return enumName([]string{"target_not_a_creature", "total_cover", "beyond_range", "target_not_seen"}, int(r))
}

// ThrownNoEffectReason is why a thrown item had no effect. The zero value is no reason.
type ThrownNoEffectReason int

// The reasons.
const (
	// ThrownNoEffectSaveSucceeded: the target made its saving throw.
	ThrownNoEffectSaveSucceeded ThrownNoEffectReason = iota + 1
	// ThrownNoEffectNotFiendOrUndead: Holy Water thrown at a creature that is neither.
	ThrownNoEffectNotFiendOrUndead
	// ThrownNoEffectTargetTooLarge: a Net thrown at a Huge or Gargantuan creature.
	ThrownNoEffectTargetTooLarge
)

// String returns "save_succeeded", "not_fiend_or_undead", "target_too_large", "none" or "unknown".
func (r ThrownNoEffectReason) String() string {
	if r == 0 {
		return "none"
	}
	return enumName([]string{"save_succeeded", "not_fiend_or_undead", "target_too_large"}, int(r))
}

// ThrownEffect is what a thrown item did to its target, as data the host applies. Variants:
// ThrownDamageDealt, ThrownDamageAndBurningStarted, ThrownOilCovered, ThrownRestrained,
// ThrownNoEffect.
type ThrownEffect interface{ isThrownEffect() }

// ThrownDamageDealt: the target takes the damage, unadjusted (pass it to ApplyDamage or
// ApplyDamageToObject).
type ThrownDamageDealt struct{ damage Damage }

// Damage returns the damage.
func (e ThrownDamageDealt) Damage() Damage { return e.damage }

// ThrownDamageAndBurningStarted: the target takes the damage and is set burning (BurningStateBurning).
type ThrownDamageAndBurningStarted struct{ damage Damage }

// Damage returns the damage.
func (e ThrownDamageAndBurningStarted) Damage() Damage { return e.damage }

// ThrownOilCovered: the target is covered in oil with a fresh coat, replacing any it carried.
type ThrownOilCovered struct{ coat OilCoat }

// Coat returns the fresh coat.
func (e ThrownOilCovered) Coat() OilCoat { return e.coat }

// ThrownRestrained: the target is Restrained by the Net until it escapes.
type ThrownRestrained struct{ restraint NetRestraint }

// Restraint returns the Net's restraint.
func (e ThrownRestrained) Restraint() NetRestraint { return e.restraint }

// ThrownNoEffect: nothing happens to the target.
type ThrownNoEffect struct{ reason ThrownNoEffectReason }

// Reason returns why.
func (e ThrownNoEffect) Reason() ThrownNoEffectReason { return e.reason }

func (ThrownDamageDealt) isThrownEffect()             {}
func (ThrownDamageAndBurningStarted) isThrownEffect() {}
func (ThrownOilCovered) isThrownEffect()              {}
func (ThrownRestrained) isThrownEffect()              {}
func (ThrownNoEffect) isThrownEffect()                {}

// ThrownSave is the target's saving throw against a thrown item. Variants: ThrownSaveRolled,
// ThrownSaveForced, ThrownSaveSkipped.
type ThrownSave interface {
	isThrownSave()
	// Succeeded reports whether the target passed the saving throw; false for a skipped save.
	Succeeded() bool
}

// ThrownSaveRolled: the save was rolled against the Difficulty Class.
type ThrownSaveRolled struct {
	dc     d20.DifficultyClass
	result d20.CheckResult
}

// DifficultyClass returns the Difficulty Class.
func (s ThrownSaveRolled) DifficultyClass() d20.DifficultyClass { return s.dc }

// Result returns the resolved saving throw.
func (s ThrownSaveRolled) Result() d20.CheckResult { return s.result }

// Succeeded reports whether the total met the Difficulty Class.
func (s ThrownSaveRolled) Succeeded() bool { return s.result.Outcome() == d20.Success }

// ThrownSaveForced: the target failed without a die, by choice or automatically.
type ThrownSaveForced struct {
	dc     d20.DifficultyClass
	result d20.ForcedFailureResult
}

// DifficultyClass returns the Difficulty Class.
func (s ThrownSaveForced) DifficultyClass() d20.DifficultyClass { return s.dc }

// Result returns the forced failure.
func (s ThrownSaveForced) Result() d20.ForcedFailureResult { return s.result }

// Succeeded returns false.
func (ThrownSaveForced) Succeeded() bool { return false }

// ThrownSaveSkipped: the target had nothing to save against.
type ThrownSaveSkipped struct{ reason ThrownNoEffectReason }

// Reason returns why the save was skipped.
func (s ThrownSaveSkipped) Reason() ThrownNoEffectReason { return s.reason }

// Succeeded returns false.
func (ThrownSaveSkipped) Succeeded() bool { return false }

func (ThrownSaveRolled) isThrownSave()  {}
func (ThrownSaveForced) isThrownSave()  {}
func (ThrownSaveSkipped) isThrownSave() {}

// ThrownItemRules is the stateless rules of throwing an item (item-use-effects.md FR-004,
// FR-006). The zero value is ready to use. Pure; safe for concurrent use.
type ThrownItemRules struct{}

// Replacement returns the attack replacement of throwing kind: AttackReplacementKindReplace
// by ReplacementByItem with the kind's ItemID. This package never records it: the host passes
// it to AttackActionProgress.RecordReplacement. It returns the zero AttackReplacement for an
// invalid kind.
func (ThrownItemRules) Replacement(kind ThrownItemKind) AttackReplacement {
	r, err := NewAttackReplacement(AttackReplacementKindReplace, ReplacementByItem(kind.ItemID()))
	if err != nil {
		return AttackReplacement{}
	}
	return r
}

// Possibility says why kind cannot be thrown at target distanceFeet away, checked in this
// order: an object for Holy Water or a Net (ThrownReasonTargetNotACreature), total Cover
// (ThrownReasonTotalCover), a distance above the kind's range (ThrownReasonBeyondRange; a
// distance equal to the range is valid) and a target the thrower cannot see for a kind that
// needs sight (ThrownReasonTargetNotSeen). It returns the zero reason when the throw is
// possible.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) for an invalid kind, target,
// sight or cover or a negative distance. It never panics.
func (r ThrownItemRules) Possibility(kind ThrownItemKind, target ThrownTarget, distanceFeet int,
	sight ThrownSight, cover ThrownCover) (ThrownNotPossibleReason, error) {
	switch {
	case !kind.Valid():
		return 0, itemUseInvalid(ItemUseFieldKind, int64(kind))
	case !validThrownTarget(target):
		return 0, itemUseInvalid(ItemUseFieldTarget, 0)
	case distanceFeet < 0:
		return 0, itemUseInvalid(ItemUseFieldDistanceFeet, int64(distanceFeet))
	case !sight.Valid():
		return 0, itemUseInvalid(ItemUseFieldSight, int64(sight))
	case !cover.Valid():
		return 0, itemUseInvalid(ItemUseFieldCover, int64(cover))
	}
	return r.possibility(kind, target, distanceFeet, sight, cover), nil
}

// possibility is Possibility for values known to be valid.
func (ThrownItemRules) possibility(kind ThrownItemKind, target ThrownTarget, distanceFeet int,
	sight ThrownSight, cover ThrownCover) ThrownNotPossibleReason {
	facts := ThrownItemFactsOf(kind)
	_, isObject := target.(ThrownAtObject)
	switch {
	case isObject && facts.creaturesOnly:
		return ThrownReasonTargetNotACreature
	case cover == ThrownCoverTotal:
		return ThrownReasonTotalCover
	case distanceFeet > facts.rangeFeet:
		return ThrownReasonBeyondRange
	case sight == ThrownSightUnseen && facts.sightRequired:
		return ThrownReasonTargetNotSeen
	}
	return 0
}

// SaveSkip says whether the target has nothing to save against: Holy Water thrown at a
// creature that is neither a Fiend nor an Undead (ThrownNoEffectNotFiendOrUndead) and a Net
// thrown at a Huge or Gargantuan creature (ThrownNoEffectTargetTooLarge). It returns the zero
// reason when the target saves, for an object and for an invalid kind.
func (ThrownItemRules) SaveSkip(kind ThrownItemKind, target ThrownTarget) ThrownNoEffectReason {
	c, ok := target.(ThrownAtCreature)
	switch {
	case !ok:
	case kind == ThrownHolyWater && c.Type != content.CreatureTypeFiend && c.Type != content.CreatureTypeUndead:
		return ThrownNoEffectNotFiendOrUndead
	case kind == ThrownNet && (NetRules{}).AutoSucceeds(c.Size):
		return ThrownNoEffectTargetTooLarge
	}
	return 0
}

// Consumption returns ItemConsumed (one) for Acid, Alchemist's Fire, Holy Water and Oil,
// whatever the save, and ItemNotConsumed for a Net and for an invalid kind (Q-7).
func (ThrownItemRules) Consumption(kind ThrownItemKind) ItemConsumption {
	if !kind.Valid() || kind == ThrownNet {
		return ItemNotConsumed{}
	}
	return consumedOne(kind.ItemUseKind())
}

// OnFailedSave returns what a failed saving throw does: for Acid, Alchemist's Fire and Holy
// Water the damage of damageTotal (the total of the damage expression, 0 or more) in the
// kind's Damage Type, with burning for Alchemist's Fire; a fresh OilCoat for Oil; the Net's
// restraint for a Net. It returns an *ItemUseError (matching srderr.ErrInvalidInput) for an
// invalid kind or a negative damageTotal. It never panics.
func (ThrownItemRules) OnFailedSave(kind ThrownItemKind, damageTotal int) (ThrownEffect, error) {
	f := ThrownItemFactsOf(kind)
	switch {
	case !kind.Valid():
		return nil, itemUseInvalid(ItemUseFieldKind, int64(kind))
	case f.restrainsTarget:
		return ThrownRestrained{NetRestraint{}}, nil
	case f.coatsInOil:
		return ThrownOilCovered{NewOilCoat()}, nil
	}
	dmg, err := NewDamage(damageTotal, f.damageType)
	if err != nil {
		return nil, itemUseInvalid(ItemUseFieldDamage, int64(damageTotal))
	}
	if f.startsBurning {
		return ThrownDamageAndBurningStarted{dmg}, nil
	}
	return ThrownDamageDealt{dmg}, nil
}

// NetRestraint is the restraint a Net puts on a creature (SRD 5.2.1, "Net", p. 98): Restrained
// until it escapes. It carries only the Net's constants, so the zero value is ready to use.
type NetRestraint struct{}

// EscapeCheck returns the check to escape: Strength (Athletics) against Difficulty Class 10,
// taken as an action by the target or a creature within 5 feet.
func (NetRestraint) EscapeCheck() HoldCheck { return HoldProfileFor(HoldKindNet).EscapeCheck() }

// Statistics returns the Net as an object: Armor Class 10, 5 Hit Points and Immunity to
// Bludgeoning, Poison and Psychic damage.
func (NetRestraint) Statistics() ObjectStatus { return NetProfile{}.Status() }

// Escape returns NetEscaped for a successful escape check and NetHoldsFast for a failed one.
func (NetRestraint) Escape(outcome d20.Outcome) NetOutcome {
	if outcome == d20.Success {
		return NetEscaped{}
	}
	return NetHoldsFast{}
}

// Destroyed returns NetEscaped: a destroyed Net frees the creature.
func (NetRestraint) Destroyed() NetOutcome { return NetEscaped{} }

// NetOutcome is the result of an attempt to get out of a Net. Variants: NetEscaped, NetHoldsFast.
type NetOutcome interface{ isNetOutcome() }

// NetEscaped: the creature is free of the Net.
type NetEscaped struct{}

// NetHoldsFast: the Net still restrains the creature.
type NetHoldsFast struct{}

func (NetEscaped) isNetOutcome()   {}
func (NetHoldsFast) isNetOutcome() {}
