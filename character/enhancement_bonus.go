package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// EnhancementFamily is a kind of item whose bonus is "determined by its
// rarity". The zero value is invalid.
type EnhancementFamily int

// The families.
const (
	EnhancementArmor EnhancementFamily = iota + 1
	EnhancementShield
	EnhancementWeapon
	EnhancementAmmunition
	EnhancementWarMageWand
)

// String returns the snake_case name of the family, or "unknown".
func (f EnhancementFamily) String() string {
	names := []string{"armor", "shield", "weapon", "ammunition", "war_mage_wand"}
	if f < 1 || int(f) > len(names) {
		return "unknown"
	}
	return names[f-1]
}

// EnhancementBonusResult is the closed set of outcomes of
// EnhancementBonusLookup.Of. Variants: EnhancementBonusFound,
// EnhancementBonusNotListed.
type EnhancementBonusResult interface{ isEnhancementBonusResult() }

// EnhancementBonusFound: the bonus of the family at the rarity.
type EnhancementBonusFound struct{ bonus modifier.Modifier }

// Modifier returns the bonus, +1 to +3.
func (r EnhancementBonusFound) Modifier() modifier.Modifier { return r.bonus }

// EnhancementBonusNotListed: the item header does not list the rarity for the
// family (or the family or rarity is invalid); nothing is guessed.
type EnhancementBonusNotListed struct {
	family EnhancementFamily
	rarity content.ItemRarity
}

// Family returns the family asked for.
func (r EnhancementBonusNotListed) Family() EnhancementFamily { return r.family }

// Rarity returns the rarity asked for.
func (r EnhancementBonusNotListed) Rarity() content.ItemRarity { return r.rarity }

func (EnhancementBonusFound) isEnhancementBonusResult()     {}
func (EnhancementBonusNotListed) isEnhancementBonusResult() {}

// EnhancementBonusLookup holds table "Enhancement bonuses" (SRD 5.2.1, "Armor,
// +1, +2, or +3", p. 210; "Shield", p. 243; "Weapon", p. 253; "Ammunition",
// p. 209; "Wand of the War Mage", p. 251). The zero value is ready to use.
type EnhancementBonusLookup struct{}

// Of returns the bonus of family at rarity: armor Rare +1, Very Rare +2,
// Legendary +3; the other families Uncommon +1, Rare +2, Very Rare +3. Any other
// pair is EnhancementBonusNotListed; it is never an error.
func (EnhancementBonusLookup) Of(family EnhancementFamily, rarity content.ItemRarity) EnhancementBonusResult {
	first := content.RarityUncommon
	switch family {
	case EnhancementArmor:
		first = content.RarityRare
	case EnhancementShield, EnhancementWeapon, EnhancementAmmunition, EnhancementWarMageWand:
	default:
		return EnhancementBonusNotListed{family: family, rarity: rarity}
	}
	if rarity < first || rarity > first+2 {
		return EnhancementBonusNotListed{family: family, rarity: rarity}
	}
	return EnhancementBonusFound{bonus: modifier.New(int(rarity-first) + 1)}
}

// AmmunitionState is whether a piece of ammunition is still magical. The zero
// value is invalid.
type AmmunitionState int

// The states.
const (
	MagicalAmmunition AmmunitionState = iota + 1
	NonmagicalAmmunition
)

// MagicAmmunition holds the rule that magic ammunition stops being magical once
// it hits (SRD 5.2.1, "Ammunition, +1, +2, or +3", p. 209). The zero value is
// ready to use.
type MagicAmmunition struct{}

// AfterAttack returns NonmagicalAmmunition when hit is true and the state is
// MagicalAmmunition, and state otherwise.
func (MagicAmmunition) AfterAttack(state AmmunitionState, hit bool) AmmunitionState {
	if hit && state == MagicalAmmunition {
		return NonmagicalAmmunition
	}
	return state
}
