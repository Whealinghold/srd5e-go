package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ItemUseKind is an item whose use item-use-effects.md models (SRD 5.2.1, "Adventuring Gear",
// pp. 94-99). The zero value is invalid. Safe for concurrent use.
type ItemUseKind int

// The nine kinds; the first five are the thrown kinds and equal the ThrownItemKind values.
const (
	ItemUseAcid ItemUseKind = iota + 1
	ItemUseAlchemistsFire
	ItemUseHolyWater
	ItemUseOil
	ItemUseNet
	ItemUseBasicPoison
	ItemUseAntitoxin
	ItemUsePotionOfHealing
	ItemUseHealersKit
)

func itemUseNames() []string {
	return []string{"acid", "alchemists_fire", "holy_water", "oil", "net", "poison_basic", "antitoxin",
		"potion_of_healing", "healers_kit"}
}

// Valid reports whether k is one of the nine kinds.
func (k ItemUseKind) Valid() bool { return k >= ItemUseAcid && k <= ItemUseHealersKit }

// String returns the content id of the item ("acid", "alchemists_fire", "holy_water", "oil",
// "net", "poison_basic", "antitoxin", "potion_of_healing", "healers_kit") or "unknown".
func (k ItemUseKind) String() string { return enumName(itemUseNames(), int(k)) }

// ItemID returns the content id of the equipment entry; the zero ItemID for an invalid kind.
func (k ItemUseKind) ItemID() content.ItemID {
	if !k.Valid() {
		return ""
	}
	return content.MustItemID(k.String())
}

// ItemActionCost is what using an item costs, reported as data: the host spends it (SRD 5.2.1,
// pp. 94-99). The zero value is invalid.
type ItemActionCost int

// The three costs.
const (
	// ItemActionCostReplacesOneAttack: the item replaces one attack of the Attack action.
	ItemActionCostReplacesOneAttack ItemActionCost = iota + 1
	// ItemActionCostBonusAction: "As a Bonus Action".
	ItemActionCostBonusAction
	// ItemActionCostUtilizeAction: "As a Utilize action".
	ItemActionCostUtilizeAction
)

// Valid reports whether c is one of the three costs.
func (c ItemActionCost) Valid() bool {
	return c >= ItemActionCostReplacesOneAttack && c <= ItemActionCostUtilizeAction
}

// String returns "replaces_one_attack", "bonus_action", "utilize_action" or "unknown".
func (c ItemActionCost) String() string {
	return enumName([]string{"replaces_one_attack", "bonus_action", "utilize_action"}, int(c))
}

// ItemUseRules is the stateless cost table of the items. The zero value is ready to use.
// Pure; safe for concurrent use.
type ItemUseRules struct{}

// CostOf returns the cost of using kind: ItemActionCostReplacesOneAttack for the five thrown
// kinds, ItemActionCostBonusAction for Basic Poison, Antitoxin and the Potion of Healing, and
// ItemActionCostUtilizeAction for the Healer's Kit. It returns the zero cost for an invalid
// kind. It never panics.
func (ItemUseRules) CostOf(kind ItemUseKind) ItemActionCost {
	switch kind {
	case ItemUseAcid, ItemUseAlchemistsFire, ItemUseHolyWater, ItemUseOil, ItemUseNet:
		return ItemActionCostReplacesOneAttack
	case ItemUseBasicPoison, ItemUseAntitoxin, ItemUsePotionOfHealing:
		return ItemActionCostBonusAction
	case ItemUseHealersKit:
		return ItemActionCostUtilizeAction
	}
	return 0
}

// ItemConsumption says whether using an item used it up. Variants: ItemConsumed,
// ItemNotConsumed. The engine holds no stock: the host decrements.
type ItemConsumption interface{ isItemConsumption() }

// ItemConsumed: Count of the item with id Item were used up.
type ItemConsumed struct {
	item  content.ItemID
	count int
}

// Item returns the content id of the item used up.
func (c ItemConsumed) Item() content.ItemID { return c.item }

// Count returns how many were used up.
func (c ItemConsumed) Count() int { return c.count }

// ItemNotConsumed: nothing was used up (a refusal, or a Net, which stays an object).
type ItemNotConsumed struct{}

func (ItemConsumed) isItemConsumption()    {}
func (ItemNotConsumed) isItemConsumption() {}

// consumedOne is one of kind used up.
func consumedOne(kind ItemUseKind) ItemConsumed { return ItemConsumed{kind.ItemID(), 1} }

// ItemSaveDCOf returns the Difficulty Class of the saving throw a thrown item forces: 8 plus
// the thrower's Dexterity modifier plus Proficiency Bonus, with no cap or floor (SRD 5.2.1,
// "Acid", p. 94). The Proficiency Bonus is always added (Q-3); the sum saturates at the limits
// of int.
func ItemSaveDCOf(dexterity ability.Modifier, proficiencyBonus character.ProficiencyBonus) d20.DifficultyClass {
	return NetRules{}.SaveDC(dexterity.Value(), proficiencyBonus.Value())
}

// itemDice is the damage expression of count dice of die.
func itemDice(count int, die dice.Die) dice.Expression { return dice.MustExpression(count, die, 0) }
