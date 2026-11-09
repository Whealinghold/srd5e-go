package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: tool-proficiency-and-crafting.md FR-001 - the set copies its
// input, ignores order and duplicates, reads and compares by value, and an
// empty set is the zero value (SRD 5.2.1, "Tool Proficiency", p. 93).
// Path: unhappy (immutability).
func TestToolProficienciesAreImmutableValues(t *testing.T) {
	t.Parallel()
	in := idsOf("thieves_tools", "herbalism_kit", "thieves_tools")
	p := character.NewToolProficiencies(in...)
	in[0] = "changed"
	if got := p.IDs(); !slices.Equal(got, idsOf("herbalism_kit", "thieves_tools")) {
		t.Errorf("IDs = %v, want sorted and deduplicated, unaffected by the input", got)
	}
	ids := p.IDs()
	ids[0] = "changed"
	if got := p.IDs(); got[0] != "herbalism_kit" {
		t.Errorf("IDs after mutating a returned slice = %v", got)
	}
	if !p.Equal(character.NewToolProficiencies(idsOf("thieves_tools", "herbalism_kit")...)) {
		t.Error("equal sets in another order are not Equal")
	}
	if p.Equal(character.NewToolProficiencies("herbalism_kit")) {
		t.Error("different sets are Equal")
	}
	if !p.Contains("thieves_tools") || p.Contains("smiths_tools") {
		t.Error("Contains is wrong")
	}
	plus := p.Plus("smiths_tools")
	if !plus.Contains("smiths_tools") || !plus.Contains("thieves_tools") || !plus.Contains("herbalism_kit") || p.Contains("smiths_tools") {
		t.Error("Plus must keep the old ids, add the new one and leave the receiver unchanged")
	}
	var zero character.ToolProficiencies
	if zero.Contains("thieves_tools") || len(zero.IDs()) != 0 || !zero.Equal(character.NewToolProficiencies()) {
		t.Error("the zero value is not the empty set")
	}
	if got := zero.Plus("a").IDs(); !slices.Equal(got, idsOf("a")) {
		t.Errorf("zero.Plus = %v", got)
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-002 - proficiency is per
// entry id; a variant does not count for another variant or for its family, and
// the family id does not count for a variant (SRD 5.2.1, "Tools", p. 93).
// Path: unhappy (rule interaction).
func TestToolProficiencyIsPerEntryID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		held        []content.ItemID
		asked       content.ItemID
		wantContain bool
	}{
		{"dice does not count for playing cards", idsOf("gaming_set_dice"), "gaming_set_playing_cards", false},
		{"a variant does not count for the family id", idsOf("gaming_set_dice"), "gaming_set", false},
		{"the family id does not count for a variant", idsOf("gaming_set"), "gaming_set_dice", false},
		{"the same variant counts", idsOf("gaming_set_dice"), "gaming_set_dice", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := character.NewToolProficiencies(c.held...).Contains(c.asked); got != c.wantContain {
				t.Errorf("Contains(%s) = %v, want %v", c.asked, got, c.wantContain)
			}
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-003 - a monster is proficient
// with exactly the tools of its stat block (SRD 5.2.1, "Tool Proficiency", p. 93).
// Path: happy.
func TestMonsterToolProficiencies(t *testing.T) {
	t.Parallel()
	m := character.MonsterToolProficiencies("thieves_tools")
	if !slices.Equal(m.IDs(), idsOf("thieves_tools")) {
		t.Errorf("IDs = %v", m.IDs())
	}
	if !character.MonsterToolProficiencies().Equal(character.ToolProficiencies{}) {
		t.Error("a stat block with no tools is not the empty set")
	}
}
