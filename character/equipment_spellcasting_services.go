package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpellcastingServices is the Spellcasting Services table (SRD 5.2.1, p. 102):
// rows whose spell level ranges cover each level 0 to 9 exactly once. Build it
// with [NewSpellcastingServices]. IsAvailable and TotalCost use no table state,
// so they work on the zero value. It is an immutable value that holds its own
// copy of the rows; it holds a slice, so it is not comparable with ==.
type SpellcastingServices struct {
	rows []content.SpellcastingService
}

// NewSpellcastingServices returns the table of rows. It copies the rows. It
// returns a *[TableCoverageError] (matching srderr.ErrInvalidInput) with the
// uncovered and overlapping levels, each ordered by level, unless the ranges
// cover each level 0 to 9 exactly once. A row whose minimum is above its maximum
// covers nothing.
func NewSpellcastingServices(rows []content.SpellcastingService) (SpellcastingServices, error) {
	var counts [content.MaxSpellLevel + 1]int
	for _, r := range rows {
		for level := max(r.SpellLevelMin(), content.MinSpellLevel); level <= min(r.SpellLevelMax(), content.MaxSpellLevel); level++ {
			counts[level]++
		}
	}
	if uncovered, overlapping := coverage(counts); len(uncovered) > 0 || len(overlapping) > 0 {
		return SpellcastingServices{}, &TableCoverageError{Uncovered: uncovered, Overlapping: overlapping}
	}
	return SpellcastingServices{rows: slices.Clone(rows)}, nil
}

// MustSpellcastingServices is like [NewSpellcastingServices] but panics with its
// error when the ranges do not cover each level once.
func MustSpellcastingServices(rows []content.SpellcastingService) SpellcastingServices {
	s, err := NewSpellcastingServices(rows)
	if err != nil {
		panic(err)
	}
	return s
}

// ForSpellLevel returns the row whose range contains level. It returns the zero
// row for the zero table or a level outside 0 to 9.
func (s SpellcastingServices) ForSpellLevel(level content.SpellLevel) content.SpellcastingService {
	for _, r := range s.rows {
		if int(level) >= r.SpellLevelMin() && int(level) <= r.SpellLevelMax() {
			return r
		}
	}
	return content.SpellcastingService{}
}

// IsAvailable reports whether row can be found in a settlement of size: in a
// Village, Town or City for "Village, town, or city", in a Town or City for
// "Town or city only", in a City only for "City only". An invalid size or row
// gives false.
func (SpellcastingServices) IsAvailable(row content.SpellcastingService, size content.SettlementSize) bool {
	least := row.Availability().MinimumSize()
	return size.Valid() && least.Valid() && size >= least
}

// TotalCost returns the row's listed cost plus componentCost, the cost of any
// expensive components, as a Copper Piece cost (SRD 5.2.1, "Spellcasting", p.
// 102). The zero cost means no components. It returns an error matching
// srderr.ErrInvalidInput for a componentCost with a rate or a total above
// [content.MaxCostAmount].
func (SpellcastingServices) TotalCost(row content.SpellcastingService, componentCost content.Cost) (content.Cost, error) {
	if _, rated := componentCost.Per(); rated {
		return content.Cost{}, invalidInput("component cost")
	}
	total := row.Price().Cost().ValueInCopper() + componentCost.ValueInCopper()
	if total > content.MaxCostAmount {
		return content.Cost{}, invalidInput("total cost out of range")
	}
	return content.NewCost(int(total), content.CoinCP, 0)
}
