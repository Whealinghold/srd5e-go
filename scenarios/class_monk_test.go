package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

func monkTool(id, name string, group content.ToolGroup) content.Entry {
	use := must(content.NewToolUtilizeAction("Use it", 15))
	return must(content.NewStandaloneTool(content.ItemID(id), name, content.SourceCitation{}, group, ability.Dexterity, []content.ToolUtilizeAction{use}, nil,
		content.MustFixedCost(content.MustCost(10, content.CoinGP, 0)), content.MustWeight(5, "")))
}

// A player creates a Monk over the bundled Monk definition with the Standard Array and the Monk creation choices, which seed the MonkState: two skills of the six offered and one
// Artisan's Tools tool are accepted (Hit Point maximum 8 plus the Constitution modifier); one skill, Arcana, a repeated skill, a tool outside Artisan's Tools and Musical Instrument, no tool and an
// equipment option C the Monk does not have are rejected with their problem kinds, and option B gives 50 GP and nothing else (SRD 5.2.1, "Core Monk Traits", p. 49). Cites class-monk.md FR-004 to
// FR-005, FR-007, FR-008, FR-012, FR-014, FR-143.
func TestCreateAMonk(t *testing.T) {
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
	source := layeredSource{top: contenttest.NewFakeSource(sage, alert,
		monkTool("calligraphers_supplies", "Calligrapher’s Supplies", content.ToolGroupArtisans), monkTool("thieves_tools", "Thieves’ Tools", content.ToolGroupOther)), base: data.MustBundledSource()}
	creator := must(character.NewCreateCharacter(character.WithSource(source), character.WithRoller(dicetest.NewFixedRoller())))
	cmd := character.CreateCharacterCommand{
		CharacterID: character.MustIdentifier("pc-1"), ClassID: "monk", BackgroundID: "sage", SpeciesID: "human",
		AbilityScores: character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
			ability.Strength: 8, ability.Dexterity: 15, ability.Constitution: 12, ability.Intelligence: 10, ability.Wisdom: 14, ability.Charisma: 13}),
		BackgroundIncrease: character.MustPlusTwoPlusOne(ability.Wisdom, ability.Constitution),
		ClassSkills:        []content.Skill{content.SkillAcrobatics, content.SkillInsight}, ClassTools: []character.ChosenTool{{Tool: "calligraphers_supplies"}},
		ClassEquipment: character.OptionA, BackgroundEquipment: character.OptionB,
		Size: character.ChosenSize{Size: content.SizeMedium}, Languages: []content.Language{content.LanguageDwarvish, content.LanguageElvish}, Alignment: content.AlignmentNeutralGood,
		ClassChoices: character.MonkCreationChoices{},
	}
	res, err := creator.Run(ctx, cmd)
	created, ok := res.(character.CreateCharacterSucceeded)
	if err != nil || !ok {
		t.Fatalf("result = %#v, %v", res, err)
	}
	pc := created.Character()
	// D8 maximum plus the Constitution modifier of 12 + 1 = 13 (+1).
	if pc.HitPointMaximum().Value() != 9 || pc.ClassLevels().LevelIn("monk") != 1 {
		t.Errorf("Hit Point maximum %d, levels %v", pc.HitPointMaximum().Value(), pc.ClassLevels())
	}
	state, stored, err := character.ReadFeatureState(pc.FeatureStates(), character.MonkStateKey())
	if err != nil || !stored || state != character.InitialMonkState() {
		t.Errorf("state = %v, %v, %v; want the initial MonkState", state, stored, err)
	}
	if gold := created.Record().Equipment().Gold(); gold != content.MustCost(11+50, content.CoinGP, 0) {
		t.Errorf("Starting Equipment gold = %v, want 11 + 50 GP", gold)
	}
	optionB := cmd
	optionB.ClassEquipment = character.OptionB
	if res, err := creator.Run(ctx, optionB); err != nil || res.(character.CreateCharacterSucceeded).Record().Equipment().Gold() != content.MustCost(50+50, content.CoinGP, 0) {
		t.Errorf("option B = %#v, %v; want 50 + 50 GP", res, err)
	}
	for _, c := range []struct {
		name   string
		mutate func(*character.CreateCharacterCommand)
		want   character.CreateCharacterProblemKind
	}{
		{"one skill", func(c *character.CreateCharacterCommand) { c.ClassSkills = []content.Skill{content.SkillAcrobatics} }, character.ClassSkillCountWrong},
		{"Arcana", func(c *character.CreateCharacterCommand) {
			c.ClassSkills = []content.Skill{content.SkillAcrobatics, content.SkillArcana}
		}, character.ClassSkillNotOffered},
		{"a repeated skill", func(c *character.CreateCharacterCommand) {
			c.ClassSkills = []content.Skill{content.SkillAcrobatics, content.SkillAcrobatics}
		}, character.ClassSkillRepeated},
		{"no tool", func(c *character.CreateCharacterCommand) { c.ClassTools = nil }, character.ClassToolCountWrong},
		{"a tool outside the categories", func(c *character.CreateCharacterCommand) {
			c.ClassTools = []character.ChosenTool{{Tool: "thieves_tools"}}
		}, character.ClassToolNotInCategory},
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

// A level 5 Monk of the Warrior of the Open Hand, over the bundled definition, spends a Focus Point on Flurry of Blows through the dice port, deflects a hit (the d10 roll
// plus Dexterity +3 plus level 5), redirects the force when the damage reaches 0 (two d8 plus Dexterity, against a DC of 8 + 3 + 3), attempts a Stunning Strike (a failed save stuns, a second
// attempt in the turn is refused), uses Uncanny Metabolism on Initiative (all points back, level plus the d8 as Hit Points, once until a Long Rest), then a Short Rest regains the points and
// leaves Uncanny Metabolism spent while a Long Rest restores it (SRD 5.2.1, "Level 2: Monk's Focus", "Level 3: Deflect Attacks", "Level 5: Stunning Strike", p. 50-51). Cites class-monk.md
// FR-034, FR-036, FR-046, FR-049, FR-051 to FR-061, FR-076, FR-077, FR-079, FR-124, FR-133.
func TestMonkFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	res, err := data.MustBundledSource().Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("monk")))
	if err != nil {
		t.Fatal(err)
	}
	progression := must(character.NewMonkProgression(res.(content.LookupFound).Entry.(content.ClassDefinition)))
	stats := progression.StatsAt(character.MustMonkLevel(5))
	classes := character.MustClassLevels([]character.ClassLevel{{Class: "monk", Level: 5}})
	scores := must(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(10), ability.Dexterity: ability.MustScore(16), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(8), ability.Wisdom: ability.MustScore(16), ability.Charisma: ability.MustScore(10),
	}))
	hero := must(character.Restore(character.Parts{
		ID: character.MustIdentifier("hero"), ExperiencePoints: character.MustExperiencePoints(6500), ClassLevels: classes, Species: "human", Background: "soldier",
		AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(40),
		HitDicePool:       character.MustHitDicePoolFromLevels(classes, map[content.ClassID]content.HitPointDie{"monk": content.HitDieD8}),
		HeroicInspiration: character.NotInspired, FeatureStates: character.MonkCreationChoices{}.SeedFeatureStates(character.FeatureStates{}),
	}))
	state := func(c character.Character) character.MonkState {
		s, _, _ := character.ReadFeatureState(c.FeatureStates(), character.MonkStateKey())
		return s
	}

	flurry := must(character.NewUseFocusFeature(progression, dicetest.NewFixedRoller()))
	afterFlurry := must(flurry.Run(ctx, character.UseFocusFeatureCommand{Character: hero, Use: character.MonkFocusUse{Feature: character.MonkFlurryOfBlows, SpendPoint: true}})).(character.FocusFeatureApplied)
	if afterFlurry.Resolution().UnarmedStrikes() != 2 || state(afterFlurry.Character()).FocusPoints(stats).Available() != 4 {
		t.Fatalf("Flurry of Blows = %#v", afterFlurry)
	}
	hero = afterFlurry.Character()

	deflect := must(character.NewResolveDeflectAttacks(progression, dicetest.NewFixedRoller(10)))
	deflected := must(deflect.Run(ctx, character.ResolveDeflectAttacksCommand{Character: hero, DamageTypes: []content.DamageType{content.DamageSlashing}, TotalDamage: 18})).(character.DeflectAttacksApplied)
	if r := deflected.Resolution(); r.Reduction() != 18 || r.DamageAfter() != 0 || !r.CanRedirect() {
		t.Fatalf("Deflect Attacks = %+v, want the damage reduced to 0", r)
	}
	redirect := must(character.NewRedirectDeflectedAttack(progression, dicetest.NewFixedRoller(4, 5)))
	redirected := must(redirect.Run(ctx, character.RedirectDeflectedAttackCommand{Character: hero, Outcome: d20.Failure, Request: character.MonkRedirectRequest{
		Attack: character.MonkMeleeAttack, DamageType: content.DamageSlashing, Target: character.MonkRedirectTarget{Seen: true, DistanceFeet: 5}}})).(character.RedirectApplied)
	if r := redirected.Resolution(); r.DamageApplied() != 12 || r.DifficultyClass() != 14 || state(redirected.Character()).FocusExpended() != 2 {
		t.Fatalf("redirect = %+v, state %v", r, state(redirected.Character()))
	}
	hero = redirected.Character()

	hit := character.MonkStunningHit{Hit: true, MonkWeaponOrUnarmed: true}
	stun := must(character.StunningStrike{}.Resolve(stats, state(hero), character.MonkAbilitiesOf(hero), hit, d20.Failure))
	if _, stunned := stun.Effect().(character.StunApplied); !stunned || stun.DifficultyClass() != 14 || stun.State().FocusExpended() != 3 {
		t.Fatalf("Stunning Strike = %+v", stun)
	}
	hero = must(character.WithFeatureState(hero, character.MonkStateKey(), stun.State()))
	again := character.StunningStrike{}.Availability(stats, state(hero), character.MonkStunningHit{Hit: true, MonkWeaponOrUnarmed: true, UsedThisTurn: true})
	if un, ok := again.(character.MonkUnavailable); !ok || un.Reason != character.MonkAlreadyUsed {
		t.Errorf("a second strike in the turn = %#v, want alreadyUsed", again)
	}

	uncanny := must(character.NewUseUncannyMetabolism(progression, dicetest.NewFixedRoller(6)))
	healed := must(uncanny.Run(ctx, character.UseUncannyMetabolismCommand{Character: hero, RolledInitiative: true})).(character.UncannyMetabolismApplied)
	if healed.HitPointsToRegain() != 11 || healed.FocusPointsRegained() != 3 || state(healed.Character()).FocusPoints(stats).Available() != 5 {
		t.Fatalf("Uncanny Metabolism = %d Hit Points, %d regained", healed.HitPointsToRegain(), healed.FocusPointsRegained())
	}
	hero = healed.Character()
	if again := must(uncanny.Run(ctx, character.UseUncannyMetabolismCommand{Character: hero, RolledInitiative: true})); again.(character.UncannyMetabolismNotAvailable).Reason != character.MonkAlreadyUsed {
		t.Errorf("a second Uncanny Metabolism = %#v", again)
	}
	hero = must(character.WithFeatureState(hero, character.MonkStateKey(), character.MustMonkState(4, true, 0, character.QuiveringPalmVibrations{})))
	short := state(hero).AfterRest(character.ShortRest)
	long := state(hero).AfterRest(character.LongRest)
	if short.FocusExpended() != 0 || !short.UncannyMetabolismUsed() || long.UncannyMetabolismUsed() {
		t.Errorf("after rests: short %v, long %v", short, long)
	}
}
