package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// dieOf gives the Hit Point Die of the sample classes used by the aggregate
// tests (SRD 5.2.1, class pages, "Hit Point Die").
func dieOf(class string) content.HitPointDie {
	switch class {
	case "barbarian":
		return content.HitDieD12
	case "fighter", "paladin":
		return content.HitDieD10
	case "cleric", "bard", "monk":
		return content.HitDieD8
	}
	return content.HitDieD6
}

func newParams(t *testing.T, class string, scores ability.Scores) character.NewParams {
	t.Helper()
	return character.NewParams{
		ID:            character.MustIdentifier("pc-1"),
		Class:         content.ClassID(class),
		HitPointDie:   dieOf(class),
		Species:       content.SpeciesID("human"),
		Background:    content.BackgroundID("soldier"),
		AbilityScores: scores,
	}
}

// created returns a level 1 character of class with scores.
func created(t *testing.T, class string, scores ability.Scores) character.Character {
	t.Helper()
	return character.MustNew(newParams(t, class, scores)).Character()
}

// restored returns a character with the given class levels, XP, scores and
// stored maximum; the pool holds one die per level of the class's die.
func restored(t *testing.T, xp, maximum int, scores ability.Scores, entries ...character.ClassLevel) character.Character {
	t.Helper()
	dice := map[content.ClassID]content.HitPointDie{}
	for _, e := range entries {
		dice[e.Class] = dieOf(string(e.Class))
	}
	lv := levels(entries...)
	c, err := character.Restore(character.Parts{
		ID:                character.MustIdentifier("pc-1"),
		ExperiencePoints:  character.MustExperiencePoints(xp),
		ClassLevels:       lv,
		Species:           content.SpeciesID("human"),
		Background:        content.BackgroundID("soldier"),
		AbilityScores:     scores,
		HitPointMaximum:   character.MustHitPointMaximum(maximum),
		HitDicePool:       character.MustHitDicePoolFromLevels(lv, dice),
		HeroicInspiration: character.NotInspired,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func allOf(abilities ...ability.Ability) content.AbilityRequirement {
	return must(content.NewAllOf(abilities...))
}

func anyOf(abilities ...ability.Ability) content.AbilityRequirement {
	return must(content.NewAnyOf(abilities...))
}

func advancement(t *testing.T, class string, req content.AbilityRequirement) character.AdvancementClass {
	t.Helper()
	return must(character.NewAdvancementClass(content.ClassID(class), dieOf(class), req))
}

func fixedGain(n int) character.FixedHitPointGain { return must(character.NewFixedHitPointGain(n)) }

func rolledGain(n int, die content.HitPointDie) character.RolledHitPointGain {
	return must(character.NewRolledHitPointGain(n, die))
}

// levelUp runs Character.LevelUp and fails on a validation error.
func levelUp(t *testing.T, c character.Character, class character.AdvancementClass, gain character.HitPointGain, reqs map[content.ClassID]content.AbilityRequirement) character.Transition[character.LevelUpRejection] {
	t.Helper()
	tr, err := c.LevelUp(class, gain, reqs)
	if err != nil {
		t.Fatalf("LevelUp: %v", err)
	}
	return tr
}

func changedLevel(t *testing.T, tr character.Transition[character.LevelUpRejection]) character.TransitionChanged[character.LevelUpRejection] {
	t.Helper()
	c, ok := tr.(character.TransitionChanged[character.LevelUpRejection])
	if !ok {
		t.Fatalf("transition = %#v, want a change", tr)
	}
	return c
}

func rejectedLevel(t *testing.T, tr character.Transition[character.LevelUpRejection]) character.LevelUpRejection {
	t.Helper()
	r, ok := tr.(character.TransitionRejected[character.LevelUpRejection])
	if !ok {
		t.Fatalf("transition = %#v, want a rejection", tr)
	}
	return r.Rejection()
}
