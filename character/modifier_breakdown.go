package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// DerivedStatSource says where one term of a ModifierBreakdown comes from. The
// zero value is invalid.
type DerivedStatSource int

// The sources of a breakdown term.
const (
	// BaseValueTerm: a printed base such as the 10 of Armor Class or Passive Perception.
	BaseValueTerm DerivedStatSource = iota + 1
	// AbilityModifierTerm: an ability modifier; the term names the Ability.
	AbilityModifierTerm
	// ProficiencyBonusTerm: the Proficiency Bonus.
	ProficiencyBonusTerm
	// ExpertiseTerm: the Proficiency Bonus doubled by Expertise.
	ExpertiseTerm
	// ArmorBaseTerm: the listed base of body armor.
	ArmorBaseTerm
	// ShieldBonusTerm: the Armor Class bonus of a trained Shield.
	ShieldBonusTerm
	// RollModeTerm: the 5 that Advantage or Disadvantage adds to or takes from a passive score.
	RollModeTerm
	// FlatBonusTerm: a flat bonus; the term names its FlatBonusSource.
	FlatBonusTerm
	// StoredHitPointMaximumTerm: the stored Hit Point maximum.
	StoredHitPointMaximumTerm
	// HitPointMaximumBonusTerm: a Hit Point maximum bonus feature.
	HitPointMaximumBonusTerm
	// FloorTerm: the lift of an Armor Class floor.
	FloorTerm
)

// String returns the lower camel case name of s, or "unknown".
func (s DerivedStatSource) String() string {
	switch s {
	case BaseValueTerm:
		return "baseValue"
	case AbilityModifierTerm:
		return "abilityModifier"
	case ProficiencyBonusTerm:
		return "proficiencyBonus"
	case ExpertiseTerm:
		return "expertise"
	case ArmorBaseTerm:
		return "armorBase"
	case ShieldBonusTerm:
		return "shieldBonus"
	case RollModeTerm:
		return "rollMode"
	case FlatBonusTerm:
		return "flatBonus"
	case StoredHitPointMaximumTerm:
		return "storedHitPointMaximum"
	case HitPointMaximumBonusTerm:
		return "hitPointMaximumBonus"
	case FloorTerm:
		return "floor"
	}
	return "unknown"
}

// BreakdownTerm is one term of a ModifierBreakdown. Ability is set only for an
// AbilityModifierTerm and Origin only for a FlatBonusTerm. Construct it with a
// keyed literal.
type BreakdownTerm struct {
	Modifier modifier.Modifier
	Source   DerivedStatSource
	Ability  ability.Ability
	Origin   FlatBonusSource
}

// ModifierBreakdown is the ordered list of terms that make up a derived number;
// the breakdown is the single source of that number. It is immutable and
// defensively copied; it holds a slice, so compare two with Equal, not ==. The
// zero value has no terms and a total of 0. Safe for concurrent use.
type ModifierBreakdown struct{ terms []BreakdownTerm }

// NewModifierBreakdown returns the breakdown of terms, copied.
//
// It returns an errors.Join of errors matching srderr.ErrInvalidInput, one per
// violation, for: more than one ProficiencyBonusTerm or ExpertiseTerm together
// ("The Bonus Doesn't Stack", SRD 5.2.1, p. 8); an AbilityModifierTerm without
// an Ability; an Ability on another term; a FlatBonusTerm without an Origin; an
// Origin on another term; and an invalid Source. Several AbilityModifierTerms
// are allowed. It never panics.
func NewModifierBreakdown(terms ...BreakdownTerm) (ModifierBreakdown, error) {
	var problems []error
	bad := func(what string) { problems = append(problems, invalidInput(what)) }
	proficiency := 0
	for _, t := range terms {
		switch t.Source {
		case ProficiencyBonusTerm, ExpertiseTerm:
			proficiency++
		case BaseValueTerm, AbilityModifierTerm, ArmorBaseTerm, ShieldBonusTerm, RollModeTerm, FlatBonusTerm,
			StoredHitPointMaximumTerm, HitPointMaximumBonusTerm, FloorTerm:
		default:
			bad("breakdown term with an invalid source")
		}
		if (t.Source == AbilityModifierTerm) != (t.Ability != 0) {
			bad("breakdown term ability does not match its source")
		}
		if (t.Source == FlatBonusTerm) != (t.Origin != nil) {
			bad("breakdown term origin does not match its source")
		}
	}
	if proficiency > 1 {
		bad("more than one Proficiency Bonus term")
	}
	if len(problems) > 0 {
		return ModifierBreakdown{}, errors.Join(problems...)
	}
	return ModifierBreakdown{terms: slices.Clone(terms)}, nil
}

