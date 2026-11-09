package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// GiantPotionDuration is how long a Potion of Giant Strength lasts (SRD 5.2.1,
// "Potion of Giant Strength", p. 235).
const GiantPotionDuration = 60 * time.Minute

// EnhancementChoice is what a +N item or a giant-strength item needs besides its
// id: the resolved rarity (through ResolvedRarity) and the giant type. Construct
// it with a keyed literal; unused fields stay zero.
type EnhancementChoice struct {
	Rarity content.ItemRarity
	Giant  GiantStrengthType
}

// MagicItemEffectLookup is the closed set of outcomes of MagicItemEffectTable.Lookup.
// Variants: MagicItemEffectsFound, MagicItemEffectsNotModelled.
type MagicItemEffectLookup interface{ isMagicItemEffectLookup() }

// MagicItemEffectsFound: the effects of the item and whether it needs Attunement.
type MagicItemEffectsFound struct {
	effects    []MagicItemEffect
	attunement bool
}

// Effects returns a new slice of the effects.
func (r MagicItemEffectsFound) Effects() []MagicItemEffect { return slices.Clone(r.effects) }

// RequiresAttunement reports whether the item needs Attunement, the fact the host
// turns into the Access of its ItemEffectContext.
func (r MagicItemEffectsFound) RequiresAttunement() bool { return r.attunement }

// MagicItemEffectsNotModelled: no effect is modelled for the id, or the choice
// does not resolve a +N bonus or a giant type; nothing is invented.
type MagicItemEffectsNotModelled struct{ id content.MagicItemID }

// ItemID returns the id asked for.
func (r MagicItemEffectsNotModelled) ItemID() content.MagicItemID { return r.id }

func (MagicItemEffectsFound) isMagicItemEffectLookup()       {}
func (MagicItemEffectsNotModelled) isMagicItemEffectLookup() {}

// MagicItemEffectTable holds the numeric effects of the SRD items (tables "Fixed
// bonus effects" and "Set and raised scores", SRD 5.2.1, pp. 209-253). The ids
// are the snake_case names of the items; the +N items are armor_plus,
// shield_plus, weapon_plus, ammunition_plus and wand_of_the_war_mage_plus. Host
// items build their MagicItemEffects directly. The zero value is ready to use.
type MagicItemEffectTable struct{}

// Lookup returns the effects of id. A +N item and a giant-strength item need the
// matching field of choice; without it, or for an id the table lacks, the result
// is MagicItemEffectsNotModelled.
func (MagicItemEffectTable) Lookup(id content.MagicItemID, choice EnhancementChoice) MagicItemEffectLookup {
	found := func(attune bool, effects ...MagicItemEffect) MagicItemEffectLookup {
		return MagicItemEffectsFound{effects: effects, attunement: attune}
	}
	bonus := func(family EnhancementFamily) (modifier.Modifier, bool) {
		f, ok := (EnhancementBonusLookup{}).Of(family, choice.Rarity).(EnhancementBonusFound)
		return f.bonus, ok
	}
	ac := func(m int, c ItemEffectCondition) MagicItemEffect { return NewArmorClassBonus(modifier.New(m), c) }
	saves := func(m int, c ItemEffectCondition) MagicItemEffect { return NewSavingThrowBonus(modifier.New(m), c) }
	spell := func(m int, c ItemEffectCondition) MagicItemEffect { return NewSpellAttackRollBonus(modifier.New(m), c) }
	hit := func(m int, c ItemEffectCondition) MagicItemEffect { return NewAttackAndDamageBonus(modifier.New(m), c) }
	switch id {
	case "armor_plus":
		if m, ok := bonus(EnhancementArmor); ok {
			return found(false, NewArmorClassBonus(m, WhileWorn))
		}
	case "shield_plus":
		if m, ok := bonus(EnhancementShield); ok {
			return found(false, NewArmorClassBonus(m, WhileHeld))
		}
	case "weapon_plus":
		if m, ok := bonus(EnhancementWeapon); ok {
			return found(false, NewAttackAndDamageBonus(m, WhileHeld))
		}
	case "ammunition_plus":
		if m, ok := bonus(EnhancementAmmunition); ok {
			return found(false, NewAttackAndDamageBonus(m, WhileOnPerson))
		}
	case "wand_of_the_war_mage_plus":
		if m, ok := bonus(EnhancementWarMageWand); ok {
			return found(true, NewSpellAttackRollBonus(m, WhileHeld), NewIgnoresHalfCoverOnSpellAttacks(WhileHeld))
		}
	case "ring_of_protection", "cloak_of_protection":
		return found(true, ac(1, WhileWorn), saves(1, WhileWorn))
	case "bracers_of_defense":
		return found(true, ac(2, WhileWornWithoutArmorOrShield))
	case "scarab_of_protection":
		return found(true, ac(1, WhileOnPerson))
	case "robe_of_stars":
		return found(true, saves(1, WhileWorn))
	case "dwarven_plate":
		return found(false, ac(2, WhileWorn))
	case "elven_chain", "glamoured_studded_leather":
		return found(false, ac(1, WhileWorn))
	case "staff_of_power":
		return found(true, ac(2, WhileHeld), saves(2, WhileHeld), spell(2, WhileHeld), hit(2, WhileHeld))
	case "staff_of_the_magi", "staff_of_the_woodlands":
		return found(true, spell(2, WhileHeld), hit(2, WhileHeld))
	case "amulet_of_health":
		return foundScore(ability.Constitution, 19)
	case "gauntlets_of_ogre_power":
		return foundScore(ability.Strength, 19, true)
	case "headband_of_intellect":
		return foundScore(ability.Intelligence, 19)
	case "belt_of_giant_strength":
		if s := choice.Giant.StrengthScore(); s > 0 {
			return foundScore(ability.Strength, s, true)
		}
	case "potion_of_giant_strength":
		if s := choice.Giant.StrengthScore(); s > 0 {
			set, _ := NewSetAbilityScore(ability.Strength, s, WhileOnPerson) // s is 21 to 29
			return found(false, set.Lasting(GiantPotionDuration))
		}
	case "belt_of_dwarvenkind":
		raise, _ := NewRaiseAbilityScore(ability.Constitution, 2, 20, WhileWorn) // constants are valid
		return found(true, raise)
	case "hammer_of_thunderbolts":
		boost, _ := NewBoostBestowedScore(4, 30, WhileOnPerson) // constants are valid
		return found(true, boost)
	}
	return MagicItemEffectsNotModelled{id: id}
}

// foundScore is the set-score row of an item worn and attuned to (the potion builds its own); the optional
// flag marks a score a Hammer of Thunderbolts can add to.
func foundScore(a ability.Ability, score int, bestowing ...bool) MagicItemEffectLookup {
	set, _ := NewSetAbilityScore(a, score, WhileWorn) // table scores are 19 to 29
	if len(bestowing) > 0 {
		set = set.ForBestowing()
	}
	return MagicItemEffectsFound{effects: []MagicItemEffect{set}, attunement: true}
}
