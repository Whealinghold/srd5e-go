package character

// RangerEvent is something that happened to a Ranger, in the past tense and immutable, the closed family of Ranger events. Every
// one is also an Event and carries the Identifier it concerns. The interface cannot be implemented outside this package.
// Members: FavoredEnemyCastSpent, TirelessUsed, NaturesVeilUsed, ExhaustionReductionGranted, RangerChoiceMade,
// RangerChoiceReplaced.
type RangerEvent interface {
	Event
	isRangerEvent()
}

// FavoredEnemyCastSpent is the event of a free cast of Hunter’s Mark spent from Favored Enemy.
type FavoredEnemyCastSpent struct {
	id        Identifier
	remaining int
}

// CharacterID returns the id of the Character.
func (e FavoredEnemyCastSpent) CharacterID() Identifier { return e.id }

// Remaining returns the free casts left after this one.
func (e FavoredEnemyCastSpent) Remaining() int { return e.remaining }

// TirelessUsed is the event of a use of Tireless.
type TirelessUsed struct {
	id                Identifier
	rolled, temporary int
}

// CharacterID returns the id of the Character.
func (e TirelessUsed) CharacterID() Identifier { return e.id }

// Rolled returns the d8 rolled.
func (e TirelessUsed) Rolled() int { return e.rolled }

// TemporaryHitPoints returns the Temporary Hit Points granted.
func (e TirelessUsed) TemporaryHitPoints() int { return e.temporary }

// NaturesVeilUsed is the event of a use of Nature’s Veil.
type NaturesVeilUsed struct {
	id        Identifier
	remaining int
}

// CharacterID returns the id of the Character.
func (e NaturesVeilUsed) CharacterID() Identifier { return e.id }

// Remaining returns the uses left after this one.
func (e NaturesVeilUsed) Remaining() int { return e.remaining }

// ExhaustionReductionGranted is the event of the Exhaustion level decrease Tireless gives on a finished Short Rest. Applying it is
// the rest rules' (rests-and-hit-dice.md).
type ExhaustionReductionGranted struct {
	id     Identifier
	amount int
}

// CharacterID returns the id of the Character.
func (e ExhaustionReductionGranted) CharacterID() Identifier { return e.id }

// Amount returns the levels of Exhaustion to remove, always 1.
func (e ExhaustionReductionGranted) Amount() int { return e.amount }

// RangerChoiceMade is the event of an initial Ranger choice.
type RangerChoiceMade struct {
	id     Identifier
	choice string
}

// CharacterID returns the id of the Character.
func (e RangerChoiceMade) CharacterID() Identifier { return e.id }

// Choice returns the srd5e.feature value of the choice, such as "deft_explorer".
func (e RangerChoiceMade) Choice() string { return e.choice }

// RangerChoiceReplaced is the event of a replacement of a Ranger choice already made.
type RangerChoiceReplaced struct {
	id     Identifier
	choice string
}

// CharacterID returns the id of the Character.
func (e RangerChoiceReplaced) CharacterID() Identifier { return e.id }

// Choice returns the srd5e.feature value of the choice, such as "hunters_prey".
func (e RangerChoiceReplaced) Choice() string { return e.choice }

func (FavoredEnemyCastSpent) isEvent()      {}
func (TirelessUsed) isEvent()               {}
func (NaturesVeilUsed) isEvent()            {}
func (ExhaustionReductionGranted) isEvent() {}
func (RangerChoiceMade) isEvent()           {}
func (RangerChoiceReplaced) isEvent()       {}

func (FavoredEnemyCastSpent) isRangerEvent()      {}
func (TirelessUsed) isRangerEvent()               {}
func (NaturesVeilUsed) isRangerEvent()            {}
func (ExhaustionReductionGranted) isRangerEvent() {}
func (RangerChoiceMade) isRangerEvent()           {}
func (RangerChoiceReplaced) isRangerEvent()       {}
