package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-rogue.md FR-086 - StrokeOfLuckUsed carries the character id and is both a RogueEvent and an Event.
// Path: happy.
func TestStrokeOfLuckUsedIsAnEvent(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("rogue-1")
	res := character.StrokeOfLuck{}.Use(id, rl(20), character.InitialRogueState(), character.StrokeOfLuckRolledTest{Roll: d20.MustRoll(modifier.Normal, 7), Outcome: d20.Failure}).(character.StrokeOfLuckSpent)
	var ev character.RogueEvent = res.Event()
	var base character.Event = ev
	if base.CharacterID() != id || res.Event().CharacterID() != id {
		t.Errorf("event character = %v, want %v", base.CharacterID(), id)
	}
}
