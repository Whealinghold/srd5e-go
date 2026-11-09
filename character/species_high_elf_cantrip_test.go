package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// cantripSpells are the synthetic spells of the High Elf swap tests.
func cantripSpells(t *testing.T) map[content.SpellID]content.Spell {
	t.Helper()
	return spellMap(t, spellSpec{"light", 0, []string{"cleric", "wizard"}}, spellSpec{"prestidigitation", 0, []string{"wizard", "bard"}},
		spellSpec{"guidance", 0, []string{"cleric", "druid"}}, spellSpec{"shield", 1, []string{"wizard"}}, spellSpec{"mage_hand", 0, []string{"wizard"}},
		spellSpec{"sacred_flame", 0, []string{"cleric"}}, spellSpec{"leveled_cleric", 2, []string{"cleric"}})
}

// openHighElf is a High Elf whose swap window is open.
func openHighElf(t *testing.T) character.SpeciesTraits {
	t.Helper()
	return must(spElf(t, character.ElvenHighElf).WithCantripWindow(true))
}

// replaceTo swaps the High Elf cantrip to spell and returns the new traits and the result.
func replaceTo(t *testing.T, spell content.SpellID) (character.SpeciesTraits, character.SpeciesTraitsTransition) {
	t.Helper()
	tr := must(character.HighElfCantripSwap{}.Replace(spChar(t, 5, 10), openHighElf(t), cantripSpells(t)[spell]))
	ch, ok := tr.(character.SpeciesTraitsChanged)
	if !ok {
		t.Fatalf("Replace(%s) = %#v", spell, tr)
	}
	return ch.Traits(), tr
}

// Requirement: doc/specs/species-trait-behaviour.md FR-025, FR-026, Table L "High Elf swap", "swap twice" (SRD 5.2.1, "High Elf", p. 85) - an open window accepts a Wizard cantrip: it replaces Prestidigitation, closes the window, emits HighElfCantripReplaced and leaves the ability and the level 3 and 5 spells; it may be repeated after each later Long Rest, replacing the previous replacement, and a second swap in one window is refused.
// Path: happy.
func TestHighElfCantripSwapReplaces(t *testing.T) {
	t.Parallel()
	first, tr := replaceTo(t, "light")
	ev := tr.(character.SpeciesTraitsChanged).Events()
	if c, _ := first.HighElfCantrip(); c != "light" || first.CantripWindowOpen() || len(ev) != 1 {
		t.Fatalf("after the swap: cantrip %v, window %v, events %v", c, first.CantripWindowOpen(), ev)
	}
	if e := ev[0].(character.HighElfCantripReplaced); e.Old() != "prestidigitation" || e.New() != "light" || e.CharacterID() != character.MustIdentifier("pc-1") {
		t.Errorf("event = %#v", e)
	}
	if first.Choices() != spElf(t, character.ElvenHighElf).Choices() {
		t.Error("the swap changed the choices")
	}
	if rj := spRejection(t, must(character.HighElfCantripSwap{}.Replace(spChar(t, 5, 10), first, cantripSpells(t)["mage_hand"]))); rj != (character.CantripWindowClosed{}) {
		t.Errorf("second swap in one window = %v", rj)
	}
	reopened := must(first.WithCantripWindow(true))
	second := must(character.HighElfCantripSwap{}.Replace(spChar(t, 5, 10), reopened, cantripSpells(t)["mage_hand"])).(character.SpeciesTraitsChanged)
	if e := second.Events()[0].(character.HighElfCantripReplaced); e.Old() != "light" || e.New() != "mage_hand" {
		t.Errorf("second swap event = %#v", e)
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-025, Table L "swap rejected" (SRD 5.2.1, "High Elf", p. 85) - a swap is refused with the first applicable reason: the zero traits, a species that is not a High Elf, a closed window, and then every spell problem is listed (not a cantrip, not on the Wizard list, the cantrip held now), the traits being unchanged.
// Path: per case.
func TestHighElfCantripSwapRejects(t *testing.T) {
	t.Parallel()
	type p = character.SpeciesChoiceProblem
	notACantrip, notOnList, dup := p{Name: character.ChoiceReplacementCantrip, Kind: character.TraitProblemNotACantrip}, p{Name: character.ChoiceReplacementCantrip, Kind: character.TraitProblemNotOnSpellList}, p{Name: character.ChoiceReplacementCantrip, Kind: character.TraitProblemDuplicate}
	for _, c := range []struct {
		name     string
		traits   character.SpeciesTraits
		spell    content.SpellID
		rejected character.SpeciesTraitRejection
		problems []p
	}{
		{"zero traits", character.SpeciesTraits{}, "light", character.SpeciesBehaviourUnknown{}, nil},
		{"a Drow", spElf(t, character.ElvenDrow), "light", character.SpeciesUseNotOffered{}, nil},
		{"a Dwarf", spDwarf(t), "light", character.SpeciesUseNotOffered{}, nil},
		{"no window", spElf(t, character.ElvenHighElf), "light", character.CantripWindowClosed{}, nil},
		{"a level 1 spell on the list", openHighElf(t), "shield", nil, []p{notACantrip}},
		{"a level 2 Cleric spell", openHighElf(t), "leveled_cleric", nil, []p{notACantrip, notOnList}},
		{"a Cleric-only cantrip", openHighElf(t), "sacred_flame", nil, []p{notOnList}},
		{"the current cantrip", openHighElf(t), "prestidigitation", nil, []p{dup}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.traits
			rj := spRejection(t, must(character.HighElfCantripSwap{}.Replace(spChar(t, 5, 10), c.traits, cantripSpells(t)[c.spell])))
			if c.problems == nil {
				if rj != c.rejected {
					t.Errorf("rejection = %#v, want %#v", rj, c.rejected)
				}
			} else {
				inv, ok := rj.(character.SpeciesChoicesInvalid)
				if !ok || len(inv.Problems()) != len(c.problems) {
					t.Fatalf("rejection = %#v, want problems %v", rj, c.problems)
				}
				for i, want := range c.problems {
					if inv.Problems()[i] != want {
						t.Errorf("problem %d = %v, want %v", i, inv.Problems()[i], want)
					}
				}
			}
			if c.traits != before {
				t.Error("the traits were altered")
			}
		})
	}
	if got, err := (character.HighElfCantripSwap{}).Replace(character.Character{}, openHighElf(t), cantripSpells(t)["light"]); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v, %v", got, err)
	}
	if (character.SpeciesChoicesInvalid{}).Problems() != nil {
		t.Error("the zero rejection lists problems")
	}
}
