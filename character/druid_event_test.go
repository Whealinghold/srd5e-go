package character_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-druid.md FR-166 - every Druid event is a past-tense immutable value implementing Event and DruidEvent, carries the CharacterID taken from the DruidCharacterFacts and
// is outside the Character aggregate's closed RecordEvent family; the cost methods of the events that have one are as the book says.
// Path: per case (happy).
func TestDruidEventsCarryTheCharacterID(t *testing.T) {
	t.Parallel()
	tables := druTables(t)
	facts := druFacts(t, 20, true)
	f := character.DruidStateFields{WildShapeUsesRemaining: 4, KnownForms: wolfKnown, ElementalFury: character.ElementalFuryPrimalStrike, LandType: character.LandArid}
	state := druState(t, 20, true, f)
	hit := rogAttack(t, true)
	var events []character.Event
	apply := func(c character.DruidChoice) {
		r, err := character.ApplyDruidChoice(character.ApplyDruidChoiceInput{State: druState(t, 20, true, character.DruidStateFields{}), Tables: tables, Facts: facts, Choice: c})
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, r.(character.ApplyDruidChoiceApplied).Events()...)
	}
	apply(character.DruidChoicePrimalOrder{Order: character.PrimalOrderMagician})
	apply(character.DruidChoiceElementalFury{Option: character.ElementalFuryPotentSpellcasting})
	apply(character.DruidChoiceLandType{Land: character.LandPolar})
	apply(character.DruidChoiceKnownForms{Forms: []character.BeastFormFacts{druForm("wolf", "1/4", false)}})
	rep, rerr := character.ApplyDruidChoice(character.ApplyDruidChoiceInput{State: druState(t, 20, true, character.DruidStateFields{KnownForms: wolfKnown, RestWindow: []character.RestWindowPermission{character.RestWindowKnownFormReplacement}}),
		Tables: tables, Facts: facts, Choice: character.DruidChoiceReplaceForm{Replace: "wolf", With: druForm("bear", "1", false)}})
	if rerr != nil {
		t.Fatal(rerr)
	}
	events = append(events, rep.(character.ApplyDruidChoiceApplied).Events()...)
	events = append(events, gained(t, gainInput(t, 19, 20, druState(t, 19, true, character.DruidStateFields{}))).Events()...)
	slots := druSlots(t, 20, map[int]int{1: 1, 2: 1})
	for _, u := range []character.DruidFeatureUse{
		character.DruidUseAssumeWildShape{Form: "wolf"}, character.DruidUseWildCompanion{Payment: character.DruidPaymentWildShape{}}, character.DruidUseLandsAid{DC: dcOf(t, 15)},
		character.DruidUseCreateSanctuary{DistanceFeet: 10}, character.DruidUsePrimalStrike{Source: character.PrimalStrikeSourceWeapon, Attack: hit},
		character.DruidUseSlotForWildShapeUse{Pool: character.PoolSpellcasting, Level: 1}, character.DruidUseNatureMagician{Pool: character.PoolSpellcasting, Uses: 1},
	} {
		in := character.UseDruidFeatureInput{State: state, Tables: tables, Facts: facts, Spellcasting: slots, Use: u}
		if _, ok := u.(character.DruidUseSlotForWildShapeUse); ok {
			in.State = druState(t, 20, true, character.DruidStateFields{KnownForms: wolfKnown, LandType: character.LandArid})
		}
		r, err := character.UseDruidFeature(in)
		if err != nil {
			t.Fatal(err)
		}
		used, ok := r.(character.UseDruidFeatureUsed)
		if !ok {
			t.Fatalf("%T rejected: %#v", u, r)
		}
		events = append(events, used.Events()...)
	}
	for _, in := range []struct {
		state character.DruidFeatureState
		use   character.DruidFeatureUse
	}{
		{druState(t, 20, true, character.DruidStateFields{WildShapeUsesRemaining: 4, KnownForms: wolfKnown, ActiveForm: "wolf"}), character.DruidUseEndWildShape{Reason: character.WildShapeEndLeftEarly}},
		{druState(t, 20, true, character.DruidStateFields{SanctuaryActive: true}), character.DruidUseMoveSanctuary{MoveFeet: 5, DistanceFeet: 5}},
		{druState(t, 20, true, character.DruidStateFields{SanctuaryActive: true}), character.DruidUseEndSanctuary{Reason: character.SanctuaryEndDied}},
		{druState(t, 20, true, character.DruidStateFields{RestWindow: []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}}), character.DruidUseNaturalRecoveryRecover{Pool: character.PoolSpellcasting, Levels: []int{1}}},
	} {
		r, err := character.UseDruidFeature(character.UseDruidFeatureInput{State: in.state, Tables: tables, Facts: facts, Spellcasting: slots, Use: in.use})
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, r.(character.UseDruidFeatureUsed).Events()...)
	}
	tr, err := character.ApplyDruidTrigger(character.ApplyDruidTriggerInput{State: druState(t, 20, true, character.DruidStateFields{FamiliarPresent: true, KnownForms: wolfKnown}), Tables: tables, Facts: facts,
		Trigger: character.DruidTriggerRestFinished{Kind: character.LongRest}})
	if err != nil {
		t.Fatal(err)
	}
	events = append(events, tr.Events()...)
	ini, _ := character.ApplyDruidTrigger(character.ApplyDruidTriggerInput{State: druState(t, 20, true, character.DruidStateFields{}), Tables: tables, Facts: facts, Trigger: character.DruidTriggerInitiativeRolled{}})
	events = append(events, ini.Events()...)
	seen := map[string]bool{}
	for _, e := range events {
		_, isDruid := e.(character.DruidEvent)
		_, isRecord := e.(character.RecordEvent)
		if !isDruid || isRecord || e.CharacterID() != druTestID {
			t.Errorf("%T: druid %v, record %v, id %v", e, isDruid, isRecord, e.CharacterID())
		}
		seen[fmt.Sprintf("%T", e)] = true
	}
	if len(seen) != 18 {
		t.Errorf("%d event types seen, want 18: %v", len(seen), seen)
	}
}
