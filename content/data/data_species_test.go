package data_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

func traitNames(s content.Species) []string {
	var out []string
	for _, tr := range s.Traits() {
		out = append(out, tr.Name())
	}
	return out
}

func traitText(t *testing.T, s content.Species, name string) string {
	t.Helper()
	for _, tr := range s.Traits() {
		if tr.Name() == name {
			return tr.Text()
		}
	}
	t.Fatalf("species %s has no trait %q", s.ID(), name)
	return ""
}

func parseSizes(t *testing.T, names []string) []content.CreatureSize {
	t.Helper()
	var out []content.CreatureSize
	for _, n := range names {
		z, err := content.ParseCreatureSize(n)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, z)
	}
	return out
}

// Requirement: doc/specs/content-thin-slice.md FR-009, FR-010, FR-014, FR-021, FR-022, FR-029 and
// content-species-backgrounds-feats.md FR-011, FR-012, FR-013, FR-015, FR-074 - each species has the creature type,
// sizes (as a set), size text, speed and the trait names in printed order of the oracle.
// Path: happy / unhappy (boundary: size is a set; exactly the printed traits).
func TestSpeciesCore(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	for _, o := range readOracle[speciesOracleFile](t, "species").Species {
		t.Run(o.ID, func(t *testing.T) {
			t.Parallel()
			s := speciesOf(t, src, o.ID)
			ct, err := content.ParseCreatureType(o.CreatureType)
			if err != nil {
				t.Fatal(err)
			}
			wantSizes := parseSizes(t, o.Sizes)
			gotSizes := s.Sizes()
			slices.Sort(wantSizes)
			slices.Sort(gotSizes)
			if s.Name() != o.Name || s.CreatureType() != ct || !slices.Equal(gotSizes, wantSizes) || s.SizeText() != o.SizeText || s.Speed() != o.Speed {
				t.Errorf("%s = %q, %v, %v, %q, %d; want %q, %v, %v, %q, %d", o.ID, s.Name(), s.CreatureType(), gotSizes, s.SizeText(), s.Speed(), o.Name, ct, wantSizes, o.SizeText, o.Speed)
			}
			var wantNames []string
			for _, tr := range o.Traits {
				wantNames = append(wantNames, tr.Name)
			}
			if got := traitNames(s); !slices.Equal(got, wantNames) {
				t.Errorf("%s trait names = %v, want %v", o.ID, got, wantNames)
			}
		})
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-010 to FR-013, FR-016, FR-018, FR-020, FR-022, FR-024, FR-026 to
// FR-028, FR-030 - every trait text contains each pinned phrase picked from the cited page.
// Path: happy.
func TestSpeciesTraitPhrases(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	for _, o := range readOracle[speciesOracleFile](t, "species").Species {
		s := speciesOf(t, src, o.ID)
		for _, tr := range o.Traits {
			t.Run(o.ID+"/"+tr.Name, func(t *testing.T) {
				t.Parallel()
				text := traitText(t, s, tr.Name)
				for _, p := range tr.Phrases {
					if !strings.Contains(text, p) {
						t.Errorf("%s %s text lacks %q", o.ID, tr.Name, p)
					}
				}
			})
		}
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-015, T1 and content-species-backgrounds-feats.md FR-017 - the
// Draconic Ancestry choice has exactly the ten ancestors of T1, each a DamageTypeOption with its damage type.
// Path: per case (T1): happy / unhappy (boundary: none missing, none extra).
func TestDraconicAncestors(t *testing.T) {
	t.Parallel()
	o := readOracle[speciesOracleFile](t, "species")
	d := speciesOf(t, newSlice(t), "dragonborn")
	choices := d.Choices()
	if len(choices) != 1 || choices[0].TraitName() != "Draconic Ancestry" {
		t.Fatalf("dragonborn choices = %+v, want one for Draconic Ancestry", choices)
	}
	opts := choices[0].Options()
	if len(opts) != len(o.DraconicAncestors) || len(opts) != 10 {
		t.Fatalf("Draconic Ancestry has %d options, want 10", len(opts))
	}
	for i, a := range o.DraconicAncestors {
		dt, err := content.ParseDamageType(a.DamageType)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := opts[i].(content.DamageTypeOption)
		if !ok || got.Name() != a.Dragon || got.DamageType() != dt {
			t.Errorf("option %d = %#v, want %s with %s", i, opts[i], a.Dragon, a.DamageType)
		}
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-017, T2 - the Breath Weapon text states the base damage and the
// three upgrade thresholds as printed, and exactly three upgrades.
// Path: per case (T2): happy / unhappy (boundary: lowest and highest threshold, exactly three).
func TestBreathWeaponDamageByLevel(t *testing.T) {
	t.Parallel()
	o := readOracle[speciesOracleFile](t, "species")
	text := traitText(t, speciesOf(t, newSlice(t), "dragonborn"), "Breath Weapon")
	if len(o.BreathWeaponDamage) != 4 {
		t.Fatalf("oracle has %d rows, want the base and three upgrades", len(o.BreathWeaponDamage))
	}
	for _, r := range o.BreathWeaponDamage {
		want := "takes " + r.Dice + " damage"
		if r.Level > 1 {
			want = describe("%d (%s)", r.Level, r.Dice)
		}
		if !strings.Contains(text, want) {
			t.Errorf("Breath Weapon text lacks %q (level %d)", want, r.Level)
		}
	}
	if strings.Count(text, "d10)") != 3 {
		t.Errorf("Breath Weapon text states %d upgrades, want 3", strings.Count(text, "d10)"))
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-019 - the Damage Resistance text depends on the Draconic Ancestry
// choice and names no fixed damage type.
// Path: unhappy (rule interaction: depends on a choice).
func TestDamageResistanceNamesNoFixedType(t *testing.T) {
	t.Parallel()
	text := traitText(t, speciesOf(t, newSlice(t), "dragonborn"), "Damage Resistance")
	for dt := content.DamageAcid; dt.Valid(); dt++ {
		if strings.Contains(text, dt.String()) {
			t.Errorf("Damage Resistance text names the fixed type %s", dt)
		}
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-023, FR-024, FR-025, T3 and content-species-backgrounds-feats.md
// FR-018, FR-061 - the Elven Lineage choice has exactly the three lineages of T3, each with level 1, 3 and 5
// benefits whose texts and spellIds are as printed, and the base elf values are unchanged by the lineages.
// Path: per case (T3): happy / unhappy (boundary: exactly three; override versus base).
func TestElvenLineages(t *testing.T) {
	t.Parallel()
	o := readOracle[speciesOracleFile](t, "species")
	elf := speciesOf(t, newSlice(t), "elf")
	choices := elf.Choices()
	if len(choices) != 1 || choices[0].TraitName() != "Elven Lineage" {
		t.Fatalf("elf choices = %+v, want one for Elven Lineage", choices)
	}
	opts := choices[0].Options()
	if len(opts) != len(o.Lineages) || len(opts) != 3 {
		t.Fatalf("Elven Lineage has %d options, want 3", len(opts))
	}
	for i, l := range o.Lineages {
		t.Run(l.Name, func(t *testing.T) {
			t.Parallel()
			got, ok := opts[i].(content.LineageOption)
			if !ok || got.Name() != l.Name {
				t.Fatalf("option %d = %#v, want lineage %s", i, opts[i], l.Name)
			}
			bs := got.LevelBenefits()
			wantText := []string{l.Level1, l.Level3, l.Level5}
			for j, level := range []int{1, 3, 5} {
				if bs[j].Level() != level || bs[j].Text() != wantText[j] || bs[j].SpellID() != content.SpellID(l.Spells[j]) {
					t.Errorf("%s benefit %d = %d, %q, %s; want %d, %q, %s", l.Name, j, bs[j].Level(), bs[j].Text(), bs[j].SpellID(), level, wantText[j], l.Spells[j])
				}
			}
		})
	}
	if elf.Speed() != 30 || !strings.Contains(traitText(t, elf, "Darkvision"), "range of 60 feet") {
		t.Errorf("base elf = speed %d, %q; want 30 and Darkvision 60 feet unchanged by lineages", elf.Speed(), traitText(t, elf, "Darkvision"))
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-032, FR-031, T4 - only Draconic Ancestry and Elven Lineage are
// structured choices; every other selection is stated in the verbatim text (or in sizes for the Human size).
// Path: per case (T4): happy / unhappy (boundary; missing content: option sets not bundled).
func TestChoicesInTheSpecies(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	o := readOracle[speciesOracleFile](t, "species")
	structured := map[string][]string{}
	for _, id := range sliceIDs()[content.KindSpecies] {
		for _, c := range speciesOf(t, src, id).Choices() {
			structured[id] = append(structured[id], c.TraitName())
		}
	}
	if want := map[string][]string{"dragonborn": {"Draconic Ancestry"}, "elf": {"Elven Lineage"}}; len(structured) != 2 ||
		!slices.Equal(structured["dragonborn"], want["dragonborn"]) || !slices.Equal(structured["elf"], want["elf"]) {
		t.Errorf("structured choices = %v, want %v", structured, want)
	}
	for _, c := range o.Choices {
		t.Run(c.Label, func(t *testing.T) {
			t.Parallel()
			s := speciesOf(t, src, c.Species)
			if c.Structured {
				if !slices.Contains(traitNames(s), "Draconic Ancestry") && !slices.Contains(traitNames(s), "Elven Lineage") {
					t.Errorf("%s has no trait for the structured choice %s", c.Species, c.Label)
				}
			}
			if !slices.ContainsFunc(speciesTexts(s), func(text string) bool { return strings.Contains(text, c.Phrase) }) {
				t.Errorf("%s texts lack %q for %s", c.Species, c.Phrase, c.Label)
			}
		})
	}
	human := speciesOf(t, src, "human")
	if got := human.Sizes(); len(got) != 2 || !slices.Contains(got, content.SizeMedium) || !slices.Contains(got, content.SizeSmall) {
		t.Errorf("human sizes = %v, want {Medium, Small}", got)
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-031, FR-062 - a Human trait that asks for a pick from a kind absent
// from the slice stays verbatim text with no reference, and the Human holds no reference site.
// Path: unhappy (missing content: choice target not bundled).
func TestHumanPicksAreText(t *testing.T) {
	t.Parallel()
	human := speciesOf(t, newSlice(t), "human")
	if refs := human.References(); len(refs) != 0 {
		t.Errorf("human references = %v, want none", refs)
	}
	if len(human.Choices()) != 0 {
		t.Errorf("human choices = %v, want none", human.Choices())
	}
}
