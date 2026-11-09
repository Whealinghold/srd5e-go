package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func scrollRow(level, days int, cost content.Cost) content.SpellScrollCost {
	return must(content.NewSpellScrollCost(content.ItemID("scroll_row_x"), "Row", content.SourceCitation{}, level, days, price(cost)))
}

// scrollRows returns the printed Spell Scroll Costs table.
func scrollRows() []content.SpellScrollCost {
	days := []int{1, 1, 3, 5, 10, 25, 40, 50, 60, 120}
	gold := []int{15, 25, 100, 150, 1000, 1500, 10000, 12500, 15000, 50000}
	rows := make([]content.SpellScrollCost, 0, 10)
	for l := range days {
		rows = append(rows, scrollRow(l, days[l], gp(gold[l])))
	}
	return rows
}

// Requirement: per case equipSpec FR-049 (SRD 5.2.1, "Scribing Spell Scrolls",
// Spell Scroll Costs, p. 103) - the ten rows, levels 0 to 9.
// Path: per case (happy; unhappy boundary for level 9).
func TestScrollScribingRows(t *testing.T) {
	t.Parallel()
	table := character.MustScrollScribing(scrollRows())
	days := []int{1, 1, 3, 5, 10, 25, 40, 50, 60, 120}
	gold := []int{15, 25, 100, 150, 1000, 1500, 10000, 12500, 15000, 50000}
	for l := range days {
		row := table.ForSpellLevel(levelOf(t, l))
		if row.SpellLevel() != l || row.Days() != days[l] || row.Price().Cost() != gp(gold[l]) {
			t.Errorf("level %d = %d days, %v", l, row.Days(), row.Price().Cost())
		}
	}
	if table.IsZero() {
		t.Error("a built table is not zero")
	}
	if got := table.ForSpellLevel(content.SpellLevel(10)); got != (content.SpellScrollCost{}) {
		t.Errorf("level 10 = %v, want the zero row", got)
	}
}

// Requirement: equipSpec FR-050, FR-051 - rows that do not cover each level
// once give a *TableCoverageError (an srderr.ErrInvalidInput error) listing the
// missing and repeated levels in order; no rows lists all ten; MustScrollScribing
// panics with that error.
// Path: unhappy (missing content, invalid input).
func TestNewScrollScribingCoverage(t *testing.T) {
	t.Parallel()
	rows := scrollRows()
	cases := []struct {
		name     string
		rows     []content.SpellScrollCost
		missing  []content.SpellLevel
		repeated []content.SpellLevel
	}{
		{"0 to 8 only", rows[:9], []content.SpellLevel{9}, nil},
		{"level 3 twice, 4 absent", slices.Concat(rows[:4], rows[3:4], rows[5:]), []content.SpellLevel{4}, []content.SpellLevel{3}},
		{"nil", nil, []content.SpellLevel{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, nil},
		{"complete plus a repeat", append(slices.Clone(rows), rows[0]), nil, []content.SpellLevel{0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			table, err := character.NewScrollScribing(c.rows)
			var cov *character.TableCoverageError
			if !errors.As(err, &cov) || !slices.Equal(cov.Missing, c.missing) || !slices.Equal(cov.Repeated, c.repeated) || !table.IsZero() {
				t.Fatalf("err = %v, table zero %v; want missing %v repeated %v", err, table.IsZero(), c.missing, c.repeated)
			}
			wantInvalid(t, err, "coverage")
			defer func() {
				if r := recover(); !errors.Is(r.(error), cov) {
					t.Errorf("Must panicked with %v, want %v", r, cov)
				}
			}()
			character.MustScrollScribing(c.rows)
			t.Error("MustScrollScribing did not panic")
		})
	}
}

// Requirement: equipSpec FR-067 - the table copies the rows it is given and the
// zero value is "no table"; upstream rejects spell levels outside 0 to 9
// (content-spells.md FR-010).
// Path: unhappy (immutability, invalid input).
func TestScrollScribingCopiesRows(t *testing.T) {
	t.Parallel()
	rows := scrollRows()
	table := character.MustScrollScribing(rows)
	rows[2] = scrollRow(2, 99, gp(1))
	if got := table.ForSpellLevel(levelOf(t, 2)); got.Days() != 3 {
		t.Errorf("level 2 after mutating the source = %d days, want 3", got.Days())
	}
	if !(character.ScrollScribing{}).IsZero() || (character.ScrollScribing{}).ForSpellLevel(levelOf(t, 1)) != (content.SpellScrollCost{}) {
		t.Error("the zero table is empty")
	}
	for _, n := range []int{-1, 10} {
		_, err := content.NewSpellLevel(n)
		wantInvalid(t, err, "NewSpellLevel")
	}
}
