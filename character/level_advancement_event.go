package character

import "github.com/Whealinghold/srd5e-go/content"

// LevelAdvancementEvent is the closed family of events of level advancement. It
// is an Event, so a host switches over it for the family and over Event for the
// context. Events are past tense and immutable and carry the Identifier of the
// character they concern. Members: SubclassChosen.
type LevelAdvancementEvent interface {
	Event
	isLevelAdvancementEvent()
}

// SubclassChosen is the event of a character choosing the subclass of a class.
type SubclassChosen struct {
	id         Identifier
	class      content.ClassID
	subclass   content.SubclassID
	classLevel int
}

// CharacterID returns the id of the character.
func (e SubclassChosen) CharacterID() Identifier { return e.id }

// Class returns the class the subclass belongs to.
func (e SubclassChosen) Class() content.ClassID { return e.class }

// Subclass returns the subclass chosen.
func (e SubclassChosen) Subclass() content.SubclassID { return e.subclass }

// ClassLevel returns the character's level in the class when it chose.
func (e SubclassChosen) ClassLevel() int { return e.classLevel }

func (SubclassChosen) isEvent()                 {}
func (SubclassChosen) isLevelAdvancementEvent() {}
