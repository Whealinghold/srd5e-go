package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/class-monk.md section 5 (events), FR-146 - every Monk event is a MonkEvent and an Event naming the Character it concerns; the family has the eleven
// members of the spec, built by the services and use cases that own them.
// Path: per case.
func TestMonkEventsAreSealedAndNameTheCharacter(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("pc-1")
	events := []character.MonkEvent{
		must(character.PerfectFocus{}.Resolve(mnkStats(t, 15), mnkState(15, false, 0), true, false)).Event(id),
		character.DisciplinedSurvivor{}.Event(id), character.SuperiorDefense{}.Event(id),
		must(character.StunningStrike{}.Resolve(mnkStats(t, 5), mnkState(0, false, 0), mnkAbilities(t, 3, 3, 5), monkHit, d20.Failure)).Event(id),
		must(character.QuiveringPalm{}.Start(mnkStats(t, 17), mnkState(0, false, 0), openHand, palmStart(true))).Event(id),
		must(character.QuiveringPalm{}.Finish(mnkStats(t, 17), character.MustMonkState(0, false, 0, mnkVibrations(3)), mnkAbilities(t, 3, 3, 17), character.QuiveringPalmEndHarmlessly, true, 0, 0)).Event(id),
	}
	for _, e := range events {
		var asEvent character.Event = e
		if asEvent.CharacterID() != id {
			t.Errorf("%T names %v, want %v", e, asEvent.CharacterID(), id)
		}
	}
}
