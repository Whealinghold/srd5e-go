package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-barbarian.md FR-107, FR-111, FR-120 - every Barbarian event is a BarbarianEvent and an Event (not a RecordEvent), carries the Identifier of the character, and is produced by the function or use case of its
// feature: RageStarted, RageEnded, RelentlessRageUsed, RagesRegained, IntimidatingPresenceUsed, BarbarianWeaponMasteriesChosen and BarbarianWeaponMasterySwapped; a refusal produces none.
// Path: per case.
func TestBarbarianEventsAreBarbarianEventsOfTheCharacter(t *testing.T) {
	t.Parallel()
	stats, id := brbStats(t, 15), brbID(t)
	var events []character.Event
	entered := must(character.BarbarianRage{}.Enter(id, stats, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, "", 30)).(character.RageEntered)
	ended := must(character.BarbarianRage{}.End(id, stats, entered.State(), character.RageHeavyArmorDonned{})).(character.RageEndApplied)
	regained := must(character.PersistentRage{}.RegainOnInitiative(id, stats, entered.State())).(character.RagesRegainedOnInitiative)
	presence := must(character.BerserkerRules{}.UseIntimidatingPresence(id, stats, character.BerserkerSubclassID, entered.State(), character.NewBarbarianTurn(), character.BarbarianHostFacts{},
		brbMod(t, 10), character.ProficiencyBonusFor(character.MustLevel(15)))).(character.IntimidatingPresenceApplied)
	relentless := must(useRelentlessRage(t, dicetest.NewFixedRoller(15)).Run(context.Background(), character.UseRelentlessRageCommand{
		Character: brbRagingBarbarian(t, 15, 0), Facts: dropped, Modifiers: d20.NewRollModifiers(modifier.New(0))})).(character.RelentlessRageSucceeded)
	src := brbSource(t)
	chosen := brbApply(t, src, brbBarbarian(t, 1), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe")}).(character.BarbarianOptionApplied)
	swapped := brbApply(t, src, chosen.Character(), character.BarbarianSwapWeaponMastery{Remove: "handaxe", Add: "mace", RestApplied: fgtRest(t, chosen.Character().ID(), character.LongRest)}).(character.BarbarianOptionApplied)
	owners := []character.Identifier{id, id, id, id, relentless.Character().ID(), chosen.Character().ID(), swapped.Character().ID()}
	for _, evs := range [][]character.Event{entered.Events(), ended.Events(), regained.Events(), presence.Events(), relentless.Events(), chosen.Events(), swapped.Events()} {
		events = append(events, evs...)
	}
	if len(events) != len(owners) {
		t.Fatalf("%d events, want %d", len(events), len(owners))
	}
	for i, e := range events {
		be, ok := e.(character.BarbarianEvent)
		_, record := e.(character.RecordEvent)
		if !ok || record || e.CharacterID() != owners[i] || be.CharacterID() != owners[i] {
			t.Errorf("event %d = %T for %v, want a BarbarianEvent of %v", i, e, e.CharacterID(), owners[i])
		}
	}
	rejected := must(character.BarbarianRage{}.Enter(id, stats, entered.State(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, "", 30)).(character.RageNotEntered)
	if rejected.Reason != character.BarbarianRageAlreadyActive {
		t.Errorf("rejection = %v", rejected.Reason)
	}
}
