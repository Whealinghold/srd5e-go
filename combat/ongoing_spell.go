package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpellOrigin is where an ongoing spell comes from. Antimagic Field suppresses
// an ordinary spell and not one cast by an Artifact or a deity (SRD 5.2.1,
// "Antimagic Field", p. 109). The zero value is invalid.
type SpellOrigin int

// The origins of an ongoing spell.
const (
	// SpellOriginOrdinary: a spell cast by a creature.
	SpellOriginOrdinary SpellOrigin = iota + 1
	// SpellOriginArtifactOrDeity: a spell cast by an Artifact or a deity.
	SpellOriginArtifactOrDeity
)

// Valid reports whether o is an origin.
func (o SpellOrigin) Valid() bool { return o == SpellOriginOrdinary || o == SpellOriginArtifactOrDeity }

// String returns "ordinary", "artifact_or_deity" or "unknown".
func (o SpellOrigin) String() string {
	return enumName([]string{"ordinary", "artifact_or_deity"}, int(o))
}

// OngoingSpell is a spell that is in effect on one target: an OngoingEffect of
// category spell, the level it was cast at, its caster, origin, remaining time,
// dispel restriction, the parent it is a part of, and the conditions it imposes
// on its holder (SRD 5.2.1, "Spells", p. 106). It is immutable and equal by value
// over every field. Build it with NewOngoingSpell; the zero value is invalid.
// Safe for concurrent use.
type OngoingSpell struct {
	effect                OngoingEffect
	castAtLevel           content.SpellLevel
	caster                CombatantID
	origin                SpellOrigin
	remaining             SpellTimeRemaining
	restriction           DispelRestriction
	partOf                OngoingEffectID
	conditions            []content.ConditionKind
	requiresConcentration bool
}

// OngoingSpellParams is the input of NewOngoingSpell. Effect must be an
// OngoingEffect of category spell with a spell source. CastAtLevel is the level
// of the slot the spell was cast with when that is higher than its own level, and
// the spell's own level otherwise (a cantrip is 0). Caster may be the zero
// CombatantID when unknown. PartOf is the parent effect, or the zero id.
// Conditions are copied.
type OngoingSpellParams struct {
	Effect                OngoingEffect
	CastAtLevel           content.SpellLevel
	Caster                CombatantID
	Origin                SpellOrigin
	Remaining             SpellTimeRemaining
	Restriction           DispelRestriction
	PartOf                OngoingEffectID
	Conditions            []content.ConditionKind
	RequiresConcentration bool
}

// NewOngoingSpell returns the ongoing spell described by p. It returns an error
// matching srderr.ErrInvalidInput when the effect is not a spell effect with a
// spell source, the cast level is not a spell level, the origin is not defined,
// the remaining time is the zero value, the restriction is nil, or the spell
// names itself as its parent. It copies Conditions and never panics.
func NewOngoingSpell(p OngoingSpellParams) (OngoingSpell, error) {
	switch {
	case p.Effect.Category() != OngoingEffectSpell || p.Effect.Source().Kind() != EffectSourceSpell:
		return OngoingSpell{}, invalidInput("an ongoing spell needs a spell effect with a spell source")
	case !p.CastAtLevel.Valid():
		return OngoingSpell{}, invalidInput("an ongoing spell needs a spell level")
	case !p.Origin.Valid():
		return OngoingSpell{}, invalidInput("an ongoing spell needs an origin")
	case p.Remaining.IsZero():
		return OngoingSpell{}, invalidInput("an ongoing spell needs its remaining time")
	case p.Restriction == nil:
		return OngoingSpell{}, invalidInput("an ongoing spell needs a dispel restriction")
	case p.PartOf == p.Effect.ID() && !p.PartOf.IsZero():
		return OngoingSpell{}, invalidInput("a spell cannot be a part of itself")
	}
	return OngoingSpell{
		effect:                p.Effect,
		castAtLevel:           p.CastAtLevel,
		caster:                p.Caster,
		origin:                p.Origin,
		remaining:             p.Remaining,
		restriction:           p.Restriction,
		partOf:                p.PartOf,
		conditions:            slices.Clone(p.Conditions),
		requiresConcentration: p.RequiresConcentration,
	}, nil
}

// Effect returns the ongoing effect of the spell.
func (s OngoingSpell) Effect() OngoingEffect { return s.effect }

// SpellID returns the id of the spell.
func (s OngoingSpell) SpellID() content.SpellID { return content.SpellID(s.effect.Source().ID()) }

// CastAtLevel returns the level the spell took when it was cast.
func (s OngoingSpell) CastAtLevel() content.SpellLevel { return s.castAtLevel }

// Origin returns where the spell comes from.
func (s OngoingSpell) Origin() SpellOrigin { return s.origin }

// Remaining returns the time left on the spell.
func (s OngoingSpell) Remaining() SpellTimeRemaining { return s.remaining }

// Restriction returns the dispel restriction of the spell.
func (s OngoingSpell) Restriction() DispelRestriction { return s.restriction }

// PartOf returns the id of the parent effect the spell is a part of, or the zero id.
func (s OngoingSpell) PartOf() OngoingEffectID { return s.partOf }

// Conditions returns a copy of the conditions the spell imposes on its holder.
func (s OngoingSpell) Conditions() []content.ConditionKind { return slices.Clone(s.conditions) }

// RequiresConcentration reports whether the spell requires Concentration.
func (s OngoingSpell) RequiresConcentration() bool { return s.requiresConcentration }

// IsZero reports whether s is the invalid zero value.
func (s OngoingSpell) IsZero() bool { return s.effect.IsZero() }

// Equal reports whether s and o are equal in every field.
func (s OngoingSpell) Equal(o OngoingSpell) bool {
	return s.effect == o.effect &&
		s.castAtLevel == o.castAtLevel &&
		s.caster == o.caster &&
		s.origin == o.origin &&
		s.remaining == o.remaining &&
		dispelRestrictionEqual(s.restriction, o.restriction) &&
		s.partOf == o.partOf &&
		slices.Equal(s.conditions, o.conditions) &&
		s.requiresConcentration == o.requiresConcentration
}
