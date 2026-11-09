package character_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// grantedIDs returns the spell ids in order.
func grantedIDs(gs []character.SpeciesGrantedSpell) []content.SpellID {
	out := []content.SpellID{}
	for _, g := range gs {
		out = append(out, g.Spell())
	}
	return out
}

// Requirement: doc/specs/species-trait-behaviour.md FR-020 to FR-023, Table L (SRD 5.2.1, "Elf" p. 85, "Gnome" p. 85, "Tiefling" p. 86) - at character level 5 each lineage and legacy lists its cantrips, its level 3 spell and its level 5 spell, the last two always prepared with their own free-cast use and every spell carrying the chosen ability; Thaumaturgy comes with every Tiefling legacy.
// Path: per case.
func TestSpeciesGrantedSpellsTableL(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		ab     ability.Ability
		want   []content.SpellID
	}{
		{"Drow", spElf(t, character.ElvenDrow), ability.Wisdom, []content.SpellID{"dancing_lights", "faerie_fire", "darkness"}},
		{"High Elf", spElf(t, character.ElvenHighElf), ability.Wisdom, []content.SpellID{"prestidigitation", "detect_magic", "misty_step"}},
		{"Wood Elf", spElf(t, character.ElvenWoodElf), ability.Wisdom, []content.SpellID{"druidcraft", "longstrider", "pass_without_trace"}},
		{"Abyssal", spTiefling(t, character.LegacyAbyssal), ability.Charisma, []content.SpellID{"poison_spray", "thaumaturgy", "ray_of_sickness", "hold_person"}},
		{"Chthonic", spTiefling(t, character.LegacyChthonic), ability.Charisma, []content.SpellID{"chill_touch", "thaumaturgy", "false_life", "ray_of_enfeeblement"}},
		{"Infernal", spTiefling(t, character.LegacyInfernal), ability.Charisma, []content.SpellID{"fire_bolt", "thaumaturgy", "hellish_rebuke", "darkness"}},
		{"Forest Gnome", spGnome(t, character.GnomishForest), ability.Intelligence, []content.SpellID{"minor_illusion", "speak_with_animals"}},
		{"Rock Gnome", spGnome(t, character.GnomishRock), ability.Intelligence, []content.SpellID{"mending", "prestidigitation"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.SpeciesGrantedSpells{}.Of(c.traits, character.MustLevel(5))
			if !slices.Equal(grantedIDs(got), c.want) {
				t.Fatalf("spells = %v, want %v", grantedIDs(got), c.want)
			}
			for _, g := range got {
				if g.Ability() != c.ab {
					t.Errorf("%s ability = %v, want %v", g.Spell(), g.Ability(), c.ab)
				}
				u, has := g.Use()
				prepared := g.AlwaysPrepared()
				leveled := slices.Contains([]content.SpellID{"faerie_fire", "darkness", "detect_magic", "misty_step", "longstrider", "pass_without_trace", "ray_of_sickness", "hold_person", "false_life", "ray_of_enfeeblement", "hellish_rebuke", "speak_with_animals"}, g.Spell())
				if leveled != prepared || leveled != has || has && !u.Valid() {
					t.Errorf("%s: always prepared %v, use %v %v", g.Spell(), prepared, u, has)
				}
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-020, FR-021, FR-023, Table L "boundaries", "Gnome level 1" (SRD 5.2.1, "Elf" p. 84) - the level 3 spell appears from character level 3 and the level 5 spell from 5; Speak with Animals and the cantrips are there from level 1; nothing is listed for level 0.
// Path: unhappy (boundary).
func TestSpeciesGrantedSpellsBoundaries(t *testing.T) {
	t.Parallel()
	elf := spElf(t, character.ElvenWoodElf)
	for level, want := range map[int][]content.SpellID{1: {"druidcraft"}, 2: {"druidcraft"}, 3: {"druidcraft", "longstrider"}, 4: {"druidcraft", "longstrider"}, 5: {"druidcraft", "longstrider", "pass_without_trace"}} {
		got := character.SpeciesGrantedSpells{}.Of(elf, character.MustLevel(level))
		if !slices.Equal(grantedIDs(got), want) {
			t.Errorf("level %d = %v, want %v", level, grantedIDs(got), want)
		}
		if u, _ := func() (character.SpeciesUse, bool) { return got[len(got)-1].Use() }(); level == 3 && u != character.LineageSpellThreeUse || level == 5 && u != character.LineageSpellFiveUse {
			t.Errorf("level %d last use = %v", level, u)
		}
	}
	if got := (character.SpeciesGrantedSpells{}).Of(elf, character.Level{}); len(got) != 0 || got == nil {
		t.Errorf("the zero level lists %v", got)
	}
	if got := (character.SpeciesGrantedSpells{}).Of(spGnome(t, character.GnomishForest), character.MustLevel(1)); !slices.Equal(grantedIDs(got), []content.SpellID{"minor_illusion", "speak_with_animals"}) {
		t.Errorf("Forest Gnome at level 1 = %v", grantedIDs(got))
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-020, FR-022, Table L "non-casting species" (SRD 5.2.1, pp. 84-86) - a species without granted spells, the zero traits and invalid choices list none, and the High Elf cantrip is the one it holds now.
// Path: unhappy (empty case).
func TestSpeciesGrantedSpellsEmptyAndSwapped(t *testing.T) {
	t.Parallel()
	for name, tr := range map[string]character.SpeciesTraits{"Dwarf": spDwarf(t), "Human": spHuman(t), "Dragonborn": spDragonborn(t, character.DraconicRed), "zero": {}} {
		if got := (character.SpeciesGrantedSpells{}).Of(tr, character.MustLevel(20)); len(got) != 0 || got == nil {
			t.Errorf("%s lists %v", name, got)
		}
	}
	swapped, _ := replaceTo(t, "light")
	if got := (character.SpeciesGrantedSpells{}).Of(swapped, character.MustLevel(1)); !slices.Equal(grantedIDs(got), []content.SpellID{"light"}) {
		t.Errorf("swapped High Elf = %v", grantedIDs(got))
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-023 (SRD 5.2.1, "Gnomish Lineage", p. 85) - a Rock Gnome has a Tiny clockwork device (10 minutes to create, AC 5, 1 Hit Point, three at a time, each ending after 8 hours); a Forest Gnome and every other species have none.
// Path: happy.
func TestSpeciesGrantedSpellsClockworkDevice(t *testing.T) {
	t.Parallel()
	got, ok := character.SpeciesGrantedSpells{}.ClockworkDevice(spGnome(t, character.GnomishRock))
	want := character.ClockworkDevice{MinutesToCreate: 10, Size: content.SizeTiny, ArmorClass: 5, HitPoints: 1, MaxAtOnce: 3, HoursLasting: 8}
	if !ok || got != want {
		t.Errorf("Rock Gnome = %v, %v", got, ok)
	}
	for name, tr := range map[string]character.SpeciesTraits{"Forest Gnome": spGnome(t, character.GnomishForest), "Dwarf": spDwarf(t), "zero": {}} {
		if _, ok := (character.SpeciesGrantedSpells{}).ClockworkDevice(tr); ok {
			t.Errorf("%s has a clockwork device", name)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-024, Q10 (content-species-backgrounds-feats.md FR-018 to FR-022) - the code constants for the printed facts agree with the bundled species: all ten Draconic Ancestors map to the bundled damage types, and each bundled Elven Lineage names the spells of Table L at levels 1, 3 and 5. The five species not bundled (Gnome, Goliath, Halfling, Orc, Tiefling) are pinned by the tests above only; checking them against data is deferred to Phase 5.
// Path: happy.
func TestSpeciesConstantsAgreeWithBundledContent(t *testing.T) {
	t.Parallel()
	src, err := data.NewBundledSource()
	if err != nil {
		t.Fatal(err)
	}
	species := func(id string) content.Species {
		res, err := src.Lookup(context.Background(), content.NewKey(content.KindSpecies, content.MustID(id)))
		if err != nil {
			t.Fatal(err)
		}
		return res.(content.LookupFound).Entry.(content.Species)
	}
	for _, o := range species("dragonborn").Choices()[0].Options() {
		opt := o.(content.DamageTypeOption)
		a, ok := character.ParseDraconicAncestry(opt.Name())
		if !ok || a.DamageType() != opt.DamageType() {
			t.Errorf("%s: code %v, content %v", opt.Name(), a.DamageType(), opt.DamageType())
		}
	}
	for _, o := range species("elf").Choices()[0].Options() {
		opt := o.(content.LineageOption)
		lineage, ok := character.ParseElvenLineage(opt.Name())
		if !ok {
			t.Fatalf("lineage %q is not in code", opt.Name())
		}
		tr := spTraits(t, "elf", character.ElfChoices{Lineage: lineage, Ability: ability.Wisdom, KeenSensesSkill: content.SkillInsight})
		var want []content.SpellID
		for _, b := range opt.LevelBenefits() {
			want = append(want, b.SpellID())
		}
		if got := grantedIDs((character.SpeciesGrantedSpells{}).Of(tr, character.MustLevel(5))); !slices.Equal(got, want) {
			t.Errorf("%s: code %v, content %v", opt.Name(), got, want)
		}
	}
}
