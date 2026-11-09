package scenarios_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// A Wizard 5 casts Magic Missile with a level 3 slot (cast at level 3, one slot spent); the host then
// reports a slot spent this turn, so a second slot spell is refused while a cantrip and a feature-granted
// cast still work; a Ritual begins as a longer casting with Concentration and an Incapacitated report fails
// it with no slot spent; a damage amount gives the Concentration save DC and a failed save ends the
// Concentration through the BreakConcentration use case and the pure transition alike (SRD 5.2.1, "Casting
// Spells", pp. 104-105; "Casting without Slots", p. 104; "Concentration", p. 179). Cites
// casting-and-concentration.md FR-057, FR-058, FR-061, FR-062, FR-072, FR-123, FR-128, FR-079.
func TestCastingThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	ctx, spells := context.Background(), bundledSpells(t)
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: character.MustIdentifier("wizard"), Levels: spellLevels(t, "wizard", 5), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("fire_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("magic_missile", "detect_magic", "fireball"), Spellbook: character.NewSpellbook(ids("magic_missile", "detect_magic", "fireball"))}}})
	if err != nil {
		t.Fatal(err)
	}
	state := created.(character.CreateSpellcastingCreated).State()
	facts := func(spent bool) character.CastingContext {
		c, err := character.NewCastingContext(character.CastingFacts{CanCastSpells: true, CanSpeak: true, CanMakeGestures: true, HasFreeHand: true, HasComponentPouch: true, TriggerMet: true, SlotSpentThisTurn: spent})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	cast := func(s character.SpellcastingState, spell string, mode character.CastMode, slot character.SpellSlotChoice, spent bool) character.CastSpellResult {
		t.Helper()
		got, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: s, Class: character.CasterWizard, Spell: spells[content.MustSpellID(spell)], Mode: mode, Slot: slot, Context: facts(spent)})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	level3 := character.NewSpellSlotChoiceSpellcasting(character.MustSlotLevel(3))

	first, ok := cast(state, "magic_missile", character.CastModeNormal, level3, false).(character.CastSpellCast)
	if !ok {
		t.Fatal("Magic Missile was not cast")
	}
	if at, _ := first.Report().CastAtLevel(); at != 3 || !first.Report().CountsAsSlotSpell() || first.State().Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(3)) != 1 {
		t.Errorf("report = %+v, level 3 slots = %d, want castAt 3 and one level 3 slot left", first.Report(), first.State().Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(3)))
	}

	if r, ok := cast(first.State(), "magic_missile", character.CastModeNormal, level3, true).(character.CastSpellRejected); !ok || r.Rejection().First() != character.CastRejectionSlotAlreadySpentThisTurn {
		t.Errorf("a second slot spell the same turn = %#v, want slot_already_spent_this_turn", r)
	}
	if _, ok := cast(first.State(), "fire_bolt", character.CastModeNormal, nil, true).(character.CastSpellCast); !ok {
		t.Error("a cantrip after a slot spell was refused")
	}
	if _, ok := cast(first.State(), "fireball", character.CastModeFeatureGranted, nil, true).(character.CastSpellCast); !ok {
		t.Error("a feature-granted cast after a slot spell was refused")
	}

	started, ok := cast(state, "detect_magic", character.CastModeRitual, nil, false).(character.CastSpellStarted)
	if !ok {
		t.Fatal("the Ritual did not start")
	}
	if c, going := started.State().CastingInProgress(); !going || c.Source() != character.CastSourceRitual || c.TotalMinutes() < 10 {
		t.Errorf("CastingInProgress = %+v %v", c, going)
	}
	broken, err := character.NewBreakConcentration().Do(ctx, character.BreakConcentrationCommand{State: started.State(), Reason: character.BreakIncapacitated})
	failed, isFailed := broken.(character.BreakConcentrationCastingFailed)
	if err != nil || !isFailed || failed.CastingFailed().Spell() != content.MustSpellID("detect_magic") {
		t.Fatalf("break = %#v, %v, want CastingFailed", broken, err)
	}
	if _, going := failed.State().CastingInProgress(); going || failed.State().Slots(character.PoolSpellcasting) != state.Slots(character.PoolSpellcasting) {
		t.Error("the casting remains or a slot was spent")
	}

	concentrating := cast(state, "dancing_lights", character.CastModeNormal, nil, false).(character.CastSpellCast).State()
	if _, ok := concentrating.Concentration(); !ok {
		t.Fatal("no Concentration after casting Dancing Lights")
	}
	dc, err := character.ConcentrationSaveDCForDamage(25)
	if err != nil || dc.Value() != 12 {
		t.Fatalf("DC = %v, %v, want 12", dc.Value(), err)
	}
	ended, err := character.NewBreakConcentration().Do(ctx, character.BreakConcentrationCommand{State: concentrating, Reason: character.BreakFailedSave})
	direct := character.ConcentrationRules{}.BreakConcentration(concentrating, character.BreakFailedSave)
	e, isEnded := ended.(character.BreakConcentrationEnded)
	if err != nil || !isEnded || !reflect.DeepEqual(e.State(), direct.State()) || !reflect.DeepEqual(e.Events(), direct.Events()) {
		t.Errorf("use case = %#v, %v; transition = %#v", ended, err, direct)
	}
}
