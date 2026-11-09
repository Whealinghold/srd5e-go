package character

import "github.com/Whealinghold/srd5e-go/content"

// SpellScrollRow is one row of the Spell Scroll table: the rarity, spell save DC
// and spell attack bonus of a scroll of one spell level (SRD 5.2.1, "Spell
// Scroll", p. 244). Rarity is data for hosts; no rule here reads it. It is
// comparable. The zero SpellScrollRow is not a row of the table.
type SpellScrollRow struct {
	level       content.SpellLevel
	rarity      content.ItemRarity
	saveDC      int
	attackBonus int
}

// SpellLevel returns the spell level the row is for.
func (r SpellScrollRow) SpellLevel() content.SpellLevel { return r.level }

// Rarity returns the rarity of a scroll of that level.
func (r SpellScrollRow) Rarity() content.ItemRarity { return r.rarity }

// SaveDC returns the spell save DC of a found scroll of that level.
func (r SpellScrollRow) SaveDC() int { return r.saveDC }

// AttackBonus returns the spell attack bonus of a found scroll of that level.
func (r SpellScrollRow) AttackBonus() int { return r.attackBonus }

// SpellScrollTableRow returns the row of the Spell Scroll table for spell level l
// (SRD 5.2.1, "Spell Scroll", p. 244; "Spell Scroll (Cantrip)", p. 99). It is
// total over levels 0 to 9 and returns the zero row for any other value, which
// only a conversion such as content.SpellLevel(10) can produce. It never
// panics. The table is a typed projection of the bundled catalogue entry, kept
// as code because package character reads no content source.
func SpellScrollTableRow(l content.SpellLevel) SpellScrollRow {
	if !content.ValidSpellLevel(int(l)) {
		return SpellScrollRow{}
	}
	row := SpellScrollRow{level: l}
	switch {
	case l <= 2:
		row.saveDC, row.attackBonus = 13, 5
	case l <= 4:
		row.saveDC, row.attackBonus = 15, 7
	case l <= 6:
		row.saveDC, row.attackBonus = 17, 9
	case l <= 8:
		row.saveDC, row.attackBonus = 18, 10
	default:
		row.saveDC, row.attackBonus = 19, 11
	}
	switch {
	case l <= 1:
		row.rarity = content.RarityCommon
	case l <= 3:
		row.rarity = content.RarityUncommon
	case l <= 5:
		row.rarity = content.RarityRare
	case l <= 8:
		row.rarity = content.RarityVeryRare
	default:
		row.rarity = content.RarityLegendary
	}
	return row
}
