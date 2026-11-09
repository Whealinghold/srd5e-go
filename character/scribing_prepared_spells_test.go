package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case tool-proficiency-and-crafting.md FR-042 (SRD 5.2.1,
// "Prerequisites for the Scribe", p. 103; spellcasting.md FR-045, FR-052, FR-097)
// - the union of prepared, always-prepared, known, extra and granted cantrips,
// deduplicated; spellbook spells count only when prepared; empty gives empty.
// Path: per case (unhappy empty and minimal).
func TestPreparedSpellSetFromSpellcasting(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	granted := must(character.NewGrantedCantrip(spellID(0, 9), character.CasterWizard, classOf("cleric")))
	wizard := character.MustClassSpellcasting(character.ClassSpellcastingSetup{
		Class: character.CasterWizard, Cantrips: []content.SpellID{spellID(0, 1), spellID(0, 2), spellID(0, 3), spellID(0, 4)},
		Prepared: []content.SpellID{spellID(1, 1)}, AlwaysPrepared: []content.SpellID{spellID(1, 7)}, ExtraCantrips: 1,
		Granted: []character.GrantedCantrip{granted}, Spellbook: character.NewSpellbook(spellIDs(1, 6)),
	}, 1, spells)
	cleric := character.MustClassSpellcasting(character.ClassSpellcastingSetup{
		Class: character.CasterCleric, Cantrips: []content.SpellID{spellID(0, 1)}, Prepared: []content.SpellID{spellID(1, 1), spellID(1, 2)},
	}, 1, spells)
	cases := []struct {
		name    string
		classes []character.ClassSpellcasting
		want    []content.SpellID
		absent  []content.SpellID
	}{
		{"empty input", nil, nil, nil},
		{"a Wizard: prepared, always prepared, cantrips, extra and granted; spellbook alone is not prepared",
			[]character.ClassSpellcasting{wizard},
			[]content.SpellID{spellID(0, 1), spellID(0, 2), spellID(0, 3), spellID(0, 4), spellID(0, 9), spellID(1, 1), spellID(1, 7)},
			[]content.SpellID{spellID(1, 2), spellID(1, 6)}},
		{"two classes are united and deduplicated", []character.ClassSpellcasting{wizard, cleric},
			[]content.SpellID{spellID(0, 1), spellID(0, 2), spellID(0, 3), spellID(0, 4), spellID(0, 9), spellID(1, 1), spellID(1, 2), spellID(1, 7)}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			set := character.PreparedSpellSetFromSpellcasting(c.classes)
			if got := set.IDs(); !slices.Equal(got, slices.Sorted(slices.Values(c.want))) {
				t.Errorf("IDs = %v, want %v", got, c.want)
			}
			for _, id := range c.want {
				if !set.Contains(id) {
					t.Errorf("set lacks %s", id)
				}
			}
			for _, id := range c.absent {
				if set.Contains(id) {
					t.Errorf("set holds %s", id)
				}
			}
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-042, FR-048 - the set is
// sorted and deduplicated, its slice is a copy on construction and on read, and
// sets compare by value.
// Path: unhappy (immutability).
func TestPreparedSpellSetIsImmutable(t *testing.T) {
	t.Parallel()
	in := []content.SpellID{"b_spell", "a_spell", "b_spell"}
	s := character.NewPreparedSpellSet(in...)
	in[0] = "changed"
	got := s.IDs()
	if !slices.Equal(got, []content.SpellID{"a_spell", "b_spell"}) {
		t.Fatalf("IDs = %v", got)
	}
	got[0] = "changed"
	if !s.Contains("a_spell") || s.Contains("changed") {
		t.Error("IDs aliases the set's own slice")
	}
	if !s.Equal(character.NewPreparedSpellSet("a_spell", "b_spell")) || s.Equal(character.NewPreparedSpellSet("a_spell")) {
		t.Error("Equal is wrong")
	}
	if (character.PreparedSpellSet{}).Contains("a_spell") {
		t.Error("the zero set must be empty")
	}
}
