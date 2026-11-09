package character

import (
	"slices"
	"strconv"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

const (
	monkSlowFallPerLevel = 5
	monkBodyAndMindBonus = 4
	monkBodyAndMindMax   = 25
)

// MonkUnarmoredDefenseBasis returns the Armor Class basis Unarmored Defense gives: 10
// plus Dexterity plus Wisdom, not usable with a Shield (SRD 5.2.1, "Level 1: Unarmored
// Defense", p. 50). The host passes it to the Armor Class calculation when the
// character holds the feature. The definition declares no basis because content is
// upstream of character.
func MonkUnarmoredDefenseBasis() AbilityBasedArmorClassBasis {
	b, _ := NewAbilityBasedArmorClassBasis("monk_unarmored_defense", []ability.Ability{ability.Wisdom}, false) // the id and the ability are valid constants
	return b
}

// IsMonkWeapon reports whether w is a Monk weapon: a Simple Melee weapon, or a Martial
// Melee weapon that has the Light property (SRD 5.2.1, "Level 1: Martial Arts", p. 50).
func IsMonkWeapon(w content.Weapon) bool {
	if w.Type() != content.WeaponMelee {
		return false
	}
	light := slices.ContainsFunc(w.Properties(), func(u content.WeaponPropertyUse) bool { return u.Property() == content.WeaponPropertyLight })
	return w.Category() == content.WeaponSimple || (w.Category() == content.WeaponMartial && light)
}

// MonkLoadout is what the Monk wields and wears, as the host reports it. Construct it
// with a keyed literal; the zero value is unarmed, unarmored and without a Shield.
type MonkLoadout struct {
	// Weapons are the weapons the Monk holds.
	Weapons []content.Weapon
	// ArmorWorn reports that the Monk wears armor.
	ArmorWorn bool
	// ShieldWielded reports that the Monk wields a Shield.
	ShieldWielded bool
}

// MartialArtsOffers are the Martial Arts benefits a Monk is offered (SRD 5.2.1, "Level
// 1: Martial Arts", p. 50). Only MonkEffects.MartialArts builds one; every method
// answers false or the zero die while the benefits do not apply. A value.
type MartialArtsOffers struct {
	applies bool
	die     dice.Die
}

// Applies reports that the Monk is unarmed or wields only Monk weapons, wears no armor
// and wields no Shield.
func (o MartialArtsOffers) Applies() bool { return o.applies }

// BonusUnarmedStrike reports that an Unarmed Strike is offered as a Bonus Action.
func (o MartialArtsOffers) BonusUnarmedStrike() bool { return o.applies }

// MartialArtsDie returns the die the Monk may roll in place of the normal damage of an
// Unarmed Strike or a Monk weapon, an option the player may take or leave; the zero die when it does not apply.
func (o MartialArtsOffers) MartialArtsDie() dice.Die { return o.die }

// DexterityForAttacks reports that the attack and damage rolls of an Unarmed Strike or
// Monk weapon may use the Dexterity modifier instead of the Strength modifier.
func (o MartialArtsOffers) DexterityForAttacks() bool { return o.applies }

// DexterityForGrappleAndShoveDC reports that the save DC of the Grapple or Shove option
// may use the Dexterity modifier instead of the Strength modifier.
func (o MartialArtsOffers) DexterityForGrappleAndShoveDC() bool { return o.applies }

// MonkAbilityChange is a new ability score and the maximum it may reach, for the host
// to feed to ChangeAbilityScore. Construct it with a keyed literal.
type MonkAbilityChange struct {
	Ability  ability.Ability
	NewScore int
	Maximum  int
}

// MonkEvasionFacts is the effect Evasion would apply to. Construct it with a keyed
// literal.
type MonkEvasionFacts struct {
	// Save is the ability of the saving throw.
	Save ability.Ability
	// Outcome is the result of the Monk's saving throw.
	Outcome d20.Outcome
	// HalfDamageOnSuccess reports that the effect lets a save take only half damage.
	HalfDamageOnSuccess bool
	// Incapacitated reports that the Monk has the Incapacitated condition.
	Incapacitated bool
	// Damage is the effect's damage before the save.
	Damage int
}

// EvasionResult is the closed outcome of MonkEffects.Evasion. Variants: EvasionApplied,
// EvasionUnchanged, EvasionRejected.
type EvasionResult interface{ isEvasionResult() }

// EvasionApplied: Evasion applies; the Monk takes DamageTaken (0 on a success, half of the damage, rounded down, on a failure).
type EvasionApplied struct{ DamageTaken int }

// EvasionUnchanged: the effect's own result stands; Evasion does not apply to it.
type EvasionUnchanged struct{}

// EvasionRejected: the Monk cannot use Evasion; the effect's own result stands. Construct it with a keyed literal.
type EvasionRejected struct{ Reason MonkReason }

func (EvasionApplied) isEvasionResult()   {}
func (EvasionUnchanged) isEvasionResult() {}
func (EvasionRejected) isEvasionResult()  {}

// SlowFallResult is the closed outcome of MonkEffects.SlowFall. Variants: SlowFallApplied, SlowFallRejected.
type SlowFallResult interface{ isSlowFallResult() }

// SlowFallApplied: the Reaction reduces the fall damage by Reduction, to DamageAfter.
type SlowFallApplied struct{ Reduction, DamageAfter int }

// SlowFallRejected: Slow Fall is not available. Construct it with a keyed literal.
type SlowFallRejected struct{ Reason MonkReason }

func (SlowFallApplied) isSlowFallResult()  {}
func (SlowFallRejected) isSlowFallResult() {}

// BodyAndMindResult is the closed outcome of MonkEffects.BodyAndMind. Variants:
// BodyAndMindApplied, BodyAndMindRejected.
type BodyAndMindResult interface{ isBodyAndMindResult() }

// BodyAndMindApplied: the changes to Dexterity then Wisdom, for the host to apply.
type BodyAndMindApplied struct{ changes []MonkAbilityChange }

// Changes returns the Dexterity change then the Wisdom change, as a new slice.
func (r BodyAndMindApplied) Changes() []MonkAbilityChange { return slices.Clone(r.changes) }

// BodyAndMindRejected: Body and Mind is not available. Construct it with a keyed literal.
type BodyAndMindRejected struct{ Reason MonkReason }

func (BodyAndMindApplied) isBodyAndMindResult()  {}
func (BodyAndMindRejected) isBodyAndMindResult() {}

// SelfRestorationOffer is the closed outcome of MonkEffects.SelfRestoration. Variants:
// SelfRestorationOffered, SelfRestorationRejected.
type SelfRestorationOffer interface{ isSelfRestorationOffer() }

// SelfRestorationOffered: the Monk may remove one of the conditions at the end of the turn.
type SelfRestorationOffered struct{ removable []content.ConditionKind }

// Removable returns the conditions on the Monk that Self-Restoration can remove (Charmed,
// Frightened, Poisoned), in condition order, as a new slice; the Monk removes one.
func (o SelfRestorationOffered) Removable() []content.ConditionKind { return slices.Clone(o.removable) }

// SelfRestorationRejected: Self-Restoration is not available. Construct it with a keyed literal.
type SelfRestorationRejected struct{ Reason MonkReason }

func (SelfRestorationOffered) isSelfRestorationOffer()  {}
func (SelfRestorationRejected) isSelfRestorationOffer() {}

// SuperiorDefenseEffect is the Resistance Superior Defense gives. Only
// SuperiorDefense.Resolve builds one. A value.
type SuperiorDefenseEffect struct{}

// Duration returns one minute (SRD 5.2.1, "Level 18: Superior Defense", p. 52) for the host to expire.
func (SuperiorDefenseEffect) Duration() time.Duration { return time.Minute }

// EndsWhenIncapacitated reports that the effect ends when the Monk has the Incapacitated condition.
func (SuperiorDefenseEffect) EndsWhenIncapacitated() bool { return true }

// ResistsAllExcept returns the damage types the effect does not give Resistance to: Force only, as a new slice.
func (SuperiorDefenseEffect) ResistsAllExcept() []content.DamageType {
	return []content.DamageType{content.DamageForce}
}

// Resists reports whether the effect gives Resistance to damage of type t: every valid type but Force.
func (SuperiorDefenseEffect) Resists(t content.DamageType) bool {
	return t.Valid() && t != content.DamageForce
}

// MonkEffects are the stateless queries of the Monk features that declare data for
// derived stats and combat to apply (SRD 5.2.1, pp. 50-52). Every method takes the
// stats of the Monk level; the zero stats are not a Monk and give no effect. The zero
// value is ready to use. Safe for concurrent use.
type MonkEffects struct{}

func (MonkEffects) level(stats MonkLevelStats) int { return stats.level.n }

// MartialArts returns the Martial Arts offers for the Monk of stats' level with loadout:
// they apply only while the Monk wields no weapon but Monk weapons, wears no armor and
// wields no Shield (SRD 5.2.1, "Level 1: Martial Arts", p. 50). Zero stats are offered nothing.
func (MonkEffects) MartialArts(stats MonkLevelStats, loadout MonkLoadout) MartialArtsOffers {
	if stats.IsZero() || loadout.ArmorWorn || loadout.ShieldWielded || slices.ContainsFunc(loadout.Weapons, func(w content.Weapon) bool { return !IsMonkWeapon(w) }) {
		return MartialArtsOffers{}
	}
	return MartialArtsOffers{applies: true, die: stats.martialArts}
}

// UnarmoredMovement returns the feet added to Speed: the Unarmored Movement bonus of the
// level while the Monk wears no armor and wields no Shield, else 0; 0 at level 1 (SRD
// 5.2.1, "Level 2: Unarmored Movement", p. 51).
func (MonkEffects) UnarmoredMovement(stats MonkLevelStats, armorWorn, shieldWielded bool) int {
	if armorWorn || shieldWielded {
		return 0
	}
	return stats.unarmoredMovementF
}

// AttacksPerAttackAction returns the attacks per Attack action from the Monk level
// alone: 2 from level 5, else 1; 0 for zero stats (SRD 5.2.1, "Level 5: Extra Attack", p. 51).
func (e MonkEffects) AttacksPerAttackAction(stats MonkLevelStats) int {
	if stats.IsZero() {
		return 0
	}
	return 1 + thresholdCount(e.level(stats), monkExtraAttackLevel)
}

// EmpoweredStrikes reports that an Unarmed Strike may deal Force damage instead of its
// normal type, from level 6 (SRD 5.2.1, "Level 6: Empowered Strikes", p. 51).
func (e MonkEffects) EmpoweredStrikes(stats MonkLevelStats) bool {
	return e.level(stats) >= monkEmpoweredLevel
}

// AcrobaticMovement reports that the Monk may move along vertical surfaces and across
// liquids on its turn without falling: from level 9 while wearing no armor and wielding
// no Shield (SRD 5.2.1, "Level 9: Acrobatic Movement", p. 51).
func (e MonkEffects) AcrobaticMovement(stats MonkLevelStats, armorWorn, shieldWielded bool) bool {
	return e.level(stats) >= monkAcrobaticLevel && !armorWorn && !shieldWielded
}

// ForgoingFoodAndDrinkGivesExhaustion reports whether forgoing food and drink gives the
// Monk levels of Exhaustion: false from level 10 (SRD 5.2.1, "Level 10: Self-Restoration", p. 52).
func (e MonkEffects) ForgoingFoodAndDrinkGivesExhaustion(stats MonkLevelStats) bool {
	return e.level(stats) < monkHeightenedLevel
}

// ProficientInAllSavingThrows reports the proficiency in all saving throws Disciplined
// Survivor gives, from level 14 (SRD 5.2.1, "Level 14: Disciplined Survivor", p. 52). Below
// it the Monk gains none from this feature.
func (e MonkEffects) ProficientInAllSavingThrows(stats MonkLevelStats) bool {
	return e.level(stats) >= monkSurvivorLevel
}

// SlowFall reduces fall damage by five times the Monk level, never below 0 (SRD 5.2.1,
// "Level 4: Slow Fall", p. 51). It returns an error matching srderr.ErrInvalidInput for
// a negative damage. It never panics.
func (e MonkEffects) SlowFall(stats MonkLevelStats, fallDamage int) (SlowFallResult, error) {
	if fallDamage < 0 {
		return nil, invalidInput("negative fall damage " + strconv.Itoa(fallDamage))
	}
	if r := monkGate(stats, monkSlowFallLevel); r != 0 {
		return SlowFallRejected{Reason: r}, nil
	}
	reduction := monkSlowFallPerLevel * e.level(stats)
	return SlowFallApplied{Reduction: reduction, DamageAfter: max(fallDamage-reduction, 0)}, nil
}

// Evasion applies Evasion to an effect (SRD 5.2.1, "Level 7: Evasion", p. 51). It returns
// EvasionRejected below level 7 or for zero stats, EvasionUnchanged for an Incapacitated
// Monk, a save that is not Dexterity or an effect with no half-damage clause, and
// otherwise EvasionApplied. It returns an error matching srderr.ErrInvalidInput for a
// negative damage, an invalid ability or an outcome other than d20.Success or d20.Failure.
// It never panics.
func (e MonkEffects) Evasion(stats MonkLevelStats, facts MonkEvasionFacts) (EvasionResult, error) {
	if facts.Damage < 0 || !facts.Save.Valid() || (facts.Outcome != d20.Success && facts.Outcome != d20.Failure) {
		return nil, invalidInput("Evasion facts: negative damage, invalid ability or outcome")
	}
	if r := monkGate(stats, monkEvasionLevel); r != 0 {
		return EvasionRejected{Reason: r}, nil
	}
	if facts.Incapacitated || facts.Save != ability.Dexterity || !facts.HalfDamageOnSuccess {
		return EvasionUnchanged{}, nil
	}
	if facts.Outcome == d20.Success {
		return EvasionApplied{DamageTaken: 0}, nil
	}
	return EvasionApplied{DamageTaken: facts.Damage / 2}, nil
}

// SelfRestoration offers the removal of one condition at the end of the Monk's turn
// (SRD 5.2.1, "Level 10: Self-Restoration", p. 52): of the conditions on the Monk, only
// Charmed, Frightened and Poisoned are removable, and the Monk removes at most one. It
// is rejected below level 10 or for zero stats; with none of the three on the Monk it
// offers an empty list.
func (e MonkEffects) SelfRestoration(stats MonkLevelStats, conditions []content.ConditionKind) SelfRestorationOffer {
	if r := monkGate(stats, monkHeightenedLevel); r != 0 {
		return SelfRestorationRejected{Reason: r}
	}
	out := []content.ConditionKind{}
	for _, c := range []content.ConditionKind{content.ConditionCharmed, content.ConditionFrightened, content.ConditionPoisoned} {
		if slices.Contains(conditions, c) {
			out = append(out, c)
		}
	}
	return SelfRestorationOffered{removable: out}
}

// BodyAndMind raises Dexterity and Wisdom by 4 to a maximum of 25 at level 20 (SRD 5.2.1,
// "Level 20: Body and Mind", p. 52). A score already above 25 is not lowered. It returns
// an error matching srderr.ErrInvalidInput for a zero ability score. It never panics.
func (e MonkEffects) BodyAndMind(stats MonkLevelStats, dexterity, wisdom ability.Score) (BodyAndMindResult, error) {
	if dexterity.IsZero() || wisdom.IsZero() {
		return nil, invalidInput("Body and Mind with a zero ability score")
	}
	if r := monkGate(stats, monkBodyAndMindLevel); r != 0 {
		return BodyAndMindRejected{Reason: r}, nil
	}
	raise := func(v int) int { return max(v, min(v+monkBodyAndMindBonus, monkBodyAndMindMax)) }
	return BodyAndMindApplied{changes: []MonkAbilityChange{
		{Ability: ability.Dexterity, NewScore: raise(dexterity.Value()), Maximum: monkBodyAndMindMax},
		{Ability: ability.Wisdom, NewScore: raise(wisdom.Value()), Maximum: monkBodyAndMindMax},
	}}, nil
}
