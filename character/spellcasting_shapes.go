package character

import "github.com/Whealinghold/srd5e-go/content"

// ConcentrationSubject is what a character concentrates on: a spell or a
// non-spell effect (SRD 5.2.1, "Concentration", p. 179). Variants:
// ConcentrationSubjectSpell, ConcentrationSubjectFeature.
type ConcentrationSubject interface{ isConcentrationSubject() }

// ConcentrationSubjectSpell: Concentration on a spell.
type ConcentrationSubjectSpell struct{ spell content.SpellID }

// NewConcentrationSubjectSpell returns the subject for spell id.
func NewConcentrationSubjectSpell(id content.SpellID) ConcentrationSubjectSpell {
	return ConcentrationSubjectSpell{spell: id}
}

// Spell returns the spell id.
func (s ConcentrationSubjectSpell) Spell() content.SpellID { return s.spell }

// ConcentrationSubjectFeature: Concentration on an effect that is not a spell.
type ConcentrationSubjectFeature struct{ feature content.FeatureID }

// NewConcentrationSubjectFeature returns the subject for feature id.
func NewConcentrationSubjectFeature(id content.FeatureID) ConcentrationSubjectFeature {
	return ConcentrationSubjectFeature{feature: id}
}

// Feature returns the feature id.
func (s ConcentrationSubjectFeature) Feature() content.FeatureID { return s.feature }

func (ConcentrationSubjectSpell) isConcentrationSubject()   {}
func (ConcentrationSubjectFeature) isConcentrationSubject() {}

// ConcentrationKind says how Concentration began. The zero value is invalid.
// Comparable and safe for concurrent use.
type ConcentrationKind int

// The Concentration kinds.
const (
	// ConcentrationEffect: Concentration on an effect already in place.
	ConcentrationEffect ConcentrationKind = iota + 1
	// ConcentrationCasting: Concentration while casting a spell over time; a spell subject only.
	ConcentrationCasting
)

// String returns "effect", "casting" or "unknown".
func (k ConcentrationKind) String() string {
	switch k {
	case ConcentrationEffect:
		return "effect"
	case ConcentrationCasting:
		return "casting"
	}
	return "unknown"
}

// Concentration is the subject a character concentrates on and the kind. The zero
// Concentration means none. Comparable and safe for concurrent use.
type Concentration struct {
	subject ConcentrationSubject
	kind    ConcentrationKind
}

// NewConcentration returns Concentration on subject of the given kind. It
// returns an error matching srderr.ErrInvalidInput for a nil subject, an
// invalid kind or a feature subject with ConcentrationCasting. It never panics.
func NewConcentration(subject ConcentrationSubject, kind ConcentrationKind) (Concentration, error) {
	_, isFeature := subject.(ConcentrationSubjectFeature)
	if subject == nil || (kind != ConcentrationEffect && kind != ConcentrationCasting) || (isFeature && kind == ConcentrationCasting) {
		return Concentration{}, invalidInput("concentration: invalid subject or kind")
	}
	return Concentration{subject: subject, kind: kind}, nil
}

// IsZero reports whether c is the zero value (no Concentration).
func (c Concentration) IsZero() bool { return c.subject == nil }

// Subject returns the subject, nil for the zero value.
func (c Concentration) Subject() ConcentrationSubject { return c.subject }

// Kind returns the kind.
func (c Concentration) Kind() ConcentrationKind { return c.kind }

// CastSource is where a cast spell comes from. The zero value is invalid.
// Comparable and safe for concurrent use.
type CastSource int

// The cast sources.
const (
	// CastSourceSlot: a prepared spell cast with a spell slot.
	CastSourceSlot CastSource = iota + 1
	// CastSourceCantrip: a cantrip, cast without a slot.
	CastSourceCantrip
	// CastSourceRitual: a Ritual, cast without a slot.
	CastSourceRitual
	// CastSourceFeatureGranted: a spell a feature lets the character cast.
	CastSourceFeatureGranted
	// CastSourceScroll: a spell cast from a Spell Scroll.
	CastSourceScroll
	// CastSourceDuplicated: a spell that Wish duplicates; never in a CastingInProgress.
	CastSourceDuplicated
)

