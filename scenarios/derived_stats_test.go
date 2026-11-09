package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A level 5 Wizard over the bundled slice: Proficiency Bonus +3 from the total
// level, the stored maximum, Armor Class 10 plus Dexterity, and the Intelligence
// saving throw handed to a D20 Test through its breakdown (SRD 5.2.1, "Fill In
// Numbers", p. 22; "Level Advancement", p. 23; "Saving Throws", p. 7).
func TestLevelFiveWizardDerivedStats(t *testing.T) {
	t.Parallel()
	scores, err := ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(8), ability.Dexterity: ability.MustScore(14), ability.Constitution: ability.MustScore(12),
		ability.Intelligence: ability.MustScore(16), ability.Wisdom: ability.MustScore(15), ability.Charisma: ability.MustScore(10),
	})
	if err != nil {
		t.Fatal(err)
	}
	levels := character.MustClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 5}})
	uc, err := character.NewCalculateDerivedStats(character.WithSource(data.MustBundledSource()))
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), character.CalculateDerivedStatsCommand{
		ClassLevels: levels, AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(27),
	})
	if err != nil {
		t.Fatal(err)
	}
	ok, isOK := res.(character.CalculateDerivedStatsSucceeded)
	if !isOK {
		t.Fatalf("result = %#v", res)
	}
	stats := ok.Stats
	if stats.ProficiencyBonus().Value() != 3 || stats.HitPointMaximum().Value() != 27 || stats.ArmorClass() != 12 {
		t.Errorf("proficiency %d, maximum %d, AC %d; want 3, 27, 12", stats.ProficiencyBonus().Value(), stats.HitPointMaximum().Value(), stats.ArmorClass())
	}
	if want := levels.TotalLevel(); stats.TotalLevel() != want {
		t.Errorf("total level %d", stats.TotalLevel().Value())
	}

	save := stats.SavingThrow(ability.Intelligence)
	rm, err := save.Breakdown().ToRollModifiers()
	if err != nil {
		t.Fatal(err)
	}
	roll, err := d20.NewRoll(modifier.Normal, []int{10})
	if err != nil {
		t.Fatal(err)
	}
	check, err := d20.ResolveCheck(d20.SavingThrow, roll, rm, d20.DifficultyClass(16))
	if err != nil {
		t.Fatal(err)
	}
	if save.Value() != 6 || check.Outcome() != d20.Success {
		t.Errorf("Intelligence save %d, outcome %v; want +6 and a success against DC 16", save.Value(), check.Outcome())
	}
}
