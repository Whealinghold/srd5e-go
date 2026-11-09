package character

import "github.com/Whealinghold/srd5e-go/content"

// ChargedItemEvent is the closed family of events of charged spell items, the
// Ring of Spell Storing and the Staff of the Magi. It is an Event. Members:
// MagicItemLastChargeResolved, RingSpellStored, RingStoredSpellCast,
// StaffSpellAbsorbed. Charge spending reuses MagicItemChargesSpent.
type ChargedItemEvent interface {
	Event
	isChargedItemEvent()
}

// MagicItemLastChargeResolved is the event of the last-charge rule of an emptied
// item being resolved.
type MagicItemLastChargeResolved struct {
	id          Identifier
	instance    MagicItemInstanceID
	consequence LastChargeConsequence
}

// CharacterID returns the holder.
func (e MagicItemLastChargeResolved) CharacterID() Identifier { return e.id }

// InstanceID returns the item copy.
func (e MagicItemLastChargeResolved) InstanceID() MagicItemInstanceID { return e.instance }

// Consequence returns what became of the item.
func (e MagicItemLastChargeResolved) Consequence() LastChargeConsequence { return e.consequence }

// RingSpellStored is the event of a spell being stored in a Ring of Spell Storing.
type RingSpellStored struct {
	id        Identifier
	instance  MagicItemInstanceID
	spell     content.SpellID
	slotLevel content.SpellLevel
}

// CharacterID returns the holder.
func (e RingSpellStored) CharacterID() Identifier { return e.id }

// InstanceID returns the ring.
func (e RingSpellStored) InstanceID() MagicItemInstanceID { return e.instance }

// SpellID returns the spell stored.
func (e RingSpellStored) SpellID() content.SpellID { return e.spell }

// SlotLevel returns the level of the slot the spell was cast with.
func (e RingSpellStored) SlotLevel() content.SpellLevel { return e.slotLevel }

// RingStoredSpellCast is the event of a stored spell being cast from the ring.
type RingStoredSpellCast struct {
	id        Identifier
	instance  MagicItemInstanceID
	spell     content.SpellID
	slotLevel content.SpellLevel
}

// CharacterID returns the wearer.
func (e RingStoredSpellCast) CharacterID() Identifier { return e.id }

// InstanceID returns the ring.
func (e RingStoredSpellCast) InstanceID() MagicItemInstanceID { return e.instance }

// SpellID returns the spell cast.
func (e RingStoredSpellCast) SpellID() content.SpellID { return e.spell }

// SlotLevel returns the original caster's slot level.
func (e RingStoredSpellCast) SlotLevel() content.SpellLevel { return e.slotLevel }

// StaffSpellAbsorbed is the event of a Staff of the Magi absorbing a spell.
type StaffSpellAbsorbed struct {
	id       Identifier
	instance MagicItemInstanceID
	level    int
}

// CharacterID returns the wearer.
func (e StaffSpellAbsorbed) CharacterID() Identifier { return e.id }

// InstanceID returns the staff.
func (e StaffSpellAbsorbed) InstanceID() MagicItemInstanceID { return e.instance }

// Level returns the level of the spell absorbed, the charges gained.
func (e StaffSpellAbsorbed) Level() int { return e.level }

func (MagicItemLastChargeResolved) isEvent()            {}
func (RingSpellStored) isEvent()                        {}
func (RingStoredSpellCast) isEvent()                    {}
func (StaffSpellAbsorbed) isEvent()                     {}
func (MagicItemLastChargeResolved) isChargedItemEvent() {}
func (RingSpellStored) isChargedItemEvent()             {}
func (RingStoredSpellCast) isChargedItemEvent()         {}
func (StaffSpellAbsorbed) isChargedItemEvent()          {}
