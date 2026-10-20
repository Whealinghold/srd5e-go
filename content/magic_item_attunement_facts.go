package content

import "slices"

// AttunementPrerequisite is a structured Attunement prerequisite, the host-side
// form of the verbatim text of [AttunementRequiredBy]: a closed family with
// four variants, [NoPrerequisite], [ClassPrerequisite],
// [SpellcasterPrerequisite] and [DescribedPrerequisite] (SRD 5.2.1, "Magic Item
// Rules", Attunement Prerequisites, p. 205; "Attunement", p. 102). Evaluating
// one, and reading an Attunement text into one, is not done here. Consumers
// switch over it exhaustively. Every variant is immutable and safe for
// concurrent use.
type AttunementPrerequisite interface{ isAttunementPrerequisite() }

// NoPrerequisite is no prerequisite. The zero value is valid.
type NoPrerequisite struct{}

func (NoPrerequisite) isAttunementPrerequisite() {}

// ClassPrerequisite is met by membership of one of its classes. The zero value
// is invalid; build it with [NewClassPrerequisite]. It holds a slice, so
// compare it with [ClassPrerequisite.Equal].
type ClassPrerequisite struct{ classes []ClassID }

// NewClassPrerequisite returns a prerequisite of a copy of classes: a
// non-empty set of valid class ids.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an empty set, BlockInvalidMember for an invalid id and
// BlockDuplicate for a repeat. It never panics.
func NewClassPrerequisite(classes []ClassID) (ClassPrerequisite, error) {
	if err := checkSet("ClassPrerequisite", classes, 1, noUpperBound, ClassID.Valid); err != nil {
		return ClassPrerequisite{}, err
	}
	return ClassPrerequisite{classes: slices.Clone(classes)}, nil
}

// Classes returns the class ids as a new slice.
func (p ClassPrerequisite) Classes() []ClassID { return slices.Clone(p.classes) }

// Equal reports whether p and q hold the same classes in the same order.
func (p ClassPrerequisite) Equal(q ClassPrerequisite) bool { return slices.Equal(p.classes, q.classes) }

func (ClassPrerequisite) isAttunementPrerequisite() {}

// SpellcasterPrerequisite is met by a creature able to cast at least one spell
// using its traits or features, not by using a magic item or the like. The zero
// value is valid.
type SpellcasterPrerequisite struct{}

func (SpellcasterPrerequisite) isAttunementPrerequisite() {}

// DescribedPrerequisite is any prerequisite that is neither a class nor a
// Spellcaster, kept as verbatim text. The zero value is invalid; build it with
// [NewDescribedPrerequisite].
type DescribedPrerequisite struct{ text string }

// NewDescribedPrerequisite returns a prerequisite of the verbatim text, which
// must contain a non-white-space character.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter otherwise. It never panics.
func NewDescribedPrerequisite(text string) (DescribedPrerequisite, error) {
	if err := checkText("DescribedPrerequisite", text); err != nil {
		return DescribedPrerequisite{}, err
	}
	return DescribedPrerequisite{text: text}, nil
}

// Text returns the verbatim text.
func (p DescribedPrerequisite) Text() string { return p.text }

func (DescribedPrerequisite) isAttunementPrerequisite() {}

// AttunementFacts holds the numbers of the Attunement rules (SRD 5.2.1, "No
// More Than Three Items", "Ending Attunement", p. 102; "Attunement", p. 177).
// They are numbers only; comparing against them is not done here. It holds no
// state; its zero value is the facts. Safe for concurrent use.
type AttunementFacts struct{}

// StandardLimit returns 3: no more than three magic items at a time.
func (AttunementFacts) StandardLimit() int { return 3 }

// MaxCopiesPerItem returns 1: no attuning to more than one copy of an item.
func (AttunementFacts) MaxCopiesPerItem() int { return 1 }

// EndingDistanceFeet returns 100: Attunement ends when the item has been more
// than this many feet away...
func (AttunementFacts) EndingDistanceFeet() int { return 100 }

// EndingHoursAway returns 24: ...for at least this many hours.
func (AttunementFacts) EndingHoursAway() int { return 24 }
