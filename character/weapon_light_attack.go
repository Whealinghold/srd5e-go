package character

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// WeaponInstanceKey tells two weapons of the same kind apart (two Daggers): a
// comparable value over a non-blank string. Build it with NewWeaponInstanceKey.
type WeaponInstanceKey struct{ v string }

// NewWeaponInstanceKey returns the key s. It returns an error matching
// srderr.ErrInvalidInput for an empty or whitespace-only s.
func NewWeaponInstanceKey(s string) (WeaponInstanceKey, error) {
	if strings.TrimSpace(s) == "" {
		return WeaponInstanceKey{}, invalidInput("weapon instance key: blank")
	}
	return WeaponInstanceKey{v: s}, nil
}

// MustWeaponInstanceKey is NewWeaponInstanceKey for values known to be valid. It
// panics with the error that function returns, and only then.
func MustWeaponInstanceKey(s string) WeaponInstanceKey {
	k, err := NewWeaponInstanceKey(s)
	if err != nil {
		panic(err)
	}
	return k
}

// String returns the key's text.
func (k WeaponInstanceKey) String() string { return k.v }

// turnMark records whether something happened this turn; the zero value is "not
// yet". It is an enum, not a bool.
type turnMark int

const (
	markNotYet turnMark = iota
	markDone
)

type turnAttack struct {
	key      WeaponInstanceKey
	hasLight bool
}

// WeaponTurnState is what one combatant has done this turn that the Light and
// Cleave rules read: whether the Attack action was taken, each weapon attacked
// with, and whether the Light and Cleave extra attacks were made. The zero value
// is the initial state of a turn and records nothing; the host starts each turn
// from it, the engine never resets it. Immutable: each Record method returns a
// new value. It holds a slice, so compare with Equal, not ==.
type WeaponTurnState struct {
	attackAction turnMark
	attacks      []turnAttack
	lightExtra   turnMark
	cleave       turnMark
}

// RecordAttackAction returns the state with the Attack action taken. The host
// calls it when its Attack action starts.
func (s WeaponTurnState) RecordAttackAction() WeaponTurnState {
	s.attacks = slices.Clone(s.attacks)
	s.attackAction = markDone
	return s
}

// RecordAttack returns the state with an attack made with the weapon key; hasLight
// says whether that weapon has the Light property.
func (s WeaponTurnState) RecordAttack(key WeaponInstanceKey, hasLight bool) WeaponTurnState {
	s.attacks = append(slices.Clone(s.attacks), turnAttack{key: key, hasLight: hasLight})
	return s
}

// RecordLightExtraAttack returns the state with the Light extra attack made.
func (s WeaponTurnState) RecordLightExtraAttack() WeaponTurnState {
	s.attacks = slices.Clone(s.attacks)
	s.lightExtra = markDone
	return s
}

// RecordCleaveAttack returns the state with the Cleave extra attack made.
func (s WeaponTurnState) RecordCleaveAttack() WeaponTurnState {
	s.attacks = slices.Clone(s.attacks)
	s.cleave = markDone
	return s
}

// Equal reports whether s and o record the same.
func (s WeaponTurnState) Equal(o WeaponTurnState) bool {
	return s.attackAction == o.attackAction && s.lightExtra == o.lightExtra && s.cleave == o.cleave &&
		slices.Equal(s.attacks, o.attacks)
}

// ExtraAttackCost is what a Light extra attack costs. The zero value is not a cost.
type ExtraAttackCost int

const (
	// ExtraAttackSpendsBonusAction: the extra attack is made as a Bonus Action.
	ExtraAttackSpendsBonusAction ExtraAttackCost = iota + 1
	// ExtraAttackPartOfAttackAction: the extra attack is part of the Attack action (Nick).
	ExtraAttackPartOfAttackAction
)

// String returns "spendsBonusAction", "partOfAttackAction" or "unknown".
func (c ExtraAttackCost) String() string {
	switch c {
	case ExtraAttackSpendsBonusAction:
		return "spendsBonusAction"
	case ExtraAttackPartOfAttackAction:
		return "partOfAttackAction"
	}
	return "unknown"
}

// ExtraAttackDamageSource is the host's report of an effect that adds the ability
// modifier to the extra attack's damage (Two-Weapon Fighting, SRD 5.2.1, p. 88):
// the closed set NoExtraAttackDamageEffect and ExtraAttackDamageEffectActive.
//
//sumtype:decl
type ExtraAttackDamageSource interface{ isExtraAttackDamageSource() }

// NoExtraAttackDamageEffect: no effect; the Light rule applies.
type NoExtraAttackDamageEffect struct{}

// ExtraAttackDamageEffectActive: the ability modifier is added whatever its sign.
type ExtraAttackDamageEffectActive struct{}

