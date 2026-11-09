package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ItemEffectCondition is when an item's effect applies. The zero value is
// invalid and never holds.
type ItemEffectCondition int

// The conditions.
const (
	WhileWorn ItemEffectCondition = iota + 1
	WhileHeld
	WhileOnPerson
	WhileWornWithoutArmorOrShield
)

// ItemUsage is how a creature has an item. The zero value is invalid.
type ItemUsage int

// The usages.
const (
	UsageWorn ItemUsage = iota + 1
	UsageHeld
	UsageCarried
)

// ItemEffectContext is what decides whether the effects of one item are on: the
// access its holder has to its magic (SRD 5.2.1, "Attunement", p. 102), how the
// item is used and whether the holder wears armor or uses a Shield. Construct it
// with a keyed literal; the zero value turns nothing on.
type ItemEffectContext struct {
	Access     MagicalPropertiesAccessLevel
	Usage      ItemUsage
	WearsArmor bool
	UsesShield bool
}

// holds reports whether c satisfies cond. WhileOnPerson holds for any use: a
// worn or held item is on the person too.
func (c ItemEffectContext) holds(cond ItemEffectCondition) bool {
	switch cond {
	case WhileWorn:
		return c.Usage == UsageWorn
	case WhileHeld:
		return c.Usage == UsageHeld
	case WhileOnPerson:
		return c.Usage >= UsageWorn && c.Usage <= UsageCarried
	case WhileWornWithoutArmorOrShield:
		return c.Usage == UsageWorn && !c.WearsArmor && !c.UsesShield
	}
	return false
}

// conditioned holds the condition every effect carries.
type conditioned struct{ cond ItemEffectCondition }

// Condition returns when the effect applies.
func (c conditioned) Condition() ItemEffectCondition { return c.cond }

// MagicItemEffect is the closed set of numeric item effects. Variants:
// ArmorClassBonus, SavingThrowBonus, AttackAndDamageBonus, SpellAttackRollBonus,
// IgnoresHalfCoverOnSpellAttacks, SetAbilityScore, RaiseAbilityScore,
// BoostBestowedScore.
type MagicItemEffect interface{ isMagicItemEffect() }

// ArmorClassBonus is a bonus to Armor Class on top of the wearer's base.
type ArmorClassBonus struct {
	conditioned
	mod modifier.Modifier
}

// NewArmorClassBonus returns the bonus m while cond holds.
func NewArmorClassBonus(m modifier.Modifier, cond ItemEffectCondition) ArmorClassBonus {
	return ArmorClassBonus{conditioned{cond}, m}
}

// Modifier returns the bonus.
func (e ArmorClassBonus) Modifier() modifier.Modifier { return e.mod }

// SavingThrowBonus is a bonus to saving throws; the host decides which ones.
type SavingThrowBonus struct {
	conditioned
	mod modifier.Modifier
}

// NewSavingThrowBonus returns the bonus m while cond holds.
func NewSavingThrowBonus(m modifier.Modifier, cond ItemEffectCondition) SavingThrowBonus {
	return SavingThrowBonus{conditioned{cond}, m}
}

// Modifier returns the bonus.
func (e SavingThrowBonus) Modifier() modifier.Modifier { return e.mod }

// AttackAndDamageBonus is one bonus to the attack rolls and the damage rolls made
// with one weapon or piece of ammunition.
type AttackAndDamageBonus struct {
	conditioned
	mod modifier.Modifier
}

// NewAttackAndDamageBonus returns the bonus m while cond holds.
func NewAttackAndDamageBonus(m modifier.Modifier, cond ItemEffectCondition) AttackAndDamageBonus {
	return AttackAndDamageBonus{conditioned{cond}, m}
}

// Modifier returns the bonus.
func (e AttackAndDamageBonus) Modifier() modifier.Modifier { return e.mod }

// SpellAttackRollBonus is a bonus to spell attack rolls. It is not a spell save
// DC bonus and not SpellAttackBonus.
type SpellAttackRollBonus struct {
	conditioned
	mod modifier.Modifier
}

// NewSpellAttackRollBonus returns the bonus m while cond holds.
func NewSpellAttackRollBonus(m modifier.Modifier, cond ItemEffectCondition) SpellAttackRollBonus {
	return SpellAttackRollBonus{conditioned{cond}, m}
}

// Modifier returns the bonus.
func (e SpellAttackRollBonus) Modifier() modifier.Modifier { return e.mod }

// IgnoresHalfCoverOnSpellAttacks: spell attack rolls ignore Half Cover.
type IgnoresHalfCoverOnSpellAttacks struct{ conditioned }

