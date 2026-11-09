package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// WeaponAttackMode is how a weapon is used in an attack: the closed set
// MeleeAttackMode, ThrownAttackMode and RangedAttackMode.
//
//sumtype:decl
type WeaponAttackMode interface{ isWeaponAttackMode() }

// MeleeAttackMode is a melee use of the weapon.
type MeleeAttackMode struct{}

// ThrownAttackMode is a use of the weapon with the Thrown property.
type ThrownAttackMode struct{}

// RangedAttackMode is a use of an Ammunition weapon.
type RangedAttackMode struct{}

func (MeleeAttackMode) isWeaponAttackMode()  {}
func (ThrownAttackMode) isWeaponAttackMode() {}
func (RangedAttackMode) isWeaponAttackMode() {}

// WeaponAttackKind is the attack form of a plan. The zero value is not a kind.
type WeaponAttackKind int

const (
	// MeleeWeaponAttackKind is a melee weapon attack.
	MeleeWeaponAttackKind WeaponAttackKind = iota + 1
	// RangedWeaponAttackKind is a ranged weapon attack.
	RangedWeaponAttackKind
)

// String returns "meleeWeaponAttack", "rangedWeaponAttack" or "unknown".
func (k WeaponAttackKind) String() string {
	switch k {
	case MeleeWeaponAttackKind:
		return "meleeWeaponAttack"
	case RangedWeaponAttackKind:
		return "rangedWeaponAttack"
	}
	return "unknown"
}

// AbilityPreference chooses the attack ability of a Finesse weapon: the closed
// set DefaultAbility and ChosenAbility.
//
//sumtype:decl
type AbilityPreference interface{ isAbilityPreference() }

// DefaultAbility uses Strength for melee and thrown Melee weapons and Dexterity
// for ranged use and thrown Ranged weapons.
type DefaultAbility struct{}

// ChosenAbility is the Strength or Dexterity chosen for a Finesse weapon (SRD
// 5.2.1, "Finesse", p. 89). Build it with NewChosenAbility.
type ChosenAbility struct{ ability ability.Ability }

// NewChosenAbility returns the choice of a. It returns an error matching
// srderr.ErrInvalidInput, naming ability, for anything but Strength or Dexterity.
func NewChosenAbility(a ability.Ability) (ChosenAbility, error) {
	if a != ability.Strength && a != ability.Dexterity {
		return ChosenAbility{}, invalidInput("ability of a Finesse choice: not Strength or Dexterity")
	}
	return ChosenAbility{ability: a}, nil
}

// MustChosenAbility is NewChosenAbility for values known to be valid. It panics
// with the error that function returns, and only then.
func MustChosenAbility(a ability.Ability) ChosenAbility {
	c, err := NewChosenAbility(a)
	if err != nil {
		panic(err)
	}
	return c
}

// Ability returns the chosen ability.
func (c ChosenAbility) Ability() ability.Ability { return c.ability }

func (DefaultAbility) isAbilityPreference() {}
func (ChosenAbility) isAbilityPreference()  {}

// AbilityModifierToDamage says whether the ability modifier is added to a
// damage roll. The zero value is not a rule.
type AbilityModifierToDamage int

const (
	// AddModifierOmitUnlessNegative adds the modifier only when it is negative
	// (the Light extra attack, SRD 5.2.1, p. 89).
	AddModifierOmitUnlessNegative AbilityModifierToDamage = iota + 1
	// AddModifierAlways adds the modifier whatever its sign.
	AddModifierAlways
	// AddModifierNever adds nothing (a fixed damage amount, SRD 5.2.1, p. 16).
	AddModifierNever
)

// String returns "omitUnlessNegative", "alwaysAdd", "neverAdd" or "unknown".
func (m AbilityModifierToDamage) String() string {
	switch m {
	case AddModifierOmitUnlessNegative:
		return "omitUnlessNegative"
	case AddModifierAlways:
		return "alwaysAdd"
	case AddModifierNever:
		return "neverAdd"
	}
	return "unknown"
}

