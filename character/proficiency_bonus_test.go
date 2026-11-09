package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/derived-stats.md FR-059 - +2 for levels 1 to 4, +3 for
// 5 to 8, +4 for 9 to 12, +5 for 13 to 16, +6 for 17 to 20 (SRD 5.2.1, "Level
// Advancement", p. 23).
// Path: per case.
func TestProficiencyBonusFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		level int
		want  int
	}{
		{"level 1", 1, 2}, {"level 4", 4, 2}, {"level 5", 5, 3}, {"level 8", 8, 3},
		{"level 9", 9, 4}, {"level 12", 12, 4}, {"level 13", 13, 5}, {"level 16", 16, 5},
		{"level 17", 17, 6}, {"level 20", 20, 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := character.ProficiencyBonusFor(character.MustLevel(c.level))
			if b.Value() != c.want || b.IsZero() {
				t.Errorf("ProficiencyBonusFor(%d) = %d, want %d [SRD 5.2.1, p. 23]", c.level, b.Value(), c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-059 - the zero Level has the zero
// bonus.
// Path: unhappy (invalid input).
func TestProficiencyBonusForZeroLevel(t *testing.T) {
	t.Parallel()
	if b := character.ProficiencyBonusFor(character.Level{}); !b.IsZero() || b.Value() != 0 {
		t.Errorf("ProficiencyBonusFor(zero) = %d, want the zero bonus", b.Value())
	}
}
