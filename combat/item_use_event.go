package combat

// ItemUseEvent is the closed family of events the item-use rules produce. Each is an
// immutable value comparable with ==, with a constant snake_case Kind, past tense. Combatants
// is a fresh copy of the creatures concerned, the user first, without duplicates. Variants:
// ThrownItemUsed, BasicPoisonApplied, BasicPoisonSpent, AntitoxinDrunk, AntitoxinWoreOff,
// PotionOfHealingConsumed, HealersKitUseSpent. Each embeds itemUseEventBase.
type ItemUseEvent interface {
	Event
	itemUseEvent()
}

// itemUseEventBase carries the creatures an event is about, the first being the user and the
// second, when not zero and not the first, the other creature concerned, and puts the event
// in the Event and ItemUseEvent families.
type itemUseEventBase struct{ first, second CombatantID }

func (itemUseEventBase) event()        {}
func (itemUseEventBase) itemUseEvent() {}

// Combatants returns a new slice: the first creature and, if it is another one, the second.
func (b itemUseEventBase) Combatants() []CombatantID {
	if b.second.IsZero() || b.second == b.first {
		return []CombatantID{b.first}
	}
	return []CombatantID{b.first, b.second}
}

// ThrownItemUsed: an item was thrown. Its Kind is thrown_item_used. Combatants is [thrower]
// or, at a creature, [thrower, target].
type ThrownItemUsed struct {
	itemUseEventBase
	item ThrownItemKind
}

// Kind returns "thrown_item_used".
func (ThrownItemUsed) Kind() string { return "thrown_item_used" }

// Item returns the item thrown.
func (e ThrownItemUsed) Item() ThrownItemKind { return e.item }

// BasicPoisonApplied: Basic Poison was applied to a weapon or ammunition. Its Kind is
// basic_poison_applied. Combatants is [owner].
type BasicPoisonApplied struct{ itemUseEventBase }

// Kind returns "basic_poison_applied".
func (BasicPoisonApplied) Kind() string { return "basic_poison_applied" }

// BasicPoisonSpent: a Basic Poison coating was spent. Its Kind is basic_poison_spent.
// Combatants is [owner].
type BasicPoisonSpent struct {
	itemUseEventBase
	cause BasicPoisonState
}

// Kind returns "basic_poison_spent".
func (BasicPoisonSpent) Kind() string { return "basic_poison_spent" }

// Cause returns BasicPoisonDelivered or BasicPoisonExpired.
func (e BasicPoisonSpent) Cause() BasicPoisonState { return e.cause }

// AntitoxinDrunk: a creature drank Antitoxin. Its Kind is antitoxin_drunk. Combatants is
// [drinker].
type AntitoxinDrunk struct{ itemUseEventBase }

// Kind returns "antitoxin_drunk".
func (AntitoxinDrunk) Kind() string { return "antitoxin_drunk" }

// AntitoxinWoreOff: the hour of Antitoxin ended. Its Kind is antitoxin_wore_off. Combatants
// is [drinker].
type AntitoxinWoreOff struct{ itemUseEventBase }

// Kind returns "antitoxin_wore_off".
func (AntitoxinWoreOff) Kind() string { return "antitoxin_wore_off" }

// PotionOfHealingConsumed: a Potion of Healing was drunk or administered. Its Kind is
// potion_of_healing_used. Combatants is [user, patient].
type PotionOfHealingConsumed struct{ itemUseEventBase }

// Kind returns "potion_of_healing_used".
func (PotionOfHealingConsumed) Kind() string { return "potion_of_healing_used" }

// HealersKitUseSpent: a use of a Healer's Kit was expended to stabilize a creature. Its Kind
// is healers_kit_used. Combatants is [user, patient].
type HealersKitUseSpent struct{ itemUseEventBase }

// Kind returns "healers_kit_used".
func (HealersKitUseSpent) Kind() string { return "healers_kit_used" }