// NewIgnoresHalfCoverOnSpellAttacks returns the effect while cond holds.
func NewIgnoresHalfCoverOnSpellAttacks(cond ItemEffectCondition) IgnoresHalfCoverOnSpellAttacks {
	return IgnoresHalfCoverOnSpellAttacks{conditioned{cond}}
}

// SetAbilityScore sets an ability score to a value that has no effect on a base
// score at or above it. Item durations are data; expiry is the host's.
type SetAbilityScore struct {
	conditioned
	ability  ability.Ability
	score    ability.Score
	duration time.Duration
	bestows  bool
}

// NewSetAbilityScore returns the effect setting a to score while cond holds. It
// returns an error matching srderr.ErrInvalidInput for an invalid ability or a
// score outside 1 to 30.
func NewSetAbilityScore(a ability.Ability, score int, cond ItemEffectCondition) (SetAbilityScore, error) {
	s, err := ability.NewScore(score)
	if err != nil || !a.Valid() {
		return SetAbilityScore{}, invalidInput("set ability score: invalid ability or score")
	}
	return SetAbilityScore{conditioned: conditioned{cond}, ability: a, score: s}, nil
}

// ForBestowing returns e marked as the kind a Hammer of Thunderbolts can add to
// (Belt of Giant Strength, Gauntlets of Ogre Power).
func (e SetAbilityScore) ForBestowing() SetAbilityScore { e.bestows = true; return e }

// Lasting returns e carrying the duration d (Potion of Giant Strength: one hour).
func (e SetAbilityScore) Lasting(d time.Duration) SetAbilityScore { e.duration = d; return e }

// Ability returns the ability set.
func (e SetAbilityScore) Ability() ability.Ability { return e.ability }

// Score returns the score granted.
func (e SetAbilityScore) Score() ability.Score { return e.score }

// Duration returns the duration, or 0 when the item states none.
func (e SetAbilityScore) Duration() time.Duration { return e.duration }

// RaiseAbilityScore raises an ability score by an amount up to a cap, never
// lowering a score already above the cap.
type RaiseAbilityScore struct {
	conditioned
	ability ability.Ability
	amount  int
	cap     int
}

// NewRaiseAbilityScore returns the effect raising a by amount to at most limit
// while cond holds. It returns an error matching srderr.ErrInvalidInput for an
// invalid ability, an amount below 1 or a limit outside 1 to 30.
func NewRaiseAbilityScore(a ability.Ability, amount, limit int, cond ItemEffectCondition) (RaiseAbilityScore, error) {
	if !a.Valid() || amount < 1 || limit < 1 || limit > 30 {
		return RaiseAbilityScore{}, invalidInput("raise ability score: invalid ability, amount or cap")
	}
	return RaiseAbilityScore{conditioned{cond}, a, amount, limit}, nil
}

// Ability returns the ability raised.
func (e RaiseAbilityScore) Ability() ability.Ability { return e.ability }

// Amount returns the raise.
func (e RaiseAbilityScore) Amount() int { return e.amount }

// Cap returns the highest score the raise reaches.
func (e RaiseAbilityScore) Cap() int { return e.cap }

// BoostBestowedScore adds to the Strength score a Belt of Giant Strength or
// Gauntlets of Ogre Power bestows (Hammer of Thunderbolts, Might of Giants).
type BoostBestowedScore struct {
	conditioned
	amount int
	cap    int
}

// NewBoostBestowedScore returns the effect adding amount, to at most limit, while
// cond holds. It returns an error matching srderr.ErrInvalidInput for an amount
// below 1 or a limit outside 1 to 30.
func NewBoostBestowedScore(amount, limit int, cond ItemEffectCondition) (BoostBestowedScore, error) {
	if amount < 1 || limit < 1 || limit > 30 {
		return BoostBestowedScore{}, invalidInput("boost bestowed score: invalid amount or cap")
	}
	return BoostBestowedScore{conditioned{cond}, amount, limit}, nil
}

// Amount returns the boost.
func (e BoostBestowedScore) Amount() int { return e.amount }

// Cap returns the highest score the boost reaches.
func (e BoostBestowedScore) Cap() int { return e.cap }

func (ArmorClassBonus) isMagicItemEffect()                {}
func (SavingThrowBonus) isMagicItemEffect()               {}
func (AttackAndDamageBonus) isMagicItemEffect()           {}
func (SpellAttackRollBonus) isMagicItemEffect()           {}
func (IgnoresHalfCoverOnSpellAttacks) isMagicItemEffect() {}
func (SetAbilityScore) isMagicItemEffect()                {}
func (RaiseAbilityScore) isMagicItemEffect()              {}
func (BoostBestowedScore) isMagicItemEffect()             {}

// MagicItemBonus is one bonus and the item copy it comes from.
type MagicItemBonus struct {
	mod      modifier.Modifier
	instance MagicItemInstanceID
}

