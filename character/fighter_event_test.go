package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-fighter.md FR-155, FR-147 - every Fighter event is a FighterEvent and an Event (not a RecordEvent), carries the CharacterId of the character, and is
// produced by the use case of its feature: SecondWindUsed, TacticalMindUsed, ActionSurgeUsed, IndomitableUsed, FighterFightingStyleChosen, FighterFightingStyleReplaced,
// FighterWeaponMasteriesChosen and FighterWeaponMasterySwapped; a rejection produces none.
// Path: per case.
func TestFighterEventsAreFighterEventsOfTheCharacter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := fgtSource(t)
	fighter := fgtFighter(t, 9)
	id := fighter.ID()
	var events []character.Event
	sw := must(useSecondWind(t, dicetest.NewFixedRoller(5)).Run(ctx, character.UseSecondWindCommand{Character: fighter})).(character.SecondWindApplied)
	tm := must(useTacticalMind(t, dicetest.NewFixedRoller(10)).Run(ctx, character.UseTacticalMindCommand{Character: fighter, FailedCheck: failedCheck(t, 5)})).(character.TacticalMindSucceeded)
	as := must(useActionSurge(t).Run(ctx, character.UseActionSurgeCommand{Character: fighter, Turn: character.FighterOwnTurn, PerTurn: character.FighterActionSurgeUnused})).(character.ActionSurgeApplied)
	in := must(useIndomitable(t, dicetest.NewFixedRoller(20)).Run(ctx, character.UseIndomitableCommand{Character: fighter, FailedSave: fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{5}, 0, 15)})).(character.IndomitableSucceeded)
	chosen := apply(t, src, fgtFighter(t, 1), character.TakenFeats{}, character.FighterChooseFightingStyle{Choice: styleChoice("defense")}).(character.FighterOptionApplied)
	facts, lg := fgtReplaceFacts(t, fgtStyle("archery"))
	replaced := apply(t, src, facts.Character, facts.Feats, character.FighterReplaceFightingStyle{Choice: styleChoice("archery"), LevelGained: lg}).(character.FighterOptionApplied)
	masteries := apply(t, src, fgtFighter(t, 1), character.TakenFeats{}, character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace")}).(character.FighterOptionApplied)
	swapped := apply(t, src, masteries.Character(), character.TakenFeats{}, character.FighterSwapWeaponMastery{Remove: "mace", Add: "rapier", RestApplied: fgtRest(t, masteries.Character().ID(), character.LongRest)}).(character.FighterOptionApplied)
	ids := []character.Identifier{id, id, id, id, chosen.Character().ID(), replaced.Character().ID(), masteries.Character().ID(), swapped.Character().ID()}
	for _, evs := range [][]character.Event{sw.Events(), tm.Events(), as.Events(), in.Events(), chosen.Events()[1:], replaced.Events()[2:], masteries.Events(), swapped.Events()} {
		events = append(events, evs...)
	}
	if len(events) != 8 {
		t.Fatalf("%d events, want 8", len(events))
	}
	for i, e := range events {
		fe, ok := e.(character.FighterEvent)
		_, record := e.(character.RecordEvent)
		if !ok || record || e.CharacterID() != ids[i] || fe.CharacterID() != ids[i] {
			t.Errorf("event %d = %T for %v, want a FighterEvent of %v", i, e, e.CharacterID(), ids[i])
		}
	}
	rejected := must(useActionSurge(t).Run(ctx, character.UseActionSurgeCommand{Character: fighter, Turn: character.FighterOtherTurn, PerTurn: character.FighterActionSurgeUnused})).(character.ActionSurgeNotAvailable)
	if rejected.Reason != character.FighterNotOwnTurn {
		t.Errorf("rejection = %v", rejected.Reason)
	}
}

// Requirement: doc/specs/class-fighter.md FR-052, FR-096 - the action cost and the additional action are enums with names; the zero value is unknown.
// Path: per case.
func TestFighterActionEnums(t *testing.T) {
	t.Parallel()
	if character.FighterBonusAction.String() != "bonus_action" || character.FighterActionCost(0).String() != "unknown" ||
		character.FighterAnyActionButMagic.String() != "any_action_but_magic" || character.FighterAdditionalAction(0).String() != "unknown" {
		t.Error("action enum names are wrong")
	}
	if character.FighterActionCost(5).String() != "unknown" || character.FighterAdditionalAction(-1).String() != "unknown" || character.FighterActionCost(-1).String() != "unknown" || character.FighterAdditionalAction(5).String() != "unknown" {
		t.Error("out-of-range action enum values must be unknown")
	}
}