// WeaponAttackBlockReason says why a weapon attack cannot be planned. When
// several apply, the one reported is the first in this order. The zero value is
// not a reason.
type WeaponAttackBlockReason int

const (
	// AttackBlockedImprovisedUse: a Ranged weapon used in melee, or a Melee
	// weapon without Thrown thrown; the host applies the improvised rules.
	AttackBlockedImprovisedUse WeaponAttackBlockReason = iota + 1
	// AttackBlockedModeNotSupportedByWeapon: another use the weapon cannot support.
	AttackBlockedModeNotSupportedByWeapon
	// AttackBlockedFinesseNotAvailable: an ability was chosen for a weapon without Finesse.
	AttackBlockedFinesseNotAvailable
	// AttackBlockedExemptionNotApplicable: an exemption was reported for a weapon whose Two-Handed has no condition, or without Two-Handed.
	AttackBlockedExemptionNotApplicable
	// AttackBlockedNeedsTwoHands: a Two-Handed weapon is not held in two hands.
	AttackBlockedNeedsTwoHands
	// AttackBlockedWrongAmmunitionType: the supply is of another type.
	AttackBlockedWrongAmmunitionType
	// AttackBlockedNoAmmunition: no piece of ammunition is left.
	AttackBlockedNoAmmunition
	// AttackBlockedNoFreeHandToLoad: a one-handed Ammunition weapon needs a free hand.
	AttackBlockedNoFreeHandToLoad
	// AttackBlockedLoadingLimit: a Loading weapon was already fired this activation.
	AttackBlockedLoadingLimit
)

func blockReasonNames() []string {
	return []string{"improvisedUse", "modeNotSupportedByWeapon", "finesseNotAvailable", "exemptionNotApplicable",
		"needsTwoHands", "wrongAmmunitionType", "noAmmunition", "noFreeHandToLoad", "loadingLimit"}
}

// String returns the short name ("improvisedUse", ...) or "unknown".
func (r WeaponAttackBlockReason) String() string {
	if r < 1 || int(r) > len(blockReasonNames()) {
		return "unknown"
	}
	return blockReasonNames()[r-1]
}

// PlannedReachOrRange is the reach of a melee plan or the range of a thrown or
// ranged plan: the closed set PlannedReach and PlannedRange.
//
//sumtype:decl
type PlannedReachOrRange interface{ isPlannedReachOrRange() }

// PlannedReach is the reach of a melee attack.
type PlannedReach struct{ Reach ReachInFeet }

// PlannedRange is the range of a thrown or ranged attack.
type PlannedRange struct{ Range content.RangePair }

func (PlannedReach) isPlannedReachOrRange() {}
func (PlannedRange) isPlannedReachOrRange() {}

// WeaponAttackPlan is what the property rules add to an attack: published
// language the host maps onto its own attack resolution. It carries the damage
// as content, never a rolled amount; the host adds the one ability modifier once
// as DamageModifier says. Immutable.
type WeaponAttackPlan struct {
	kind          WeaponAttackKind
	ability       ability.Ability
	modifier      ability.Modifier
	proficiency   WeaponProficiencyStatus
	damage        content.WeaponDamage
	damageMod     AbilityModifierToDamage
	reachOrRange  PlannedReachOrRange
	circumstances []modifier.RollCircumstance
	ammunition    AmmunitionSupply
	hasAmmunition bool
	loadingAfter  LoadingState
}

// Kind returns the attack form.
func (p WeaponAttackPlan) Kind() WeaponAttackKind { return p.kind }

// Ability returns the ability of the attack roll and the damage roll.
func (p WeaponAttackPlan) Ability() ability.Ability { return p.ability }

// AbilityModifier returns the one ability modifier for both rolls.
func (p WeaponAttackPlan) AbilityModifier() ability.Modifier { return p.modifier }

// Proficiency returns whether the Proficiency Bonus is added.
func (p WeaponAttackPlan) Proficiency() WeaponProficiencyStatus { return p.proficiency }

// Damage returns the weapon damage to roll.
func (p WeaponAttackPlan) Damage() content.WeaponDamage { return p.damage }