// String returns the snake_case name or "unknown".
func (s CastSource) String() string {
	names := [...]string{"slot", "cantrip", "ritual", "feature_granted", "scroll", "duplicated"}
	if s < CastSourceSlot || int(s) > len(names) {
		return "unknown"
	}
	return names[s-1]
}

// SpellSlotChoice is the slot a cast spends. Variants: SpellSlotChoiceSpellcasting,
// SpellSlotChoicePactMagic.
type SpellSlotChoice interface{ isSpellSlotChoice() }

// SpellSlotChoiceSpellcasting: a slot of the Spellcasting pool at a level.
type SpellSlotChoiceSpellcasting struct{ level SlotLevel }

// NewSpellSlotChoiceSpellcasting returns the choice of a Spellcasting slot of level l.
func NewSpellSlotChoiceSpellcasting(l SlotLevel) SpellSlotChoiceSpellcasting {
	return SpellSlotChoiceSpellcasting{level: l}
}

// Level returns the slot level.
func (c SpellSlotChoiceSpellcasting) Level() SlotLevel { return c.level }

// SpellSlotChoicePactMagic: a Pact Magic slot.
type SpellSlotChoicePactMagic struct{}

func (SpellSlotChoiceSpellcasting) isSpellSlotChoice() {}
func (SpellSlotChoicePactMagic) isSpellSlotChoice()    {}

// CastingInProgress is a spell whose casting takes more than one action and is
// not finished: the spell, its source, the class, the casting time mode chosen,
// the slot chosen when there is one, and the whole minutes the casting takes. The
// zero value means none. A casting from a scroll or a feature with no casting
// class has the zero class. Comparable and safe for concurrent use.
type CastingInProgress struct {
	spell   content.SpellID
	source  CastSource
	class   SpellcastingClass
	mode    content.CastingTimeMode
	slot    SpellSlotChoice
	minutes int
}

// NewCastingInProgress returns the casting. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell id, an invalid class (the zero
// class is allowed only for CastSourceScroll and CastSourceFeatureGranted), an
// invalid source, CastSourceDuplicated or fewer than 1 minute. A nil slot or a zero mode means
// none was chosen. It never panics.
func NewCastingInProgress(spell content.SpellID, source CastSource, class SpellcastingClass, mode content.CastingTimeMode, slot SpellSlotChoice, totalMinutes int) (CastingInProgress, error) {
	classless := class == 0 && (source == CastSourceScroll || source == CastSourceFeatureGranted)
	if !spell.Valid() || (!class.Valid() && !classless) || source < CastSourceSlot || source >= CastSourceDuplicated || totalMinutes < 1 {
		return CastingInProgress{}, invalidInput("casting in progress: invalid spell, class, source or minutes")
	}
	return CastingInProgress{spell: spell, source: source, class: class, mode: mode, slot: slot, minutes: totalMinutes}, nil
}

// IsZero reports whether c is the zero value (nothing being cast).
func (c CastingInProgress) IsZero() bool { return !c.spell.Valid() }

// Spell returns the spell id.
func (c CastingInProgress) Spell() content.SpellID { return c.spell }

// Source returns the source.
func (c CastingInProgress) Source() CastSource { return c.source }

// Class returns the casting class.
func (c CastingInProgress) Class() SpellcastingClass { return c.class }

// Mode returns the casting time mode; the zero mode when none was chosen.
func (c CastingInProgress) Mode() content.CastingTimeMode { return c.mode }

// Slot returns the slot chosen, nil when none.
func (c CastingInProgress) Slot() SpellSlotChoice { return c.slot }

// TotalMinutes returns the whole minutes the casting takes, at least 1.
func (c CastingInProgress) TotalMinutes() int { return c.minutes }
