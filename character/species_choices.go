package character

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The nine species with a behaviour (SRD 5.2.1, "Species", pp. 83-86).
const (
	speciesDragonborn content.SpeciesID = "dragonborn"
	speciesDwarf      content.SpeciesID = "dwarf"
	speciesElf        content.SpeciesID = "elf"
	speciesGnome      content.SpeciesID = "gnome"
	speciesGoliath    content.SpeciesID = "goliath"
	speciesHalfling   content.SpeciesID = "halfling"
	speciesHuman      content.SpeciesID = "human"
	speciesOrc        content.SpeciesID = "orc"
	speciesTiefling   content.SpeciesID = "tiefling"
)

// The printed names of the traits that ask for a choice.
const (
	traitDraconicAncestry = "Draconic Ancestry"
	traitElvenLineage     = "Elven Lineage"
	traitGnomishLineage   = "Gnomish Lineage"
	traitGiantAncestry    = "Giant Ancestry"
	traitFiendishLegacy   = "Fiendish Legacy"
)

// behaviourKnown reports whether id is one of the nine species of this context.
func behaviourKnown(id content.SpeciesID) bool {
	switch id {
	case speciesDragonborn, speciesDwarf, speciesElf, speciesGnome, speciesGoliath, speciesHalfling, speciesHuman, speciesOrc, speciesTiefling:
		return true
	}
	return false
}

// SpeciesTraitChoices is what the player chose for the traits of a species, a
// closed set with one variant per species that has a choice. Variants:
// DragonbornChoices, ElfChoices, GnomeChoices, GoliathChoices, TieflingChoices,
// HumanChoices, NoSpeciesChoices (Dwarf, Halfling and Orc). All are comparable.
type SpeciesTraitChoices interface{ isSpeciesTraitChoices() }

// DragonbornChoices holds the Draconic Ancestry. Construct it with a keyed literal.
type DragonbornChoices struct{ Ancestry DraconicAncestry }

// ElfChoices holds the Elven Lineage, its spellcasting ability (Intelligence,
// Wisdom or Charisma) and the Keen Senses skill (Insight, Perception or
// Survival). Construct it with a keyed literal.
type ElfChoices struct {
	Lineage         ElvenLineage
	Ability         ability.Ability
	KeenSensesSkill content.Skill
}

// GnomeChoices holds the Gnomish Lineage and its spellcasting ability. Construct
// it with a keyed literal.
type GnomeChoices struct {
	Lineage GnomishLineage
	Ability ability.Ability
}

// GoliathChoices holds the Giant Ancestry. Construct it with a keyed literal.
type GoliathChoices struct{ Ancestry GiantAncestry }

// TieflingChoices holds the Fiendish Legacy and its spellcasting ability.
// Construct it with a keyed literal.
type TieflingChoices struct {
	Legacy  FiendishLegacy
	Ability ability.Ability
}

// HumanChoices holds the Skillful skill (any skill). Construct it with a keyed literal.
type HumanChoices struct{ Skill content.Skill }

// NoSpeciesChoices is the choices of a species that has none (Dwarf, Halfling, Orc).
type NoSpeciesChoices struct{}

func (DragonbornChoices) isSpeciesTraitChoices() {}
func (ElfChoices) isSpeciesTraitChoices()        {}
func (GnomeChoices) isSpeciesTraitChoices()      {}
func (GoliathChoices) isSpeciesTraitChoices()    {}
func (TieflingChoices) isSpeciesTraitChoices()   {}
func (HumanChoices) isSpeciesTraitChoices()      {}
func (NoSpeciesChoices) isSpeciesTraitChoices()  {}

// SpeciesChoiceName names the part of a species' choices a problem concerns. The
// zero value is not a name. Comparable and safe for concurrent use.
type SpeciesChoiceName int

// The names of the choices.
const (
	ChoiceDraconicAncestry SpeciesChoiceName = iota + 1
	ChoiceElvenLineage
	ChoiceGnomishLineage
	ChoiceGiantAncestry
	ChoiceFiendishLegacy
	ChoiceSpellcastingAbility
	ChoiceKeenSensesSkill
	ChoiceSkillfulSkill
	ChoiceReplacementCantrip
	ChoiceShape
)

// String returns the snake_case name, or "unknown".
func (n SpeciesChoiceName) String() string {
	return featNameOf(int(n), "draconic_ancestry", "elven_lineage", "gnomish_lineage", "giant_ancestry", "fiendish_legacy",
		"spellcasting_ability", "keen_senses_skill", "skillful_skill", "replacement_cantrip", "shape")
}

