package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// PaladinEvent is something that happened to a Paladin, in the past tense and immutable, the closed family of Paladin events.
// Every one is also an Event and carries the Identifier it concerns. The interface cannot be implemented outside this package.
// Members: LayOnHandsUsed, PaladinChannelDivinityUsed, FreeCastSpent, HolyNimbusActivated, HolyNimbusRestored,
// SmiteOfProtectionTriggered, PaladinEffectEnded, PaladinChoiceMade, PaladinLevelOneGained, PaladinAdvanced,
// PaladinResourcesRestored.
type PaladinEvent interface {
	Event
	isPaladinEvent()
}

// LayOnHandsUsed is the event of a Lay On Hands use.
type LayOnHandsUsed struct {
	id         Identifier
	hitPoints  int
	conditions []content.ConditionKind
	remaining  int
}

// CharacterID returns the id of the Character.
func (e LayOnHandsUsed) CharacterID() Identifier { return e.id }

// HitPoints returns the Hit Points restored.
func (e LayOnHandsUsed) HitPoints() int { return e.hitPoints }

// Conditions returns the conditions removed, as a new slice.
func (e LayOnHandsUsed) Conditions() []content.ConditionKind { return slices.Clone(e.conditions) }

// Remaining returns the Hit Points left in the pool.
func (e LayOnHandsUsed) Remaining() int { return e.remaining }

// PaladinChannelDivinityUsed is the event of a Channel Divinity use.
type PaladinChannelDivinityUsed struct {
	id        Identifier
	option    string
	remaining int
}

// CharacterID returns the id of the Character.
func (e PaladinChannelDivinityUsed) CharacterID() Identifier { return e.id }

// Option returns "divine_sense", "abjure_foes" or "sacred_weapon".
func (e PaladinChannelDivinityUsed) Option() string { return e.option }

// Remaining returns the uses left after the use.
func (e PaladinChannelDivinityUsed) Remaining() int { return e.remaining }

// FreeCastSpent is the event of a free cast spent; the host then casts the spell with CastModeFeatureGranted.
type FreeCastSpent struct {
	id      Identifier
	feature FreeCastFeature
	spell   content.SpellID
}

// CharacterID returns the id of the Character.
func (e FreeCastSpent) CharacterID() Identifier { return e.id }

// Feature returns the free cast spent.
func (e FreeCastSpent) Feature() FreeCastFeature { return e.feature }

// Spell returns the spell to cast.
func (e FreeCastSpent) Spell() content.SpellID { return e.spell }

// HolyNimbusActivated is the event of Holy Nimbus activated.
type HolyNimbusActivated struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e HolyNimbusActivated) CharacterID() Identifier { return e.id }

// HolyNimbusRestored is the event of the Holy Nimbus use restored by expending a level 5 spell slot.
type HolyNimbusRestored struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e HolyNimbusRestored) CharacterID() Identifier { return e.id }

// SmiteOfProtectionTriggered is the event of Smite of Protection triggered by a cast of Divine Smite.
type SmiteOfProtectionTriggered struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e SmiteOfProtectionTriggered) CharacterID() Identifier { return e.id }

// PaladinEffectEnded is the event of an active effect ending.
type PaladinEffectEnded struct {
	id     Identifier
	effect PaladinEffectKind
	reason PaladinEndReason
}

// CharacterID returns the id of the Character.
func (e PaladinEffectEnded) CharacterID() Identifier { return e.id }

// Effect returns the effect that ended.
func (e PaladinEffectEnded) Effect() PaladinEffectKind { return e.effect }

// Reason returns why it ended.
func (e PaladinEffectEnded) Reason() PaladinEndReason { return e.reason }

// PaladinChoiceMade is the event of a Fighting Style or Blessed Warrior choice.
type PaladinChoiceMade struct {
	id     Identifier
	choice string
}

// CharacterID returns the id of the Character.
func (e PaladinChoiceMade) CharacterID() Identifier { return e.id }

// Choice returns "fighting_style_feat", "blessed_warrior" or "blessed_warrior_cantrip_replaced".
func (e PaladinChoiceMade) Choice() string { return e.choice }

