package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpeciesTraitEvent is something that happened to a character's species traits, in
// the past tense and immutable. Members: SpeciesUseExpended, SpeciesUsesRenewed,
// HighElfCantripReplaced, HalflingLuckUsed, RelentlessEnduranceUsed,
// StonesEnduranceUsed.
type SpeciesTraitEvent interface {
	Event
	isSpeciesTraitEvent()
}

// SpeciesUseExpended is the event of a limited use being expended.
type SpeciesUseExpended struct {
	id  Identifier
	use SpeciesUse
}

// CharacterID returns the id of the character.
func (e SpeciesUseExpended) CharacterID() Identifier { return e.id }

// Use returns the use expended.
func (e SpeciesUseExpended) Use() SpeciesUse { return e.use }

// SpeciesUsesRenewed is the event of a rest renewing uses and opening windows. It
// holds slices, so compare two with Equal.
type SpeciesUsesRenewed struct {
	id      Identifier
	uses    []SpeciesUse
	windows []SpeciesWindow
}

// CharacterID returns the id of the character.
func (e SpeciesUsesRenewed) CharacterID() Identifier { return e.id }

// Uses returns a new slice of the uses cleared, in the order of SpeciesUse.
func (e SpeciesUsesRenewed) Uses() []SpeciesUse { return slices.Clone(e.uses) }

// Windows returns a new slice of the windows opened.
func (e SpeciesUsesRenewed) Windows() []SpeciesWindow { return slices.Clone(e.windows) }

// Equal reports whether e and o record the same renewal.
func (e SpeciesUsesRenewed) Equal(o SpeciesUsesRenewed) bool {
	return e.id == o.id && slices.Equal(e.uses, o.uses) && slices.Equal(e.windows, o.windows)
}

// HighElfCantripReplaced is the event of a High Elf replacing the lineage cantrip.
type HighElfCantripReplaced struct {
	id        Identifier
	old, repl content.SpellID
}

// CharacterID returns the id of the character.
func (e HighElfCantripReplaced) CharacterID() Identifier { return e.id }

// Old returns the cantrip replaced.
func (e HighElfCantripReplaced) Old() content.SpellID { return e.old }

// New returns the cantrip now held.
func (e HighElfCantripReplaced) New() content.SpellID { return e.repl }

// HalflingLuckUsed is the event of a Halfling rerolling a 1 on a d20.
type HalflingLuckUsed struct {
	id          Identifier
	index, face int
}

// CharacterID returns the id of the character.
func (e HalflingLuckUsed) CharacterID() Identifier { return e.id }

// DieIndex returns the index of the die rerolled.
func (e HalflingLuckUsed) DieIndex() int { return e.index }

// NewValue returns the face the new roll shows.
func (e HalflingLuckUsed) NewValue() int { return e.face }

// RelentlessEnduranceUsed is the event of an Orc dropping to 1 Hit Point instead of 0.
type RelentlessEnduranceUsed struct{ id Identifier }

// CharacterID returns the id of the character.
func (e RelentlessEnduranceUsed) CharacterID() Identifier { return e.id }

// StonesEnduranceUsed is the event of a Goliath reducing damage with Stone's Endurance.
type StonesEnduranceUsed struct {
	id              Identifier
	roll, reduction int
}

// CharacterID returns the id of the character.
func (e StonesEnduranceUsed) CharacterID() Identifier { return e.id }

// Roll returns the 1d12 rolled.
func (e StonesEnduranceUsed) Roll() int { return e.roll }

// Reduction returns the damage taken off, never below 0.
func (e StonesEnduranceUsed) Reduction() int { return e.reduction }

func (SpeciesUseExpended) isEvent()      {}
func (SpeciesUsesRenewed) isEvent()      {}
func (HighElfCantripReplaced) isEvent()  {}
func (HalflingLuckUsed) isEvent()        {}
func (RelentlessEnduranceUsed) isEvent() {}
func (StonesEnduranceUsed) isEvent()     {}

func (SpeciesUseExpended) isSpeciesTraitEvent()      {}
func (SpeciesUsesRenewed) isSpeciesTraitEvent()      {}
func (HighElfCantripReplaced) isSpeciesTraitEvent()  {}
func (HalflingLuckUsed) isSpeciesTraitEvent()        {}
func (RelentlessEnduranceUsed) isSpeciesTraitEvent() {}
func (StonesEnduranceUsed) isSpeciesTraitEvent()     {}