// Modifier returns the bonus.
func (b MagicItemBonus) Modifier() modifier.Modifier { return b.mod }

// InstanceID returns the item copy.
func (b MagicItemBonus) InstanceID() MagicItemInstanceID { return b.instance }

// EquippedItem is one item copy, its effects and the context that decides them.
// Construct it with a keyed literal.
type EquippedItem struct {
	Instance MagicItemInstanceID
	Effects  []MagicItemEffect
	Context  ItemEffectContext
}

// activeScoreEffect is a score effect that is on, with the copy it comes from.
type activeScoreEffect struct {
	instance MagicItemInstanceID
	effect   MagicItemEffect
	// worn reports whether the item is worn, which the Hammer's boost needs of a Belt or Gauntlets.
	worn bool
}

// ActiveItemEffects are the effects that are on, grouped by kind in input order.
// Bonuses from different items add without a cap. The zero value is empty.
type ActiveItemEffects struct {
	armorClass       []MagicItemBonus
	savingThrows     []MagicItemBonus
	spellAttackRolls []MagicItemBonus
	attackAndDamage  []MagicItemBonus
	ignoresHalfCover bool
	scores           []activeScoreEffect
}

// ActiveEffectsOf keeps an effect of an item only when the item's Access is AccessFullMagical
// and the effect's condition holds for its Context (SRD 5.2.1, "Attunement",
// p. 102; "Bracers of Defense", p. 214). Nothing caps or merges the bonuses of
// different items. It returns an error matching srderr.ErrInvalidInput for a
// blank instance. The input is not changed.
func ActiveEffectsOf(items []EquippedItem) (ActiveItemEffects, error) {
	var out ActiveItemEffects
	for _, it := range items {
		if !it.Instance.Valid() {
			return ActiveItemEffects{}, invalidInput("active item effects: blank instance")
		}
		if it.Context.Access != AccessFullMagical {
			continue
		}
		for _, e := range it.Effects {
			switch e := e.(type) {
			case ArmorClassBonus:
				out.armorClass = addBonus(out.armorClass, it, e.cond, e.mod)
			case SavingThrowBonus:
				out.savingThrows = addBonus(out.savingThrows, it, e.cond, e.mod)
			case SpellAttackRollBonus:
				out.spellAttackRolls = addBonus(out.spellAttackRolls, it, e.cond, e.mod)
			case AttackAndDamageBonus:
				out.attackAndDamage = addBonus(out.attackAndDamage, it, e.cond, e.mod)
			case IgnoresHalfCoverOnSpellAttacks:
				out.ignoresHalfCover = out.ignoresHalfCover || it.Context.holds(e.cond)
			case SetAbilityScore:
				out.scores = addScore(out.scores, it, e.cond, e)
			case RaiseAbilityScore:
				out.scores = addScore(out.scores, it, e.cond, e)
			case BoostBestowedScore:
				out.scores = addScore(out.scores, it, e.cond, e)
			}
		}
	}
	return out, nil
}

func addBonus(list []MagicItemBonus, it EquippedItem, cond ItemEffectCondition, m modifier.Modifier) []MagicItemBonus {
	if !it.Context.holds(cond) {
		return list
	}
	return append(list, MagicItemBonus{mod: m, instance: it.Instance})
}

func addScore(list []activeScoreEffect, it EquippedItem, cond ItemEffectCondition, e MagicItemEffect) []activeScoreEffect {
	if !it.Context.holds(cond) {
		return list
	}
	return append(list, activeScoreEffect{instance: it.Instance, effect: e, worn: it.Context.Usage == UsageWorn})
}

// ArmorClass returns a new slice of the Armor Class bonuses.
func (a ActiveItemEffects) ArmorClass() []MagicItemBonus { return slices.Clone(a.armorClass) }

// SavingThrows returns a new slice of the saving throw bonuses.
func (a ActiveItemEffects) SavingThrows() []MagicItemBonus { return slices.Clone(a.savingThrows) }

// SpellAttackRolls returns a new slice of the spell attack roll bonuses.
func (a ActiveItemEffects) SpellAttackRolls() []MagicItemBonus {
	return slices.Clone(a.spellAttackRolls)
}

// AttackAndDamage returns a new slice of the attack and damage bonuses, one per
// item copy.
func (a ActiveItemEffects) AttackAndDamage() []MagicItemBonus { return slices.Clone(a.attackAndDamage) }

// IgnoresHalfCoverOnSpellAttacks reports whether spell attack rolls ignore Half Cover.
func (a ActiveItemEffects) IgnoresHalfCoverOnSpellAttacks() bool { return a.ignoresHalfCover }