// DamageModifier says whether the ability modifier is added to the damage.
func (p WeaponAttackPlan) DamageModifier() AbilityModifierToDamage { return p.damageMod }

// ReachOrRange returns the PlannedReach or PlannedRange.
func (p WeaponAttackPlan) ReachOrRange() PlannedReachOrRange { return p.reachOrRange }

// Circumstances returns the roll circumstances (the Heavy Disadvantage) as a new slice.
func (p WeaponAttackPlan) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(p.circumstances)
}

// AmmunitionAfter returns the supply after the attack and true, or false when
// the weapon has no Ammunition.
func (p WeaponAttackPlan) AmmunitionAfter() (AmmunitionSupply, bool) {
	return p.ammunition, p.hasAmmunition
}

// LoadingAfter returns the loading state after the attack.
func (p WeaponAttackPlan) LoadingAfter() LoadingState { return p.loadingAfter }

// WeaponAttackPlanResult is the closed outcome of PlanWeaponAttack:
// WeaponAttackPlanned or WeaponAttackPlanBlocked.
//
//sumtype:decl
type WeaponAttackPlanResult interface{ isWeaponAttackPlanResult() }

// WeaponAttackPlanned carries the plan.
type WeaponAttackPlanned struct{ Plan WeaponAttackPlan }

// WeaponAttackPlanBlocked carries the first reason the attack cannot be made.
type WeaponAttackPlanBlocked struct{ Reason WeaponAttackBlockReason }

func (WeaponAttackPlanned) isWeaponAttackPlanResult()     {}
func (WeaponAttackPlanBlocked) isWeaponAttackPlanResult() {}

// WeaponAttackInput is the input of PlanWeaponAttack. Construct it with a keyed
// literal. A nil Preference reads as DefaultAbility; a zero AttackerReach reads
// as 5 feet.
type WeaponAttackInput struct {
	Weapon        content.Weapon
	Mode          WeaponAttackMode
	Strength      ability.Score
	Dexterity     ability.Score
	Preference    AbilityPreference
	Wielding      WeaponWielding
	AttackerReach ReachInFeet
	Proficiencies WeaponProficiencies
}

// weaponUses indexes the data-carrying properties of a weapon.
type weaponUses struct {
	ammunition *content.AmmunitionUse
	thrown     *content.ThrownUse
	versatile  *content.VersatileUse
	twoHanded  *content.TwoHandedUse
}

func usesOf(w content.Weapon) weaponUses {
	var u weaponUses
	for _, p := range w.Properties() {
		switch v := p.(type) {
		case content.AmmunitionUse:
			u.ammunition = &v
		case content.ThrownUse:
			u.thrown = &v
		case content.VersatileUse:
			u.versatile = &v
		case content.TwoHandedUse:
			u.twoHanded = &v
		case content.PlainUse:
		}
	}
	return u
}

func blocked(r WeaponAttackBlockReason) WeaponAttackPlanResult {
	return WeaponAttackPlanBlocked{Reason: r}
}

