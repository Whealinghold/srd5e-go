package character

import "github.com/Whealinghold/srd5e-go/content"

// CastingEvent is the closed family of events of casting spells and of
// Concentration. It is an Event, so a host switches over it for the family and
// over Event for the context. Events are past tense and immutable and carry the
// Identifier of the character whose SpellcastingState produced them. SpellSlotExpended
// belongs to SpellcastingEvent and appears in the same lists. Members: SpellCast,
// CastingCountered, ConcentrationBegan, ConcentrationEnded, CastingFailed.
type CastingEvent interface {
	Event
	isCastingEvent()
}

// ConcentrationEndReason is why Concentration ended: a ConcentrationBreakReason
// value or CastingCompleted. The zero value is invalid. Comparable and safe for
// concurrent use.
type ConcentrationEndReason int

// The end reasons; the first five equal the ConcentrationBreakReason values.
const (
	// EndedIncapacitated: the caster became Incapacitated.
	EndedIncapacitated ConcentrationEndReason = iota + 1
	// EndedDead: the caster died.
	EndedDead
	// EndedByCaster: the caster ended it (or the host reports a loss such as Rage).
	EndedByCaster
	// EndedFailedSave: the Constitution saving throw after damage failed.
	EndedFailedSave
	// EndedAnotherEffect: another effect that needs Concentration began.
	EndedAnotherEffect
	// EndedCastingCompleted: a longer casting finished and the spell needs no Concentration.
	EndedCastingCompleted
)

// String returns the snake_case name or "unknown".
func (r ConcentrationEndReason) String() string {
	names := [...]string{"incapacitated", "dead", "ended", "failed_save", "another_effect", "casting_completed"}
	if r < EndedIncapacitated || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// ConcentrationBreakReason is why a caller ends Concentration (SRD 5.2.1,
// "Concentration", p. 179). CastingCompleted is not one. The zero value is
// invalid. Comparable and safe for concurrent use.
type ConcentrationBreakReason int

// The break reasons.
const (
	// BreakIncapacitated: the caster is Incapacitated.
	BreakIncapacitated ConcentrationBreakReason = iota + 1
	// BreakDead: the caster died.
	BreakDead
	// BreakEnded: the caster ends Concentration, or the host reports a loss such as Rage.
	BreakEnded
	// BreakFailedSave: the Constitution saving throw after damage failed.
	BreakFailedSave
	// BreakAnotherEffect: another effect that needs Concentration began.
	BreakAnotherEffect
)

// String returns the snake_case name or "unknown".
func (r ConcentrationBreakReason) String() string {
	if !r.Valid() {
		return "unknown"
	}
	return ConcentrationEndReason(r).String()
}

// Valid reports whether r is one of the five break reasons.
func (r ConcentrationBreakReason) Valid() bool {
	return r >= BreakIncapacitated && r <= BreakAnotherEffect
}

// SpellCast is the event of a spell being cast. Class is absent for a cast with
// no casting class; CastAtLevel is 0 for a cantrip or a Ritual.
type SpellCast struct {
	id           Identifier
	spell        content.SpellID
	class        SpellcastingClass
	source       CastSource
	castAt       int
	scalingLevel int
}

// CharacterID returns the id of the character.
func (e SpellCast) CharacterID() Identifier { return e.id }

// Spell returns the spell id.
func (e SpellCast) Spell() content.SpellID { return e.spell }

// Class returns the casting class and whether there is one.
func (e SpellCast) Class() (SpellcastingClass, bool) { return e.class, e.class.Valid() }

// Source returns the source.
func (e SpellCast) Source() CastSource { return e.source }

// CastAtLevel returns the level the spell was cast at and whether there is one.
func (e SpellCast) CastAtLevel() (int, bool) { return e.castAt, e.castAt > 0 }

// ScalingLevel returns the level the spell scales with and whether there is one.
func (e SpellCast) ScalingLevel() (int, bool) { return e.scalingLevel, e.scalingLevel > 0 }

// CastingCountered is the event of a cast being countered ("Counterspell", p. 120).
type CastingCountered struct {
	id    Identifier
	spell content.SpellID
}

// CharacterID returns the id of the character.
func (e CastingCountered) CharacterID() Identifier { return e.id }

// Spell returns the spell id.
func (e CastingCountered) Spell() content.SpellID { return e.spell }

// ConcentrationBegan is the event of Concentration beginning.
type ConcentrationBegan struct {
	id      Identifier
	subject ConcentrationSubject
	kind    ConcentrationKind
}

// CharacterID returns the id of the character.
func (e ConcentrationBegan) CharacterID() Identifier { return e.id }

// Subject returns what is concentrated on.
func (e ConcentrationBegan) Subject() ConcentrationSubject { return e.subject }

// Kind returns how Concentration began.
func (e ConcentrationBegan) Kind() ConcentrationKind { return e.kind }

// ConcentrationEnded is the event of Concentration ending.
type ConcentrationEnded struct {
	id      Identifier
	subject ConcentrationSubject
	reason  ConcentrationEndReason
}

// CharacterID returns the id of the character.
func (e ConcentrationEnded) CharacterID() Identifier { return e.id }

// Subject returns what was concentrated on.
func (e ConcentrationEnded) Subject() ConcentrationSubject { return e.subject }

// Reason returns why it ended.
func (e ConcentrationEnded) Reason() ConcentrationEndReason { return e.reason }

// CastingFailed is the event of a longer casting ending without effect and
// without a slot spent, because Concentration ended ("Longer Casting Times", p. 105).
// Its reason is never EndedCastingCompleted.
type CastingFailed struct {
	id     Identifier
	spell  content.SpellID
	reason ConcentrationEndReason
}

// CharacterID returns the id of the character.
func (e CastingFailed) CharacterID() Identifier { return e.id }

// Spell returns the spell id.
func (e CastingFailed) Spell() content.SpellID { return e.spell }

// Reason returns why the casting failed.
func (e CastingFailed) Reason() ConcentrationEndReason { return e.reason }

func (SpellCast) isEvent()                 {}
func (CastingCountered) isEvent()          {}
func (ConcentrationBegan) isEvent()        {}
func (ConcentrationEnded) isEvent()        {}
func (CastingFailed) isEvent()             {}
func (SpellCast) isCastingEvent()          {}
func (CastingCountered) isCastingEvent()   {}
func (ConcentrationBegan) isCastingEvent() {}
func (ConcentrationEnded) isCastingEvent() {}
func (CastingFailed) isCastingEvent()      {}
