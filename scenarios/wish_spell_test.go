package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A Wizard 17 casts Wish for Reshape Reality and endures the stress (5 days of Strength 3, no lockout); casting
// Fireball with a level 3 slot then costs 3d10 Necrotic damage that nothing reduces; two days of rest bring the
// weakness down to 1 day; a finished Long Rest ends the cast damage but not the weakness. A second Wizard duplicates
// a spell with no stress and no dice (SRD 5.2.1, "Wish", p. 175; "Long Rest", p. 185). Cites wish-spell.md FR-004,
// FR-022, FR-030, FR-041, FR-050, FR-052.
func TestWishThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	ctx, id, spells := context.Background(), character.MustIdentifier("wizard"), bundledSpells(t)
	levels := spellLevels(t, "wizard", 17)
	pc, err := character.Restore(character.Parts{
		ID: id, ClassLevels: levels, Species: "human", Background: "sage", AbilityScores: scoresFor(t, 14, 10, 14, 20, 10, 10),
		HitPointMaximum: character.MustHitPointMaximum(110), HitDicePool: character.MustHitDicePoolFromLevels(levels, map[content.ClassID]content.HitPointDie{"wizard": content.HitDieD6}),
		HeroicInspiration: character.NotInspired,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The Wish: Reshape Reality, 2d4 = 2 + 3, percentile 50.
	roller := dicetest.NewFixedRoller(2, 3, 50)
	resolver, err := character.NewWishResolver(roller)
	if err != nil {
		t.Fatal(err)
	}
	statement, _ := character.NewWishStatement("A boon for the realm")
	choice, _ := character.NewReshapeReality(statement)
	res, err := resolver.ResolveWish(ctx, character.ResolveWishCommand{Character: pc, Choice: choice})
	applied, ok := res.(character.ResolveWishApplied)
	if err != nil || !ok {
		t.Fatalf("ResolveWish = %#v, %v", res, err)
	}
	stress, _, _ := character.ReadFeatureState(applied.Character().FeatureStates(), character.WishStressKey())
	weak, isWeak := stress.Weakness().(character.WishWeakened)
	if !isWeak || weak.DaysRemaining() != 5 || stress.CastDamage() != (character.WishCastDamageActive{}) || !stress.CanCastWish() {
		t.Fatalf("stress = %+v, want 5 days, cast damage active, no lockout", stress)
	}
	if got := character.WishEffectiveScores(applied.Character().AbilityScores(), stress).Score(ability.Strength); got != ability.MustScore(3) {
		t.Errorf("effective Strength = %v, want 3 (stored 14)", got)
	}
	pc = applied.Character()

	// Fireball with a level 3 slot costs 3d10 Necrotic damage.
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: id, Levels: levels, Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("fire_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("fireball"), Spellbook: character.NewSpellbook(ids("fireball"))}}})
	if err != nil {
		t.Fatal(err)
	}
	facts, err := character.NewCastingContext(character.CastingFacts{CanCastSpells: true, CanSpeak: true, CanMakeGestures: true, HasFreeHand: true, HasComponentPouch: true, TriggerMet: true, CanReadSpellbook: true})
	if err != nil {
		t.Fatal(err)
	}
	fireball := spells[content.MustSpellID("fireball")]
	cast, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: created.(character.CreateSpellcastingCreated).State(), Class: character.CasterWizard, Spell: fireball,
		Mode: character.CastModeNormal, Slot: character.NewSpellSlotChoiceSpellcasting(character.MustSlotLevel(3)), Context: facts})
	if err != nil {
		t.Fatal(err)
	}
	var spellCast character.SpellCast
	for _, e := range cast.(character.CastSpellCast).Events() {
		if sc, ok := e.(character.SpellCast); ok {
			spellCast = sc
		}
	}
	damageRoller := dicetest.NewFixedRoller(4, 7, 9)
	resolver, _ = character.NewWishResolver(damageRoller)
	dmg, err := resolver.RollWishStressDamage(ctx, character.RollWishStressDamageCommand{Character: pc, SpellCast: spellCast, Spell: fireball})
	rolled, ok := dmg.(character.RollWishStressDamageRolled)
	if err != nil || !ok || rolled.Damage().Type() != content.DamageNecrotic || rolled.Damage().Amount() != 20 {
		t.Fatalf("RollWishStressDamage = %#v, %v, want 20 Necrotic", dmg, err)
	}
	if reqs := damageRoller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(3, dice.D10(), 0) {
		t.Errorf("requests = %v, want one 3d10", reqs)
	}

	// Two days of rest, then a Long Rest.
	for range 2 {
		tr := character.WishAfterDay(id, stress, character.WishRestingDayWithLightActivity)
		stress = tr.Stress()
	}
	if w, ok := stress.Weakness().(character.WishWeakened); !ok || w.DaysRemaining() != 1 {
		t.Errorf("after two resting days = %+v, want weakened for 1 day", stress.Weakness())
	}
	rep := character.MustRestReport(character.LongRest, character.StandardLongRest, character.NewRestRanItsCourse(8, 0))
	long := character.NewRestTaker().TakeRest(ctx, character.MustTakeRestCommand(id, rep, 110, 0, nil, character.SpentHitDice{})).(character.TakeRestApplied)
	tr, err := character.WishAfterRest(id, stress, long.Applied())
	if err != nil || tr.Stress().CastDamage() != (character.WishCastDamageInactive{}) || tr.Stress().Weakness() != stress.Weakness() || len(tr.Events()) != 1 {
		t.Errorf("WishAfterRest = %+v, %v", tr, err)
	}

	// A duplication takes effect with no stress and no dice.
	idle := dicetest.NewFixedRoller()
	resolver, _ = character.NewWishResolver(idle)
	dupChoice, _ := character.NewDuplicateSpell(fireball)
	dup, err := resolver.ResolveWish(ctx, character.ResolveWishCommand{Character: applied.Character(), Choice: dupChoice})
	if _, ok := dup.(character.ResolveWishDuplicated); err != nil || !ok || len(idle.Requests()) != 0 {
		t.Errorf("duplication = %#v, %v, requests %v", dup, err, idle.Requests())
	}
}
