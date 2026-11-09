package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// WarlockEvent is something that happened to a Warlock, in the past tense and immutable, the closed family of Warlock events. Every
// one is also an Event and carries the Identifier of the Warlock. The interface cannot be implemented outside this package. Members:
// InvocationsChosen, MysticArcanumChosen, MagicalCunningUsed, ContactPatronUsed, MysticArcanumCast, InvocationSpellCastWithoutSlot,
// DarkOnesBlessingGained, FiendishResilienceChosen, DarkOnesOwnLuckUsed, HurlThroughHellUsed, HurlThroughHellRestored,
// EldritchSmiteUsed, WarlockFeaturesRestored, PactWeaponBonded, PactWeaponBondEnded, FamiliarFormChosen, FamiliarSpeedChosen, FamiliarAttackForgone,
// FamiliarAttackCommanded, FamiliarResistanceGranted, BookOfShadowsConjured, BookOfShadowsLost, GiftOfTheProtectorsNameWritten,
// GiftOfTheProtectorsNameErased, GiftOfTheProtectorsTriggered, LifedrinkerUsed, GazeLinkBegun, GazeLinkMaintained.
type WarlockEvent interface {
	Event
	isWarlockEvent()
}

// InvocationsChosen is the event of the Eldritch Invocations being chosen or replaced.
type InvocationsChosen struct {
	id    Identifier
	picks []InvocationPick
}

// CharacterID returns the id of the Warlock.
func (e InvocationsChosen) CharacterID() Identifier { return e.id }

// Picks returns the invocations held after the choice as a new slice.
func (e InvocationsChosen) Picks() []InvocationPick { return slices.Clone(e.picks) }

// MysticArcanumChosen is the event of an arcanum spell being chosen or replaced.
type MysticArcanumChosen struct {
	id         Identifier
	spellLevel int
	spell      content.SpellID
	replaced   bool
}

// CharacterID returns the id of the Warlock.
func (e MysticArcanumChosen) CharacterID() Identifier { return e.id }

// SpellLevel returns the level of the arcanum spell, 6 to 9.
func (e MysticArcanumChosen) SpellLevel() int { return e.spellLevel }

// Spell returns the arcanum spell chosen.
func (e MysticArcanumChosen) Spell() content.SpellID { return e.spell }

// Replaced reports whether the spell replaced an earlier one of the same level.
func (e MysticArcanumChosen) Replaced() bool { return e.replaced }

// MagicalCunningUsed is the event of the rite of Magical Cunning completed.
type MagicalCunningUsed struct {
	id      Identifier
	regains int
}

// CharacterID returns the id of the Warlock.
func (e MagicalCunningUsed) CharacterID() Identifier { return e.id }

// SlotsRegained returns the Pact Magic slots regained, 0 when none was expended.
func (e MagicalCunningUsed) SlotsRegained() int { return e.regains }

// ContactPatronUsed is the event of Contact Other Plane cast through Contact Patron.
type ContactPatronUsed struct{ id Identifier }

// CharacterID returns the id of the Warlock.
func (e ContactPatronUsed) CharacterID() Identifier { return e.id }

// MysticArcanumCast is the event of an arcanum spell cast without a spell slot.
type MysticArcanumCast struct {
	id         Identifier
	spellLevel int
	spell      content.SpellID
}

// CharacterID returns the id of the Warlock.
func (e MysticArcanumCast) CharacterID() Identifier { return e.id }

// SpellLevel returns the level of the arcanum spell.
func (e MysticArcanumCast) SpellLevel() int { return e.spellLevel }

// Spell returns the spell cast.
func (e MysticArcanumCast) Spell() content.SpellID { return e.spell }

// InvocationSpellCastWithoutSlot is the event of a spell cast through a spell-granting invocation.
type InvocationSpellCastWithoutSlot struct {
	id     Identifier
	option content.FeatureOptionID
	spell  content.SpellID
}

// CharacterID returns the id of the Warlock.
func (e InvocationSpellCastWithoutSlot) CharacterID() Identifier { return e.id }

// Option returns the invocation.
func (e InvocationSpellCastWithoutSlot) Option() content.FeatureOptionID { return e.option }

// Spell returns the spell cast.
func (e InvocationSpellCastWithoutSlot) Spell() content.SpellID { return e.spell }

// DarkOnesBlessingGained is the event of Temporary Hit Points gained from Dark One's Blessing.
type DarkOnesBlessingGained struct {
	id  Identifier
	thp int
}

// CharacterID returns the id of the Warlock.
func (e DarkOnesBlessingGained) CharacterID() Identifier { return e.id }

