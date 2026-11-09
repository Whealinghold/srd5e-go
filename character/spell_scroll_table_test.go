package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestSpellScrollTableRow(t *testing.T) {
	t.Parallel()
	const req = `spell-scroll-use.md FR-001, FR-002 (SRD 5.2.1, "Spell Scroll", p. 244)`
	cases := []struct {
		name, path string
		level      int
		rarity     content.ItemRarity
		dc, atk    int
	}{
		{"cantrip is the lowest row", "unhappy (boundary)", 0, content.RarityCommon, 13, 5},
		{"level 1", "happy", 1, content.RarityCommon, 13, 5},
		{"level 2 is the last 13", "unhappy (boundary)", 2, content.RarityUncommon, 13, 5},
		{"level 3 is the first DC step", "unhappy (boundary)", 3, content.RarityUncommon, 15, 7},
		{"level 4", "happy", 4, content.RarityRare, 15, 7},
		{"level 5", "happy", 5, content.RarityRare, 17, 9},
		{"level 6 is the last 17", "unhappy (boundary)", 6, content.RarityVeryRare, 17, 9},
		{"level 7 is the first plus 10", "unhappy (boundary)", 7, content.RarityVeryRare, 18, 10},
		{"level 8", "happy", 8, content.RarityVeryRare, 18, 10},
		{"level 9 is the highest row", "unhappy (boundary)", 9, content.RarityLegendary, 19, 11},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			row := character.SpellScrollTableRow(content.SpellLevel(c.level))
			if row.SpellLevel() != content.SpellLevel(c.level) || row.Rarity() != c.rarity || row.SaveDC() != c.dc || row.AttackBonus() != c.atk {
				t.Errorf("row(%d) = %d %v DC %d %+d, want %v DC %d %+d [%s %s]", c.level, row.SpellLevel(), row.Rarity(), row.SaveDC(), row.AttackBonus(), c.rarity, c.dc, c.atk, req, c.path)
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-001 - a level outside 0 to 9 (reachable only by conversion) gives the zero row.
// Path: unhappy (invalid input).
func TestSpellScrollTableRowOutOfRangeIsZero(t *testing.T) {
	t.Parallel()
	for _, l := range []int{-1, 10} {
		if row := character.SpellScrollTableRow(content.SpellLevel(l)); row != (character.SpellScrollRow{}) {
			t.Errorf("row(%d) = %+v, want the zero row", l, row)
		}
	}
}
