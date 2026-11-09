package character

import "github.com/Whealinghold/srd5e-go/content"

// WishEvent is something that happened to a character through the Wish spell, in the past tense
// and immutable. It is an Event. Members: WishDuplicated, WishStressEndured, WishWeaknessEnded,
// WishCastDamageEnded, WishStressDamageRolled. The events of Sudden Learning's feat replacement are
// FeatEvents.
type WishEvent interface {
	Event
	isWishEvent()
}

// WishDuplicated is the event of a Wish that duplicates a spell.
type WishDuplicated struct {
	id    Identifier
	spell content.SpellID
	level content.SpellLevel
}

// CharacterID returns the id of the character.
func (e WishDuplicated) CharacterID() Identifier { return e.id }

// Spell returns the spell duplicated.
func (e WishDuplicated) Spell() content.SpellID { return e.spell }

// Level returns the level of the spell duplicated, 0 to 8.
func (e WishDuplicated) Level() content.SpellLevel { return e.level }

// WishStressEndured is the event of the stress of a Wish other than a duplication: the days of the
// Strength 3 effect, the percentile and whether it locked the creature out of Wish for good.
type WishStressEndured struct {
	id         Identifier
	days       int
	percentile int
	lockedOut  bool
}

// CharacterID returns the id of the character.
func (e WishStressEndured) CharacterID() Identifier { return e.id }

// Days returns the 2d4 days of Strength 3, 2 to 8.
func (e WishStressEndured) Days() int { return e.days }

// Percentile returns the 1d100, 1 to 100.
func (e WishStressEndured) Percentile() int { return e.percentile }

// LockedOut reports whether the creature can never cast Wish again.
func (e WishStressEndured) LockedOut() bool { return e.lockedOut }

// WishWeaknessEnded is the event of the Strength 3 effect running out.
type WishWeaknessEnded struct{ id Identifier }

// CharacterID returns the id of the character.
func (e WishWeaknessEnded) CharacterID() Identifier { return e.id }

// WishCastDamageEnded is the event of a Long Rest ending the Necrotic damage of later casts.
type WishCastDamageEnded struct{ id Identifier }

// CharacterID returns the id of the character.
func (e WishCastDamageEnded) CharacterID() Identifier { return e.id }

// WishStressDamageRolled is the event of the Necrotic damage of a cast made under the Wish stress:
// the level of the spell and the damage rolled.
type WishStressDamageRolled struct {
	id     Identifier
	level  content.SpellLevel
	amount int
}

// CharacterID returns the id of the character.
func (e WishStressDamageRolled) CharacterID() Identifier { return e.id }

// Level returns the level of the cast, 1 to 9.
func (e WishStressDamageRolled) Level() content.SpellLevel { return e.level }

// Amount returns the Necrotic damage rolled.
func (e WishStressDamageRolled) Amount() int { return e.amount }

func (WishDuplicated) isEvent()         {}
func (WishStressEndured) isEvent()      {}
func (WishWeaknessEnded) isEvent()      {}
func (WishCastDamageEnded) isEvent()    {}
func (WishStressDamageRolled) isEvent() {}

func (WishDuplicated) isWishEvent()         {}
func (WishStressEndured) isWishEvent()      {}
func (WishWeaknessEnded) isWishEvent()      {}
func (WishCastDamageEnded) isWishEvent()    {}
func (WishStressDamageRolled) isWishEvent() {}
