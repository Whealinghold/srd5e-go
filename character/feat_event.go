package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// FeatEvent is something that happened to a character's feats, in the past
// tense and immutable. Increases a feat gives are reported with the aggregate's
// AbilityScoreChanged and HitPointMaximumChanged, which are not FeatEvents.
// Members: FeatTaken, FeatSpellReplaced, FeatUseExpended, FeatUsesRenewed,
// FeatReplaced.
type FeatEvent interface {
	Event
	isFeatEvent()
}

// FeatTaken is the event of a character taking a feat.
type FeatTaken struct {
	id       Identifier
	feat     content.FeatID
	category content.FeatCategory
	source   FeatSource
}

// CharacterID returns the id of the character.
func (e FeatTaken) CharacterID() Identifier { return e.id }

// Feat returns the feat taken.
func (e FeatTaken) Feat() content.FeatID { return e.feat }

// Category returns the category of the feat.
func (e FeatTaken) Category() content.FeatCategory { return e.category }

// Source returns where the feat came from.
func (e FeatTaken) Source() FeatSource { return e.source }

// FeatSpellReplaced is the event of a Magic Initiate Spell Change.
type FeatSpellReplaced struct {
	id        Identifier
	list      content.ClassID
	old, repl content.SpellID
}

// CharacterID returns the id of the character.
func (e FeatSpellReplaced) CharacterID() Identifier { return e.id }

// List returns the spell list of the Magic Initiate instance changed.
func (e FeatSpellReplaced) List() content.ClassID { return e.list }

// Old returns the spell replaced.
func (e FeatSpellReplaced) Old() content.SpellID { return e.old }

// New returns the spell now held.
func (e FeatSpellReplaced) New() content.SpellID { return e.repl }

// FeatUseExpended is the event of a feat's limited use being expended; the state
// is FeatUseIsExpended.
type FeatUseExpended struct {
	id   Identifier
	feat content.FeatID
}

// CharacterID returns the id of the character.
func (e FeatUseExpended) CharacterID() Identifier { return e.id }

// Feat returns the feat whose use was expended.
func (e FeatUseExpended) Feat() content.FeatID { return e.feat }

// FeatUsesRenewed is the event of expended feat uses being renewed. It holds a
// slice, so compare two with Equal.
type FeatUsesRenewed struct {
	id      Identifier
	feats   []content.FeatID
	renewal FeatRenewal
}

// CharacterID returns the id of the character.
func (e FeatUsesRenewed) CharacterID() Identifier { return e.id }

// Feats returns a new slice of the feats renewed, in the order held.
func (e FeatUsesRenewed) Feats() []content.FeatID { return slices.Clone(e.feats) }

// Renewal returns the trigger that renewed them.
func (e FeatUsesRenewed) Renewal() FeatRenewal { return e.renewal }

// Equal reports whether e and o record the same renewal.
func (e FeatUsesRenewed) Equal(o FeatUsesRenewed) bool {
	return e.id == o.id && e.renewal == o.renewal && slices.Equal(e.feats, o.feats)
}

// FeatReplaced is the event of a held feat being replaced by another.
type FeatReplaced struct {
	id        Identifier
	old, repl content.FeatID
}

// CharacterID returns the id of the character.
func (e FeatReplaced) CharacterID() Identifier { return e.id }

// Old returns the feat lost.
func (e FeatReplaced) Old() content.FeatID { return e.old }

// New returns the feat gained.
func (e FeatReplaced) New() content.FeatID { return e.repl }

func (FeatTaken) isEvent()         {}
func (FeatSpellReplaced) isEvent() {}
func (FeatUseExpended) isEvent()   {}
func (FeatUsesRenewed) isEvent()   {}
func (FeatReplaced) isEvent()      {}

func (FeatTaken) isFeatEvent()         {}
func (FeatSpellReplaced) isFeatEvent() {}
func (FeatUseExpended) isFeatEvent()   {}
func (FeatUsesRenewed) isFeatEvent()   {}
func (FeatReplaced) isFeatEvent()      {}