// TemporaryHitPoints returns the Temporary Hit Points gained.
func (e DarkOnesBlessingGained) TemporaryHitPoints() int { return e.thp }

// FiendishResilienceChosen is the event of the Fiendish Resilience damage type being chosen.
type FiendishResilienceChosen struct {
	id         Identifier
	damageType content.DamageType
}

// CharacterID returns the id of the Warlock.
func (e FiendishResilienceChosen) CharacterID() Identifier { return e.id }

// DamageType returns the type the Warlock has Resistance to.
func (e FiendishResilienceChosen) DamageType() content.DamageType { return e.damageType }

// DarkOnesOwnLuckUsed is the event of Dark One's Own Luck adding 1d10 to a roll.
type DarkOnesOwnLuckUsed struct {
	id   Identifier
	kind d20.Kind
	d10  int
}

// CharacterID returns the id of the Warlock.
func (e DarkOnesOwnLuckUsed) CharacterID() Identifier { return e.id }

// Kind returns the kind of the D20 Test the die was added to.
func (e DarkOnesOwnLuckUsed) Kind() d20.Kind { return e.kind }

// Rolled returns the d10 added.
func (e DarkOnesOwnLuckUsed) Rolled() int { return e.d10 }

// HurlThroughHellUsed is the event of Hurl Through Hell used on a hit.
type HurlThroughHellUsed struct{ id Identifier }

// CharacterID returns the id of the Warlock.
func (e HurlThroughHellUsed) CharacterID() Identifier { return e.id }

// HurlThroughHellRestored is the event of the use of Hurl Through Hell restored by expending a Pact Magic slot.
type HurlThroughHellRestored struct {
	id        Identifier
	slotLevel int
}

// CharacterID returns the id of the Warlock.
func (e HurlThroughHellRestored) CharacterID() Identifier { return e.id }

// SlotLevel returns the level of the Pact Magic slot the host expends.
func (e HurlThroughHellRestored) SlotLevel() int { return e.slotLevel }

// EldritchSmiteUsed is the event of Eldritch Smite used on a pact weapon hit.
type EldritchSmiteUsed struct {
	id        Identifier
	slotLevel int
	prone     bool
}

// CharacterID returns the id of the Warlock.
func (e EldritchSmiteUsed) CharacterID() Identifier { return e.id }

// SlotLevel returns the level of the Pact Magic slot the host expends.
func (e EldritchSmiteUsed) SlotLevel() int { return e.slotLevel }

// Prone reports whether the Warlock gave the target the Prone condition.
func (e EldritchSmiteUsed) Prone() bool { return e.prone }

// WarlockFeaturesRestored is the event of a Long Rest restoring spent Warlock features.
type WarlockFeaturesRestored struct {
	id       Identifier
	features []WarlockRestorableFeature
}

// CharacterID returns the id of the Warlock.
func (e WarlockFeaturesRestored) CharacterID() Identifier { return e.id }

// Features returns the features restored, in the order of WarlockRestorableFeature, as a new slice.
func (e WarlockFeaturesRestored) Features() []WarlockRestorableFeature {
	return slices.Clone(e.features)
}

func (InvocationsChosen) isEvent()                     {}
func (MysticArcanumChosen) isEvent()                   {}
func (MagicalCunningUsed) isEvent()                    {}
func (ContactPatronUsed) isEvent()                     {}
func (MysticArcanumCast) isEvent()                     {}
func (InvocationSpellCastWithoutSlot) isEvent()        {}
func (DarkOnesBlessingGained) isEvent()                {}
func (FiendishResilienceChosen) isEvent()              {}
func (DarkOnesOwnLuckUsed) isEvent()                   {}
func (HurlThroughHellUsed) isEvent()                   {}
func (HurlThroughHellRestored) isEvent()               {}
func (EldritchSmiteUsed) isEvent()                     {}
func (WarlockFeaturesRestored) isEvent()               {}
func (InvocationsChosen) isWarlockEvent()              {}
func (MysticArcanumChosen) isWarlockEvent()            {}
func (MagicalCunningUsed) isWarlockEvent()             {}
func (ContactPatronUsed) isWarlockEvent()              {}
func (MysticArcanumCast) isWarlockEvent()              {}
func (InvocationSpellCastWithoutSlot) isWarlockEvent() {}
func (DarkOnesBlessingGained) isWarlockEvent()         {}
func (FiendishResilienceChosen) isWarlockEvent()       {}
func (DarkOnesOwnLuckUsed) isWarlockEvent()            {}
func (HurlThroughHellUsed) isWarlockEvent()            {}
func (HurlThroughHellRestored) isWarlockEvent()        {}
func (EldritchSmiteUsed) isWarlockEvent()              {}
func (WarlockFeaturesRestored) isWarlockEvent()        {}

