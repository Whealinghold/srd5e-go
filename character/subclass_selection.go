package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// ChosenSubclass is the subclass a character holds in one class, stored in the
// character's FeatureStates under SubclassStateKey. It is a FeatureState,
// immutable and comparable by value. The zero ChosenSubclass is invalid.
type ChosenSubclass struct{ subclass content.SubclassID }

// NewChosenSubclass returns the state holding subclass. It returns an error
// matching srderr.ErrInvalidInput for an invalid id. It never panics.
func NewChosenSubclass(subclass content.SubclassID) (ChosenSubclass, error) {
	if !subclass.Valid() {
		return ChosenSubclass{}, invalidInput("chosen subclass id")
	}
	return ChosenSubclass{subclass: subclass}, nil
}

// Subclass returns the subclass held.
func (s ChosenSubclass) Subclass() content.SubclassID { return s.subclass }

// Equal reports whether other is a ChosenSubclass of the same subclass.
func (s ChosenSubclass) Equal(other FeatureState) bool {
	o, ok := other.(ChosenSubclass)
	return ok && o == s
}

// SubclassStateKey builds the FeatureStateKey under which a character holds the
// subclass of one class, so a class holds at most one. The zero value is ready
// to use.
type SubclassStateKey struct{}

// ForClass returns the key "subclass_" plus the class id. It returns an error
// matching srderr.ErrInvalidInput for an invalid class id. It never panics.
func (SubclassStateKey) ForClass(class content.ClassID) (FeatureStateKey[ChosenSubclass], error) {
	if !class.Valid() {
		return FeatureStateKey[ChosenSubclass]{}, invalidInput("subclass state key of an invalid class")
	}
	return NewFeatureStateKey[ChosenSubclass]("subclass_" + string(class))
}

// SubclassRejectionReason is why a subclass choice was refused, in the order the
// reasons are checked. The zero value is not a reason.
type SubclassRejectionReason int

// The reasons a subclass choice is refused.
const (
	// SubclassNotInClass: the character has no levels in the class.
	SubclassNotInClass SubclassRejectionReason = iota + 1
	// SubclassBelowSubclassLevel: its level in the class is below the class's subclass level.
	SubclassBelowSubclassLevel
	// SubclassAlreadyChosen: it already holds a subclass of the class.
	SubclassAlreadyChosen
	// SubclassUnknownSubclass: the class lists no such subclass.
	SubclassUnknownSubclass
)

// String returns the snake_case name: "not_in_class", "below_subclass_level",
// "already_chosen", "unknown_subclass" or "unknown".
func (r SubclassRejectionReason) String() string {
	switch r {
	case SubclassNotInClass:
		return "not_in_class"
	case SubclassBelowSubclassLevel:
		return "below_subclass_level"
	case SubclassAlreadyChosen:
		return "already_chosen"
	case SubclassUnknownSubclass:
		return "unknown_subclass"
	}
	return "unknown"
}

// SubclassRejected is the rejection of a subclass choice. Construct it with a
// keyed literal.
type SubclassRejected struct{ Reason SubclassRejectionReason }

// SubclassSelection is the pure rules service for choosing a subclass (SRD
// 5.2.1, "Level 3: Cleric Subclass", p. 37; "Level 3: Rogue Subclass", p. 62).
// The zero value is ready to use. It is stateless and safe for concurrent use.
type SubclassSelection struct{}

// Choose gives the character the subclass of class. The choice is accepted when
// the character has levels in the class, its level in that class (not its total
// level) is at least class.SubclassLevel, it holds no subclass of the class and
// the class lists the subclass. The new Character holds a ChosenSubclass under
// SubclassStateKey.ForClass and every other state is unchanged; the event is one
// SubclassChosen. When several reasons hold, the one reported is the first of
// SubclassNotInClass, SubclassBelowSubclassLevel, SubclassAlreadyChosen,
// SubclassUnknownSubclass, as a TransitionRejected holding a SubclassRejected.
// The receiver is unchanged.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, an
// invalid subclass id, or a stored state under the class's key of another type.
// It never panics.
func (SubclassSelection) Choose(c Character, class content.ClassDefinition, subclass content.SubclassID) (Transition[SubclassRejected], error) {
	if c.IsZero() || !subclass.Valid() {
		return nil, invalidInput("subclass choice of a zero character or an invalid subclass")
	}
	key, err := SubclassStateKey{}.ForClass(class.ID())
	if err != nil {
		return nil, err
	}
	level := c.classes.LevelIn(class.ID())
	_, isHeld, err := ReadFeatureState(c.states, key)
	if err != nil {
		return nil, err
	}
	_, listed := class.Subclass(subclass).(content.SubclassFound)
	switch {
	case level == 0:
		return rejected(SubclassRejected{Reason: SubclassNotInClass}), nil
	case level < class.SubclassLevel():
		return rejected(SubclassRejected{Reason: SubclassBelowSubclassLevel}), nil
	case isHeld:
		return rejected(SubclassRejected{Reason: SubclassAlreadyChosen}), nil
	case !listed:
		return rejected(SubclassRejected{Reason: SubclassUnknownSubclass}), nil
	}
	state, err := NewChosenSubclass(subclass)
	if err != nil { // unreachable: the id was checked
		return nil, err
	}
	next, err := WithFeatureState(c, key, state)
	if err != nil { // unreachable: the key and state are valid and the read succeeded
		return nil, err
	}
	return changed[SubclassRejected](next, SubclassChosen{id: c.id, class: class.ID(), subclass: subclass, classLevel: level}), nil
}

// IsPending reports whether a subclass choice is owed in class: the character
// has levels in it, its level in the class is at least class.SubclassLevel and
// it holds no subclass of the class. It returns an error matching
// srderr.ErrInvalidInput for a zero Character or a stored state under the
// class's key of another type. It never panics.
func (SubclassSelection) IsPending(c Character, class content.ClassDefinition) (bool, error) {
	if c.IsZero() {
		return false, invalidInput("subclass pending check of a zero character")
	}
	key, err := SubclassStateKey{}.ForClass(class.ID())
	if err != nil {
		return false, err
	}
	_, isHeld, err := ReadFeatureState(c.states, key)
	if err != nil {
		return false, err
	}
	level := c.classes.LevelIn(class.ID())
	return level > 0 && level >= class.SubclassLevel() && !isHeld, nil
}
