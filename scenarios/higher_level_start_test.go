package scenarios_test

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A GM starts a Wizard at level 5 (SRD 5.2.1, "Starting at Higher Levels", p. 24): the plan rolls
// the level 2 Hit Die, takes the fixed value after that, chooses the Evoker at level 3, takes an
// Ability Score Improvement for +2 Constitution at level 4 (reported as a consumed grant, not a
// feat taken) and stops at level 5 with the minimum Experience Points. The same level's extra gold
// is rolled through the use case and through TotalGold with a roll made by the host, and one Trinket
// is rolled on a table the host built from its own copy of the SRD.
// Cites higher-level-start-equipment-and-trinkets.md FR-065, FR-068, FR-085, FR-094, FR-118.
func TestStartAtHigherLevelWizardWithMoneyAndTrinket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wizard, _ := classesFromWizard(t)
	src := contenttest.NewFakeSource(wizard, humanSpecies(t), soldierBackground(t))

	start, err := character.NewStartAtHigherLevel(character.WithSource(src), character.WithRoller(dicetest.NewFixedRoller(6)))
	if err != nil {
		t.Fatal(err)
	}
	fixed := character.TakeFixedValue{}
	asi, err := character.NewAbilityScoreImprovementOneByTwo(ability.Constitution)
	if err != nil {
		t.Fatal(err)
	}
	res, err := start.Start(ctx, character.StartAtHigherLevelCommand{
		CharacterID: character.MustIdentifier("pc-5"), ClassID: wizard.ID(), SpeciesID: "human", BackgroundID: "soldier",
		AbilityScores: scoresFor(t, 8, 10, 14, 15, 12, 10), StartingLevel: character.MustLevel(5),
		Steps: []character.StartingLevelStep{
			{ClassID: wizard.ID(), HitPointChoice: character.RollHitDie{}},
			{ClassID: wizard.ID(), HitPointChoice: fixed, SubclassID: "evoker"},
			{ClassID: wizard.ID(), HitPointChoice: fixed, AbilityScoreIncrease: asi},
			{ClassID: wizard.ID(), HitPointChoice: fixed},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	started, ok := res.(character.StartAtHigherLevelStarted)
	if !ok {
		t.Fatalf("result = %#v, want a started character", res)
	}
	pc := started.Character()
	// 8 (d6 and +2) + 8 (a roll of 6, +2) + 6 (fixed 4, +2) + 6 (fixed 4, +2), then Constitution 16 adds 1 for each
	// of 4 levels, then 7 (fixed 4, +3).
	if pc.TotalLevel().Value() != 5 || pc.ExperiencePoints().Value() != 6500 || pc.HitPointMaximum().Value() != 8+8+6+6+4+7 ||
		pc.AbilityScores().Score(ability.Constitution).Value() != 16 {
		t.Errorf("level %d, XP %d, Hit Points %d, Constitution %d", pc.TotalLevel().Value(), pc.ExperiencePoints().Value(), pc.HitPointMaximum().Value(),
			pc.AbilityScores().Score(ability.Constitution).Value())
	}
	grants := started.ConsumedGrants()
	if len(grants) != 1 || grants[0].Class != wizard.ID() || grants[0].ClassLevel != 4 {
		t.Errorf("ConsumedGrants = %v, want the level 4 grant", grants)
	}
	if _, isCreated := started.Events()[0].(character.Created); !isCreated {
		t.Errorf("first event = %T, want Created", started.Events()[0])
	}

	// The extra gold of level 5: 500 GP plus 1d10 x 25 GP, through the use case and through TotalGold.
	money, err := character.NewRollHigherLevelStartingMoney(dicetest.NewFixedRoller(4))
	if err != nil {
		t.Fatal(err)
	}
	moneyRes, err := money.Roll(ctx, character.RollHigherLevelStartingMoneyCommand{StartingLevel: character.MustLevel(5)})
	if err != nil {
		t.Fatal(err)
	}
	rolled, ok := moneyRes.(character.RollHigherLevelStartingMoneyRolled)
	if !ok {
		t.Fatalf("money = %#v", moneyRes)
	}
	roll := 4
	byHand, err := character.HigherLevelEquipmentRules{}.ForStartingLevel(character.MustLevel(5)).(character.HigherLevelEquipmentAllowance).TotalGold(&roll)
	if err != nil || rolled.Total().Amount() != 600 || byHand != 600 {
		t.Errorf("money total = %v, by hand %d (%v), want 600 GP", rolled.Total(), byHand, err)
	}

	// One Trinket from a table built from the host's own copy of the SRD.
	f, err := os.Open("../character/testdata/srd/trinkets.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var entries []character.Trinket
	var texts []string
	for sc := bufio.NewScanner(f); sc.Scan(); {
		if strings.HasPrefix(sc.Text(), "#") {
			continue
		}
		_, text, _ := strings.Cut(sc.Text(), " ")
		trinket, err := character.NewTrinket(text)
		if err != nil {
			t.Fatal(err)
		}
		entries, texts = append(entries, trinket), append(texts, text)
	}
	table, err := character.NewTrinketTable(entries)
	if err != nil {
		t.Fatal(err)
	}
	roller, err := character.NewRollTrinket(dicetest.NewFixedRoller(35))
	if err != nil {
		t.Fatal(err)
	}
	trinketRes, err := roller.Roll(ctx, character.RollTrinketCommand{Table: table})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := trinketRes.(character.RollTrinketRolled); !ok || got.Roll() != 35 || got.Trinket().Description() != texts[34] {
		t.Errorf("trinket = %#v, want entry 35", trinketRes)
	}
}

func humanSpecies(t *testing.T) content.Entry {
	t.Helper()
	raw, err := fs.ReadFile(data.Files(), "species.json")
	if err != nil {
		t.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range set.Entries() {
		if s, ok := e.(content.Species); ok && s.ID() == "human" {
			return s
		}
	}
	t.Fatal("no human species in the bundle")
	return nil
}

func soldierBackground(t *testing.T) content.Entry {
	t.Helper()
	tool, err := content.NewSpecificToolProficiency("Tool")
	if err != nil {
		t.Fatal(err)
	}
	b, err := content.NewBackground("soldier", "Soldier", content.SourceCitation{}, content.BackgroundFields{
		AbilityScores:      []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution},
		Feat:               content.MustBackgroundFeatReference("feat_a", ""),
		SkillProficiencies: []content.Skill{content.SkillAthletics, content.SkillStealth},
		ToolProficiency:    tool,
		EquipmentOptions:   content.NewEquipmentOptions(content.EquipmentOption{}, content.EquipmentOption{}),
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
