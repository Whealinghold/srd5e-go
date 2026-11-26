package combat

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// ActiveEffectID is the id of one active effect, supplied by the host: the module never
// generates ids (active-effects.md FR-001). It is comparable and is not trimmed or case-folded.
// The zero ActiveEffectID is invalid. Safe for concurrent use.
type ActiveEffectID string

// NewActiveEffectID returns value unchanged as an ActiveEffectID. It returns an error matching
// srderr.ErrInvalidInput when strings.TrimSpace(value) is empty. It never panics.
func NewActiveEffectID(value string) (ActiveEffectID, error) {
	if strings.TrimSpace(value) == "" {
		return "", invalidInput("active effect id is blank")
	}
	return ActiveEffectID(value), nil
}

// MustActiveEffectID is NewActiveEffectID for literals known to be valid. It panics with the
// error NewActiveEffectID returns, and only then.
func MustActiveEffectID(value string) ActiveEffectID {
	id, err := NewActiveEffectID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// IsZero reports whether id is the invalid zero ActiveEffectID.
func (id ActiveEffectID) IsZero() bool { return id == "" }

// EffectPlaceID is the opaque host id of a place an effect is on (a sphere, a bead). It carries
// no coordinates, size or shape (FR-004). The zero EffectPlaceID is invalid. Safe for concurrent use.
type EffectPlaceID string

// NewEffectPlaceID returns value unchanged as an EffectPlaceID. It returns an error matching
// srderr.ErrInvalidInput when strings.TrimSpace(value) is empty. It never panics.
func NewEffectPlaceID(value string) (EffectPlaceID, error) {
	if strings.TrimSpace(value) == "" {
		return "", invalidInput("effect place id is blank")
	}
	return EffectPlaceID(value), nil
}

// EffectOrigin is what an effect comes from (SRD 5.2.1, "Magical Effect", p. 185). Variants:
// SpellEffectOrigin, OtherEffectOrigin. Only spell origins combine (FR-015).
type EffectOrigin interface{ isEffectOrigin() }

// SpellEffectOrigin is an effect of a spell cast at a level. Build it with NewSpellEffectOrigin;
// the zero value is invalid. Immutable and safe for concurrent use.
type SpellEffectOrigin struct {
	spell content.SpellID
	level int
}

// NewSpellEffectOrigin returns the origin of spell cast at castAtLevel (0 to 9). It returns an
// error matching srderr.ErrInvalidInput for an invalid spell id or a level outside 0 to 9. It
// never panics.
func NewSpellEffectOrigin(spell content.SpellID, castAtLevel int) (SpellEffectOrigin, error) {
	if !spell.Valid() || castAtLevel < 0 || castAtLevel > 9 {
		return SpellEffectOrigin{}, invalidInput("spell effect origin needs a spell and a level of 0 to 9")
	}
	return SpellEffectOrigin{spell, castAtLevel}, nil
}

// SpellID returns the spell.
func (o SpellEffectOrigin) SpellID() content.SpellID { return o.spell }

// CastAtLevel returns the level the spell was cast at.
func (o SpellEffectOrigin) CastAtLevel() int { return o.level }

// OtherEffectOrigin is an effect of a magic item or phenomenon, named by a host key. It never
// combines. Build it with NewOtherEffectOrigin; the zero value is invalid.
type OtherEffectOrigin struct{ key string }

// NewOtherEffectOrigin returns the origin named key. It returns an error matching
// srderr.ErrInvalidInput for a blank key. It never panics.
func NewOtherEffectOrigin(key string) (OtherEffectOrigin, error) {
	if strings.TrimSpace(key) == "" {
		return OtherEffectOrigin{}, invalidInput("other effect origin key is blank")
	}
	return OtherEffectOrigin{key}, nil
}

// Key returns the host key.
func (o OtherEffectOrigin) Key() string { return o.key }

func (SpellEffectOrigin) isEffectOrigin() {}
func (OtherEffectOrigin) isEffectOrigin() {}

// EffectSubject is what an effect is on (SRD 5.2.1, "Targets", p. 106). Variants:
// CreatureEffectSubject, PlaceEffectSubject.
type EffectSubject interface{ isEffectSubject() }

// CreatureEffectSubject is an effect on a creature. Build it with NewCreatureEffectSubject; the
// zero value is invalid. No function of this package returns a CombatantID, so read the id with Is
// or from an event's Combatants.
type CreatureEffectSubject struct{ id CombatantID }

// NewCreatureEffectSubject returns the subject id. It returns an error matching
// srderr.ErrInvalidInput for a zero id. It never panics.
func NewCreatureEffectSubject(id CombatantID) (CreatureEffectSubject, error) {
	if id.IsZero() {
		return CreatureEffectSubject{}, invalidInput("creature effect subject has a zero id")
	}
	return CreatureEffectSubject{id}, nil
}

// Is reports whether the subject is the creature id.
func (s CreatureEffectSubject) Is(id CombatantID) bool { return s.id == id }

// PlaceEffectSubject is an effect on a place (a sphere, a bead). The zero value is invalid.
type PlaceEffectSubject struct{ place EffectPlaceID }

// NewPlaceEffectSubject returns the subject place. It returns an error matching
// srderr.ErrInvalidInput for a blank place. It never panics.
func NewPlaceEffectSubject(place EffectPlaceID) (PlaceEffectSubject, error) {
	if strings.TrimSpace(string(place)) == "" {
		return PlaceEffectSubject{}, invalidInput("place effect subject is blank")
	}
	return PlaceEffectSubject{place}, nil
}

// Place returns the host id of the place.
func (s PlaceEffectSubject) Place() EffectPlaceID { return s.place }

func (CreatureEffectSubject) isEffectSubject() {}
func (PlaceEffectSubject) isEffectSubject()    {}

// creatureOf returns the creature subject of s, if it is one. It returns the subject, not its id, because no
// function of this package returns a CombatantID.
func creatureOf(s EffectSubject) (CreatureEffectSubject, bool) {
	c, ok := s.(CreatureEffectSubject)
	return c, ok
}

// validEffectSubject reports whether s is a variant its constructor would accept.
func validEffectSubject(s EffectSubject) bool {
	switch v := s.(type) {
	case CreatureEffectSubject:
		return !v.id.IsZero()
	case PlaceEffectSubject:
		return strings.TrimSpace(string(v.place)) != ""
	}
	return false
}

// ConcentrationBinding ties an effect to its caster's Concentration on a subject (SRD 5.2.1,
// "Concentration", p. 179). The zero value is the absence of a binding. Build one with
// NewConcentrationBinding. Comparable and safe for concurrent use.
type ConcentrationBinding struct {
	caster  CombatantID
	subject character.ConcentrationSubject
}

// NewConcentrationBinding returns the binding of caster's Concentration on subject. It returns an
// error matching srderr.ErrInvalidInput for a zero caster or a nil subject. It never panics.
func NewConcentrationBinding(caster CombatantID, subject character.ConcentrationSubject) (ConcentrationBinding, error) {
	if caster.IsZero() || subject == nil {
		return ConcentrationBinding{}, invalidInput("concentration binding needs a caster and a subject")
	}
	return ConcentrationBinding{caster, subject}, nil
}

// IsZero reports whether b is the absence of a binding.
func (b ConcentrationBinding) IsZero() bool { return b.subject == nil }

// CasterIs reports whether the binding is the Concentration of caster.
func (b ConcentrationBinding) CasterIs(caster CombatantID) bool { return b.caster == caster }

// Subject returns what the caster concentrates on, nil for the zero binding.
func (b ConcentrationBinding) Subject() character.ConcentrationSubject { return b.subject }
