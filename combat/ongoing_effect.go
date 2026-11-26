package combat

import (
	"slices"
	"strings"
)

// OngoingEffectID names one non-condition effect on a creature in an
// OngoingEffects list. It is a host-chosen, non-blank string and not a content id.
// Build it with NewOngoingEffectID; the zero value is invalid. Comparable and safe
// for concurrent use.
type OngoingEffectID string

// NewOngoingEffectID returns the id for s. It returns an error matching
// srderr.ErrInvalidInput for an empty or all-whitespace s. It never panics.
func NewOngoingEffectID(s string) (OngoingEffectID, error) {
	if strings.TrimSpace(s) == "" {
		return "", invalidInput("ongoing effect id must not be blank")
	}
	return OngoingEffectID(s), nil
}

// String returns the id text.
func (id OngoingEffectID) String() string { return string(id) }

// IsZero reports whether id is the invalid zero value.
func (id OngoingEffectID) IsZero() bool { return id == "" }

// OngoingEffectCategory is the kind of an ongoing effect that is not a condition:
// a spell, a curse, a magical contagion or another Magical Effect. The zero value
// is not a category.
type OngoingEffectCategory int

// The categories of ongoing effect.
const (
	// OngoingEffectSpell: an effect of a spell; its source must be a spell.
	OngoingEffectSpell OngoingEffectCategory = iota + 1
	// OngoingEffectCurse: a curse (SRD 5.2.1, "Curses", p. 179).
	OngoingEffectCurse
	// OngoingEffectMagicalContagion: a magical contagion; its source must be magical.
	OngoingEffectMagicalContagion
	// OngoingEffectMagicalEffect: another Magical Effect; its source must be magical.
	OngoingEffectMagicalEffect
)

// String returns "spell", "curse", "magical_contagion", "magical_effect" or "unknown".
func (c OngoingEffectCategory) String() string {
	return enumName([]string{"spell", "curse", "magical_contagion", "magical_effect"}, int(c))
}

// OngoingEffect is one non-condition effect on one creature (a spell, a curse, a
// magical contagion or another Magical Effect), equal to another by value
// (SRD 5.2.1, "Shape-Shifting", p. 187: "conditions, spells, curses, and the
// like"). Conditions stay in ConditionSet. Build it with NewOngoingEffect; the
// zero value is invalid. Comparable and safe for concurrent use.
type OngoingEffect struct {
	id       OngoingEffectID
	category OngoingEffectCategory
	source   EffectSource
}

// NewOngoingEffect returns the effect. It returns an error matching
// srderr.ErrInvalidInput for a blank id, an undefined category, a zero source, a
// spell whose source is not a SpellSource, and a magical contagion or Magical
// Effect whose source is not magical (SRD 5.2.1, "Magical Effect", p. 185). It
// never panics.
func NewOngoingEffect(id OngoingEffectID, category OngoingEffectCategory, source EffectSource) (OngoingEffect, error) {
	switch {
	case strings.TrimSpace(string(id)) == "":
		return OngoingEffect{}, invalidInput("ongoing effect needs an id")
	case category < OngoingEffectSpell || category > OngoingEffectMagicalEffect:
		return OngoingEffect{}, invalidInput("ongoing effect needs a category")
	case source.IsZero():
		return OngoingEffect{}, invalidInput("ongoing effect needs a source")
	case category == OngoingEffectSpell && source.Kind() != EffectSourceSpell:
		return OngoingEffect{}, invalidInput("a spell effect needs a spell source")
	case (category == OngoingEffectMagicalContagion || category == OngoingEffectMagicalEffect) && !source.IsMagical():
		return OngoingEffect{}, invalidInput("a magical contagion or Magical Effect needs a magical source")
	}
	return OngoingEffect{id, category, source}, nil
}

// ID returns the effect id.
func (e OngoingEffect) ID() OngoingEffectID { return e.id }

// Category returns the category.
func (e OngoingEffect) Category() OngoingEffectCategory { return e.category }

// Source returns the effect source.
func (e OngoingEffect) Source() EffectSource { return e.source }

// IsZero reports whether e is the invalid zero value.
func (e OngoingEffect) IsZero() bool { return e == OngoingEffect{} }

// OngoingEffects is the ordered list of a creature's non-condition effects, with
// unique ids. The zero value is the empty list and is valid. It is immutable: the
// list given to the constructor is copied and All returns a copy. It holds a
// slice, so compare with Equal, not ==. Safe for concurrent use.
type OngoingEffects struct{ list []OngoingEffect }

// NewOngoingEffects returns the list holding effects in the order given; nil and
// empty both give the empty list. It returns an error matching
// srderr.ErrInvalidInput for a zero effect or when two entries share an id, and a LimitExceededError for more than 64 effects (plan P10: the cap of effects per creature, checked first). It
// copies effects and never panics.
func NewOngoingEffects(effects []OngoingEffect) (OngoingEffects, error) {
	if err := overCap("ongoing effects", len(effects), maxEffectsPerCreature); err != nil {
		return OngoingEffects{}, err
	}
	seen := make(map[OngoingEffectID]bool, len(effects))
	for _, e := range effects {
		if e.IsZero() {
			return OngoingEffects{}, invalidInput("ongoing effects hold a zero effect")
		}
		if seen[e.id] {
			return OngoingEffects{}, invalidInput("two ongoing effects share an id")
		}
		seen[e.id] = true
	}
	return OngoingEffects{slices.Clone(effects)}, nil
}

// All returns a copy of the effects in order.
func (l OngoingEffects) All() []OngoingEffect { return slices.Clone(l.list) }

// Len returns the number of effects.
func (l OngoingEffects) Len() int { return len(l.list) }

// Find returns the effect with the id, and whether the list holds it.
func (l OngoingEffects) Find(id OngoingEffectID) (OngoingEffect, bool) {
	for _, e := range l.list {
		if e.id == id {
			return e, true
		}
	}
	return OngoingEffect{}, false
}

// Equal reports whether l and o hold equal effects in the same order.
func (l OngoingEffects) Equal(o OngoingEffects) bool { return slices.Equal(l.list, o.list) }

// without returns the list minus the effects whose ids are in ids, order kept.
func (l OngoingEffects) without(ids []OngoingEffectID) OngoingEffects {
	var kept []OngoingEffect
	for _, e := range l.list {
		if !slices.Contains(ids, e.id) {
			kept = append(kept, e)
		}
	}
	return OngoingEffects{kept}
}
