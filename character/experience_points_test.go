package character_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/character-aggregate.md FR-004 - MinimumForLevel returns
// the Character Advancement XP column for levels 1 to 20 (SRD 5.2.1, "Level
// Advancement", p. 23).
// Path: happy.
func TestMinimumForLevel(t *testing.T) {
	t.Parallel()
	want := []int{0, 300, 900, 2700, 6500, 14000, 23000, 34000, 48000, 64000, 85000, 100000, 120000, 140000, 165000, 195000, 225000, 265000, 305000, 355000}
	for i, w := range want {
		if got := (character.ExperiencePoints{}).MinimumForLevel(character.MustLevel(i + 1)).Value(); got != w {
			t.Errorf("level %d: %d, want %d", i+1, got, w)
		}
	}
	if got := (character.ExperiencePoints{}).MinimumForLevel(character.Level{}); got.Value() != 0 {
		t.Errorf("zero level: %d", got.Value())
	}
}

// Requirement: doc/specs/character-aggregate.md FR-006, FR-007 - LevelFor returns
// the highest level whose minimum the total equals or exceeds, 20 above 355,000.
// Path: per case.
func TestLevelFor(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ xp, level int }{
		{0, 1}, {299, 1}, {300, 2}, {899, 2}, {900, 3}, {6500, 5}, {354999, 19}, {355000, 20}, {400000, 20}, {math.MaxInt, 20},
	} {
		if got := character.MustExperiencePoints(c.xp).LevelFor().Value(); got != c.level {
			t.Errorf("LevelFor(%d) = %d, want %d", c.xp, got, c.level)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-113, FR-114 - bonus feats: none
// for NoBonusFeats; one per complete 30,000 XP above the level 20 minimum,
// rounded down (SRD 5.2.1, "Bonus Feats at Level 20", p. 24; "Round Down", p. 5).
// Path: per case.
func TestBonusFeats(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ xp, feats int }{
		{0, 0}, {355000, 0}, {355001, 0}, {384999, 0}, {385000, 1}, {414999, 1}, {415000, 2}, {1000000, 21},
	} {
		xp := character.MustExperiencePoints(c.xp)
		if got := (character.BonusFeatsFromXp{}).FeatsEarned(xp); got != c.feats {
			t.Errorf("BonusFeatsFromXp(%d) = %d, want %d", c.xp, got, c.feats)
		}
		if got := (character.NoBonusFeats{}).FeatsEarned(xp); got != 0 {
			t.Errorf("NoBonusFeats(%d) = %d, want 0", c.xp, got)
		}
	}
}
