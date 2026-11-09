package character

import "github.com/Whealinghold/srd5e-go/content"

// FighterEvent is something that happened to a Fighter, in the past tense and
// immutable, the closed family of Fighter events. Every one is also an Event and
// carries the Identifier it concerns. The interface cannot be implemented outside
// this package. Members: SecondWindUsed, TacticalMindUsed, ActionSurgeUsed,
// IndomitableUsed, FighterFightingStyleChosen, FighterFightingStyleReplaced,
// FighterWeaponMasteriesChosen, FighterWeaponMasterySwapped.
type FighterEvent interface {
	Event
	isFighterEvent()
}

// SecondWindUsed is the event of a use of Second Wind.
type SecondWindUsed struct {
	id                Identifier
	rolled, hitPoints int
}

// CharacterID returns the id of the Character.
func (e SecondWindUsed) CharacterID() Identifier { return e.id }

// Rolled returns the d10 rolled.
func (e SecondWindUsed) Rolled() int { return e.rolled }

// HitPointsToRegain returns the uncapped Hit Points to regain.
func (e SecondWindUsed) HitPointsToRegain() int { return e.hitPoints }

// TacticalMindUsed is the event of a use of Tactical Mind that succeeded and
// expended a Second Wind use.
type TacticalMindUsed struct {
	id     Identifier
	rolled int
}

// CharacterID returns the id of the Character.
func (e TacticalMindUsed) CharacterID() Identifier { return e.id }

// Rolled returns the d10 rolled.
func (e TacticalMindUsed) Rolled() int { return e.rolled }

// ActionSurgeUsed is the event of a use of Action Surge.
type ActionSurgeUsed struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e ActionSurgeUsed) CharacterID() Identifier { return e.id }

// IndomitableUsed is the event of a use of Indomitable, which expends a use
// whether or not the new roll succeeds.
type IndomitableUsed struct {
	id     Identifier
	rolled int
}

// CharacterID returns the id of the Character.
func (e IndomitableUsed) CharacterID() Identifier { return e.id }

// Rolled returns the d20 rolled.
func (e IndomitableUsed) Rolled() int { return e.rolled }

// FighterFightingStyleChosen is the event of a Fighting Style feat chosen: the
// level 1 style, or the Champion's Additional Fighting Style.
type FighterFightingStyleChosen struct {
	id         Identifier
	feat       content.FeatID
	additional bool
}

// CharacterID returns the id of the Character.
func (e FighterFightingStyleChosen) CharacterID() Identifier { return e.id }

// Feat returns the feat chosen.
func (e FighterFightingStyleChosen) Feat() content.FeatID { return e.feat }

// Additional reports whether it is the Additional Fighting Style.
func (e FighterFightingStyleChosen) Additional() bool { return e.additional }

// FighterFightingStyleReplaced is the event of the level 1 Fighting Style being
// replaced on gaining a Fighter level.
type FighterFightingStyleReplaced struct {
	id        Identifier
	old, feat content.FeatID
}

// CharacterID returns the id of the Character.
func (e FighterFightingStyleReplaced) CharacterID() Identifier { return e.id }

// Old returns the feat replaced.
func (e FighterFightingStyleReplaced) Old() content.FeatID { return e.old }

// Feat returns the feat now chosen.
func (e FighterFightingStyleReplaced) Feat() content.FeatID { return e.feat }

// FighterWeaponMasteriesChosen is the event of the Weapon Mastery kinds chosen or
// extended on gaining a level.
type FighterWeaponMasteriesChosen struct {
	id    Identifier
	kinds []content.ItemID
}

// CharacterID returns the id of the Character.
func (e FighterWeaponMasteriesChosen) CharacterID() Identifier { return e.id }

// Kinds returns the kinds now chosen, as a new slice.
func (e FighterWeaponMasteriesChosen) Kinds() []content.ItemID {
	return append([]content.ItemID(nil), e.kinds...)
}

// FighterWeaponMasterySwapped is the event of one Weapon Mastery kind changed
// after a Long Rest.
type FighterWeaponMasterySwapped struct {
	id             Identifier
	removed, added content.ItemID
}

// CharacterID returns the id of the Character.
func (e FighterWeaponMasterySwapped) CharacterID() Identifier { return e.id }

// Removed returns the kind removed.
func (e FighterWeaponMasterySwapped) Removed() content.ItemID { return e.removed }

// Added returns the kind added.
func (e FighterWeaponMasterySwapped) Added() content.ItemID { return e.added }

func (SecondWindUsed) isEvent()                      {}
func (TacticalMindUsed) isEvent()                    {}
func (ActionSurgeUsed) isEvent()                     {}
func (IndomitableUsed) isEvent()                     {}
func (FighterFightingStyleChosen) isEvent()          {}
func (FighterFightingStyleReplaced) isEvent()        {}
func (FighterWeaponMasteriesChosen) isEvent()        {}
func (FighterWeaponMasterySwapped) isEvent()         {}
func (SecondWindUsed) isFighterEvent()               {}
func (TacticalMindUsed) isFighterEvent()             {}
func (ActionSurgeUsed) isFighterEvent()              {}
func (IndomitableUsed) isFighterEvent()              {}
func (FighterFightingStyleChosen) isFighterEvent()   {}
func (FighterFightingStyleReplaced) isFighterEvent() {}
func (FighterWeaponMasteriesChosen) isFighterEvent() {}
func (FighterWeaponMasterySwapped) isFighterEvent()  {}
