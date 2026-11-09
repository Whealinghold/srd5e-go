package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-warlock.md FR-069 to FR-072, FR-151 - Gaze of Two Minds begins a link only on a willing creature the Warlock touches with the Bonus Action (GazeLinkBegun, an active link with the creature); with an active link the same Bonus Action on the same plane maintains it (GazeLinkMaintained, the same creature); a different plane ends the maintaining (notOnSamePlane), a lapsed link is linkEnded, and the checks run in the order invocation, link state, plane, willingness, contact, Bonus Action; nothing changes on a refusal.
// Path: per case (happy; unhappy: state error).
func TestUseGazeOfTwoMinds(t *testing.T) {
	t.Parallel()
	knows := warKnows("gaze_of_two_minds")
	target := character.GazeTargetFacts{Creature: warCreature, Willingness: character.GazeWilling, Contact: character.GazeTouching}
	active := warPact(character.WarlockPactStateParts{Link: warCreature, LinkStatus: character.GazeLinkActive})
	lapsed := warPact(character.WarlockPactStateParts{Link: warCreature, LinkStatus: character.GazeLinkLapsed})
	spent := character.WarlockTurnFacts{BonusAction: character.ActionSpent}
	begin := warPactChanged(t, character.UseGazeOfTwoMinds(warID, knows, character.WarlockPactState{}, target, 0, character.WarlockTurnFacts{}))
	if ev, ok := begin.Events()[0].(character.GazeLinkBegun); !ok || ev.Creature() != warCreature || ev.CharacterID() != warID || len(begin.Events()) != 1 ||
		begin.State().Link() != character.GazeLinkOf(warCreature, character.GazeLinkActive) {
		t.Errorf("begin: events %v, link %+v", begin.Events(), begin.State().Link())
	}
	keep := warPactChanged(t, character.UseGazeOfTwoMinds(warID, knows, active, character.GazeTargetFacts{}, character.GazeSamePlane, character.WarlockTurnFacts{}))
	if ev, ok := keep.Events()[0].(character.GazeLinkMaintained); !ok || ev.Creature() != warCreature || ev.CharacterID() != warID || len(keep.Events()) != 1 || !keep.State().Link().Active() {
		t.Errorf("maintain: events %v, link %+v", keep.Events(), keep.State().Link())
	}
	for _, c := range []struct {
		name   string
		s      character.WarlockState
		pact   character.WarlockPactState
		target character.GazeTargetFacts
		plane  character.GazePlaneRelation
		turn   character.WarlockTurnFacts
		want   character.WarlockRejectionKind
	}{
		{"not known", warKnows("pact_of_the_tome"), character.WarlockPactState{}, target, 0, character.WarlockTurnFacts{}, character.WarlockInvocationNotKnown},
		{"unwilling", knows, character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Willingness: character.GazeUnwilling, Contact: character.GazeTouching}, 0, character.WarlockTurnFacts{}, character.WarlockNotWilling},
		{"willingness unset", knows, character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Contact: character.GazeTouching}, 0, character.WarlockTurnFacts{}, character.WarlockNotWilling},
		{"not touched", knows, character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Willingness: character.GazeWilling, Contact: character.GazeNotTouching}, 0, character.WarlockTurnFacts{}, character.WarlockNotTouched},
		{"contact unset", knows, character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Willingness: character.GazeWilling}, 0, character.WarlockTurnFacts{}, character.WarlockNotTouched},
		{"begin without Bonus Action", knows, character.WarlockPactState{}, target, 0, spent, character.WarlockNoBonusAction},
		{"maintain without Bonus Action", knows, active, character.GazeTargetFacts{}, character.GazeSamePlane, spent, character.WarlockNoBonusAction},
		{"other plane", knows, active, character.GazeTargetFacts{}, character.GazeDifferentPlane, character.WarlockTurnFacts{}, character.WarlockNotOnSamePlane},
		{"plane unset", knows, active, character.GazeTargetFacts{}, 0, character.WarlockTurnFacts{}, character.WarlockNotOnSamePlane},
		{"lapsed", knows, lapsed, target, character.GazeSamePlane, character.WarlockTurnFacts{}, character.WarlockLinkEnded},
	} {
		if got := warPactRefused(t, character.UseGazeOfTwoMinds(warID, c.s, c.pact, c.target, c.plane, c.turn)); got.Kind != c.want {
			t.Errorf("%s: %v, want %v", c.name, got.Kind, c.want)
		}
	}
	// A new link replaces nothing when begun after a lapse reported as the zero link.
	if got := warPactChanged(t, character.UseGazeOfTwoMinds(warID, knows, character.WarlockPactState{}, target, character.GazeDifferentPlane, character.WarlockTurnFacts{})); !got.State().Link().Active() {
		t.Error("the plane is not checked when a link begins")
	}
}

// Requirement: doc/specs/class-warlock.md FR-069 to FR-072 - an out-of-range willingness, contact or plane value is invalid and read as not willing, not touching and a different plane, never as the accepting value.
// Path: per case (unhappy: state error).
func TestUseGazeOfTwoMindsInvalidFacts(t *testing.T) {
	t.Parallel()
	knows := warKnows("gaze_of_two_minds")
	active := warPact(character.WarlockPactStateParts{Link: warCreature, LinkStatus: character.GazeLinkActive})
	for _, c := range []struct {
		name   string
		pact   character.WarlockPactState
		target character.GazeTargetFacts
		plane  character.GazePlaneRelation
		want   character.WarlockRejectionKind
	}{
		{"willingness below range", character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Willingness: -1, Contact: character.GazeTouching}, 0, character.WarlockNotWilling},
		{"willingness above range", character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Willingness: 3, Contact: character.GazeTouching}, 0, character.WarlockNotWilling},
		{"contact below range", character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Willingness: character.GazeWilling, Contact: -1}, 0, character.WarlockNotTouched},
		{"contact above range", character.WarlockPactState{}, character.GazeTargetFacts{Creature: warCreature, Willingness: character.GazeWilling, Contact: 3}, 0, character.WarlockNotTouched},
		{"plane below range", active, character.GazeTargetFacts{}, -1, character.WarlockNotOnSamePlane},
		{"plane above range", active, character.GazeTargetFacts{}, 3, character.WarlockNotOnSamePlane},
	} {
		if got := warPactRefused(t, character.UseGazeOfTwoMinds(warID, knows, c.pact, c.target, c.plane, character.WarlockTurnFacts{})); got.Kind != c.want {
			t.Errorf("%s: %v, want %v", c.name, got.Kind, c.want)
		}
	}
}
