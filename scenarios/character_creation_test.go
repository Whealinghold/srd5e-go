package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// layeredSource answers from top first and from base otherwise.
type layeredSource struct{ top, base character.ContentSource }

func (l layeredSource) Lookup(ctx context.Context, key content.Key) (content.LookupResult, error) {
	res, err := l.top.Lookup(ctx, key)
	if _, found := res.(content.LookupFound); err != nil || found {
		return res, err
	}
	return l.base.Lookup(ctx, key)
}

// A player rolls six ability scores and two languages, then creates a Wizard with a Sage background and a
// Human of the chosen Small size, rolling for a trinket: the bundled Wizard and species are used with a
// synthetic background and Origin feat. The Hit Point maximum uses the Constitution the background raised, the
// record keeps the choices behind the scores, the Origin feat is applied, and the events are Created and
// FeatTaken. A class the source lacks gives CreateCharacterContentMissing and rolls nothing (SRD 5.2.1,
// "Create Your Character", p. 19; "Random Generation", p. 21; "Choose Languages", p. 20; "Trinkets", p. 26).
// Cites character-creation.md FR-002, FR-013, FR-028, FR-050, FR-057, FR-072, FR-088, FR-105.
func TestCreateAWizard(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sage := content.MustBackground("sage", "Sage", content.SourceCitation{}, content.BackgroundFields{
		AbilityScores:      []ability.Ability{ability.Constitution, ability.Intelligence, ability.Wisdom},
		Feat:               content.MustBackgroundFeatReference("alert", ""),
		SkillProficiencies: []content.Skill{content.SkillArcana, content.SkillHistory},
		ToolProficiency:    must(content.NewSpecificToolProficiency("Calligrapher’s Supplies")),
		EquipmentOptions:   content.NewEquipmentOptions(must(content.NewEquipmentOption(nil, 8)), must(content.NewEquipmentOption(nil, 50))),
	})
	alert := content.MustFeat("alert", "Alert", content.SourceCitation{}, content.FeatFields{
		Category: content.FeatOrigin, Benefits: []content.FeatBenefit{content.MustFeatBenefit("Initiative Proficiency", "Text.")},
	})
	source := layeredSource{top: contenttest.NewFakeSource(sage, alert), base: data.MustBundledSource()}

	roller := dicetest.NewFixedRoller(
		6, 5, 4, 1, 5, 5, 4, 1, 4, 4, 4, 1, 6, 6, 1, 1, 3, 3, 3, 3, 2, 2, 2, 2, // six 4d6: 15, 14, 12, 13, 9, 6
		3, 5, // two 1d12: Dwarvish, Elvish
		42, // the trinket
	)
	scores, err := must(character.NewRollAbilityScores(roller)).Run(ctx, character.RollAbilityScoresCommand{})
	if err != nil {
		t.Fatal(err)
	}
	totals := scores.(character.RollAbilityScoresSucceeded).Scores().Totals()
	if len(totals) != 6 || totals[0] != 15 || totals[5] != 6 {
		t.Fatalf("totals = %v", totals)
	}
	languages, err := must(character.NewRollLanguages(roller)).Run(ctx, character.RollLanguagesCommand{})
	rolled, ok := languages.(character.RollLanguagesSucceeded)
	if err != nil || !ok {
		t.Fatalf("languages = %v, %v", languages, err)
	}

	assign := character.MustAbilityScoreAssignment(character.RandomGeneration, map[ability.Ability]int{
		ability.Strength: totals[4], ability.Dexterity: totals[3], ability.Constitution: totals[1],
		ability.Intelligence: totals[0], ability.Wisdom: totals[2], ability.Charisma: totals[5]})
	cmd := character.CreateCharacterCommand{
		CharacterID: character.MustIdentifier("pc-1"), ClassID: "wizard", BackgroundID: "sage", SpeciesID: "human",
		AbilityScores: assign, BackgroundIncrease: character.MustPlusTwoPlusOne(ability.Constitution, ability.Intelligence),
		ClassSkills: []content.Skill{content.SkillInsight, content.SkillInvestigation}, ClassEquipment: character.OptionA, BackgroundEquipment: character.OptionB,
		Size: character.ChosenSize{Size: content.SizeSmall}, Languages: rolled.Languages(), Alignment: content.AlignmentNeutralGood,
		Trinket: character.RollForTrinket{},
	}
	creator := must(character.NewCreateCharacter(character.WithSource(source), character.WithRoller(roller)))
	res, err := creator.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	created, ok := res.(character.CreateCharacterSucceeded)
	if !ok {
		t.Fatalf("result = %#v", res)
	}
	pc, record := created.Character(), created.Record()
	// Constitution 14 + 2 = 16 (+3): the d6 maximum plus 3.
	if pc.HitPointMaximum().Value() != 9 || pc.AbilityScores().Score(ability.Constitution).Value() != 16 || pc.AbilityScores().Score(ability.Intelligence).Value() != 16 {
		t.Errorf("character = %+v", pc)
	}
	if roll, has := record.Trinket(); !has || roll.Value() != 42 || record.Method() != character.RandomGeneration || record.AssignedScores().Score(ability.Constitution).Value() != 14 {
		t.Errorf("record = %+v", record)
	}
	if !record.Feats().Holds("alert") || record.Size() != content.SizeSmall || record.Equipment().Gold() != content.MustCost(5+50, content.CoinGP, 0) || pc.ExperiencePoints().Value() != 0 {
		t.Errorf("record = %+v", record)
	}
	if events := created.Events(); len(events) != 2 {
		t.Errorf("events = %v", events)
	}

	cmd.ClassID = "artificer"
	res, err = creator.Run(ctx, cmd)
	if missing, ok := res.(character.CreateCharacterContentMissing); err != nil || !ok || len(missing.Missing()) != 1 {
		t.Errorf("unknown class = %#v, %v", res, err)
	}
	if left, _ := roller.Remaining(); left != 0 {
		t.Errorf("%d dice left over", left)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err) // a broken fixture, not a failure of the code under test
	}
	return v
}
