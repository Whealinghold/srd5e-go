package character

import "github.com/Whealinghold/srd5e-go/content"

// ClericEvent is something that happened to a Cleric, in the past tense and immutable,
// the closed family of Cleric events. Every one is also an Event and carries the
// Identifier it concerns. The interface cannot be implemented outside this package.
// Members: ClericOptionChosen, ClericAdvanced, ClericChannelDivinityUsed,
// ClericChannelDivinityRegained, DivineInterventionUsed, DivineInterventionRecharged.
type ClericEvent interface {
	Event
	isClericEvent()
}

// ClericOptionChosen is the event of a Divine Order or a Blessed Strikes option chosen.
type ClericOptionChosen struct {
	id            Identifier
	feature, name string
}

// CharacterID returns the id of the Character.
func (e ClericOptionChosen) CharacterID() Identifier { return e.id }

// Feature returns "divine_order" or "blessed_strikes".
func (e ClericOptionChosen) Feature() string { return e.feature }

// Option returns the option chosen ("protector", "divine_strike", ...).
func (e ClericOptionChosen) Option() string { return e.name }

// ClericAdvanced is the event of a successful AdvanceCleric: one per advance, with a
// possibly empty payload.
type ClericAdvanced struct {
	id         Identifier
	level      int
	features   []content.FeatureID
	pending    []ClericPendingChoice
	usesGained int
	spells     []content.SpellID
}

// CharacterID returns the id of the Character.
func (e ClericAdvanced) CharacterID() Identifier { return e.id }

// Level returns the new Cleric level.
func (e ClericAdvanced) Level() int { return e.level }

// FeaturesGained returns the ids of the features gained, as a new slice.
func (e ClericAdvanced) FeaturesGained() []content.FeatureID {
	return append([]content.FeatureID{}, e.features...)
}

// PendingChoices returns the Cleric-specific choices the level adds, as a new slice.
func (e ClericAdvanced) PendingChoices() []ClericPendingChoice {
	return append([]ClericPendingChoice{}, e.pending...)
}

// ChannelDivinityUsesGained returns how much the Channel Divinity maximum rose.
func (e ClericAdvanced) ChannelDivinityUsesGained() int { return e.usesGained }

// AlwaysPreparedSpellsGranted returns the always-prepared spells newly granted, as a new slice.
func (e ClericAdvanced) AlwaysPreparedSpellsGranted() []content.SpellID {
	return append([]content.SpellID{}, e.spells...)
}

// ClericChannelDivinityUsed is the event of a Channel Divinity use.
type ClericChannelDivinityUsed struct {
	id        Identifier
	effect    string
	remaining int
}

// CharacterID returns the id of the Character.
func (e ClericChannelDivinityUsed) CharacterID() Identifier { return e.id }

// Effect returns "divine_spark", "turn_undead" or "preserve_life".
func (e ClericChannelDivinityUsed) Effect() string { return e.effect }

// Remaining returns the uses left after the use.
func (e ClericChannelDivinityUsed) Remaining() int { return e.remaining }

// ClericChannelDivinityRegained is the event of Channel Divinity uses regained on a rest.
type ClericChannelDivinityRegained struct {
	id   Identifier
	uses int
}

// CharacterID returns the id of the Character.
func (e ClericChannelDivinityRegained) CharacterID() Identifier { return e.id }

// Uses returns how many uses were regained.
func (e ClericChannelDivinityRegained) Uses() int { return e.uses }

// DivineInterventionUsed is the event of a Divine Intervention use.
type DivineInterventionUsed struct {
	id            Identifier
	spell         content.SpellID
	longRestsToGo int
}

// CharacterID returns the id of the Character.
func (e DivineInterventionUsed) CharacterID() Identifier { return e.id }

// Spell returns the spell chosen.
func (e DivineInterventionUsed) Spell() content.SpellID { return e.spell }

// LongRestsToGo returns the Long Rests before it recharges when a granted spell was
// chosen, or 0 when only one Long Rest is needed.
func (e DivineInterventionUsed) LongRestsToGo() int { return e.longRestsToGo }

// DivineInterventionRecharged is the event of Divine Intervention becoming available again.
type DivineInterventionRecharged struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e DivineInterventionRecharged) CharacterID() Identifier { return e.id }

func (ClericOptionChosen) isEvent()                  {}
func (ClericAdvanced) isEvent()                      {}
func (ClericChannelDivinityUsed) isEvent()           {}
func (ClericChannelDivinityRegained) isEvent()       {}
func (DivineInterventionUsed) isEvent()              {}
func (DivineInterventionRecharged) isEvent()         {}
func (ClericOptionChosen) isClericEvent()            {}
func (ClericAdvanced) isClericEvent()                {}
func (ClericChannelDivinityUsed) isClericEvent()     {}
func (ClericChannelDivinityRegained) isClericEvent() {}
func (DivineInterventionUsed) isClericEvent()        {}
func (DivineInterventionRecharged) isClericEvent()   {}