// PaladinLevelOneGained is the event of Paladin level 1 gained by multiclassing.
type PaladinLevelOneGained struct {
	id    Identifier
	entry ClassEntry
}

// CharacterID returns the id of the Character.
func (e PaladinLevelOneGained) CharacterID() Identifier { return e.id }

// Entry returns how the class was entered.
func (e PaladinLevelOneGained) Entry() ClassEntry { return e.entry }

// PaladinAdvanced is the event of a successful AdvancePaladin: one per advance, with a possibly empty payload.
type PaladinAdvanced struct {
	id         Identifier
	level      int
	features   []content.FeatureID
	pending    []PaladinPendingChoice
	usesGained int
	spells     []content.SpellID
}

// CharacterID returns the id of the Character.
func (e PaladinAdvanced) CharacterID() Identifier { return e.id }

// Level returns the new Paladin level.
func (e PaladinAdvanced) Level() int { return e.level }

// FeaturesGained returns the ids of the features gained, as a new slice.
func (e PaladinAdvanced) FeaturesGained() []content.FeatureID { return slices.Clone(e.features) }

// PendingChoices returns the Paladin-specific choices the level adds, as a new slice.
func (e PaladinAdvanced) PendingChoices() []PaladinPendingChoice { return slices.Clone(e.pending) }

// ChannelDivinityUsesGained returns how much the Channel Divinity maximum rose.
func (e PaladinAdvanced) ChannelDivinityUsesGained() int { return e.usesGained }

// AlwaysPreparedSpellsGranted returns the always-prepared spells newly granted, as a new slice.
func (e PaladinAdvanced) AlwaysPreparedSpellsGranted() []content.SpellID {
	return slices.Clone(e.spells)
}

// PaladinResourcesRestored is the event of a rest restoring Paladin resources.
type PaladinResourcesRestored struct {
	id          Identifier
	layOnHands  int
	channelUses int
	freeCasts   []FreeCastFeature
	holyNimbus  bool
}

// CharacterID returns the id of the Character.
func (e PaladinResourcesRestored) CharacterID() Identifier { return e.id }

// LayOnHands returns the Hit Points restored to the pool.
func (e PaladinResourcesRestored) LayOnHands() int { return e.layOnHands }

// ChannelDivinityUses returns the Channel Divinity uses regained.
func (e PaladinResourcesRestored) ChannelDivinityUses() int { return e.channelUses }

// FreeCasts returns the free casts made available again, as a new slice.
func (e PaladinResourcesRestored) FreeCasts() []FreeCastFeature { return slices.Clone(e.freeCasts) }

// HolyNimbus reports whether the Holy Nimbus use was made available again.
func (e PaladinResourcesRestored) HolyNimbus() bool { return e.holyNimbus }

func (LayOnHandsUsed) isEvent()                    {}
func (PaladinChannelDivinityUsed) isEvent()        {}
func (FreeCastSpent) isEvent()                     {}
func (HolyNimbusActivated) isEvent()               {}
func (HolyNimbusRestored) isEvent()                {}
func (SmiteOfProtectionTriggered) isEvent()        {}
func (PaladinEffectEnded) isEvent()                {}
func (PaladinChoiceMade) isEvent()                 {}
func (PaladinLevelOneGained) isEvent()             {}
func (PaladinAdvanced) isEvent()                   {}
func (PaladinResourcesRestored) isEvent()          {}
func (LayOnHandsUsed) isPaladinEvent()             {}
func (PaladinChannelDivinityUsed) isPaladinEvent() {}
func (FreeCastSpent) isPaladinEvent()              {}
func (HolyNimbusActivated) isPaladinEvent()        {}
func (HolyNimbusRestored) isPaladinEvent()         {}
func (SmiteOfProtectionTriggered) isPaladinEvent() {}
func (PaladinEffectEnded) isPaladinEvent()         {}
func (PaladinChoiceMade) isPaladinEvent()          {}
func (PaladinLevelOneGained) isPaladinEvent()      {}
func (PaladinAdvanced) isPaladinEvent()            {}
func (PaladinResourcesRestored) isPaladinEvent()   {}
