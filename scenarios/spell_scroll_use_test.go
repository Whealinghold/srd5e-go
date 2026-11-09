package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A Wizard 1 reads a Fireball scroll (level 3, above their level 1 slots): the ability check is
// Intelligence against DC 13; a failed check erases the spell with no cast and no slot spent; a passed
// check yields terms CastSpell accepts with CastModeSpellScroll (DC 15, +7, no slot spent), after which
// SettleScrollReading consumes the scroll; a cast the rules refuse leaves it intact (SRD 5.2.1, "Spell
// Scroll", p. 244; "Casting without Slots", p. 104). Cites spell-scroll-use.md FR-008, FR-010, FR-011,
// FR-016, FR-018, FR-033, FR-036.
func TestSpellScrollUseThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	ctx, spells := context.Background(), bundledSpells(t)
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: character.MustIdentifier("wizard"), Levels: spellLevels(t, "wizard", 1), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("fire_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("magic_missile", "detect_magic"), Spellbook: character.NewSpellbook(ids("magic_missile", "detect_magic"))}}})
	if err != nil {
		t.Fatal(err)
	}
	state := created.(character.CreateSpellcastingCreated).State()
	scroll, err := character.NewSpellScroll(spells[content.MustSpellID("fireball")], character.FoundScroll{})
	if err != nil {
		t.Fatal(err)
	}
	route, err := character.NewScrollClassRoute(content.MustClassID("wizard"), ability.Intelligence)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := character.NewScrollReaderProfile(character.ScrollReaderFacts{UnderstandsWrittenLanguage: true, HighestSlotLevel: 1, ClassRoutes: []character.ScrollClassRoute{route}})
	if err != nil {
		t.Fatal(err)
	}
	check := func(natural int) d20.CheckResult {
		roll, err := d20.NewRoll(modifier.Normal, []int{natural})
		if err != nil {
			t.Fatal(err)
		}
		res, err := d20.ResolveCheck(d20.AbilityCheck, roll, d20.NewRollModifiers(modifier.New(0)), 13)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	facts := func(canCast bool) character.CastingContext {
		c, err := character.NewCastingContext(character.CastingFacts{CanCastSpells: canCast, CanSpeak: true, CanMakeGestures: true, HasFreeHand: true, TriggerMet: true})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	cast := func(terms character.ScrollCastTerms, canCast bool) character.CastSpellResult {
		t.Helper()
		castTerms, err := terms.CastTerms()
		if err != nil {
			t.Fatal(err)
		}
		got, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: state, Spell: spells[terms.SpellID()], Mode: character.CastModeSpellScroll, Context: facts(canCast), Terms: &castTerms})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	required, ok := character.AssessScrollReading(reader, scroll, character.ItemIntact).(character.ScrollCheckRequired)
	if !ok || required.DC() != 13 || len(required.Options()) != 1 {
		t.Fatalf("assessment = %#v, want a DC 13 check with one option", required)
	}
	option := required.Options()[0]

	failed, err := character.ResolveScrollCheck(required, option, check(12))
	erased, isErased := failed.(character.ScrollSpellErased)
	if err != nil || !isErased || erased.Disposition() != character.DispositionErased {
		t.Fatalf("failed check = %#v, %v, want ScrollSpellErased", failed, err)
	}

	passed, err := character.ResolveScrollCheck(required, option, check(13))
	if err != nil {
		t.Fatal(err)
	}
	terms := passed.(character.ScrollCheckPassed).Terms()
	done, ok := cast(terms, true).(character.CastSpellCast)
	if !ok {
		t.Fatal("the scroll cast was not performed")
	}
	if at, _ := done.Report().CastAtLevel(); at != 3 || done.Report().CountsAsSlotSpell() || done.State().Slots(character.PoolSpellcasting) != state.Slots(character.PoolSpellcasting) {
		t.Errorf("report = %+v, want cast at 3, no slot spent, not a slot spell", done.Report())
	}
	if got := character.SettleScrollReading(character.ProgressCompleted); got != character.DispositionConsumed {
		t.Errorf("disposition after the cast = %v, want consumed", got)
	}

	if _, ok := cast(terms, false).(character.CastSpellRejected); !ok {
		t.Error("a caster who cannot cast was not rejected")
	}
	if got := character.SettleScrollReading(character.ProgressNotStarted); got != character.DispositionRetained {
		t.Errorf("disposition after a rejected cast = %v, want retained", got)
	}
}
