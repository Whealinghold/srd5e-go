package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SorcererEvent is something that happened to a Sorcerer, in the past tense and immutable, the closed family of Sorcerer
// events. Every one is also an Event and carries the Identifier of the Sorcerer. The interface cannot be implemented
// outside this package. Members: InnateSorceryActivated, InnateSorceryEnded, SpellSlotConvertedToSorceryPoints,
// SpellSlotCreated, CreatedSlotSpent, CreatedSlotsExpired, MetamagicUsed, SorceryPointsRestored, MetamagicOptionsChosen,
// ElementalAffinityChosen, ElementalAffinityApplied, DragonWingsActivated, DragonWingsEnded, DragonWingsUseRestored,
// DragonCompanionCast.
type SorcererEvent interface {
	Event
	isSorcererEvent()
}

// InnateSorceryActivated is the event of Innate Sorcery becoming active.
type InnateSorceryActivated struct {
	id         Identifier
	paidPoints bool
}

// CharacterID returns the id of the Sorcerer.
func (e InnateSorceryActivated) CharacterID() Identifier { return e.id }

// PaidWithPoints reports whether the activation cost 2 Sorcery Points (Sorcery Incarnate) rather than a use.
func (e InnateSorceryActivated) PaidWithPoints() bool { return e.paidPoints }

// InnateSorceryEnded is the event of Innate Sorcery ending.
type InnateSorceryEnded struct{ id Identifier }

// CharacterID returns the id of the Sorcerer.
func (e InnateSorceryEnded) CharacterID() Identifier { return e.id }

// SpellSlotConvertedToSorceryPoints is the event of a spell slot expended for Sorcery Points.
type SpellSlotConvertedToSorceryPoints struct {
	id      Identifier
	level   int
	gained  int
	created bool
}

// CharacterID returns the id of the Sorcerer.
func (e SpellSlotConvertedToSorceryPoints) CharacterID() Identifier { return e.id }

// SlotLevel returns the level of the slot, which is also the number of points gained.
func (e SpellSlotConvertedToSorceryPoints) SlotLevel() int { return e.level }

// PointsGained returns the Sorcery Points gained.
func (e SpellSlotConvertedToSorceryPoints) PointsGained() int { return e.gained }

// WasCreatedSlot reports whether the slot was a created slot rather than an ordinary one.
func (e SpellSlotConvertedToSorceryPoints) WasCreatedSlot() bool { return e.created }

// SpellSlotCreated is the event of a spell slot created from Sorcery Points.
type SpellSlotCreated struct {
	id          Identifier
	level, cost int
}

// CharacterID returns the id of the Sorcerer.
func (e SpellSlotCreated) CharacterID() Identifier { return e.id }

// SlotLevel returns the level of the slot created.
func (e SpellSlotCreated) SlotLevel() int { return e.level }

// Cost returns the Sorcery Points spent.
func (e SpellSlotCreated) Cost() int { return e.cost }

// CreatedSlotSpent is the event of a created spell slot spent on a spell.
type CreatedSlotSpent struct {
	id                    Identifier
	slotLevel, spellLevel int
}

// CharacterID returns the id of the Sorcerer.
func (e CreatedSlotSpent) CharacterID() Identifier { return e.id }

// SlotLevel returns the level of the created slot, which is the level the spell is cast at.
func (e CreatedSlotSpent) SlotLevel() int { return e.slotLevel }

// SpellLevel returns the level of the spell cast.
func (e CreatedSlotSpent) SpellLevel() int { return e.spellLevel }

// CreatedSlotsExpired is the event of created spell slots vanishing at a Long Rest.
type CreatedSlotsExpired struct {
	id    Identifier
	count int
}

// CharacterID returns the id of the Sorcerer.
func (e CreatedSlotsExpired) CharacterID() Identifier { return e.id }

// Count returns how many created slots vanished.
func (e CreatedSlotsExpired) Count() int { return e.count }

// MetamagicUsed is the event of a Metamagic option used on a casting.
type MetamagicUsed struct {
	id     Identifier
	effect MetamagicEffect
	cost   int
	free   bool
}

// CharacterID returns the id of the Sorcerer.
func (e MetamagicUsed) CharacterID() Identifier { return e.id }

// Effect returns the option used.
func (e MetamagicUsed) Effect() MetamagicEffect { return e.effect }

// Cost returns the Sorcery Points spent, 0 for the free use of Arcane Apotheosis.
func (e MetamagicUsed) Cost() int { return e.cost }

// Free reports whether the use was the free use of Arcane Apotheosis.
func (e MetamagicUsed) Free() bool { return e.free }

// SorceryPointsRestored is the event of Sorcerous Restoration regaining Sorcery Points.
type SorceryPointsRestored struct {
	id                Identifier
	amount, remaining int
}

// CharacterID returns the id of the Sorcerer.
func (e SorceryPointsRestored) CharacterID() Identifier { return e.id }

