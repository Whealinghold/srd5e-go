package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func partsOf(t *testing.T) character.Parts {
	t.Helper()
	lv := levels(cl("fighter", 2))
	return character.Parts{
		ID: character.MustIdentifier("pc-1"), ExperiencePoints: character.MustExperiencePoints(300), ClassLevels: lv,
		Species: "human", Background: "soldier", AbilityScores: flatScores(t, 12), HitPointMaximum: character.MustHitPointMaximum(20),
		HitDicePool:       character.MustHitDicePoolFromLevels(lv, map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10}),
		HeroicInspiration: character.NotInspired,
	}
}

// Requirement: doc/specs/character-aggregate.md FR-051, FR-052, FR-053, FR-084 -
// Restore rejects a zero id, no class, an invalid species, background, scores or
// Heroic Inspiration, a pool of another size than the total level and a maximum
// below the total level; a valid set round-trips.
// Path: per case.
func TestRestore(t *testing.T) {
	t.Parallel()
	good, err := character.Restore(partsOf(t))
	if err != nil || good.TotalLevel().Value() != 2 || good.IsZero() {
		t.Fatalf("valid parts: %v", err)
	}
	for _, c := range []struct {
		name   string
		mutate func(*character.Parts)
	}{
		{"zero id", func(p *character.Parts) { p.ID = character.Identifier{} }},
		{"no class", func(p *character.Parts) { p.ClassLevels = character.ClassLevels{} }},
		{"species", func(p *character.Parts) { p.Species = "" }},
		{"background", func(p *character.Parts) { p.Background = "" }},
		{"scores", func(p *character.Parts) { p.AbilityScores = ability.Scores{} }},
		{"inspiration", func(p *character.Parts) { p.HeroicInspiration = 0 }},
		{"pool too small", func(p *character.Parts) { p.HitDicePool = character.HitDicePool{} }},
		{"pool too large", func(p *character.Parts) { p.HitDicePool = p.HitDicePool.Plus(content.HitDieD10) }},
		{"maximum below the total level", func(p *character.Parts) { p.HitPointMaximum = character.MustHitPointMaximum(1) }},
		{"zero maximum", func(p *character.Parts) { p.HitPointMaximum = character.HitPointMaximum{} }},
	} {
		p := partsOf(t)
		c.mutate(&p)
		if _, err := character.Restore(p); !isInvalid(err) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	p := partsOf(t)
	p.HitPointMaximum = character.MustHitPointMaximum(2) // exactly the total level is allowed
	if _, err := character.Restore(p); err != nil {
		t.Errorf("maximum equal to the total level: %v", err)
	}
	if !(character.Character{}).IsZero() {
		t.Error("zero Character must report IsZero")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-010 - the Proficiency Bonus
// comes from the total level (SRD 5.2.1, "Multiclassing", p. 25).
// Path: per case.
func TestCharacterProficiencyBonus(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		entries []character.ClassLevel
		want    int
	}{
		{"Fighter 3 Rogue 2", []character.ClassLevel{cl("fighter", 3), cl("rogue", 2)}, 3},
		{"Cleric 4 Fighter 4", []character.ClassLevel{cl("cleric", 4), cl("fighter", 4)}, 3},
		{"Wizard 1", []character.ClassLevel{cl("wizard", 1)}, 2},
		{"single class 5", []character.ClassLevel{cl("wizard", 5)}, 3},
	} {
		if got := restored(t, 0, 40, flatScores(t, 10), c.entries...).ProficiencyBonus().Value(); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-024 - IsEligibleForNextLevel is
// true exactly below level 20 with XP at least the next level's minimum.
// Path: per case.
func TestIsEligibleForNextLevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level, xp int
		want      bool
	}{
		{1, 299, false}, {1, 300, true}, {1, 6500, true}, {19, 354999, false}, {19, 355000, true}, {20, 400000, false},
	} {
		got := restored(t, c.xp, 400, flatScores(t, 10), cl("fighter", c.level)).IsEligibleForNextLevel()
		if got != c.want {
			t.Errorf("level %d xp %d: %v", c.level, c.xp, got)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-055, FR-056 - characters
// compare by value: class order, feature states, Heroic Inspiration and every
// scalar count; the exposed collections are copies.
// Path: per case.
func TestCharacterEqual(t *testing.T) {
	t.Parallel()
	base := partsOf(t)
	mk := func(f func(*character.Parts)) character.Character {
		p := base
		if f != nil {
			f(&p)
		}
		return must(character.Restore(p))
	}
	rage := character.MustFeatureStateKey[rageState]("rage")
	want := mk(nil)
	if !want.Equal(mk(nil)) {
		t.Fatal("equal parts must give equal characters")
	}
	for name, f := range map[string]func(*character.Parts){
		"id":         func(p *character.Parts) { p.ID = character.MustIdentifier("pc-2") },
		"xp":         func(p *character.Parts) { p.ExperiencePoints = character.MustExperiencePoints(301) },
		"classes":    func(p *character.Parts) { p.ClassLevels = levels(cl("rogue", 2)) },
		"species":    func(p *character.Parts) { p.Species = "elf" },
		"background": func(p *character.Parts) { p.Background = "sage" },
		"scores":     func(p *character.Parts) { p.AbilityScores = flatScores(t, 13) },
		"maximum":    func(p *character.Parts) { p.HitPointMaximum = character.MustHitPointMaximum(21) },
		"pool": func(p *character.Parts) {
			p.HitDicePool = character.MustHitDicePoolFromLevels(levels(cl("fighter", 2)), map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD12})
		},
		"inspiration": func(p *character.Parts) { p.HeroicInspiration = character.Inspired },
		"states": func(p *character.Parts) {
			p.FeatureStates = must(character.SetFeatureState(character.FeatureStates{}, rage, rageState{1}))
		},
	} {
		if want.Equal(mk(f)) {
			t.Errorf("a different %s must be unequal", name)
		}
	}
	two := mk(func(p *character.Parts) {
		p.ClassLevels = levels(cl("fighter", 1), cl("rogue", 1))
		p.HitDicePool = character.MustHitDicePoolFromLevels(p.ClassLevels, map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10, "rogue": content.HitDieD8})
	})
	swapped := mk(func(p *character.Parts) {
		p.ClassLevels = levels(cl("rogue", 1), cl("fighter", 1))
		p.HitDicePool = two.HitDicePool()
	})
	if two.Equal(swapped) || two.InitialClass() != "fighter" || swapped.InitialClass() != "rogue" {
		t.Error("a different initial class is unequal")
	}
	entries := want.ClassLevels().Entries()
	entries[0].Level = 9
	pool := want.HitDicePool().Entries()
	pool[0].Count = 9
	if want.ClassLevels().LevelIn("fighter") != 2 || want.HitDicePool().Count(content.HitDieD10) != 2 {
		t.Error("exposed collections must be copies")
	}
}
