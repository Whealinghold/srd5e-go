package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// wizardList is the Wizard spell list a High Elf cantrip is replaced from.
const wizardList content.ClassID = "wizard"

// HighElfCantripSwap is the High Elf replacement of the lineage cantrip after a Long
// Rest (SRD 5.2.1, "High Elf", p. 85). The zero value is ready to use. It is
// stateless and safe for concurrent use.
type HighElfCantripSwap struct{}

// Replace replaces the cantrip a High Elf holds with spell, closes the window and
// emits HighElfCantripReplaced. The chosen ability and the level 3 and 5 spells are
// unchanged, and the swap may be repeated at every later Long Rest.
//
// It checks, in this order: the zero SpeciesTraits (SpeciesBehaviourUnknown), traits
// that are not a High Elf (SpeciesUseNotOffered), a closed window
// (CantripWindowClosed), and then the spell: it must be a cantrip, on the Wizard list
// and not the cantrip held now; every problem found is listed in
// SpeciesChoicesInvalid (NotACantrip, NotOnSpellList, duplicate). It returns an error
// matching srderr.ErrInvalidInput for a zero Character. The inputs are unchanged. It
// never panics.
func (HighElfCantripSwap) Replace(c Character, t SpeciesTraits, spell content.Spell) (SpeciesTraitsTransition, error) {
	if c.IsZero() {
		return nil, invalidInput("replace a High Elf cantrip of a zero character")
	}
	reject := func(r SpeciesTraitRejection) (SpeciesTraitsTransition, error) {
		return SpeciesTraitsRejected{Rejection: r}, nil
	}
	switch {
	case t.IsZero():
		return reject(SpeciesBehaviourUnknown{})
	case t.cantrip == "":
		return reject(SpeciesUseNotOffered{})
	case !t.window:
		return reject(CantripWindowClosed{})
	}
	var ps problemAccumulator
	if spell.Level() != 0 {
		ps.add(ChoiceReplacementCantrip, TraitProblemNotACantrip)
	}
	if !slices.Contains(spell.Classes(), wizardList) {
		ps.add(ChoiceReplacementCantrip, TraitProblemNotOnSpellList)
	}
	if spell.ID() == t.cantrip {
		ps.add(ChoiceReplacementCantrip, TraitProblemDuplicate)
	}
	if len(ps) > 0 {
		return reject(SpeciesChoicesInvalid{problems: ps})
	}
	next := t
	next.cantrip, next.window = spell.ID(), false
	return SpeciesTraitsChanged{traits: next, events: []SpeciesTraitEvent{HighElfCantripReplaced{id: c.ID(), old: t.cantrip, repl: spell.ID()}}}, nil
}