// Terms returns a new slice of the terms in order.
func (b ModifierBreakdown) Terms() []BreakdownTerm { return slices.Clone(b.terms) }

// Total returns the sum of the terms.
func (b ModifierBreakdown) Total() modifier.Modifier {
	var sum modifier.Modifier
	for _, t := range b.terms {
		sum = sum.Add(t.Modifier)
	}
	return sum
}

// Equal reports whether b and o hold the same terms in the same order.
func (b ModifierBreakdown) Equal(o ModifierBreakdown) bool { return slices.Equal(b.terms, o.terms) }

// BreakdownMappingProblem names a reason a breakdown cannot be a D20 Test's
// modifiers. The zero value is not a problem.
type BreakdownMappingProblem int

// The breakdown mapping problems, in the order ToRollModifiers reports them.
const (
	// BreakdownMissingAbilityTerm: there is no AbilityModifierTerm.
	BreakdownMissingAbilityTerm BreakdownMappingProblem = iota + 1
	// BreakdownMultipleAbilityTerms: there is more than one AbilityModifierTerm.
	BreakdownMultipleAbilityTerms
	// BreakdownUnmappableSource: a term's source has no place in a D20 Test.
	BreakdownUnmappableSource
)

// String returns "missingAbilityTerm", "multipleAbilityTerms",
// "unmappableSource" or "unknown".
func (p BreakdownMappingProblem) String() string {
	switch p {
	case BreakdownMissingAbilityTerm:
		return "missingAbilityTerm"
	case BreakdownMultipleAbilityTerms:
		return "multipleAbilityTerms"
	case BreakdownUnmappableSource:
		return "unmappableSource"
	}
	return "unknown"
}

// BreakdownMappingError reports one problem of ModifierBreakdown.ToRollModifiers.
// It matches srderr.ErrInvalidInput with errors.Is.
type BreakdownMappingError struct {
	Problem BreakdownMappingProblem
}

// Error renders the error. The text is not a contract.
func (e *BreakdownMappingError) Error() string {
	return "character: breakdown is not a D20 Test: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *BreakdownMappingError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *BreakdownMappingError with the same Problem.
func (e *BreakdownMappingError) Is(target error) bool {
	t, ok := target.(*BreakdownMappingError)
	return ok && t != nil && e != nil && t.Problem == e.Problem
}

// ToRollModifiers maps the breakdown to the modifiers of a D20 Test (SRD 5.2.1,
// "D20 Tests", "Add Modifiers", p. 6): the one ability term is the ability
// modifier, the one ProficiencyBonusTerm or ExpertiseTerm is the Proficiency
// Bonus slot and each FlatBonusTerm is one other modifier, in term order. A
// saving throw or skill breakdown maps; Passive Perception, Armor Class and the
// Hit Point maximum do not. This is the only mapping from a breakdown to a D20
// Test.
//
// It returns an errors.Join of one *BreakdownMappingError per problem found,
// each at most once, in the order of BreakdownMappingProblem, each matching
// srderr.ErrInvalidInput. It never panics.
func (b ModifierBreakdown) ToRollModifiers() (d20.RollModifiers, error) {
	var abilityTerm, proficiency *modifier.Modifier
	abilities, unmappable := 0, false
	var others []modifier.Modifier
	for _, t := range b.terms {
		m := t.Modifier
		switch t.Source { //nolint:exhaustive // every other source has no place in a D20 Test
		case AbilityModifierTerm:
			abilities++
			abilityTerm = &m
		case ProficiencyBonusTerm, ExpertiseTerm:
			proficiency = &m
		case FlatBonusTerm:
			others = append(others, m)
		default:
			unmappable = true
		}
	}
	var problems []error
	if abilities == 0 {
		problems = append(problems, &BreakdownMappingError{Problem: BreakdownMissingAbilityTerm})
	}
	if abilities > 1 {
		problems = append(problems, &BreakdownMappingError{Problem: BreakdownMultipleAbilityTerms})
	}
	if unmappable {
		problems = append(problems, &BreakdownMappingError{Problem: BreakdownUnmappableSource})
	}
	if len(problems) > 0 {
		return d20.RollModifiers{}, errors.Join(problems...)
	}
	rm := d20.NewRollModifiers(*abilityTerm)
	if proficiency != nil {
		rm = rm.WithProficiencyBonus(*proficiency)
	}
	return rm.WithOthers(others...), nil
}
