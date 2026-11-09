package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// fgtDef is the bundled Fighter ClassDefinition (content/data/class_fighter.json).
func fgtDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		tb.Fatal(err)
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("fighter")))
	if err != nil {
		tb.Fatal(err)
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

func fgtProgression(tb testing.TB) character.FighterProgression {
	tb.Helper()
	return must(character.NewFighterProgression(fgtDef(tb)))
}

// fgtStats are the stats of the Fighter level n.
func fgtStats(tb testing.TB, n int) character.FighterLevelStats {
	tb.Helper()
	return fgtProgression(tb).StatsAt(character.MustFighterLevel(n))
}

// fgtChar is a character with the given class levels, enough Experience Points for level 20, flat scores of 10 and a stored maximum of 200.
func fgtChar(t *testing.T, entries ...character.ClassLevel) character.Character {
	t.Helper()
	return restored(t, 355000, 200, flatScores(t, 10), entries...)
}

// fgtFighter is a Fighter of level n and nothing else.
func fgtFighter(t *testing.T, n int) character.Character {
	t.Helper()
	return fgtChar(t, cl("fighter", n))
}

// fgtWith returns c holding the FighterState state.
func fgtWith(t *testing.T, c character.Character, state character.FighterState) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.FighterStateKey(), state))
}

// fgtStateOf reads the FighterState c stores.
func fgtStateOf(t *testing.T, c character.Character) character.FighterState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.FighterStateKey())
	if err != nil || !ok {
		t.Fatalf("FighterState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// fgtCheck is a rolled check of kind with the dice, the roll mode they were rolled under, a total modifier of mods
// and the Difficulty Class dc.
func fgtCheck(tb testing.TB, kind d20.Kind, mode modifier.RollMode, dice []int, mods, dc int) d20.CheckResult {
	tb.Helper()
	roll := must(d20.NewRoll(mode, dice))
	return must(d20.ResolveCheck(kind, roll, d20.NewRollModifiers(modifier.New(mods)), d20.DifficultyClass(dc)))
}

// fgtState builds a FighterState from expended counts.
func fgtState(secondWind, actionSurge, indomitable int) character.FighterState {
	return character.MustFighterState(secondWind, actionSurge, indomitable, "", "")
}

// fgtStyle is a Fighting Style feat double (the SRD Fighting Style feats carry the Fighting Style Feature prerequisite).
func fgtStyle(id string) content.Feat {
	return featOf(id, content.FeatFightingStyle, false, featureReq("Fighting Style Feature"))
}

// fgtRepeatableStyle is a Repeatable Fighting Style feat double (no bundled one is Repeatable).
func fgtRepeatableStyle(id string) content.Feat {
	return featOf(id, content.FeatFightingStyle, true, featureReq("Fighting Style Feature"))
}

var fgtStyleSource = character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}

// fgtFeats holds the named feats as taken Fighting Style feats (category cat).
func fgtFeats(t *testing.T, cat content.FeatCategory, ids ...string) character.TakenFeats {
	t.Helper()
	var taken []character.TakenFeat
	for _, id := range ids {
		taken = append(taken, must(character.NewTakenFeat(content.FeatID(id), cat, fgtStyleSource, character.NoFeatChoices{})))
	}
	return must(character.NewTakenFeats(taken...))
}

// fgtRest is the RestApplied of a finished rest of kind for the character id.
func fgtRest(t *testing.T, id character.Identifier, kind character.RestKind) character.RestApplied {
	t.Helper()
	end := ran(1, 0)
	if kind == character.LongRest {
		end = ran(8, 0)
	}
	cmd := must(character.NewTakeRestCommand(id, must(character.NewRestReport(kind, character.StandardLongRest, end)), 12, 0, nil, character.SpentHitDice{}))
	return character.NewRestTaker().TakeRest(t.Context(), cmd).(character.TakeRestApplied).Applied()
}

// fgtLevelUp gains one level in class and returns the new character and the LevelGained event.
func fgtLevelUp(t *testing.T, c character.Character, class string) (character.Character, character.LevelGained) {
	t.Helper()
	reqs := map[content.ClassID]content.AbilityRequirement{"fighter": anyOf(ability.Strength, ability.Dexterity), "wizard": allOf(ability.Intelligence)}
	ch := changedLevel(t, levelUp(t, c, advancement(t, class, reqs[content.ClassID(class)]), fixedGain(map[string]int{"fighter": 6, "wizard": 4}[class]), reqs))
	for _, e := range ch.Events() {
		if lg, ok := e.(character.LevelGained); ok {
			return ch.Character(), lg
		}
	}
	t.Fatal("no LevelGained event")
	return character.Character{}, character.LevelGained{}
}

// fgtWeapons are the weapons the Weapon Mastery tests resolve.
func fgtWeapons(t *testing.T, names ...string) map[content.ItemID]content.Weapon {
	t.Helper()
	return resolvedOf(weaponFixtures(t), names...)
}

// fgtTelemetry is a recording bundle and the options that inject it with a clock stepping one second per read.
func fgtTelemetry() (*telemetrytest.Recorder, []character.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), time.Second)
	return rec, []character.Option{character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)}
}

// errBoom is the failure the scripted collaborators return.
var errBoom = errors.New("boom")
