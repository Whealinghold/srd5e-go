package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// highElfStartCantrip is the cantrip of the High Elf lineage before any swap (SRD
// 5.2.1, "High Elf", p. 85).
const highElfStartCantrip content.SpellID = "prestidigitation"

// SpeciesTraits is the state of a character's species traits: the species, the
// choices, how many times each limited use is expended, and the High Elf cantrip and
// its swap window (SRD 5.2.1, pp. 83-86). It is held beside the Character, as
// TakenFeats is. It is comparable with ==. The zero SpeciesTraits holds no species
// (a species without behaviour here, such as a house species); the use cases reject
// it with SpeciesBehaviourUnknown. Safe for concurrent use.
type SpeciesTraits struct {
	species  content.SpeciesID
	choices  SpeciesTraitChoices
	expended [speciesUseCount]int
	cantrip  content.SpellID
	window   bool
}

// NewSpeciesTraits returns the traits of species with choices and no use expended.
//
// It returns a *SpeciesBehaviourUnknownError for an id outside the nine species, a
// *SpeciesMismatchError for choices of another species (a nil value included), and
// the errors.Join of every SpeciesChoiceProblem for choices with a missing or
// undefined part (an ability other than Intelligence, Wisdom or Charisma, a Keen
// Senses skill other than Insight, Perception or Survival), each matching
// srderr.ErrInvalidInput. It never panics.
func NewSpeciesTraits(species content.SpeciesID, choices SpeciesTraitChoices) (SpeciesTraits, error) {
	if !behaviourKnown(species) {
		return SpeciesTraits{}, &SpeciesBehaviourUnknownError{Species: species}
	}
	if !choicesFit(species, choices) {
		return SpeciesTraits{}, &SpeciesMismatchError{Species: species, Choices: choices}
	}
	if ps := choiceProblems(choices); len(ps) > 0 {
		return SpeciesTraits{}, joinProblems(ps)
	}
	t := SpeciesTraits{species: species, choices: choices}
	if elf, ok := choices.(ElfChoices); ok && elf.Lineage == ElvenHighElf {
		t.cantrip = highElfStartCantrip
	}
	return t, nil
}

// MustSpeciesTraits is NewSpeciesTraits for literals known to be valid. It panics
// with the error NewSpeciesTraits returns, and only then.
func MustSpeciesTraits(species content.SpeciesID, choices SpeciesTraitChoices) SpeciesTraits {
	t, err := NewSpeciesTraits(species, choices)
	if err != nil {
		panic(err)
	}
	return t
}

// IsZero reports whether t is the zero SpeciesTraits.
func (t SpeciesTraits) IsZero() bool { return t.species == "" }

// Species returns the species id.
func (t SpeciesTraits) Species() content.SpeciesID { return t.species }

// Choices returns the choices, nil for the zero value.
func (t SpeciesTraits) Choices() SpeciesTraitChoices { return t.choices }

// ExpendedOf returns how many times u is expended, 0 for an invalid use.
func (t SpeciesTraits) ExpendedOf(u SpeciesUse) int {
	if !u.Valid() {
		return 0
	}
	return t.expended[u-1]
}

// HighElfCantrip returns the cantrip a High Elf holds from the lineage (the
// Prestidigitation of the lineage, or the last replacement) and true, or false for
// any other character.
func (t SpeciesTraits) HighElfCantrip() (content.SpellID, bool) { return t.cantrip, t.cantrip != "" }

// CantripWindowOpen reports whether a High Elf may replace the lineage cantrip now.
func (t SpeciesTraits) CantripWindowOpen() bool { return t.window }

// Offers reports whether the species and lineage give the use: Breath Weapon and
// Draconic Flight to a Dragonborn, Stonecunning to a Dwarf, Giant Ancestry and
// Large Form to a Goliath, Adrenaline Rush and Relentless Endurance to an Orc, the
// level 3 and 5 lineage spells to an Elf or a Tiefling, and the Speak with Animals
// casts to a Forest Gnome only.
func (t SpeciesTraits) Offers(u SpeciesUse) bool {
	switch u {
	case BreathWeaponUse, DraconicFlightUse:
		return t.species == speciesDragonborn
	case StonecunningUse:
		return t.species == speciesDwarf
	case GiantAncestryUse, LargeFormUse:
		return t.species == speciesGoliath
	case AdrenalineRushUse, RelentlessEnduranceUse:
		return t.species == speciesOrc
	case LineageSpellThreeUse, LineageSpellFiveUse:
		return t.species == speciesElf || t.species == speciesTiefling
	case GnomeSpellUse:
		g, ok := t.choices.(GnomeChoices)
		return ok && g.Lineage == GnomishForest
	}
	return false
}

// WithExpended returns t with u expended n times.
//
// It returns an error matching srderr.ErrInvalidInput for a negative n, an invalid
// use, or a use the species does not have. It never panics.
func (t SpeciesTraits) WithExpended(u SpeciesUse, n int) (SpeciesTraits, error) {
	if n < 0 || !t.Offers(u) {
		return SpeciesTraits{}, invalidInput("species use state for a use the species lacks, or a negative count")
	}
	t.expended[u-1] = n
	return t, nil
}

// WithCantripWindow returns t with the High Elf swap window open or closed.
//
// It returns an error matching srderr.ErrInvalidInput for traits that are not an
// Elf with the High Elf lineage. It never panics.
func (t SpeciesTraits) WithCantripWindow(open bool) (SpeciesTraits, error) {
	if t.cantrip == "" {
		return SpeciesTraits{}, invalidInput("a cantrip window for a species that is not a High Elf")
	}
	t.window = open
	return t, nil
}
