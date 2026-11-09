package character_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// fixtureTables reads testdata/srd/spellcaster_slots.txt: section name to the
// slots (spell levels 1 to 9) at each level 1 to 20.
func fixtureTables(t *testing.T) map[string][][9]int {
	t.Helper()
	raw, err := os.ReadFile("testdata/srd/spellcaster_slots.txt")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][][9]int{}
	section := ""
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		switch {
		case len(f) == 0 || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "["):
			section = strings.Trim(line, "[]")
		default:
			var row [9]int
			for i, s := range f[1:] {
				row[i], _ = strconv.Atoi(s)
			}
			out[section] = append(out[section], row)
		}
	}
	return out
}

// Requirement: doc/specs/spellcasting.md FR-039, FR-010 - the Multiclass Spellcaster
// table (p. 26) and each full caster's own Features table, transcribed apart from the
// production tables in testdata/srd, agree with the production lookup for every
// level 1 to 20.
// Path: happy.
func TestFullCasterSlotsMatchIndependentFixtures(t *testing.T) {
	t.Parallel()
	tables := fixtureTables(t)
	for name, rows := range tables {
		if len(rows) != 20 {
			t.Fatalf("fixture %s has %d rows, want 20", name, len(rows))
		}
	}
	for _, class := range []string{"bard", "cleric", "druid", "sorcerer", "wizard"} {
		for level := 1; level <= 20; level++ {
			single := character.SlotsFor(classLevelsOf(t, class, level)).Spellcasting()
			// A second class that adds a level at the same total spellcaster level checks the multiclass lookup.
			for _, want := range [][9]int{tables[class][level-1], tables["multiclass-spellcaster"][level-1]} {
				for l := 1; l <= 9; l++ {
					if got := single.Maximum(character.MustSlotLevel(l)); got != want[l-1] {
						t.Errorf("%s %d: spell level %d maximum = %d, want %d", class, level, l, got, want[l-1])
					}
				}
			}
		}
	}
	for total := 2; total <= 20; total++ {
		pools := character.SlotsFor(classLevelsOf(t, "wizard", total-1, "cleric", 1))
		for l := 1; l <= 9; l++ {
			if got, want := pools.Spellcasting().Maximum(character.MustSlotLevel(l)), tables["multiclass-spellcaster"][total-1][l-1]; got != want {
				t.Errorf("wizard %d / cleric 1: spell level %d = %d, want %d", total-1, l, got, want)
			}
		}
	}
}

// Requirement: per case.
// Path: per case.
func TestHalfCasterSlotsTableD(t *testing.T) {
	t.Parallel()
	want := [][]int{{2}, {2}, {3}, {3}, {4, 2}, {4, 2}, {4, 3}, {4, 3}, {4, 3, 2}, {4, 3, 2}, {4, 3, 3}, {4, 3, 3}, {4, 3, 3, 1}, {4, 3, 3, 1},
		{4, 3, 3, 2}, {4, 3, 3, 2}, {4, 3, 3, 3, 1}, {4, 3, 3, 3, 1}, {4, 3, 3, 3, 2}, {4, 3, 3, 3, 2}}
	for _, class := range []string{"paladin", "ranger"} {
		for level := 1; level <= 20; level++ {
			t.Run(class+"_"+strconv.Itoa(level), func(t *testing.T) {
				t.Parallel()
				s := character.SlotsFor(classLevelsOf(t, class, level)).Spellcasting()
				for l := 1; l <= 9; l++ {
					w := 0
					if l <= len(want[level-1]) {
						w = want[level-1][l-1]
					}
					if got := s.Maximum(character.MustSlotLevel(l)); got != w {
						t.Errorf("FR-011/FR-013 %s %d: spell level %d = %d, want %d", class, level, l, got, w)
					}
				}
			})
		}
	}
}

// Requirement: doc/specs/spellcasting.md FR-012, FR-016, Table E - Warlock Pact Magic
// slot count and Slot Level at every level 1 to 20.
// Path: happy.
func TestPactMagicTableE(t *testing.T) {
	t.Parallel()
	counts := []int{1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4}
	slotLevels := []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	for level := 1; level <= 20; level++ {
		p := character.SlotsFor(classLevelsOf(t, "warlock", level)).PactMagic()
		if !p.HasPactMagic() || p.Count() != counts[level-1] || p.Remaining() != counts[level-1] || p.Level().Value() != slotLevels[level-1] {
			t.Errorf("warlock %d: %d x level %d (remaining %d), want %d x level %d", level, p.Count(), p.Level().Value(), p.Remaining(), counts[level-1], slotLevels[level-1])
		}
		if got := character.MaxPreparableLevel(character.CasterWarlock, level); got != slotLevels[level-1] {
			t.Errorf("MaxPreparableLevel(warlock, %d) = %d, want %d", level, got, slotLevels[level-1])
		}
	}
}

