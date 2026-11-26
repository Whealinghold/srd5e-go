package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/attacks.md FR-013, FR-014, FR-016 - Half Cover gives +2 to AC and Dexterity saves,
// Three-Quarters +5; Total Cover and no cover give none (SRD 5.2.1, "Cover", p. 15).
// Path: per case.
func TestCoverBonuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		cover                   combat.Cover
		bonus                   int
		text                    string
		valid                   bool
	}{
		{"none", "FR-016", "unhappy (empty and minimal case)", 0, 0, "none", true},
		{"half", "FR-013", "happy", combat.HalfCover, 2, "half", true},
		{"three-quarters", "FR-013", "happy", combat.ThreeQuartersCover, 5, "three_quarters", true},
		{"total", "FR-014", "happy", combat.TotalCover, 0, "total", true},
		{"past total", "FR-014", "unhappy (invalid input)", combat.TotalCover + 1, 0, "unknown", false},
		{"negative", "FR-014", "unhappy (invalid input)", -1, 0, "unknown", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.cover.ACBonus() != c.bonus || c.cover.DexteritySaveBonus() != c.bonus || c.cover.String() != c.text || c.cover.Valid() != c.valid {
				t.Errorf("%d = AC %d Dex %d %q valid %v, want %d %q %v [%s]", c.cover, c.cover.ACBonus(), c.cover.DexteritySaveBonus(),
					c.cover, c.cover.Valid(), c.bonus, c.text, c.valid, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-017, FR-018 - only the most protective cover on the opposite side counts, degrees
// are never added, Total over Three-Quarters over Half (SRD 5.2.1, "Cover", p. 15).
// Path: per case.
func TestCoverResolverApplicable(t *testing.T) {
	t.Parallel()
	half, tq, total := combat.HalfCover, combat.ThreeQuartersCover, combat.TotalCover
	on := func(c combat.Cover) combat.CoverSource {
		return combat.CoverSource{Cover: c, AttackOriginatesOppositeSide: true}
	}
	off := func(c combat.Cover) combat.CoverSource { return combat.CoverSource{Cover: c} }
	cases := []struct {
		name, requirement, path string
		sources                 []combat.CoverSource
		want                    combat.Cover
	}{
		{"nil", "FR-016", "unhappy (empty and minimal case)", nil, 0},
		{"empty", "FR-016", "unhappy (empty and minimal case)", []combat.CoverSource{}, 0},
		{"half", "FR-017", "happy", []combat.CoverSource{on(half)}, half},
		{"two half are +2, not +4", "FR-017", "unhappy (rule interaction)", []combat.CoverSource{on(half), on(half)}, half},
		{"half and three-quarters", "FR-017", "unhappy (rule interaction)", []combat.CoverSource{on(half), on(tq)}, tq},
		{"three-quarters first", "FR-017", "unhappy (rule interaction)", []combat.CoverSource{on(tq), on(half)}, tq},
		{"total over half", "FR-017", "unhappy (rule interaction)", []combat.CoverSource{on(half), on(total)}, total},
		{"total over three-quarters", "FR-017", "unhappy (rule interaction)", []combat.CoverSource{on(total), on(tq)}, total},
		{"not on the opposite side", "FR-018", "unhappy (rule interaction)", []combat.CoverSource{off(tq)}, 0},
		{"half and a far three-quarters", "FR-018", "unhappy (rule interaction)", []combat.CoverSource{on(half), off(tq)}, half},
		{"an invalid degree is ignored", "FR-018", "unhappy (invalid input)", []combat.CoverSource{on(total + 3), on(half)}, half},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (combat.CoverResolver{}).Applicable(c.sources); got != c.want {
				t.Errorf("Applicable(%v) = %v, want %v [%s]", c.sources, got, c.want, c.requirement)
			}
		})
	}
}