// BookLossCause is why a Book of Shadows disappears, a closed set (class-warlock.md FR-101).
type BookLossCause int

const (
	// BookLostToAnotherBook: the Warlock conjured another book with the feature.
	BookLostToAnotherBook BookLossCause = iota + 1
	// BookLostToDeath: the Warlock died.
	BookLostToDeath
)

// PactWeaponBonded is the event of a weapon bonded with Pact of the Blade.
type PactWeaponBonded struct {
	id       Identifier
	item     content.ItemID
	conjured bool
}

// CharacterID returns the id of the Warlock.
func (e PactWeaponBonded) CharacterID() Identifier { return e.id }

// Weapon returns the bonded weapon.
func (e PactWeaponBonded) Weapon() content.ItemID { return e.item }

// Conjured reports whether the weapon was conjured.
func (e PactWeaponBonded) Conjured() bool { return e.conjured }

// PactWeaponBondEnded is the event of the pact weapon bond ending; a conjured weapon disappears and a magic weapon remains.
type PactWeaponBondEnded struct {
	id       Identifier
	item     content.ItemID
	conjured bool
}

// CharacterID returns the id of the Warlock.
func (e PactWeaponBondEnded) CharacterID() Identifier { return e.id }

// Weapon returns the weapon that was bonded.
func (e PactWeaponBondEnded) Weapon() content.ItemID { return e.item }

// Disappeared reports whether the weapon was conjured and so is gone.
func (e PactWeaponBondEnded) Disappeared() bool { return e.conjured }

// FamiliarFormChosen is the event of the familiar form chosen.
type FamiliarFormChosen struct {
	id   Identifier
	form FamiliarForm
}

// CharacterID returns the id of the Warlock.
func (e FamiliarFormChosen) CharacterID() Identifier { return e.id }

// Form returns the form.
func (e FamiliarFormChosen) Form() FamiliarForm { return e.form }

// FamiliarSpeedChosen is the event of the familiar given a 40 feet Fly or Swim Speed.
type FamiliarSpeedChosen struct {
	id   Identifier
	kind FamiliarSpeedKind
}

// CharacterID returns the id of the Warlock.
func (e FamiliarSpeedChosen) CharacterID() Identifier { return e.id }

// Speed returns the Speed chosen.
func (e FamiliarSpeedChosen) Speed() FamiliarSpeedKind { return e.kind }

// FamiliarAttackForgone is the event of the Warlock forgoing an attack so the familiar attacks with its Reaction.
type FamiliarAttackForgone struct{ id Identifier }

// CharacterID returns the id of the Warlock.
func (e FamiliarAttackForgone) CharacterID() Identifier { return e.id }

// FamiliarAttackCommanded is the event of the familiar commanded to take the Attack action as a Bonus Action.
type FamiliarAttackCommanded struct{ id Identifier }

// CharacterID returns the id of the Warlock.
func (e FamiliarAttackCommanded) CharacterID() Identifier { return e.id }

// FamiliarResistanceGranted is the event of the familiar given Resistance to the damage it took.
type FamiliarResistanceGranted struct {
	id         Identifier
	damageType content.DamageType
}

// CharacterID returns the id of the Warlock.
func (e FamiliarResistanceGranted) CharacterID() Identifier { return e.id }

// DamageType returns the type the familiar has Resistance to.
func (e FamiliarResistanceGranted) DamageType() content.DamageType { return e.damageType }

// BookOfShadowsConjured is the event of a Book of Shadows appearing.
type BookOfShadowsConjured struct {
	id   Identifier
	book BookOfShadows
}

// CharacterID returns the id of the Warlock.
func (e BookOfShadowsConjured) CharacterID() Identifier { return e.id }

// Book returns the book.
func (e BookOfShadowsConjured) Book() BookOfShadows { return e.book }

// BookOfShadowsLost is the event of the Book of Shadows disappearing, so its spells stop being prepared.
type BookOfShadowsLost struct {
	id    Identifier
	cause BookLossCause
}

// CharacterID returns the id of the Warlock.
func (e BookOfShadowsLost) CharacterID() Identifier { return e.id }

// Cause returns why the book disappeared.
func (e BookOfShadowsLost) Cause() BookLossCause { return e.cause }

// GiftOfTheProtectorsNameWritten is the event of a creature's name written on the page.
type GiftOfTheProtectorsNameWritten struct {
	id       Identifier
	creature Identifier
}

