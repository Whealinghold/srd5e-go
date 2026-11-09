package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func dieMap() map[content.ClassID]content.HitPointDie {
	return map[content.ClassID]content.HitPointDie{
		"fighter": content.HitDieD10, "paladin": content.HitDieD10, "cleric": content.HitDieD8, "wizard": content.HitDieD6,
	}
}

func pool(d content.HitPointDie, n int) character.HitDiceCount {
	return character.HitDiceCount{Die: d, Count: n}
}

// Requirement: doc/specs/derived-stats.md FR-061 - one die per class level,
// same dice pooled, different dice apart, ordered by faces, total equals the
// total level (SRD 5.2.1, "Multiclassing", p. 25).
// Path: per case.
func TestHitDicePoolFromLevels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		levels     []character.ClassLevel
		dice       map[content.ClassID]content.HitPointDie
		want       []character.HitDiceCount
	}{
		{"single class", "happy", []character.ClassLevel{cl("fighter", 3)}, dieMap(), []character.HitDiceCount{pool(content.HitDieD10, 3)}},
		{"same die types pool", "unhappy (rule interaction)", []character.ClassLevel{cl("fighter", 5), cl("paladin", 5)}, dieMap(), []character.HitDiceCount{pool(content.HitDieD10, 10)}},
		{"different die types stay separate", "unhappy (rule interaction)", []character.ClassLevel{cl("cleric", 5), cl("paladin", 5)}, dieMap(), []character.HitDiceCount{pool(content.HitDieD8, 5), pool(content.HitDieD10, 5)}},
		{"minimal", "unhappy (empty and minimal)", []character.ClassLevel{cl("wizard", 1)}, dieMap(), []character.HitDiceCount{pool(content.HitDieD6, 1)}},
		{"entries ordered by faces", "unhappy (determinism)", []character.ClassLevel{cl("fighter", 1), cl("wizard", 1)}, dieMap(), []character.HitDiceCount{pool(content.HitDieD6, 1), pool(content.HitDieD10, 1)}},
		{"extra definition ignored", "unhappy (rule interaction)", []character.ClassLevel{cl("fighter", 3)}, dieMap(), []character.HitDiceCount{pool(content.HitDieD10, 3)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := character.HitDicePoolFromLevels(character.MustClassLevels(c.levels), c.dice)
			if err != nil || !slices.Equal(p.Entries(), c.want) {
				t.Fatalf("pool = %v, %v, want %v", p.Entries(), err, c.want)
			}
			if p.Total() != character.MustClassLevels(c.levels).TotalLevel().Value() {
				t.Errorf("Total = %d, want the total level", p.Total())
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-061 - a class without a die is an
// error, also with an empty or nil map; the Must form panics with it.
// Path: unhappy (invalid input).
func TestHitDicePoolFromLevelsClassWithoutDie(t *testing.T) {
	t.Parallel()
	l := character.MustClassLevels([]character.ClassLevel{cl("fighter", 3), cl("rogue", 2)})
	for name, dice := range map[string]map[content.ClassID]content.HitPointDie{
		"only fighter": {"fighter": content.HitDieD10}, "empty": {}, "nil": nil, "invalid die": {"fighter": 0, "rogue": content.HitDieD8},
	} {
		if _, err := character.HitDicePoolFromLevels(l, dice); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
	if got := recovered(t, func() { character.MustHitDicePoolFromLevels(l, nil) }); !errors.Is(got, srderr.ErrInvalidInput) {
		t.Errorf("Must panic = %v", got)
	}
	if got := recovered(t, func() {
		character.MustHitDicePoolFromLevels(l, map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10, "rogue": content.HitDieD8})
	}); got != nil {
		t.Errorf("Must panicked: %v", got)
	}
}

// Requirement: doc/specs/derived-stats.md FR-070 - the constructor drops zero
// counts, rejects negative counts and invalid dice, and enforces no relation to
// the level.
// Path: per case.
func TestNewHitDicePool(t *testing.T) {
	t.Parallel()
	p, err := character.NewHitDicePool(map[content.HitPointDie]int{content.HitDieD8: 2, content.HitDieD10: 0})
	eq, _ := character.NewHitDicePool(map[content.HitPointDie]int{content.HitDieD8: 2})
	if err != nil || !p.Equal(eq) || p.Total() != 2 || p.Count(content.HitDieD8) != 2 || p.Count(content.HitDieD10) != 0 {
		t.Errorf("zero count not dropped: %v, %v", p.Entries(), err)
	}
	for name, m := range map[string]map[content.HitPointDie]int{"negative": {content.HitDieD8: -1}, "invalid die": {0: 1}} {
		if _, err := character.NewHitDicePool(m); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
	if got, err := character.NewHitDicePool(nil); err != nil || got.Total() != 0 || !got.Equal(character.HitDicePool{}) {
		t.Errorf("nil map = %v, %v", got.Entries(), err)
	}
}

// Requirement: doc/specs/derived-stats.md FR-073 - Plus adds one die, keeps the
// order, leaves the receiver, and matches HitDicePoolFromLevels.
// Path: per case.
func TestHitDicePoolPlus(t *testing.T) {
	t.Parallel()
	mk := func(m map[content.HitPointDie]int) character.HitDicePool {
		p, err := character.NewHitDicePool(m)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name, path string
		start      character.HitDicePool
		die        content.HitPointDie
		want       []character.HitDiceCount
	}{
		{"die absent, empty pool", "unhappy (empty and minimal)", character.HitDicePool{}, content.HitDieD6, []character.HitDiceCount{pool(content.HitDieD6, 1)}},
		{"same die", "unhappy (empty and minimal)", mk(map[content.HitPointDie]int{content.HitDieD10: 5}), content.HitDieD10, []character.HitDiceCount{pool(content.HitDieD10, 6)}},
		{"different die", "unhappy (empty and minimal)", mk(map[content.HitPointDie]int{content.HitDieD8: 5}), content.HitDieD10, []character.HitDiceCount{pool(content.HitDieD8, 5), pool(content.HitDieD10, 1)}},
		{"keeps order", "unhappy (determinism)", mk(map[content.HitPointDie]int{content.HitDieD10: 2}), content.HitDieD6, []character.HitDiceCount{pool(content.HitDieD6, 1), pool(content.HitDieD10, 2)}},
		{"invalid die ignored", "unhappy (invalid input)", mk(map[content.HitPointDie]int{content.HitDieD10: 2}), 0, []character.HitDiceCount{pool(content.HitDieD10, 2)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.start.Entries()
			got := c.start.Plus(c.die)
			if !slices.Equal(got.Entries(), c.want) || !slices.Equal(c.start.Entries(), before) {
				t.Errorf("Plus = %v (receiver %v), want %v", got.Entries(), c.start.Entries(), c.want)
			}
		})
	}
	var p character.HitDicePool
	for range 5 {
		p = p.Plus(content.HitDieD10)
	}
	for range 5 {
		p = p.Plus(content.HitDieD10)
	}
	want := character.MustHitDicePoolFromLevels(character.MustClassLevels([]character.ClassLevel{cl("fighter", 5), cl("paladin", 5)}), dieMap())
	if !p.Equal(want) {
		t.Errorf("ten Plus = %v, want %v", p.Entries(), want.Entries())
	}
}
