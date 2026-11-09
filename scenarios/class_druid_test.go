package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A Druid gains levels 1 to 6 with the Circle of the Land: the Primal Order at level 1, Wild Shape with two uses and four known forms at level 2, the land type after a
// Long Rest at level 3 (temperate), a Wild Shape form for 1 hour 30 minutes and 3 Temporary Hit Points, and at level 6 Natural Recovery: a Short Rest opens the recovery,
// a level 3 slot is recovered once, a second Short Rest does not allow another, and a Long Rest restores the uses and clears the limit (SRD 5.2.1, "Druid Features", p. 41;
// "Level 2: Wild Shape", p. 42; "Level 3: Circle of the Land Spells", "Level 6: Natural Recovery", p. 46). Cites class-druid.md FR-025 to FR-027, FR-030, FR-060, FR-065, FR-070, FR-118,
// FR-119, FR-127, FR-128, FR-153, FR-156, FR-158.
func TestDruidLevelsOneToSix(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	res, err := data.MustBundledSource().Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("druid")))
	if err != nil {
		t.Fatal(err)
	}
	tables := must(character.DruidTablesFromClass(res.(content.LookupFound).Entry.(content.ClassDefinition)))
	id := character.MustIdentifier("druid")
	subclass := character.CircleOfTheLandSubclassID
	var state character.DruidFeatureState
	for level := 1; level <= 6; level++ {
		gain := must(character.GainDruidLevel(character.GainDruidLevelInput{ID: id, Previous: level - 1, Target: level, State: state, Tables: tables, Subclass: subclass}))
		applied, ok := gain.(character.GainDruidLevelApplied)
		if !ok {
			t.Fatalf("level %d: %#v", level, gain)
		}
		state = applied.State()
		facts := must(character.NewDruidCharacterFacts(id, level, subclass, character.ProficiencyBonusFor(character.MustLevel(level))))
		choose := func(c character.DruidChoice) {
			t.Helper()
			r := must(character.ApplyDruidChoice(character.ApplyDruidChoiceInput{State: state, Tables: tables, Facts: facts, Choice: c}))
			a, ok := r.(character.ApplyDruidChoiceApplied)
			if !ok {
				t.Fatalf("level %d: choice %#v = %#v", level, c, r)
			}
			state = a.State()
		}
		trigger := func(tr character.DruidTrigger) []character.Event {
			t.Helper()
			r := must(character.ApplyDruidTrigger(character.ApplyDruidTriggerInput{State: state, Tables: tables, Facts: facts, Trigger: tr}))
			state = r.State()
			return r.Events()
		}
		switch level {
		case 1:
			if p := applied.PendingChoices(); len(p) != 1 || p[0].Kind != character.DruidPendingPrimalOrder {
				t.Errorf("level 1 pending = %v", p)
			}
			choose(character.DruidChoicePrimalOrder{Order: character.PrimalOrderMagician})
		case 2:
			if p := applied.PendingChoices(); len(p) != 1 || p[0].Kind != character.DruidPendingKnownForms || p[0].Count != 4 || state.WildShapeUsesRemaining() != 2 {
				t.Errorf("level 2 pending = %v, uses %d", p, state.WildShapeUsesRemaining())
			}
			var forms []character.BeastFormFacts
			for _, name := range []string{"rat", "riding_horse", "spider", "wolf"} {
				forms = append(forms, must(character.NewBeastFormFacts(content.MonsterID(name), content.CreatureTypeBeast, content.MustChallengeRating("1/4"), false)))
			}
			choose(character.DruidChoiceKnownForms{Forms: forms})
		case 3:
			trigger(character.DruidTriggerRestFinished{Kind: character.LongRest}) // opens the land type window
			choose(character.DruidChoiceLandType{Land: character.LandTemperate})
			spells := character.DruidRules{}.CircleSpells(tables, state.LandType(), character.MustDruidLevel(level))
			if !slices.Equal(spells, []content.SpellID{"misty_step", "shocking_grasp", "sleep"}) {
				t.Errorf("Circle Spells = %v", spells)
			}
			use := must(character.UseDruidFeature(character.UseDruidFeatureInput{State: state, Tables: tables, Facts: facts, Use: character.DruidUseAssumeWildShape{Form: "wolf",
				Items: []character.DruidItemPlan{{Key: "pack", Disposition: character.EquipmentDispositionMergesIntoForm}}}}))
			used, ok := use.(character.UseDruidFeatureUsed)
			if !ok {
				t.Fatalf("Wild Shape = %#v", use)
			}
			ev := used.Events()[0].(character.WildShapeAssumed)
			if ev.Duration().Minutes() != 90 || ev.TemporaryHitPoints() != 3 || used.State().WildShapeUsesRemaining() != 1 {
				t.Errorf("Wild Shape = %v, %d Temporary Hit Points, %d uses", ev.Duration(), ev.TemporaryHitPoints(), used.State().WildShapeUsesRemaining())
			}
			state = used.State()
		case 6:
			slots := must(character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: id, Levels: must(character.NewClassLevels([]character.ClassLevel{{Class: "druid", Level: 6}})), Spells: bundledSpells(t)})).(character.CreateSpellcastingCreated).State()
			slots = slots.ExpendSlot(character.PoolSpellcasting, character.MustSlotLevel(3)).(character.SlotExpenditureExpended).State()
			trigger(character.DruidTriggerRestFinished{Kind: character.ShortRest})
			recover := character.DruidUseNaturalRecoveryRecover{Pool: character.PoolSpellcasting, Levels: []int{3}}
			run := func() character.UseDruidFeatureResult {
				return must(character.UseDruidFeature(character.UseDruidFeatureInput{State: state, Tables: tables, Facts: facts, Spellcasting: slots, Use: recover}))
			}
			first, ok := run().(character.UseDruidFeatureUsed)
			if !ok || len(first.SlotChanges()) != 1 {
				t.Fatalf("recovery = %#v", first)
			}
			regained := slots.RegainSlot(character.PoolSpellcasting, character.MustSlotLevel(3), 1).(character.SlotRecoveryRegained).State()
			if regained.Slots(character.PoolSpellcasting).Expended(character.MustSlotLevel(3)) != 0 {
				t.Error("the host did not apply the slot change")
			}
			state = first.State()
			trigger(character.DruidTriggerRestFinished{Kind: character.ShortRest})
			if rej, ok := run().(character.UseDruidFeatureRejected); !ok || rej.Reason != character.DruidAlreadyUsedSinceLongRest {
				t.Errorf("a second recovery after a Short Rest = %#v", rej)
			}
			trigger(character.DruidTriggerRestFinished{Kind: character.LongRest})
			if state.NaturalRecoveryRecovery() || state.WildShapeUsesRemaining() != 3 {
				t.Errorf("after a Long Rest: %+v", state)
			}
		}
	}
}

