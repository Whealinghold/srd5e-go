package character

// SpellcastingEvent is the closed family of events of spell slots and prepared
// spells. It is an Event, so a host switches over it for the family and over
// Event for the context. Events are past tense and immutable and carry the
// Identifier of the character whose SpellcastingState produced them. Members:
// SpellSlotExpended, SpellSlotsRestored, PreparedSpellsChanged.
type SpellcastingEvent interface {
	Event
	isSpellcastingEvent()
}

// SpellSlotExpended is the event of one spell slot being expended.
type SpellSlotExpended struct {
	id    Identifier
	level SlotLevel
	pool  SpellSlotPool
}

// CharacterID returns the id of the character.
func (e SpellSlotExpended) CharacterID() Identifier { return e.id }

// Level returns the level of the slot expended.
func (e SpellSlotExpended) Level() SlotLevel { return e.level }

// Pool returns the pool the slot came from.
func (e SpellSlotExpended) Pool() SpellSlotPool { return e.pool }

// SpellSlotsRestored is the event of expended slots being restored: how many in
// each pool.
type SpellSlotsRestored struct {
	id                  Identifier
	spellcasting, pacts int
}

// CharacterID returns the id of the character.
func (e SpellSlotsRestored) CharacterID() Identifier { return e.id }

// Spellcasting returns the number of Spellcasting-pool slots restored.
func (e SpellSlotsRestored) Spellcasting() int { return e.spellcasting }

// PactMagic returns the number of Pact Magic slots restored.
func (e SpellSlotsRestored) PactMagic() int { return e.pacts }

// PreparedSpellsChanged is the event of a class's prepared spells, cantrips or
// granted cantrips changing.
type PreparedSpellsChanged struct {
	id    Identifier
	class SpellcastingClass
}

// CharacterID returns the id of the character.
func (e PreparedSpellsChanged) CharacterID() Identifier { return e.id }

// Class returns the class whose lists changed.
func (e PreparedSpellsChanged) Class() SpellcastingClass { return e.class }

func (SpellSlotExpended) isEvent()                 {}
func (SpellSlotsRestored) isEvent()                {}
func (PreparedSpellsChanged) isEvent()             {}
func (SpellSlotExpended) isSpellcastingEvent()     {}
func (SpellSlotsRestored) isSpellcastingEvent()    {}
func (PreparedSpellsChanged) isSpellcastingEvent() {}
