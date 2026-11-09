package character

// RogueEvent is something that happened to a Rogue, in the past tense and immutable, the closed family of Rogue events. Every
// one is also an Event and carries the Identifier it concerns. The interface cannot be implemented outside this package.
// Members: StrokeOfLuckUsed.
type RogueEvent interface {
	Event
	isRogueEvent()
}

// StrokeOfLuckUsed is the event of a use of Stroke of Luck (SRD 5.2.1, "Level 20: Stroke of Luck", p. 63). It is outside the
// Character aggregate's RecordEvent family.
type StrokeOfLuckUsed struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e StrokeOfLuckUsed) CharacterID() Identifier { return e.id }

func (StrokeOfLuckUsed) isEvent()      {}
func (StrokeOfLuckUsed) isRogueEvent() {}