// A player creates a Druid over the bundled Druid definition with the Standard Array and DruidCreationChoices, which seed a level 1 DruidFeatureState holding the Primal Order: two skills of
// the eight offered and Option A are accepted (Hit Point maximum 8 plus the Constitution modifier); one skill, Acrobatics, a repeated skill and an equipment option C the Druid does not have are
// rejected with their problem kinds, and Primal Order Warden is chosen with the creation choices, not with the skills (SRD 5.2.1, "Core Druid Traits", p. 41; "Level 1: Primal Order", p. 42). Cites
// class-druid.md FR-004 to FR-007, FR-011, FR-012, FR-163, FR-170.
func TestCreateADruid(t *testing.T) {
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
	source := layeredSource{top: contenttest.NewFakeSource(sage, alert, monkTool("calligraphers_supplies", "Calligrapher’s Supplies", content.ToolGroupArtisans)), base: data.MustBundledSource()}
	creator := must(character.NewCreateCharacter(character.WithSource(source), character.WithRoller(dicetest.NewFixedRoller())))
	choices := must(character.NewDruidCreationChoices(character.PrimalOrderWarden))
	cmd := character.CreateCharacterCommand{
		CharacterID: character.MustIdentifier("pc-druid"), ClassID: "druid", BackgroundID: "sage", SpeciesID: "human",
		AbilityScores: character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
			ability.Strength: 8, ability.Dexterity: 13, ability.Constitution: 14, ability.Intelligence: 10, ability.Wisdom: 15, ability.Charisma: 12}),
		BackgroundIncrease: character.MustPlusTwoPlusOne(ability.Wisdom, ability.Constitution),
		ClassSkills:        []content.Skill{content.SkillNature, content.SkillSurvival},
		ClassEquipment:     character.OptionA, BackgroundEquipment: character.OptionB,
		Size: character.ChosenSize{Size: content.SizeMedium}, Languages: []content.Language{content.LanguageDwarvish, content.LanguageElvish}, Alignment: content.AlignmentNeutralGood,
		ClassChoices: choices,
	}
	res, err := creator.Run(ctx, cmd)
	created, ok := res.(character.CreateCharacterSucceeded)
	if err != nil || !ok {
		t.Fatalf("result = %#v, %v", res, err)
	}
	pc := created.Character()
	state, stored, err := character.ReadFeatureState(pc.FeatureStates(), character.DruidFeatureStateKey())
	if err != nil || !stored || state.PrimalOrder() != character.PrimalOrderWarden || state.WildShapeUsesRemaining() != 0 || pc.ClassLevels().LevelIn("druid") != 1 {
		t.Errorf("state = %+v, %v, %v; levels %v", state, stored, err, pc.ClassLevels())
	}
	for _, c := range []struct {
		name   string
		mutate func(*character.CreateCharacterCommand)
		want   character.CreateCharacterProblemKind
	}{
		{"one skill", func(c *character.CreateCharacterCommand) { c.ClassSkills = []content.Skill{content.SkillNature} }, character.ClassSkillCountWrong},
		{"Acrobatics", func(c *character.CreateCharacterCommand) {
			c.ClassSkills = []content.Skill{content.SkillNature, content.SkillAcrobatics}
		}, character.ClassSkillNotOffered},
		{"a repeated skill", func(c *character.CreateCharacterCommand) {
			c.ClassSkills = []content.Skill{content.SkillNature, content.SkillNature}
		}, character.ClassSkillRepeated},
		{"option C", func(c *character.CreateCharacterCommand) { c.ClassEquipment = character.OptionC }, character.ClassEquipmentOptionUnknown},
	} {
		bad := cmd
		c.mutate(&bad)
		res, err := creator.Run(ctx, bad)
		rej, isRej := res.(character.CreateCharacterRejected)
		if err != nil || !isRej || rej.Problems()[0].Kind() != c.want {
			t.Errorf("%s: %#v, %v; want a rejection with %v", c.name, res, err, c.want)
		}
	}
}