// CharacterID returns the id of the Warlock.
func (e GiftOfTheProtectorsNameWritten) CharacterID() Identifier { return e.id }

// Creature returns the creature whose name was written.
func (e GiftOfTheProtectorsNameWritten) Creature() Identifier { return e.creature }

// GiftOfTheProtectorsNameErased is the event of a creature's name erased from the page.
type GiftOfTheProtectorsNameErased struct {
	id       Identifier
	creature Identifier
}

// CharacterID returns the id of the Warlock.
func (e GiftOfTheProtectorsNameErased) CharacterID() Identifier { return e.id }

// Creature returns the creature whose name was erased.
func (e GiftOfTheProtectorsNameErased) Creature() Identifier { return e.creature }

// GiftOfTheProtectorsTriggered is the event of a named creature dropping to 1 Hit Point instead of 0.
type GiftOfTheProtectorsTriggered struct {
	id       Identifier
	creature Identifier
}

// CharacterID returns the id of the Warlock.
func (e GiftOfTheProtectorsTriggered) CharacterID() Identifier { return e.id }

// Creature returns the creature that dropped to 1 Hit Point.
func (e GiftOfTheProtectorsTriggered) Creature() Identifier { return e.creature }

// LifedrinkerUsed is the event of Lifedrinker adding 1d6 damage and, when a Hit Point Die was expended, healing the Warlock.
type LifedrinkerUsed struct {
	id         Identifier
	damageType content.DamageType
	healed     int
}

// CharacterID returns the id of the Warlock.
func (e LifedrinkerUsed) CharacterID() Identifier { return e.id }

// DamageType returns the type of the extra damage.
func (e LifedrinkerUsed) DamageType() content.DamageType { return e.damageType }

// HitPointsRegained returns the Hit Points regained, 0 when no Hit Point Die was expended.
func (e LifedrinkerUsed) HitPointsRegained() int { return e.healed }

// GazeLinkBegun is the event of a Gaze of Two Minds link beginning.
type GazeLinkBegun struct {
	id       Identifier
	creature Identifier
}

// CharacterID returns the id of the Warlock.
func (e GazeLinkBegun) CharacterID() Identifier { return e.id }

// Creature returns the linked creature.
func (e GazeLinkBegun) Creature() Identifier { return e.creature }

// GazeLinkMaintained is the event of a Gaze of Two Minds link extended to the end of the next turn.
type GazeLinkMaintained struct {
	id       Identifier
	creature Identifier
}

// CharacterID returns the id of the Warlock.
func (e GazeLinkMaintained) CharacterID() Identifier { return e.id }

// Creature returns the linked creature.
func (e GazeLinkMaintained) Creature() Identifier { return e.creature }

func (PactWeaponBonded) isEvent()                      {}
func (PactWeaponBondEnded) isEvent()                   {}
func (FamiliarFormChosen) isEvent()                    {}
func (FamiliarSpeedChosen) isEvent()                   {}
func (FamiliarAttackForgone) isEvent()                 {}
func (FamiliarAttackCommanded) isEvent()               {}
func (FamiliarResistanceGranted) isEvent()             {}
func (BookOfShadowsConjured) isEvent()                 {}
func (BookOfShadowsLost) isEvent()                     {}
func (GiftOfTheProtectorsNameWritten) isEvent()        {}
func (GiftOfTheProtectorsNameErased) isEvent()         {}
func (GiftOfTheProtectorsTriggered) isEvent()          {}
func (LifedrinkerUsed) isEvent()                       {}
func (GazeLinkBegun) isEvent()                         {}
func (GazeLinkMaintained) isEvent()                    {}
func (PactWeaponBonded) isWarlockEvent()               {}
func (PactWeaponBondEnded) isWarlockEvent()            {}
func (FamiliarFormChosen) isWarlockEvent()             {}
func (FamiliarSpeedChosen) isWarlockEvent()            {}
func (FamiliarAttackForgone) isWarlockEvent()          {}
func (FamiliarAttackCommanded) isWarlockEvent()        {}
func (FamiliarResistanceGranted) isWarlockEvent()      {}
func (BookOfShadowsConjured) isWarlockEvent()          {}
func (BookOfShadowsLost) isWarlockEvent()              {}
func (GiftOfTheProtectorsNameWritten) isWarlockEvent() {}
func (GiftOfTheProtectorsNameErased) isWarlockEvent()  {}
func (GiftOfTheProtectorsTriggered) isWarlockEvent()   {}
func (LifedrinkerUsed) isWarlockEvent()                {}
func (GazeLinkBegun) isWarlockEvent()                  {}
func (GazeLinkMaintained) isWarlockEvent()             {}
