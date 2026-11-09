package character

import "github.com/Whealinghold/srd5e-go/content"

// BarbarianEvent is something that happened to a Barbarian, in the past tense and
// immutable, the closed family of Barbarian events. Every one is also an Event and
// carries the Identifier it concerns. The interface cannot be implemented outside this
// package. Members: RageStarted, RageEnded, RelentlessRageUsed, RagesRegained,
// IntimidatingPresenceUsed, BarbarianWeaponMasteriesChosen, BarbarianWeaponMasterySwapped.
type BarbarianEvent interface {
	Event
	isBarbarianEvent()
}

// RageStarted is the event of a Rage entered.
type RageStarted struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e RageStarted) CharacterID() Identifier { return e.id }

// RageEnded is the event of a Rage ended, with the reason.
type RageEnded struct {
	id     Identifier
	reason RageEndReason
}

// CharacterID returns the id of the Character.
func (e RageEnded) CharacterID() Identifier { return e.id }

// Reason returns why the Rage ended.
func (e RageEnded) Reason() RageEndReason { return e.reason }

// RelentlessRageUsed is the event of a use of Relentless Rage, whether or not the save
// succeeded.
type RelentlessRageUsed struct {
	id        Identifier
	succeeded bool
	hitPoints int
	nextDC    int
}

// CharacterID returns the id of the Character.
func (e RelentlessRageUsed) CharacterID() Identifier { return e.id }

// Succeeded reports whether the Constitution saving throw succeeded.
func (e RelentlessRageUsed) Succeeded() bool { return e.succeeded }

// HitPoints returns the Hit Points to set (twice the Barbarian level) on a success, 0 on a failure.
func (e RelentlessRageUsed) HitPoints() int { return e.hitPoints }

// NextDC returns the Difficulty Class of the next Relentless Rage.
func (e RelentlessRageUsed) NextDC() int { return e.nextDC }

// RagesRegained is the event of Persistent Rage regaining every expended Rage use.
type RagesRegained struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e RagesRegained) CharacterID() Identifier { return e.id }

// IntimidatingPresenceUsed is the event of a use of Intimidating Presence.
type IntimidatingPresenceUsed struct {
	id Identifier
	dc int
}

// CharacterID returns the id of the Character.
func (e IntimidatingPresenceUsed) CharacterID() Identifier { return e.id }

// DC returns the Wisdom saving throw Difficulty Class.
func (e IntimidatingPresenceUsed) DC() int { return e.dc }

func (RageStarted) isEvent()                       {}
func (RageEnded) isEvent()                         {}
func (RelentlessRageUsed) isEvent()                {}
func (RagesRegained) isEvent()                     {}
func (IntimidatingPresenceUsed) isEvent()          {}
func (RageStarted) isBarbarianEvent()              {}
func (RageEnded) isBarbarianEvent()                {}
func (RelentlessRageUsed) isBarbarianEvent()       {}
func (RagesRegained) isBarbarianEvent()            {}
func (IntimidatingPresenceUsed) isBarbarianEvent() {}

// BarbarianWeaponMasteriesChosen is the event of the Weapon Mastery kinds chosen or
// extended on gaining a level.
type BarbarianWeaponMasteriesChosen struct {
	id    Identifier
	kinds []content.ItemID
}

// CharacterID returns the id of the Character.
func (e BarbarianWeaponMasteriesChosen) CharacterID() Identifier { return e.id }

// Kinds returns the kinds now chosen, as a new slice.
func (e BarbarianWeaponMasteriesChosen) Kinds() []content.ItemID {
	return append([]content.ItemID(nil), e.kinds...)
}

// BarbarianWeaponMasterySwapped is the event of one Weapon Mastery kind changed after a Long Rest.
type BarbarianWeaponMasterySwapped struct {
	id             Identifier
	removed, added content.ItemID
}

// CharacterID returns the id of the Character.
func (e BarbarianWeaponMasterySwapped) CharacterID() Identifier { return e.id }

// Removed returns the kind removed.
func (e BarbarianWeaponMasterySwapped) Removed() content.ItemID { return e.removed }

// Added returns the kind added.
func (e BarbarianWeaponMasterySwapped) Added() content.ItemID { return e.added }

func (BarbarianWeaponMasteriesChosen) isEvent()          {}
func (BarbarianWeaponMasterySwapped) isEvent()           {}
func (BarbarianWeaponMasteriesChosen) isBarbarianEvent() {}
func (BarbarianWeaponMasterySwapped) isBarbarianEvent()  {}
