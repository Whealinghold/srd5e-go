package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/class-wizard-evoker.md section 5 (Event), class-wizard.md FR-091 - OverchannelUsed is a past-tense WizardEvent and Event outside the aggregate's closed family; it carries the CharacterID of the command,
// the use number and the slot level.
// Path: happy.
func TestOverchannelUsedEvent(t *testing.T) {
	t.Parallel()
	res, err := must(character.NewUseOverchannel(dicetest.NewMaximumRoller())).Run(t.Context(), character.UseOverchannelCommand{CharacterID: wzID, ClassLevels: wzLevels(14), Subclass: evkSub(t),
		Resources: evkResources(t, 1), Request: character.OverchannelRequest{Spell: wzFact("s", 3, evkDmg), Source: character.CastSourceSlot, SlotLevel: 3}})
	if err != nil {
		t.Fatal(err)
	}
	events := res.(character.OverchannelApplied).Events()
	if len(events) != 1 {
		t.Fatalf("%d events, want 1", len(events))
	}
	ev, ok := events[0].(character.OverchannelUsed)
	if _, wiz := events[0].(character.WizardEvent); !ok || !wiz || ev.CharacterID() != wzID || ev.Use() != 2 || ev.SlotLevel() != 3 {
		t.Errorf("event = %#v", events[0])
	}
	if _, record := events[0].(character.RecordEvent); record {
		t.Error("OverchannelUsed is in the aggregate's closed family")
	}
}
