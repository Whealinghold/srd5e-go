package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// The bounds of a level prerequisite; an ability score prerequisite is
// bounded by ability.MinScore and ability.MaxScore.
const (
	prerequisiteMinLevel = 1
	prerequisiteMaxLevel = 20
)

// Prerequisite is the closed set of feat prerequisites (SRD 5.2.1, "Parts of
// a Feat: Prerequisite", p. 87). Variants: LevelPrerequisite,
// AbilityScorePrerequisite, FeaturePrerequisite. Consumers switch over it
// exhaustively; the variants are immutable and safe for concurrent use.
type Prerequisite interface{ isPrerequisite() }

// LevelPrerequisite requires a minimum character level. The zero value is
// invalid; build it with [NewLevelPrerequisite]. Comparable.
type LevelPrerequisite struct{ minimum int }

// NewLevelPrerequisite returns a prerequisite of the given minimum level, an
// integer from 1 to 20.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange otherwise. It never panics.
func NewLevelPrerequisite(minimum int) (LevelPrerequisite, error) {
	if minimum < prerequisiteMinLevel || minimum > prerequisiteMaxLevel {
		return LevelPrerequisite{}, blockError("LevelPrerequisite", BlockOutOfRange)
	}
	return LevelPrerequisite{minimum: minimum}, nil
}

// Minimum returns the minimum level.
func (l LevelPrerequisite) Minimum() int { return l.minimum }

func (LevelPrerequisite) isPrerequisite() {}

// AbilityScorePrerequisite requires a minimum score in any one of a set of
// abilities (for example "Strength or Dexterity 13"). The zero value is
// invalid; build it with [NewAbilityScorePrerequisite]. It holds a slice, so
// it is compared with [AbilityScorePrerequisite.Equal], not ==.
type AbilityScorePrerequisite struct {
	abilities []ability.Ability
	minimum   int
}

// NewAbilityScorePrerequisite returns a prerequisite that any one of
// abilities has a score of at least minimum. The abilities are copied and
// kept in order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockEmpty for
// no abilities, BlockInvalidMember for an invalid ability, BlockDuplicate for
// a repeated ability (each checked in slice order), and BlockOutOfRange when
// minimum is outside ability.MinScore to ability.MaxScore. It never panics.
func NewAbilityScorePrerequisite(abilities []ability.Ability, minimum int) (AbilityScorePrerequisite, error) {
	const block = "AbilityScorePrerequisite"
	if len(abilities) == 0 { // equivalent mutant: == and <= agree for a length
		return AbilityScorePrerequisite{}, blockError(block, BlockEmpty)
	}
	for i, a := range abilities {
		if !a.Valid() {
			return AbilityScorePrerequisite{}, blockError(block, BlockInvalidMember)
		}
		if slices.Contains(abilities[:i], a) {
			return AbilityScorePrerequisite{}, blockError(block, BlockDuplicate)
		}
	}
	if minimum < ability.MinScore || minimum > ability.MaxScore {
		return AbilityScorePrerequisite{}, blockError(block, BlockOutOfRange)
	}
	return AbilityScorePrerequisite{abilities: slices.Clone(abilities), minimum: minimum}, nil
}

// Abilities returns the abilities, any one of which suffices, as a new slice.
func (a AbilityScorePrerequisite) Abilities() []ability.Ability { return slices.Clone(a.abilities) }

// Minimum returns the minimum ability score.
func (a AbilityScorePrerequisite) Minimum() int { return a.minimum }

// Equal reports whether a and b have the same abilities in the same order and
// the same minimum.
func (a AbilityScorePrerequisite) Equal(b AbilityScorePrerequisite) bool {
	return a.minimum == b.minimum && slices.Equal(a.abilities, b.abilities)
}

func (AbilityScorePrerequisite) isPrerequisite() {}

// FeaturePrerequisite requires a named class feature, such as Spellcasting.
// The zero value is invalid; build it with [NewFeaturePrerequisite].
// Comparable.
type FeaturePrerequisite struct{ name string }

// NewFeaturePrerequisite returns a prerequisite of the feature called name,
// which must be non-empty with no leading or trailing white space.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName otherwise. It never panics.
func NewFeaturePrerequisite(name string) (FeaturePrerequisite, error) {
	if err := checkName("FeaturePrerequisite", name); err != nil {
		return FeaturePrerequisite{}, err
	}
	return FeaturePrerequisite{name: name}, nil
}

// Name returns the feature name.
func (f FeaturePrerequisite) Name() string { return f.name }

func (FeaturePrerequisite) isPrerequisite() {}
