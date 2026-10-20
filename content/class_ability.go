package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// AbilityRequirement is the Primary Ability of a class (SRD 5.2.1, "Class
// Overview", p. 19): a closed set with exactly two variants, [AllOf] ("Strength
// and Charisma") and [AnyOf] ("Strength or Dexterity"). Consumers switch over
// it exhaustively. The variants are immutable and safe for concurrent use.
type AbilityRequirement interface {
	// Abilities returns the listed abilities in printed order, as a new slice.
	Abilities() []ability.Ability
	// IsMetBy reports whether the abilities in meeting satisfy the
	// requirement: every listed ability for [AllOf], at least one for
	// [AnyOf]. Abilities in meeting that the requirement does not list are
	// ignored. Which abilities "meet" it (a score threshold) is the rule
	// owner's, never this type's.
	IsMetBy(meeting []ability.Ability) bool
	isAbilityRequirement()
}

// AllOf requires every one of its abilities. A single primary ability is an
// AllOf of one. The zero value is invalid; build it with [NewAllOf]. It holds
// a slice, so it is not comparable with ==.
type AllOf struct{ abilities []ability.Ability }

// NewAllOf returns the requirement of every one of abilities, which are
// copied and kept in order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for no ability or more than six, BlockInvalidMember for an invalid ability
// and BlockDuplicate for a repeated one. It never panics.
func NewAllOf(abilities ...ability.Ability) (AllOf, error) {
	if err := checkSet("AllOf", abilities, 1, len(ability.All()), ability.Ability.Valid); err != nil {
		return AllOf{}, err
	}
	return AllOf{abilities: slices.Clone(abilities)}, nil
}

// Abilities returns the listed abilities in printed order, as a new slice.
func (a AllOf) Abilities() []ability.Ability { return slices.Clone(a.abilities) }

// IsMetBy reports whether every listed ability is in meeting.
func (a AllOf) IsMetBy(meeting []ability.Ability) bool {
	for _, x := range a.abilities {
		if !slices.Contains(meeting, x) {
			return false
		}
	}
	return len(a.abilities) > 0 // equivalent mutant: a length is never negative, so > and != agree
}

func (AllOf) isAbilityRequirement() {}

// AnyOf requires at least one of its abilities. It lists at least two: an
// AnyOf of one is the same fact as an AllOf of one and is not representable.
// The zero value is invalid; build it with [NewAnyOf]. It holds a slice, so
// it is not comparable with ==.
type AnyOf struct{ abilities []ability.Ability }

// NewAnyOf returns the requirement of at least one of abilities, which are
// copied and kept in order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for fewer than two abilities or more than six, BlockInvalidMember for an
// invalid ability and BlockDuplicate for a repeated one. It never panics.
func NewAnyOf(abilities ...ability.Ability) (AnyOf, error) {
	if err := checkSet("AnyOf", abilities, 2, len(ability.All()), ability.Ability.Valid); err != nil {
		return AnyOf{}, err
	}
	return AnyOf{abilities: slices.Clone(abilities)}, nil
}

// Abilities returns the listed abilities in printed order, as a new slice.
func (a AnyOf) Abilities() []ability.Ability { return slices.Clone(a.abilities) }

// IsMetBy reports whether at least one listed ability is in meeting.
func (a AnyOf) IsMetBy(meeting []ability.Ability) bool {
	return slices.ContainsFunc(a.abilities, func(x ability.Ability) bool { return slices.Contains(meeting, x) })
}

func (AnyOf) isAbilityRequirement() {}

// validAbilityRequirement reports whether r is a variant built by its
// constructor.
func validAbilityRequirement(r AbilityRequirement) bool {
	switch v := r.(type) {
	case AllOf:
		_, err := NewAllOf(v.abilities...)
		return err == nil
	case AnyOf:
		_, err := NewAnyOf(v.abilities...)
		return err == nil
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}
