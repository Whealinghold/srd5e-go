package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-002, Table D "ancestry to type" (SRD 5.2.1, "Dragonborn", Draconic Ancestors, p. 84) - each of the ten ancestries maps to its damage type and parses from its printed name.
// Path: per case.
func TestDraconicAncestryDamageTypes(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		a    character.DraconicAncestry
		want content.DamageType
	}{
		{"Black", character.DraconicBlack, content.DamageAcid}, {"Blue", character.DraconicBlue, content.DamageLightning},
		{"Brass", character.DraconicBrass, content.DamageFire}, {"Bronze", character.DraconicBronze, content.DamageLightning},
		{"Copper", character.DraconicCopper, content.DamageAcid}, {"Gold", character.DraconicGold, content.DamageFire},
		{"Green", character.DraconicGreen, content.DamagePoison}, {"Red", character.DraconicRed, content.DamageFire},
		{"Silver", character.DraconicSilver, content.DamageCold}, {"White", character.DraconicWhite, content.DamageCold},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := character.ParseDraconicAncestry(c.name)
			if !ok || got != c.a || !got.Valid() || got.String() != c.name || got.DamageType() != c.want {
				t.Errorf("Parse(%q) = %v, %v, type %v; want %v, %v", c.name, got, ok, got.DamageType(), c.a, c.want)
			}
		})
	}
	for _, name := range []string{"", "black", "Sonic", "Black "} {
		if a, ok := character.ParseDraconicAncestry(name); ok || a.Valid() || a.DamageType() != 0 || a.String() != "unknown" {
			t.Errorf("Parse(%q) = %v, %v; want the invalid zero value", name, a, ok)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-003, Table C (SRD 5.2.1, "Elf" p. 85, "Gnome" p. 85, "Goliath" pp. 85-86, "Tiefling" p. 86) - lineage, ancestry and legacy names parse by exact, case-sensitive equality with the printed heading, the Giant Ancestry names with the descriptor and the typographic apostrophe; anything else is not a member.
// Path: unhappy (invalid input).
func TestLineageNamesParseExactly(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]character.ElvenLineage{"Drow": character.ElvenDrow, "High Elf": character.ElvenHighElf, "Wood Elf": character.ElvenWoodElf} {
		if got, ok := character.ParseElvenLineage(name); !ok || got != want || got.String() != name {
			t.Errorf("ParseElvenLineage(%q) = %v, %v", name, got, ok)
		}
	}
	for name, want := range map[string]character.GnomishLineage{"Forest Gnome": character.GnomishForest, "Rock Gnome": character.GnomishRock} {
		if got, ok := character.ParseGnomishLineage(name); !ok || got != want || got.String() != name {
			t.Errorf("ParseGnomishLineage(%q) = %v, %v", name, got, ok)
		}
	}
	legacies := map[string]struct {
		l character.FiendishLegacy
		r content.DamageType
	}{"Abyssal": {character.LegacyAbyssal, content.DamagePoison}, "Chthonic": {character.LegacyChthonic, content.DamageNecrotic}, "Infernal": {character.LegacyInfernal, content.DamageFire}}
	for name, want := range legacies {
		if got, ok := character.ParseFiendishLegacy(name); !ok || got != want.l || got.String() != name || got.Resistance() != want.r {
			t.Errorf("ParseFiendishLegacy(%q) = %v, %v", name, got, ok)
		}
	}
	giants := map[string]character.GiantAncestry{
		"Cloud’s Jaunt (Cloud Giant)": character.AncestryCloud, "Fire’s Burn (Fire Giant)": character.AncestryFire,
		"Frost’s Chill (Frost Giant)": character.AncestryFrost, "Hill’s Tumble (Hill Giant)": character.AncestryHill,
		"Stone’s Endurance (Stone Giant)": character.AncestryStone, "Storm’s Thunder (Storm Giant)": character.AncestryStorm,
	}
	for name, want := range giants {
		if got, ok := character.ParseGiantAncestry(name); !ok || got != want || got.String() != name {
			t.Errorf("ParseGiantAncestry(%q) = %v, %v", name, got, ok)
		}
	}
	for _, bad := range []string{"", "drow", "Cloud's Jaunt (Cloud Giant)", "Cloud’s Jaunt", "Abyssal "} {
		_, e := character.ParseElvenLineage(bad)
		_, g := character.ParseGnomishLineage(bad)
		_, f := character.ParseFiendishLegacy(bad)
		_, a := character.ParseGiantAncestry(bad)
		if e || g || f || a {
			t.Errorf("%q parsed", bad)
		}
	}
	var zeroL character.FiendishLegacy
	if zeroL.Valid() || zeroL.Resistance() != 0 || zeroL.String() != "unknown" || character.ElvenLineage(0).Valid() || character.GnomishLineage(9).Valid() || character.GiantAncestry(7).Valid() {
		t.Error("an undefined value is valid")
	}
}
