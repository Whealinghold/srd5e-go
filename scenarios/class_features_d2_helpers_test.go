package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpD2Die is the Hit Point Die of the classes the group D2 scenarios build.
func grpD2Die(class string) content.HitPointDie {
	switch class {
	case "barbarian":
		return content.HitDieD12
	case "fighter", "paladin":
		return content.HitDieD10
	case "cleric", "bard", "monk", "warlock", "druid", "rogue", "ranger":
		return content.HitDieD8
	}
	return content.HitDieD6
}

// grpD2Hero is a character with the given class levels, Experience Points for level 20, the given scores and a stored maximum of 200.
func grpD2Hero(t *testing.T, scores ability.Scores, entries ...character.ClassLevel) character.Character {
	t.Helper()
	dice := map[content.ClassID]content.HitPointDie{}
	for _, e := range entries {
		dice[e.Class] = grpD2Die(string(e.Class))
	}
	lv := character.MustClassLevels(entries)
	return must2(character.Restore(character.Parts{
		ID: character.MustIdentifier("pc-d2"), ExperiencePoints: character.MustExperiencePoints(355000), ClassLevels: lv, Species: "human", Background: "soldier",
		AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(200), HitDicePool: character.MustHitDicePoolFromLevels(lv, dice),
		HeroicInspiration: character.NotInspired,
	}))
}

// grpD2Flat is a character whose six scores are all s.
func grpD2Flat(t *testing.T, s int, entries ...character.ClassLevel) character.Character {
	t.Helper()
	return grpD2Hero(t, scoresFor(t, s, s, s, s, s, s), entries...)
}

func grpD2CL(class string, level int) character.ClassLevel {
	return character.ClassLevel{Class: content.ClassID(class), Level: level}
}

// grpD2Class returns the bundled definition of the class.
func grpD2Class(t *testing.T, id string) content.ClassDefinition {
	t.Helper()
	src := must2(data.NewBundledSource())
	return must2(src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID(id)))).(content.LookupFound).Entry.(content.ClassDefinition)
}

// grpD2Check is a rolled check of kind with the dice, the mode they were rolled under, a total modifier and the Difficulty Class.
func grpD2Check(kind d20.Kind, mode modifier.RollMode, dice []int, mods, dc int) d20.CheckResult {
	roll := must2(d20.NewRoll(mode, dice))
	return must2(d20.ResolveCheck(kind, roll, d20.NewRollModifiers(modifier.New(mods)), d20.DifficultyClass(dc)))
}

// grpD2Rest is the RestApplied of a finished rest of the kind for the character id.
func grpD2Rest(t *testing.T, id character.Identifier, kind character.RestKind) character.RestApplied {
	t.Helper()
	end := character.NewRestRanItsCourse(1, 0)
	if kind == character.LongRest {
		end = character.NewRestRanItsCourse(8, 0)
	}
	cmd := must2(character.NewTakeRestCommand(id, must2(character.NewRestReport(kind, character.StandardLongRest, end)), 12, 0, nil, character.SpentHitDice{}))
	return character.NewRestTaker().TakeRest(context.Background(), cmd).(character.TakeRestApplied).Applied()
}

// grpD2WantInvalid fails the test unless err matches srderr.ErrInvalidInput.
func grpD2WantInvalid(t *testing.T, err error, what string) {
	t.Helper()
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("%s: error = %v, want srderr.ErrInvalidInput", what, err)
	}
}

// grpD2WantPortFailure fails the test unless err is a *srderr.PortError of the port wrapping cause.
func grpD2WantPortFailure(t *testing.T, err error, port string, cause error) {
	t.Helper()
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != port || (cause != nil && !errors.Is(err, cause)) {
		t.Errorf("error = %v, want a *srderr.PortError of port %q wrapping %v", err, port, cause)
	}
}
