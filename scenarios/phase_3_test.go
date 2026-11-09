package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// The whole of Phase 3 for one bundled Wizard, through the public packages only: the character is built over the
// bundled Wizard, gains levels 2 and 3 with the bundled progression, chooses the bundled Evoker, takes its
// spellcasting state, casts Magic Missile with a level 2 slot, finishes a Short Rest and uses Arcane Recovery to
// get the slot back (SRD 5.2.1, "Gaining a Level", p. 23; "Wizard Subclass: Evoker", p. 81; "Casting Spells",
// pp. 104-105; "Level 1: Arcane Recovery", p. 78). Cites character-aggregate.md FR-001,
// level-advancement-and-multiclassing.md FR-009, FR-031, spellcasting.md FR-021, casting-and-concentration.md
// FR-057, class-wizard.md FR-050, FR-051, class-wizard-evoker.md FR-001.
func TestPhaseThreeBundledWizardLifetime(t *testing.T) {
	t.Parallel()
	ctx, src := context.Background(), data.MustBundledSource()
	wizard := content.MustClassID("wizard")
	created, err := character.New(character.NewParams{
		ID: character.MustIdentifier("wiz-1"), Class: wizard, HitPointDie: content.HitDieD6,
		Species: content.MustSpeciesID("human"), Background: content.MustBackgroundID("sage"),
		AbilityScores: scoresFor(t, 8, 14, 13, 16, 12, 10),
	})
	if err != nil {
		t.Fatal(err)
	}
	pc := created.Character().AwardExperience(character.MustExperienceAmount(900)).Character() // the minimum for level 3

	levelUp := must(character.NewLevelUpCharacter(character.WithSource(src), character.WithRoller(dicetest.NewFixedRoller())))
	for want := 2; want <= 3; want++ {
		step, err := levelUp.Run(ctx, character.LevelUpCharacterCommand{Character: pc, Class: wizard, HitPointChoice: character.TakeFixedValue{}})
		stepped, ok := step.(character.LevelUpCharacterSucceeded)
		if err != nil || !ok {
			t.Fatalf("level %d = %#v, %v", want, step, err)
		}
		pc = stepped.Character()
	}
	chosen, err := must(character.NewChooseSubclass(character.WithSource(src))).Run(ctx, character.ChooseSubclassCommand{Character: pc, Class: wizard, Subclass: "evoker"})
	subclass, ok := chosen.(character.ChooseSubclassSucceeded)
	if err != nil || !ok {
		t.Fatalf("ChooseSubclass = %#v, %v", chosen, err)
	}
	pc = subclass.Character()
	if pc.ClassLevels().TotalLevel().Value() != 3 {
		t.Fatalf("total level = %d, want 3", pc.ClassLevels().TotalLevel().Value())
	}

	spells := bundledSpells(t)
	setup, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: pc.ID(), Levels: pc.ClassLevels(), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("fire_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("magic_missile", "detect_magic"), Spellbook: character.NewSpellbook(ids("magic_missile", "detect_magic"))}}})
	if err != nil {
		t.Fatal(err)
	}
	state := setup.(character.CreateSpellcastingCreated).State()
	facts, err := character.NewCastingContext(character.CastingFacts{CanCastSpells: true, CanSpeak: true, CanMakeGestures: true, HasFreeHand: true, HasComponentPouch: true, TriggerMet: true})
	if err != nil {
		t.Fatal(err)
	}
	level2 := character.MustSlotLevel(2)
	cast, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: state, Class: character.CasterWizard, Spell: spells[content.MustSpellID("magic_missile")],
		Mode: character.CastModeNormal, Slot: character.NewSpellSlotChoiceSpellcasting(level2), Context: facts})
	casted, ok := cast.(character.CastSpellCast)
	if err != nil || !ok || casted.State().Slots(character.PoolSpellcasting).Remaining(level2) != 1 {
		t.Fatalf("CastSpell = %#v, %v", cast, err)
	}

	rest := finishedRest(t, pc.ID(), character.ShortRest)
	recovered, err := mustOK(character.NewUseArcaneRecovery(src)).Run(ctx, character.UseArcaneRecoveryCommand{CharacterID: pc.ID(), ClassLevels: pc.ClassLevels(),
		Expended: map[int]int{2: 1}, Chosen: map[int]int{2: 1}, Trigger: rest})
	if applied, ok := recovered.(character.ArcaneRecoveryApplied); err != nil || !ok || applied.Recovered().Levels() != 2 {
		t.Fatalf("UseArcaneRecovery = %#v, %v", recovered, err)
	}
}

// A level 1 Dragonborn uses Breath Weapon, takes a Long Rest and has the use back; a second use before the rest
// is counted and a use beyond the Proficiency Bonus limit (2) is refused (SRD 5.2.1, "Dragonborn Traits", p. 84;
// "Long Rest", p. 187). Cites species-trait-behaviour.md FR-030, FR-031, FR-032, FR-063.
func TestPhaseThreeDragonbornBreathWeapon(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	hero := must(character.New(character.NewParams{
		ID: character.MustIdentifier("dragon-1"), Class: content.MustClassID("fighter"), HitPointDie: content.HitDieD10,
		Species: content.MustSpeciesID("dragonborn"), Background: content.MustBackgroundID("soldier"),
		AbilityScores: scoresFor(t, 16, 12, 14, 10, 10, 10),
	})).Character()
	traits := must(character.NewSpeciesTraits(content.MustSpeciesID("dragonborn"), character.DragonbornChoices{Ancestry: character.DraconicRed}))
	expend := character.NewExpendSpeciesTraitUse()
	breath := character.BreathWeaponRequest{Shape: character.BreathCone15}
	for range 2 {
		res, err := expend.Run(ctx, character.ExpendSpeciesTraitUseCommand{Character: hero, Traits: traits, Request: breath})
		used, ok := res.(character.ExpendSpeciesTraitUseSucceeded)
		if err != nil || !ok {
			t.Fatalf("Breath Weapon = %#v, %v", res, err)
		}
		traits = used.Traits()
	}
	third, err := expend.Run(ctx, character.ExpendSpeciesTraitUseCommand{Character: hero, Traits: traits, Request: breath})
	if rejected, ok := third.(character.ExpendSpeciesTraitUseRejected); err != nil || !ok || rejected.Rejection.String() != "species_use_exhausted" {
		t.Fatalf("a third use = %#v, %v, want species_use_exhausted", third, err)
	}

	renewed, err := character.NewRenewSpeciesTraitUses().Run(ctx, character.RenewSpeciesTraitUsesCommand{Character: hero, Traits: traits, Rest: finishedRest(t, hero.ID(), character.LongRest)})
	back, ok := renewed.(character.RenewSpeciesTraitUsesSucceeded)
	if err != nil || !ok || back.Traits().ExpendedOf(character.BreathWeaponUse) != 0 {
		t.Fatalf("after a Long Rest = %#v, %v, want no use expended", renewed, err)
	}
}
