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
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A player creates a Barbarian over the bundled Barbarian definition with the Standard Array and the Barbarian creation choices, which seed the
// BarbarianState: two skills of the six offered are accepted, three, a skill outside the list (Arcana), a repeated one and an equipment option C
// the Barbarian does not have are rejected with their problem kinds, and option B gives 75 GP and nothing else (SRD 5.2.1, "Core Barbarian Traits",
// p. 28). Cites class-barbarian.md FR-004 to FR-006, FR-009 to FR-011, FR-109, FR-117.
func TestCreateABarbarian(t *testing.T) {
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
	creator := must(character.NewCreateCharacter(character.WithSource(source), character.WithRoller(dicetest.NewFixedRoller())))
	cmd := character.CreateCharacterCommand{
		CharacterID: character.MustIdentifier("pc-1"), ClassID: "barbarian", BackgroundID: "sage", SpeciesID: "human",
		AbilityScores: character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
			ability.Strength: 15, ability.Dexterity: 12, ability.Constitution: 13, ability.Intelligence: 8, ability.Wisdom: 10, ability.Charisma: 14}),
		BackgroundIncrease: character.MustPlusTwoPlusOne(ability.Constitution, ability.Intelligence),
		ClassSkills:        []content.Skill{content.SkillAthletics, content.SkillIntimidation}, ClassEquipment: character.OptionA, BackgroundEquipment: character.OptionB,
		Size: character.ChosenSize{Size: content.SizeMedium}, Languages: []content.Language{content.LanguageDwarvish, content.LanguageElvish}, Alignment: content.AlignmentNeutralGood,
		ClassChoices: character.BarbarianCreationChoices{},
	}
	res, err := creator.Run(ctx, cmd)
	created, ok := res.(character.CreateCharacterSucceeded)
	if err != nil || !ok {
		t.Fatalf("result = %#v, %v", res, err)
	}
	pc := created.Character()
	// D12 maximum plus the Constitution modifier of 13 + 2 = 15 (+2).
	if pc.HitPointMaximum().Value() != 14 || pc.ClassLevels().LevelIn("barbarian") != 1 {
		t.Errorf("Hit Point maximum %d, levels %v", pc.HitPointMaximum().Value(), pc.ClassLevels())
	}
	state, stored, err := character.ReadFeatureState(pc.FeatureStates(), character.BarbarianStateKey())
	if err != nil || !stored || state != character.InitialBarbarianState() {
		t.Errorf("state = %v, %v, %v; want the initial BarbarianState", state, stored, err)
	}
	if gold := created.Record().Equipment().Gold(); gold != content.MustCost(15+50, content.CoinGP, 0) {
		t.Errorf("Starting Equipment gold = %v, want 15 + 50 GP", gold)
	}
	optionB := cmd
	optionB.ClassEquipment = character.OptionB
	if res, err := creator.Run(ctx, optionB); err != nil || res.(character.CreateCharacterSucceeded).Record().Equipment().Gold() != content.MustCost(75+50, content.CoinGP, 0) {
		t.Errorf("option B = %#v, %v; want 75 + 50 GP", res, err)
	}
	for _, c := range []struct {
		name   string
		mutate func(*character.CreateCharacterCommand)
		want   character.CreateCharacterProblemKind
	}{
		{"three skills", func(c *character.CreateCharacterCommand) {
			c.ClassSkills = []content.Skill{content.SkillAthletics, content.SkillIntimidation, content.SkillNature}
		}, character.ClassSkillCountWrong},
		{"one skill", func(c *character.CreateCharacterCommand) { c.ClassSkills = []content.Skill{content.SkillAthletics} }, character.ClassSkillCountWrong},
		{"Arcana", func(c *character.CreateCharacterCommand) {
			c.ClassSkills = []content.Skill{content.SkillAthletics, content.SkillArcana}
		}, character.ClassSkillNotOffered},
		{"a repeated skill", func(c *character.CreateCharacterCommand) {
			c.ClassSkills = []content.Skill{content.SkillAthletics, content.SkillAthletics}
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

// A level 11 Barbarian of the Path of the Berserker, over the bundled definition, enters a Rage (Instinctive Pounce and the effects are reported),
// is not allowed to enter it again, keeps the Rage through the end of its first turn, extends it on the next, loses it at the end of a later turn
// it did not extend, then rages again, drops to 0 Hit Points and uses Relentless Rage through the dice port: a successful save sets Hit Points to
// twice the Barbarian level and the next DC is 15. A Short Rest then resets the DC and regains one Rage use (SRD 5.2.1, "Level 1: Rage", p. 28-29;
// "Level 11: Relentless Rage", p. 30; "Short Rest", p. 187). Cites class-barbarian.md FR-021 to FR-025, FR-031 to FR-033, FR-059, FR-065, FR-066, FR-099, FR-110.
func TestBarbarianFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundle := data.MustBundledSource()
	res, err := bundle.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("barbarian")))
	if err != nil {
		t.Fatal(err)
	}
	progression := must(character.NewBarbarianProgression(res.(content.LookupFound).Entry.(content.ClassDefinition)))
	level := character.MustBarbarianLevel(11)
	stats := progression.StatsAt(level)
	id := character.MustIdentifier("hero")
	classes := character.MustClassLevels([]character.ClassLevel{{Class: "barbarian", Level: 11}})
	scores := must(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(18), ability.Dexterity: ability.MustScore(12), ability.Constitution: ability.MustScore(16),
		ability.Intelligence: ability.MustScore(8), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(10),
	}))
	hero := must(character.Restore(character.Parts{
		ID: id, ExperiencePoints: character.MustExperiencePoints(85000), ClassLevels: classes, Species: "human", Background: "soldier",
		AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(120),
		HitDicePool:       character.MustHitDicePoolFromLevels(classes, map[content.ClassID]content.HitPointDie{"barbarian": content.HitDieD12}),
		HeroicInspiration: character.NotInspired, FeatureStates: character.BarbarianCreationChoices{}.SeedFeatureStates(character.FeatureStates{}),
	}))
	state := func() character.BarbarianState {
		s, _, _ := character.ReadFeatureState(hero.FeatureStates(), character.BarbarianStateKey())
		return s
	}

	rage := character.BarbarianRage{}
	turn := character.NewBarbarianTurn()
	entry := must(rage.Enter(id, stats, state(), turn, character.BarbarianHostFacts{}, character.BerserkerSubclassID, 30)).(character.RageEntered)
	if entry.State().RagesRemaining(stats) != 3 || !entry.Pounce().Allowed() || entry.Pounce().HalfSpeed() != 15 || entry.Turn().BonusActionAvailable() {
		t.Fatalf("entry = %#v", entry)
	}
	if again, ok := must(rage.Enter(id, stats, entry.State(), entry.Turn().StartNextTurn(), character.BarbarianHostFacts{}, "", 30)).(character.RageNotEntered); !ok || again.Reason != character.BarbarianRageAlreadyActive {
		t.Errorf("a second Rage = %#v", again)
	}
	effects := character.NewBarbarianEffects(stats, character.BerserkerSubclassID, entry.State(), entry.Turn(), character.BarbarianHostFacts{})
	if len(effects.DamageResistances()) != 3 || effects.RageDamageBonusFor(character.BarbarianAttackFacts{UsesStrength: true, Kind: character.BarbarianWeaponAttack, DealsDamage: true}) != 3 || len(effects.Immunities()) != 2 {
		t.Errorf("effects = %+v", effects)
	}
	first := must(rage.End(id, stats, entry.State(), character.RageTurnEnded{})).(character.RageEndApplied)
	extended := must(rage.Extend(stats, first.State(), character.NewBarbarianTurn(), character.RageExtendedByAttackRoll)).(character.RageExtended)
	second := must(rage.End(id, stats, extended.State(), character.RageTurnEnded{})).(character.RageEndApplied)
	third := must(rage.End(id, stats, second.State(), character.RageTurnEnded{})).(character.RageEndApplied)
	if !first.State().RageActive() || !second.State().RageActive() || third.State().RageActive() || len(third.Events()) != 1 {
		t.Fatalf("turn ends: %v %v %v", first.State(), second.State(), third.State())
	}

	hero = must(character.WithFeatureState(hero, character.BarbarianStateKey(),
		must(rage.Enter(id, stats, third.State(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, "", 30)).(character.RageEntered).State()))
	relentless := must(character.NewUseRelentlessRage(progression, dicetest.NewFixedRoller(12)))
	out := must(relentless.Run(ctx, character.UseRelentlessRageCommand{Character: hero, Facts: character.BarbarianHostFacts{DroppedToZeroHitPoints: true},
		RollMode: modifier.Normal, Modifiers: d20.NewRollModifiers(modifier.New(3))}))
	won, ok := out.(character.RelentlessRageSucceeded)
	if !ok || won.HitPoints() != 22 || won.NextDC() != 15 {
		t.Fatalf("Relentless Rage = %#v", out)
	}
	rested := must(character.WithFeatureState(won.Character(), character.BarbarianStateKey(),
		func() character.BarbarianState {
			s, _, _ := character.ReadFeatureState(won.Character().FeatureStates(), character.BarbarianStateKey())
			return s.AfterRest(character.ShortRest)
		}()))
	if s, _, _ := character.ReadFeatureState(rested.FeatureStates(), character.BarbarianStateKey()); s.RelentlessRageDC() != 10 || s.RagesExpended() != 1 || s.RageActive() {
		t.Errorf("after a Short Rest: %v", s)
	}
}