// SpeciesChoiceKind is what is wrong with a choice. The zero value is not a kind.
// Comparable and safe for concurrent use.
type SpeciesChoiceKind int

// The kinds of problem.
const (
	TraitProblemMissing SpeciesChoiceKind = iota + 1
	TraitProblemUnexpected
	TraitProblemOptionUnknown
	TraitProblemAbilityNotAllowed
	TraitProblemSkillNotAllowed
	TraitProblemNotACantrip
	TraitProblemNotOnSpellList
	TraitProblemDuplicate
)

// String returns the snake_case name, or "unknown".
func (k SpeciesChoiceKind) String() string {
	return featNameOf(int(k), "missing", "unexpected", "option_unknown", "ability_not_allowed", "skill_not_allowed",
		"not_a_cantrip", "not_on_spell_list", "duplicate")
}

// SpeciesChoiceProblem is one thing wrong with the choices of a species. It
// matches srderr.ErrInvalidInput with errors.Is, and errors.Is(err, p) matches an
// equal problem. It carries no display text. Comparable.
type SpeciesChoiceProblem struct { //nolint:errname // a typed problem named by the spec, as the other problem types
	Name SpeciesChoiceName
	Kind SpeciesChoiceKind
}

// Error renders the problem. The text is not a contract.
func (p SpeciesChoiceProblem) Error() string {
	return "character: species choice " + p.Name.String() + ": " + p.Kind.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (p SpeciesChoiceProblem) Unwrap() error { return srderr.ErrInvalidInput }

// SpeciesBehaviourUnknownError reports a species with no behaviour here (a house
// species, or any id outside the nine of SRD 5.2.1). It matches
// srderr.ErrInvalidInput and ErrSpeciesBehaviourUnknown with errors.Is.
type SpeciesBehaviourUnknownError struct{ Species content.SpeciesID }

// Error renders the error. The text is not a contract.
func (e *SpeciesBehaviourUnknownError) Error() string {
	return "character: no species behaviour for " + string(e.Species)
}

// Unwrap returns ErrSpeciesBehaviourUnknown.
func (e *SpeciesBehaviourUnknownError) Unwrap() error { return ErrSpeciesBehaviourUnknown }

// ErrSpeciesBehaviourUnknown is matched by every *SpeciesBehaviourUnknownError.
var ErrSpeciesBehaviourUnknown = fmt.Errorf("character: species behaviour unknown: %w", srderr.ErrInvalidInput)

// SpeciesMismatchError reports choices of another species than the one given
// (FR-001). It matches srderr.ErrInvalidInput with errors.Is.
type SpeciesMismatchError struct {
	Species content.SpeciesID
	Choices SpeciesTraitChoices
}

// Error renders the error. The text is not a contract.
func (e *SpeciesMismatchError) Error() string {
	return "character: choices do not fit species " + string(e.Species)
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *SpeciesMismatchError) Unwrap() error { return srderr.ErrInvalidInput }

// choicesFit reports whether c is the variant of species id.
func choicesFit(id content.SpeciesID, c SpeciesTraitChoices) bool {
	switch c.(type) {
	case DragonbornChoices:
		return id == speciesDragonborn
	case ElfChoices:
		return id == speciesElf
	case GnomeChoices:
		return id == speciesGnome
	case GoliathChoices:
		return id == speciesGoliath
	case TieflingChoices:
		return id == speciesTiefling
	case HumanChoices:
		return id == speciesHuman
	case NoSpeciesChoices:
		return id == speciesDwarf || id == speciesHalfling || id == speciesOrc
	}
	return false
}

// spellcastingAbilityAllowed reports whether a may be a lineage's spellcasting ability.
func spellcastingAbilityAllowed(a ability.Ability) bool {
	return a == ability.Intelligence || a == ability.Wisdom || a == ability.Charisma
}

// keenSensesAllowed reports whether s may be the Keen Senses skill.
func keenSensesAllowed(s content.Skill) bool {
	return s == content.SkillInsight || s == content.SkillPerception || s == content.SkillSurvival
}

// problemAccumulator collects problems in order.
type problemAccumulator []SpeciesChoiceProblem

func (a *problemAccumulator) add(n SpeciesChoiceName, k SpeciesChoiceKind) {
	*a = append(*a, SpeciesChoiceProblem{Name: n, Kind: k})
}

// option checks a enum choice: the zero value is missing, an undefined one unknown.
func (a *problemAccumulator) option(n SpeciesChoiceName, v int, valid bool) {
	switch {
	case v == 0:
		a.add(n, TraitProblemMissing)
	case !valid:
		a.add(n, TraitProblemOptionUnknown)
	}
}

func (a *problemAccumulator) spellAbility(v ability.Ability) {
	switch {
	case v == 0:
		a.add(ChoiceSpellcastingAbility, TraitProblemMissing)
	case !spellcastingAbilityAllowed(v):
		a.add(ChoiceSpellcastingAbility, TraitProblemAbilityNotAllowed)
	}
}

// choiceProblems returns every problem of the contents of c, in field order.
func choiceProblems(c SpeciesTraitChoices) []SpeciesChoiceProblem {
	var ps problemAccumulator
	switch v := c.(type) {
	case DragonbornChoices:
		ps.option(ChoiceDraconicAncestry, int(v.Ancestry), v.Ancestry.Valid())
	case ElfChoices:
		ps.option(ChoiceElvenLineage, int(v.Lineage), v.Lineage.Valid())
		ps.spellAbility(v.Ability)
		switch {
		case v.KeenSensesSkill == 0:
			ps.add(ChoiceKeenSensesSkill, TraitProblemMissing)
		case !keenSensesAllowed(v.KeenSensesSkill):
			ps.add(ChoiceKeenSensesSkill, TraitProblemSkillNotAllowed)
		}
	case GnomeChoices:
		ps.option(ChoiceGnomishLineage, int(v.Lineage), v.Lineage.Valid())
		ps.spellAbility(v.Ability)
	case GoliathChoices:
		ps.option(ChoiceGiantAncestry, int(v.Ancestry), v.Ancestry.Valid())
	case TieflingChoices:
		ps.option(ChoiceFiendishLegacy, int(v.Legacy), v.Legacy.Valid())
		ps.spellAbility(v.Ability)
	case HumanChoices:
		switch {
		case v.Skill == 0:
			ps.add(ChoiceSkillfulSkill, TraitProblemMissing)
		case !v.Skill.Valid():
			ps.add(ChoiceSkillfulSkill, TraitProblemSkillNotAllowed)
		}
	}
	return ps
}

// joinProblems joins the problems into one error, nil for none.
func joinProblems(ps []SpeciesChoiceProblem) error {
	errs := make([]error, len(ps))
	for i, p := range ps {
		errs[i] = p
	}
	return errors.Join(errs...)
}

// offeredTrait returns the printed name of the trait of species id that asks for a
// choice, or "" for a species with none.
func offeredTrait(id content.SpeciesID) string {
	switch id {
	case speciesDragonborn:
		return traitDraconicAncestry
	case speciesElf:
		return traitElvenLineage
	case speciesGnome:
		return traitGnomishLineage
	case speciesGoliath:
		return traitGiantAncestry
	case speciesTiefling:
		return traitFiendishLegacy
	}
	return ""
}

// traitChoiceName returns the name of the choice of the trait called trait.
func traitChoiceName(trait string) (SpeciesChoiceName, bool) {
	i := slices.Index([]string{traitDraconicAncestry, traitElvenLineage, traitGnomishLineage, traitGiantAncestry, traitFiendishLegacy}, trait)
	return SpeciesChoiceName(i + 1), i >= 0
}

// hasSpellAbility reports whether the trait of species id asks for a spellcasting ability.
func hasSpellAbility(id content.SpeciesID) bool {
	return id == speciesElf || id == speciesGnome || id == speciesTiefling
}

// ParseSpeciesTraitChoices turns the selections made at creation into the choices
// of species (FR-003 to FR-007): the option names are matched exactly against the
// printed headings, the spellcasting ability is read from grants under the name of
// the lineage trait, and the Keen Senses or Skillful skill from the skill of grants.
// The Versatile feat of grants is ignored (it is taken with TakeFeat).
//
// It returns the choices and a nil error, or a nil value and the errors.Join of
// every SpeciesChoiceProblem found, in input order (selections, then the ability,
// then the skill), each matching srderr.ErrInvalidInput. An id outside the nine
// species is a *SpeciesBehaviourUnknownError. It never panics.
func ParseSpeciesTraitChoices(species content.SpeciesID, selections []SpeciesTraitSelection, grants SpeciesGrantSelections) (SpeciesTraitChoices, error) {
	if !behaviourKnown(species) {
		return nil, &SpeciesBehaviourUnknownError{Species: species}
	}
	var ps problemAccumulator
	offered := offeredTrait(species)
	option, seen := "", false
	for _, s := range selections {
		name, known := traitChoiceName(s.Trait)
		switch {
		case offered != "" && s.Trait == offered && !seen:
			option, seen = s.Option, true
		case offered != "" && s.Trait == offered:
			ps.add(name, TraitProblemDuplicate)
		case known:
			ps.add(name, TraitProblemUnexpected)
		default:
			ps.add(ChoiceShape, TraitProblemUnexpected)
		}
	}
	choices := parseOption(species, option, seen, &ps)
	abilities := grants.Abilities()
	var spellAbility ability.Ability
	for _, key := range slices.Sorted(maps.Keys(abilities)) {
		if key != offered || !hasSpellAbility(species) {
			ps.add(ChoiceSpellcastingAbility, TraitProblemUnexpected)
			continue
		}
		spellAbility = abilities[key]
		if !spellcastingAbilityAllowed(spellAbility) {
			ps.add(ChoiceSpellcastingAbility, TraitProblemAbilityNotAllowed)
		}
	}
	if _, given := abilities[offered]; hasSpellAbility(species) && !given {
		ps.add(ChoiceSpellcastingAbility, TraitProblemMissing)
	}
	skill, hasSkill := grants.Skill()
	switch {
	case species == speciesElf && !hasSkill:
		ps.add(ChoiceKeenSensesSkill, TraitProblemMissing)
	case species == speciesElf && !keenSensesAllowed(skill):
		ps.add(ChoiceKeenSensesSkill, TraitProblemSkillNotAllowed)
	case species == speciesHuman && !hasSkill:
		ps.add(ChoiceSkillfulSkill, TraitProblemMissing)
	case hasSkill && species != speciesElf && species != speciesHuman:
		ps.add(ChoiceShape, TraitProblemUnexpected)
	}
	if len(ps) > 0 {
		return nil, joinProblems(ps)
	}
	return withLineageDetails(choices, spellAbility, skill), nil
}

// parseOption parses the chosen option name for species into a variant with the
// option set, adding OptionUnknown or Missing to ps.
func parseOption(species content.SpeciesID, name string, seen bool, ps *problemAccumulator) SpeciesTraitChoices {
	bad := func(n SpeciesChoiceName) {
		if seen {
			ps.add(n, TraitProblemOptionUnknown)
		} else {
			ps.add(n, TraitProblemMissing)
		}
	}
	switch species {
	case speciesDragonborn:
		a, ok := ParseDraconicAncestry(name)
		if !ok {
			bad(ChoiceDraconicAncestry)
		}
		return DragonbornChoices{Ancestry: a}
	case speciesElf:
		l, ok := ParseElvenLineage(name)
		if !ok {
			bad(ChoiceElvenLineage)
		}
		return ElfChoices{Lineage: l}
	case speciesGnome:
		l, ok := ParseGnomishLineage(name)
		if !ok {
			bad(ChoiceGnomishLineage)
		}
		return GnomeChoices{Lineage: l}
	case speciesGoliath:
		a, ok := ParseGiantAncestry(name)
		if !ok {
			bad(ChoiceGiantAncestry)
		}
		return GoliathChoices{Ancestry: a}
	case speciesTiefling:
		l, ok := ParseFiendishLegacy(name)
		if !ok {
			bad(ChoiceFiendishLegacy)
		}
		return TieflingChoices{Legacy: l}
	case speciesHuman:
		return HumanChoices{}
	}
	return NoSpeciesChoices{}
}

// withLineageDetails sets the spellcasting ability and the skill on the variant
// parseOption built.
func withLineageDetails(c SpeciesTraitChoices, a ability.Ability, skill content.Skill) SpeciesTraitChoices {
	switch v := c.(type) {
	case ElfChoices:
		v.Ability, v.KeenSensesSkill = a, skill
		return v
	case GnomeChoices:
		v.Ability = a
		return v
	case TieflingChoices:
		v.Ability = a
		return v
	case HumanChoices:
		v.Skill = skill
		return v
	}
	return c
}

// MustSpeciesTraitChoices is ParseSpeciesTraitChoices for literals known to be
// valid. It panics with the error ParseSpeciesTraitChoices returns, and only then.
func MustSpeciesTraitChoices(species content.SpeciesID, selections []SpeciesTraitSelection, grants SpeciesGrantSelections) SpeciesTraitChoices {
	c, err := ParseSpeciesTraitChoices(species, selections, grants)
	if err != nil {
		panic(err)
	}
	return c
}
