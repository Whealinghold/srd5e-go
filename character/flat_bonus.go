package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// FlatBonusSource is the typed origin of a flat bonus. It is a closed set.
// Variants: SpellFlatBonusSource, MagicItemFlatBonusSource,
// FeatureFlatBonusSource.
type FlatBonusSource interface{ isFlatBonusSource() }

// SpellFlatBonusSource is a bonus a spell gives, such as Shield of Faith (SRD
// 5.2.1, p. 162).
type SpellFlatBonusSource struct{ Spell content.SpellID }

// MagicItemFlatBonusSource is a bonus a magic item gives, such as a Ring of
// Protection (SRD 5.2.1, p. 238).
type MagicItemFlatBonusSource struct{ Item content.MagicItemID }

// FeatureFlatBonusSource is a bonus a class, species or feat feature gives.
type FeatureFlatBonusSource struct{ Feature content.FeatureID }

func (SpellFlatBonusSource) isFlatBonusSource()     {}
func (MagicItemFlatBonusSource) isFlatBonusSource() {}
func (FeatureFlatBonusSource) isFlatBonusSource()   {}

// FlatBonus is a signed bonus and where it comes from. It is an immutable
// value, comparable with ==. The zero FlatBonus is invalid (no source).
type FlatBonus struct {
	modifier modifier.Modifier
	source   FlatBonusSource
}

// NewFlatBonus returns a flat bonus of m from source. It returns an error
// matching srderr.ErrInvalidInput for a nil source or a source that is not one
// of the FlatBonusSource variants. It never panics.
func NewFlatBonus(m modifier.Modifier, source FlatBonusSource) (FlatBonus, error) {
	switch source.(type) {
	case SpellFlatBonusSource, MagicItemFlatBonusSource, FeatureFlatBonusSource:
		return FlatBonus{modifier: m, source: source}, nil
	}
	return FlatBonus{}, invalidInput("flat bonus without a source")
}

// Modifier returns the bonus.
func (b FlatBonus) Modifier() modifier.Modifier { return b.modifier }

// Source returns the origin; nil for the zero FlatBonus.
func (b FlatBonus) Source() FlatBonusSource { return b.source }

// IsZero reports whether b is the invalid zero FlatBonus.
func (b FlatBonus) IsZero() bool { return b.source == nil }

// FlatSavingThrowBonus is a flat bonus to saving throws, to all six or to a
// chosen set (SRD 5.2.1, "Ring of Protection", p. 238). It holds a slice, so
// compare two with care. The zero value is invalid.
type FlatSavingThrowBonus struct {
	bonus     FlatBonus
	abilities []ability.Ability // empty: all six
}

// NewFlatSavingThrowBonus returns a bonus to the saves of abilities. A nil
// abilities means all six saves.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid bonus, an
// empty non-nil abilities or an invalid ability. It never panics.
func NewFlatSavingThrowBonus(b FlatBonus, abilities []ability.Ability) (FlatSavingThrowBonus, error) {
	if b.IsZero() {
		return FlatSavingThrowBonus{}, invalidInput("flat saving throw bonus without a source")
	}
	if abilities != nil && len(abilities) == 0 {
		return FlatSavingThrowBonus{}, invalidInput("empty ability scope")
	}
	if slices.ContainsFunc(abilities, func(a ability.Ability) bool { return !validAbility(a) }) {
		return FlatSavingThrowBonus{}, invalidInput("invalid ability in scope")
	}
	return FlatSavingThrowBonus{bonus: b, abilities: slices.Clone(abilities)}, nil
}

// Bonus returns the flat bonus.
func (b FlatSavingThrowBonus) Bonus() FlatBonus { return b.bonus }

// IsZero reports whether b is the invalid zero value.
func (b FlatSavingThrowBonus) IsZero() bool { return b.bonus.IsZero() }

// Covers reports whether the bonus applies to the save of a.
func (b FlatSavingThrowBonus) Covers(a ability.Ability) bool {
	return len(b.abilities) == 0 || slices.Contains(b.abilities, a)
}

func validAbility(a ability.Ability) bool { return a >= ability.Strength && a <= ability.Charisma }