// Amount returns the points regained.
func (e SorceryPointsRestored) Amount() int { return e.amount }

// Total returns the Sorcery Points on hand after the restoration.
func (e SorceryPointsRestored) Total() int { return e.remaining }

// MetamagicOptionsChosen is the event of the Metamagic options being chosen or replaced.
type MetamagicOptionsChosen struct {
	id      Identifier
	options []MetamagicEffect
}

// CharacterID returns the id of the Sorcerer.
func (e MetamagicOptionsChosen) CharacterID() Identifier { return e.id }

// Options returns the options known after the choice, in the printed order, as a new slice.
func (e MetamagicOptionsChosen) Options() []MetamagicEffect { return slices.Clone(e.options) }

// ElementalAffinityChosen is the event of the Elemental Affinity damage type being chosen.
type ElementalAffinityChosen struct {
	id         Identifier
	damageType content.DamageType
}

// CharacterID returns the id of the Sorcerer.
func (e ElementalAffinityChosen) CharacterID() Identifier { return e.id }

// DamageType returns the type chosen.
func (e ElementalAffinityChosen) DamageType() content.DamageType { return e.damageType }

// ElementalAffinityApplied is the event of Elemental Affinity adding the Charisma modifier to a damage roll.
type ElementalAffinityApplied struct {
	id         Identifier
	damageType content.DamageType
	bonus      int
}

// CharacterID returns the id of the Sorcerer.
func (e ElementalAffinityApplied) CharacterID() Identifier { return e.id }

// DamageType returns the type of the damage roll the bonus goes to.
func (e ElementalAffinityApplied) DamageType() content.DamageType { return e.damageType }

// Bonus returns the Charisma modifier to add to one damage roll of the spell.
func (e ElementalAffinityApplied) Bonus() int { return e.bonus }

// DragonWingsActivated is the event of the wings appearing.
type DragonWingsActivated struct{ id Identifier }

// CharacterID returns the id of the Sorcerer.
func (e DragonWingsActivated) CharacterID() Identifier { return e.id }

// DragonWingsEnded is the event of the wings being dismissed.
type DragonWingsEnded struct{ id Identifier }

// CharacterID returns the id of the Sorcerer.
func (e DragonWingsEnded) CharacterID() Identifier { return e.id }

// DragonWingsUseRestored is the event of the use of Dragon Wings restored for 3 Sorcery Points.
type DragonWingsUseRestored struct{ id Identifier }

// CharacterID returns the id of the Sorcerer.
func (e DragonWingsUseRestored) CharacterID() Identifier { return e.id }

// DragonCompanionCast is the event of Summon Dragon cast without a spell slot through Dragon Companion.
type DragonCompanionCast struct {
	id                  Identifier
	waivedConcentration bool
}

// CharacterID returns the id of the Sorcerer.
func (e DragonCompanionCast) CharacterID() Identifier { return e.id }

// WaivedConcentration reports whether the casting does not require Concentration, its duration then 1 minute.
func (e DragonCompanionCast) WaivedConcentration() bool { return e.waivedConcentration }

func (InnateSorceryActivated) isEvent()                    {}
func (InnateSorceryEnded) isEvent()                        {}
func (SpellSlotConvertedToSorceryPoints) isEvent()         {}
func (SpellSlotCreated) isEvent()                          {}
func (CreatedSlotSpent) isEvent()                          {}
func (CreatedSlotsExpired) isEvent()                       {}
func (MetamagicUsed) isEvent()                             {}
func (SorceryPointsRestored) isEvent()                     {}
func (MetamagicOptionsChosen) isEvent()                    {}
func (ElementalAffinityChosen) isEvent()                   {}
func (ElementalAffinityApplied) isEvent()                  {}
func (DragonWingsActivated) isEvent()                      {}
func (DragonWingsEnded) isEvent()                          {}
func (DragonWingsUseRestored) isEvent()                    {}
func (DragonCompanionCast) isEvent()                       {}
func (InnateSorceryActivated) isSorcererEvent()            {}
func (InnateSorceryEnded) isSorcererEvent()                {}
func (SpellSlotConvertedToSorceryPoints) isSorcererEvent() {}
func (SpellSlotCreated) isSorcererEvent()                  {}
func (CreatedSlotSpent) isSorcererEvent()                  {}
func (CreatedSlotsExpired) isSorcererEvent()               {}
func (MetamagicUsed) isSorcererEvent()                     {}
func (SorceryPointsRestored) isSorcererEvent()             {}
func (MetamagicOptionsChosen) isSorcererEvent()            {}
func (ElementalAffinityChosen) isSorcererEvent()           {}
func (ElementalAffinityApplied) isSorcererEvent()          {}
func (DragonWingsActivated) isSorcererEvent()              {}
func (DragonWingsEnded) isSorcererEvent()                  {}
func (DragonWingsUseRestored) isSorcererEvent()            {}
func (DragonCompanionCast) isSorcererEvent()               {}