func (NoExtraAttackDamageEffect) isExtraAttackDamageSource()     {}
func (ExtraAttackDamageEffectActive) isExtraAttackDamageSource() {}

// LightExtraAttackBlockReason says why the Light extra attack is not allowed.
// When several apply, the first in this order is reported. The zero value is not
// a reason.
type LightExtraAttackBlockReason int

const (
	// LightBlockedAttackActionNotTaken: the Attack action was not taken this turn.
	LightBlockedAttackActionNotTaken LightExtraAttackBlockReason = iota + 1
	// LightBlockedWeaponNotLight: the extra-attack weapon lacks Light.
	LightBlockedWeaponNotLight
	// LightBlockedAlreadyMadeThisTurn: the Light extra attack was already made.
	LightBlockedAlreadyMadeThisTurn
	// LightBlockedNoLightWeaponAttackMade: no attack with a Light weapon this turn.
	LightBlockedNoLightWeaponAttackMade
	// LightBlockedSameLightWeapon: every Light attack used this very weapon.
	LightBlockedSameLightWeapon
)

// String returns the short name ("attackActionNotTaken", ...) or "unknown".
func (r LightExtraAttackBlockReason) String() string {
	names := []string{"attackActionNotTaken", "weaponNotLight", "alreadyMadeThisTurn", "noLightWeaponAttackMade", "sameLightWeapon"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// LightExtraAttackInput is the input of PlanLightExtraAttack. Construct it with
// a keyed literal. A nil InPlay reads as WeaponsOwnMastery, a nil Access as
// NoMasteryFeature and a nil DamageSource as NoExtraAttackDamageEffect.
type LightExtraAttackInput struct {
	State           WeaponTurnState
	Weapon          content.Weapon
	InstanceKey     WeaponInstanceKey
	Access          WeaponMasteryAccess
	InPlay          MasteryInPlay
	AbilityModifier ability.Modifier
	DamageSource    ExtraAttackDamageSource
}

// LightExtraAttackResult is the closed outcome of PlanLightExtraAttack:
// LightExtraAttackAllowed or LightExtraAttackBlocked.
//
//sumtype:decl
type LightExtraAttackResult interface{ isLightExtraAttackResult() }

// LightExtraAttackAllowed: the extra attack may be made at Cost. DamageModifier
// says how the ability modifier applies to its damage and AddedModifier is the
// modifier to add (0 when none).
type LightExtraAttackAllowed struct {
	Cost           ExtraAttackCost
	DamageModifier AbilityModifierToDamage
	AddedModifier  modifier.Modifier
}

// LightExtraAttackBlocked: the extra attack may not be made.
type LightExtraAttackBlocked struct{ Reason LightExtraAttackBlockReason }

func (LightExtraAttackAllowed) isLightExtraAttackResult() {}
func (LightExtraAttackBlocked) isLightExtraAttackResult() {}

// PlanLightExtraAttack decides whether the Light extra attack may be made (SRD
// 5.2.1, "Light", p. 89), at what cost (Nick makes it part of the Attack action,
// p. 90) and how its damage takes the ability modifier. It is pure.
func PlanLightExtraAttack(in LightExtraAttackInput) LightExtraAttackResult {
	s := in.State
	hasLight := hasProperty(in.Weapon, content.WeaponPropertyLight)
	lightAttacks, others := 0, 0
	for _, a := range s.attacks {
		if a.hasLight {
			lightAttacks++
			if a.key != in.InstanceKey {
				others++
			}
		}
	}
	switch {
	case s.attackAction != markDone:
		return LightExtraAttackBlocked{Reason: LightBlockedAttackActionNotTaken}
	case !hasLight:
		return LightExtraAttackBlocked{Reason: LightBlockedWeaponNotLight}
	case s.lightExtra == markDone:
		return LightExtraAttackBlocked{Reason: LightBlockedAlreadyMadeThisTurn}
	case lightAttacks == 0:
		return LightExtraAttackBlocked{Reason: LightBlockedNoLightWeaponAttackMade}
	case others == 0:
		return LightExtraAttackBlocked{Reason: LightBlockedSameLightWeapon}
	}
	cost := ExtraAttackSpendsBonusAction
	if in.Access != nil && in.Access.Permits(in.Weapon.ID()) && masteryInPlay(in.Weapon, in.InPlay) == content.MasteryNick {
		cost = ExtraAttackPartOfAttackAction
	}
	mode, added := AddModifierOmitUnlessNegative, min(in.AbilityModifier.Value(), 0)
	if _, active := in.DamageSource.(ExtraAttackDamageEffectActive); active {
		mode, added = AddModifierAlways, in.AbilityModifier.Value()
	}
	return LightExtraAttackAllowed{Cost: cost, DamageModifier: mode, AddedModifier: modifier.New(added)}
}