// Requirement: doc/specs/spellcasting.md FR-014, FR-018, Table F - cantrips known by
// band, with the first and last level of each band pinned, and none for Paladin and Ranger.
// Path: unhappy (boundary).
func TestCantripLimitTableF(t *testing.T) {
	t.Parallel()
	bands := map[character.SpellcastingClass][3]int{
		character.CasterBard: {2, 3, 4}, character.CasterCleric: {3, 4, 5}, character.CasterDruid: {2, 3, 4},
		character.CasterSorcerer: {4, 5, 6}, character.CasterWarlock: {2, 3, 4}, character.CasterWizard: {3, 4, 5},
	}
	for class, b := range bands {
		for level, want := range map[int]int{1: b[0], 3: b[0], 4: b[1], 9: b[1], 10: b[2], 20: b[2]} {
			if got := character.CantripLimit(class, level); got != want {
				t.Errorf("CantripLimit(%v, %d) = %d, want %d", class, level, got, want)
			}
		}
	}
	for _, class := range []character.SpellcastingClass{character.CasterPaladin, character.CasterRanger, 0} {
		for level := 1; level <= 20; level++ {
			if got := character.CantripLimit(class, level); got != 0 {
				t.Errorf("CantripLimit(%v, %d) = %d, want 0", class, level, got)
			}
		}
	}
	if character.CantripLimit(character.CasterWizard, 0) != 0 || character.CantripLimit(character.CasterWizard, 21) != 0 {
		t.Error("CantripLimit outside 1 to 20 is not 0")
	}
}

// Requirement: doc/specs/spellcasting.md FR-015, Table G - prepared spells for every
// class at every level 1 to 20.
// Path: happy.
func TestPreparedLimitTableG(t *testing.T) {
	t.Parallel()
	bcd := []int{4, 5, 6, 7, 9, 10, 11, 12, 14, 15, 16, 16, 17, 17, 18, 18, 19, 20, 21, 22}
	half := []int{2, 3, 4, 5, 6, 6, 7, 7, 9, 9, 10, 10, 11, 11, 12, 12, 14, 14, 15, 15}
	sorc := []int{2, 4, 6, 7, 9, 10, 11, 12, 14, 15, 16, 16, 17, 17, 18, 18, 19, 20, 21, 22}
	lock := []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 10, 11, 11, 12, 12, 13, 13, 14, 14, 15, 15}
	wiz := []int{4, 5, 6, 7, 9, 10, 11, 12, 14, 15, 16, 16, 17, 18, 19, 21, 22, 23, 24, 25}
	table := map[character.SpellcastingClass][]int{
		character.CasterBard: bcd, character.CasterCleric: bcd, character.CasterDruid: bcd, character.CasterPaladin: half,
		character.CasterRanger: half, character.CasterSorcerer: sorc, character.CasterWarlock: lock, character.CasterWizard: wiz,
	}
	for class, want := range table {
		for level := 1; level <= 20; level++ {
			if got := character.PreparedLimit(class, level); got != want[level-1] {
				t.Errorf("PreparedLimit(%v, %d) = %d, want %d", class, level, got, want[level-1])
			}
		}
		if character.PreparedLimit(class, 0) != 0 || character.PreparedLimit(class, 21) != 0 {
			t.Errorf("PreparedLimit(%v) outside 1 to 20 is not 0", class)
		}
	}
	if character.PreparedLimit(0, 5) != 0 {
		t.Error("PreparedLimit(invalid class) is not 0")
	}
}

// Requirement: per case.
// Path: per case.
func TestMaxPreparableLevelTableJ(t *testing.T) {
	t.Parallel()
	cases := []struct {
		label string
		class character.SpellcastingClass
		level int
		want  int
		path  string
	}{
		{"J1 wizard 1", character.CasterWizard, 1, 1, "unhappy (boundary)"}, {"J1 wizard 2", character.CasterWizard, 2, 1, "unhappy (boundary)"},
		{"J2 wizard 3", character.CasterWizard, 3, 2, "unhappy (boundary)"}, {"J3 wizard 17", character.CasterWizard, 17, 9, "unhappy (boundary)"},
		{"J4 paladin 4", character.CasterPaladin, 4, 1, "unhappy (boundary)"}, {"J5 paladin 5", character.CasterPaladin, 5, 2, "unhappy (boundary)"},
		{"J6 paladin 17", character.CasterPaladin, 17, 5, "unhappy (boundary)"}, {"J7 warlock 2", character.CasterWarlock, 2, 1, "unhappy (boundary)"},
		{"J8 warlock 3", character.CasterWarlock, 3, 2, "unhappy (boundary)"}, {"J9 warlock 9", character.CasterWarlock, 9, 5, "unhappy (boundary)"},
		{"J10 wizard 3 of a Wizard 3 / Cleric 6 build uses its own table", character.CasterWizard, 3, 2, "unhappy (rule interaction)"},
		{"invalid class", 0, 5, 0, "unhappy (invalid input)"}, {"level 0", character.CasterWizard, 0, 0, "unhappy (invalid input)"},
		{"level 21", character.CasterWizard, 21, 0, "unhappy (invalid input)"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			if got := character.MaxPreparableLevel(c.class, c.level); got != c.want {
				t.Errorf("MaxPreparableLevel(%v, %d) = %d, want %d [FR-016 FR-017 FR-036 Table J]", c.class, c.level, got, c.want)
			}
		})
	}
}
