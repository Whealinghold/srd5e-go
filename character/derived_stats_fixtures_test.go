package character_test

import (
	"context"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// The fixtures are synthetic: the bundled Wizard document with its id, Hit
// Point Die, saving throws and armor training replaced, so each fixture class
// carries only the numbers a case table prints.

var (
	idPattern    = regexp.MustCompile(`"wizard(_[a-z_]*)?"`)
	diePattern   = regexp.MustCompile(`"hitPointDie": "D6"`)
	savesPattern = regexp.MustCompile(`(?s)"savingThrowProficiencies": \[.*?\]`)
)

// classFixture returns a class with the given id, die, saving throws and armor
// training (the wire spellings, e.g. "Light").
func classFixture(t *testing.T, id, die string, saves, armor []string) content.ClassDefinition {
	t.Helper()
	raw, err := fs.ReadFile(data.Files(), "class_wizard.json")
	if err != nil {
		t.Fatal(err)
	}
	quote := func(ss []string) string {
		for i, s := range ss {
			ss[i] = `"` + s + `"`
		}
		return strings.Join(ss, ",")
	}
	text := idPattern.ReplaceAllString(string(raw), `"`+id+`$1"`)
	text = diePattern.ReplaceAllString(text, `"hitPointDie": "`+die+`"`)
	text = savesPattern.ReplaceAllString(text, `"savingThrowProficiencies": [`+quote(saves)+`]`)
	text = strings.Replace(text, `"armorTraining": []`, `"armorTraining": [`+quote(armor)+`]`, 1)
	set, err := content.JSONDecoder{}.Decode([]byte(text))
	if err != nil {
		t.Fatalf("fixture class %s: %v", id, err)
	}
	for _, e := range set.Entries() {
		if def, ok := e.(content.ClassDefinition); ok {
			return def
		}
	}
	t.Fatalf("fixture class %s: no class decoded", id)
	return content.ClassDefinition{}
}

func wizardClass(t *testing.T) content.ClassDefinition {
	t.Helper()
	return classFixture(t, "wizard", "D6", []string{"intelligence", "wisdom"}, nil)
}

func fighterClass(t *testing.T) content.ClassDefinition {
	t.Helper()
	return classFixture(t, "fighter", "D10", []string{"strength", "constitution"}, []string{"Light", "Medium", "Heavy", "Shield"})
}

func scoresOf(t *testing.T, str, dex, con, intl, wis, cha int) ability.Scores {
	t.Helper()
	return must(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: score2(str), ability.Dexterity: score2(dex), ability.Constitution: score2(con),
		ability.Intelligence: score2(intl), ability.Wisdom: score2(wis), ability.Charisma: score2(cha),
	}))
}

func score2(n int) ability.Score { return ability.MustScore(n) }

// flatScores gives every ability the score s.
func flatScores(t *testing.T, s int) ability.Scores {
	t.Helper()
	return scoresOf(t, s, s, s, s, s, s)
}

func levels(entries ...character.ClassLevel) character.ClassLevels {
	return character.MustClassLevels(entries)
}

func cl(id string, n int) character.ClassLevel {
	return character.ClassLevel{Class: content.ClassID(id), Level: n}
}

func leatherArmor(t *testing.T) content.Armor {
	t.Helper()
	return armorOf(t, "leather_armor", content.ArmorLight, fullRule(t, 11), 0, false, 10, gp(10))
}

func hideArmor(t *testing.T) content.Armor {
	t.Helper()
	return armorOf(t, "hide_armor", content.ArmorMedium, cappedRule(t, 12, 2), 0, false, 12, gp(10))
}

func chainMail(t *testing.T) content.Armor {
	t.Helper()
	return armorOf(t, "chain_mail", content.ArmorHeavy, noneRule(t, 16), 0, true, 55, gp(75))
}

func plateArmor(t *testing.T) content.Armor {
	t.Helper()
	return armorOf(t, "plate_armor", content.ArmorHeavy, noneRule(t, 18), 0, true, 65, gp(1500))
}

func newUseCase(t *testing.T, entries ...content.Entry) (character.CalculateDerivedStats, *contenttest.FakeSource) {
	t.Helper()
	src := contenttest.NewFakeSource(entries...)
	uc, err := character.NewCalculateDerivedStats(character.WithSource(src))
	if err != nil {
		t.Fatal(err)
	}
	return uc, src
}

func run(t *testing.T, uc character.CalculateDerivedStats, cmd character.CalculateDerivedStatsCommand) character.CalculateDerivedStatsResult {
	t.Helper()
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

func succeeded(t *testing.T, res character.CalculateDerivedStatsResult) character.DerivedStats {
	t.Helper()
	s, ok := res.(character.CalculateDerivedStatsSucceeded)
	if !ok {
		t.Fatalf("result = %#v, want CalculateDerivedStatsSucceeded", res)
	}
	return s.Stats
}

// joined returns the errors an errors.Join holds, or err alone.
func joined(err error) []error {
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		return j.Unwrap()
	}
	return []error{err}
}

// recovered runs f and returns the error it panicked with, or nil.
func recovered(t *testing.T, f func()) (got error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				t.Fatalf("panic value %v is not an error", r)
			}
			got = err
		}
	}()
	f()
	return nil
}