// PlanWeaponAttack applies the weapon property rules to one attack (SRD 5.2.1,
// "Properties", pp. 89-90): the ability, Heavy Disadvantage, reach or range,
// damage dice (Versatile), Ammunition and Loading. It is pure, never panics and
// returns a blocked result, not an error, for an attack that cannot be made. A
// non-proficient attacker and a Heavy shortfall do not block it.
func PlanWeaponAttack(in WeaponAttackInput) WeaponAttackPlanResult {
	w, uses := in.Weapon, usesOf(in.Weapon)
	melee, thrown, ranged := false, false, false
	switch in.Mode.(type) {
	case MeleeAttackMode:
		melee = true
	case ThrownAttackMode:
		thrown = true
	case RangedAttackMode:
		ranged = true
	}
	switch {
	case melee && w.Type() == content.WeaponRanged,
		thrown && w.Type() == content.WeaponMelee && uses.thrown == nil:
		return blocked(AttackBlockedImprovisedUse)
	case !melee && !thrown && !ranged,
		ranged && uses.ammunition == nil,
		thrown && uses.thrown == nil:
		return blocked(AttackBlockedModeNotSupportedByWeapon)
	}
	chosen, isChosen := in.Preference.(ChosenAbility)
	if isChosen && !hasProperty(w, content.WeaponPropertyFinesse) {
		return blocked(AttackBlockedFinesseNotAvailable)
	}
	exempt := in.Wielding.Exemption == TwoHandedExemptionApplies
	if exempt && (uses.twoHanded == nil || uses.twoHanded.Condition() == content.TwoHandedAlways) {
		return blocked(AttackBlockedExemptionNotApplicable)
	}
	twoHanded := in.Wielding.Grip == GripTwoHanded
	if uses.twoHanded != nil && !twoHanded && !exempt {
		return blocked(AttackBlockedNeedsTwoHands)
	}
	supply, hasSupply := in.Wielding.Ammunition()
	if ranged {
		switch {
		case hasSupply && supply.Type() != uses.ammunition.Ammunition():
			return blocked(AttackBlockedWrongAmmunitionType)
		case !hasSupply || supply.Pieces() == 0:
			return blocked(AttackBlockedNoAmmunition)
		case uses.twoHanded == nil && in.Wielding.Hands == HandStateNoFreeHand:
			return blocked(AttackBlockedNoFreeHandToLoad)
		}
	}
	loadingAfter := in.Wielding.Loading
	if hasProperty(w, content.WeaponPropertyLoading) {
		if loadingAfter == LoadingAlreadyFiredThisActivation {
			return blocked(AttackBlockedLoadingLimit)
		}
		loadingAfter = LoadingAlreadyFiredThisActivation
	}

	plan := WeaponAttackPlan{proficiency: in.Proficiencies.StatusFor(w), loadingAfter: loadingAfter}
	plan.kind = RangedWeaponAttackKind
	if melee {
		plan.kind = MeleeWeaponAttackKind
	}
	plan.ability = ability.Strength
	if ranged || thrown && w.Type() == content.WeaponRanged {
		plan.ability = ability.Dexterity
	}
	if isChosen {
		plan.ability = chosen.ability
	}
	score := in.Strength
	if plan.ability == ability.Dexterity {
		score = in.Dexterity
	}
	plan.modifier = score.Modifier()
	if hasProperty(w, content.WeaponPropertyHeavy) {
		classification := in.Strength
		if w.Type() == content.WeaponRanged {
			classification = in.Dexterity
		}
		if classification.Value() < 13 {
			plan.circumstances = []modifier.RollCircumstance{modifier.Disadvantage{}}
		}
	}
	switch {
	case melee:
		plan.reachOrRange = PlannedReach{Reach: weaponReach(w, in.AttackerReach)}
	case thrown:
		plan.reachOrRange = PlannedRange{Range: uses.thrown.Range()}
	default:
		plan.reachOrRange = PlannedRange{Range: uses.ammunition.Range()}
	}
	plan.damage, plan.damageMod = planDamage(w, uses, melee && twoHanded)
	if ranged {
		plan.ammunition = AmmunitionSupply{pieces: supply.Pieces() - 1, kind: supply.Type()}
		plan.hasAmmunition = true
	}
	return WeaponAttackPlanned{Plan: plan}
}

// planDamage returns the weapon damage and how the ability modifier applies:
// the Versatile dice for a two-handed melee use, a fixed amount never taking the
// modifier (SRD 5.2.1, "Damage Rolls", p. 16).
func planDamage(w content.Weapon, uses weaponUses, twoHandedMelee bool) (content.WeaponDamage, AbilityModifierToDamage) {
	base := w.Damage()
	if _, fixed := base.(content.WeaponFixedDamage); fixed {
		return base, AddModifierNever
	}
	if twoHandedMelee && uses.versatile != nil {
		// An error is unreachable: both parts come from a valid Weapon.
		if d, err := content.NewWeaponDiceDamage(uses.versatile.Damage(), base.DamageType()); err == nil {
			return d, AddModifierAlways
		}
	}
	return base, AddModifierAlways
}
