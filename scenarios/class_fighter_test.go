package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A level 5 Fighter, built over the bundled Fighter definition, is seeded by FighterCreationChoices, chooses four Weapon Mastery kinds and
// a Fighting Style through ApplyFighterOptionChoice (content from a FakeSource: the bundle holds no weapons or feats yet), uses Second Wind
// (Tactical Shift offered at level 5), uses Action Surge on the same turn and is refused a second one, then takes a Short Rest and regains one
// Second Wind use and the Action Surge use (SRD 5.2.1, "Level 1: Second Wind", "Level 2: Action Surge", pp. 48; "Short Rest", p. 187). Cites
// class-fighter.md FR-032, FR-046, FR-051 to FR-056, FR-059, FR-063, FR-071 to FR-075, FR-144, FR-148.
func TestFighterFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundle, err := data.NewBundledSource()
	if err != nil {
		t.Fatal(err)
	}
	res, err := bundle.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("fighter")))
	if err != nil {
		t.Fatal(err)
	}
	progression, err := character.NewFighterProgression(res.(content.LookupFound).Entry.(content.ClassDefinition))
	if err != nil {
		t.Fatal(err)
	}

	scores := must2(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(16), ability.Dexterity: ability.MustScore(12), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(10),
	}))
	classes := character.MustClassLevels([]character.ClassLevel{{Class: "fighter", Level: 5}})
	hero := must2(character.Restore(character.Parts{
		ID: character.MustIdentifier("hero"), ExperiencePoints: character.MustExperiencePoints(6500), ClassLevels: classes, Species: "human", Background: "soldier",
		AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(44),
		HitDicePool:       character.MustHitDicePoolFromLevels(classes, map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10}),
		HeroicInspiration: character.NotInspired,
		FeatureStates:     character.FighterCreationChoices{}.SeedFeatureStates(character.FeatureStates{}),
	}))

	d := func(count int, die dice.Die) content.WeaponDamage {
		return must2(content.NewWeaponDiceDamage(dice.MustExpression(count, die, 0), content.DamageSlashing))
	}
	heavy := must2(content.NewPlainUse(content.WeaponPropertyHeavy))
	names := map[string]content.MasteryProperty{"greatsword": content.MasteryGraze, "longsword": content.MasterySap, "javelin": content.MasterySlow, "mace": content.MasterySap}
	entries := []content.Entry{
		must2(content.NewFeat("defense", "Defense", content.SourceCitation{}, content.FeatFields{Category: content.FeatFightingStyle, Benefits: []content.FeatBenefit{content.MustFeatBenefit("Defense", "A benefit.")}})),
	}
	for id, m := range names {
		entries = append(entries, weapon(t, id, content.WeaponMartial, d(1, dice.D8()), m, heavy))
	}
	src := contenttest.NewFakeSource(entries...)

	apply := must2(character.NewApplyFighterOptionChoice(progression, src))
	chosen := must2(apply.Run(ctx, character.ApplyFighterOptionChoiceCommand{Character: hero, Choice: character.FighterChooseWeaponMasteries{
		Kinds: []content.ItemID{"greatsword", "longsword", "javelin", "mace"}}}))
	applied, ok := chosen.(character.FighterOptionApplied)
	if !ok {
		t.Fatalf("masteries = %#v, want applied", chosen)
	}
	style := must2(apply.Run(ctx, character.ApplyFighterOptionChoiceCommand{Character: applied.Character(), Choice: character.FighterChooseFightingStyle{
		Choice: must2(character.NewFightingStyleChoice("defense"))}}))
	styled, ok := style.(character.FighterOptionApplied)
	if !ok || !styled.Feats().Holds("defense") {
		t.Fatalf("style = %#v, want applied with the feat held", style)
	}
	hero = styled.Character()
	access := must2(character.FighterEffects{}.MasteryAccess(hero.ClassLevels(), hero.FeatureStates()))
	if !access.Permits("greatsword") || access.Permits("dagger") {
		t.Error("the Fighter has Weapon Mastery with the four chosen kinds only")
	}

	secondWind := must2(character.NewUseSecondWind(progression, dicetest.NewFixedRoller(6)))
	used := must2(secondWind.Run(ctx, character.UseSecondWindCommand{Character: hero})).(character.SecondWindApplied)
	if used.HitPointsToRegain() != 11 {
		t.Errorf("Second Wind regains %d, want 6 + 5", used.HitPointsToRegain())
	}
	if shift, ok := used.TacticalShift().(character.TacticalShiftOffered); !ok {
		t.Error("Tactical Shift is offered at level 5")
	} else if half := must2(shift.HalfOf(30)); half != 15 {
		t.Errorf("half of Speed 30 = %d, want 15", half)
	}

	surge := must2(character.NewUseActionSurge(progression))
	first := must2(surge.Run(ctx, character.UseActionSurgeCommand{Character: used.Character(), Turn: character.FighterOwnTurn, PerTurn: character.FighterActionSurgeUnused}))
	afterSurge := first.(character.ActionSurgeApplied).Character()
	again := must2(surge.Run(ctx, character.UseActionSurgeCommand{Character: afterSurge, Turn: character.FighterOwnTurn, PerTurn: character.FighterActionSurgeUsed}))
	if refused, ok := again.(character.ActionSurgeNotAvailable); !ok || refused.Reason != character.FighterAlreadyUsedThisTurn {
		t.Errorf("a second Action Surge on the turn = %#v, want already used this turn", again)
	}

	rest := must2(character.NewRestReport(character.ShortRest, character.StandardLongRest, character.NewRestRanItsCourse(1, 0)))
	cmd := must2(character.NewTakeRestCommand(afterSurge.ID(), rest, 30, 0, nil, character.SpentHitDice{}))
	short := character.NewRestTaker().TakeRest(ctx, cmd).(character.TakeRestApplied).Applied()
	state, _, err := character.ReadFeatureState(afterSurge.FeatureStates(), character.FighterStateKey())
	if err != nil {
		t.Fatal(err)
	}
	if after := state.AfterRest(short.BenefitsOf()); after.SecondWindExpended() != 0 || after.ActionSurgeExpended() != 0 {
		t.Errorf("after a Short Rest: Second Wind expended %d, Action Surge expended %d; want 0 and 0", after.SecondWindExpended(), after.ActionSurgeExpended())
	}
}
