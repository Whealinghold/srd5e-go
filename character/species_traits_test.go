package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-050, FR-056, Table U (SRD 5.2.1, pp. 84-86) - the traits compare by value, start with nothing expended, and offer a use only to the species and lineage that have it.
// Path: per case.
func TestSpeciesTraitsOffers(t *testing.T) {
	t.Parallel()
	all := []character.SpeciesUse{character.BreathWeaponUse, character.DraconicFlightUse, character.StonecunningUse, character.GiantAncestryUse, character.LargeFormUse,
		character.AdrenalineRushUse, character.RelentlessEnduranceUse, character.LineageSpellThreeUse, character.LineageSpellFiveUse, character.GnomeSpellUse}
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		offers []character.SpeciesUse
	}{
		{"Dragonborn", spDragonborn(t, character.DraconicRed), []character.SpeciesUse{character.BreathWeaponUse, character.DraconicFlightUse}},
		{"Dwarf", spDwarf(t), []character.SpeciesUse{character.StonecunningUse}},
		{"Goliath", spGoliath(t, character.AncestryHill), []character.SpeciesUse{character.GiantAncestryUse, character.LargeFormUse}},
		{"Orc", spOrc(t), []character.SpeciesUse{character.AdrenalineRushUse, character.RelentlessEnduranceUse}},
		{"Elf", spElf(t, character.ElvenDrow), []character.SpeciesUse{character.LineageSpellThreeUse, character.LineageSpellFiveUse}},
		{"Tiefling", spTiefling(t, character.LegacyAbyssal), []character.SpeciesUse{character.LineageSpellThreeUse, character.LineageSpellFiveUse}},
		{"Forest Gnome", spGnome(t, character.GnomishForest), []character.SpeciesUse{character.GnomeSpellUse}},
		{"Rock Gnome", spGnome(t, character.GnomishRock), nil},
		{"Halfling", spHalfling(t), nil},
		{"Human", spHuman(t), nil},
		{"zero", character.SpeciesTraits{}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, u := range all {
				want := false
				for _, o := range c.offers {
					want = want || o == u
				}
				if got := c.traits.Offers(u); got != want {
					t.Errorf("Offers(%v) = %v, want %v", u, got, want)
				}
				if c.traits.ExpendedOf(u) != 0 {
					t.Errorf("ExpendedOf(%v) = %d at the start", u, c.traits.ExpendedOf(u))
				}
			}
		})
	}
	if spDwarf(t).Offers(0) || spDwarf(t).Offers(11) || spDwarf(t).ExpendedOf(0) != 0 {
		t.Error("an invalid use is offered")
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-050, FR-051, FR-052, FR-056 - WithExpended returns a new value and leaves the receiver alone, and rejects a negative count, an invalid use and a use the species does not have with an error matching srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestSpeciesTraitsWithExpended(t *testing.T) {
	t.Parallel()
	orc := spOrc(t)
	next := must(orc.WithExpended(character.AdrenalineRushUse, 2))
	if next.ExpendedOf(character.AdrenalineRushUse) != 2 || orc.ExpendedOf(character.AdrenalineRushUse) != 0 || next == orc {
		t.Errorf("WithExpended = %v from %v", next, orc)
	}
	if again := must(orc.WithExpended(character.AdrenalineRushUse, 2)); again != next {
		t.Error("equal traits are not ==")
	}
	if zero, err := orc.WithExpended(character.AdrenalineRushUse, 0); err != nil || zero != orc {
		t.Errorf("a count of 0 is valid and changes nothing: %v, %v", zero, err)
	}
	for name, c := range map[string]struct {
		u character.SpeciesUse
		n int
	}{"negative": {character.AdrenalineRushUse, -1}, "not offered": {character.BreathWeaponUse, 1}, "invalid use": {0, 1}} {
		if got, err := orc.WithExpended(c.u, c.n); !got.IsZero() || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, got, err)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-025, FR-056 - only a High Elf holds a cantrip, starting with Prestidigitation and a closed window, and only a High Elf accepts a cantrip window; any other species is an error matching srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestSpeciesTraitsCantripWindow(t *testing.T) {
	t.Parallel()
	high := spElf(t, character.ElvenHighElf)
	if c, ok := high.HighElfCantrip(); !ok || c != "prestidigitation" || high.CantripWindowOpen() {
		t.Errorf("start = %v, %v, window %v", c, ok, high.CantripWindowOpen())
	}
	open := must(high.WithCantripWindow(true))
	if !open.CantripWindowOpen() || high.CantripWindowOpen() || must(open.WithCantripWindow(false)) != high {
		t.Error("the window did not toggle on a copy")
	}
	for name, tr := range map[string]character.SpeciesTraits{"drow": spElf(t, character.ElvenDrow), "dwarf": spDwarf(t), "zero": {}} {
		if _, ok := tr.HighElfCantrip(); ok {
			t.Errorf("%s holds a cantrip", name)
		}
		if got, err := tr.WithCantripWindow(true); !got.IsZero() || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, got, err)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-001, FR-050 - the species and choices are read back, the zero value is zero and holds neither.
// Path: happy.
func TestSpeciesTraitsAccessors(t *testing.T) {
	t.Parallel()
	choices := character.ElfChoices{Lineage: character.ElvenWoodElf, Ability: ability.Wisdom, KeenSensesSkill: content.SkillInsight}
	tr := spTraits(t, "elf", choices)
	if tr.Species() != "elf" || tr.Choices() != choices || tr.IsZero() {
		t.Errorf("accessors = %v %v", tr.Species(), tr.Choices())
	}
	var zero character.SpeciesTraits
	if !zero.IsZero() || zero.Species() != "" || zero.Choices() != nil {
		t.Error("the zero value holds something")
	}
	if character.MustSpeciesTraits("elf", choices) != tr {
		t.Error("MustSpeciesTraits disagrees")
	}
}
