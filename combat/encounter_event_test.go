package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: doc/specs/encounter-aggregate.md FR-050, FR-051 - every event the encounter produces is an EncounterEvent carrying the encounter id and the round, with a constant snake_case kind and a fresh, duplicate-free combatants list; a wrapper reports the wrapped kind and combatants; kinds of the three own events are unique.
// Path: unhappy (immutability).
func TestEncounterEventsHonourTheEventContract(t *testing.T) {
	t.Parallel()
	e, started := encStart(t, encC(t, "a", combat.SideA, dyingStatus(t, 20, 0, 0), combat.Unconscious{}), encPC(t, "b", combat.SideB, 20, 20))
	combattest.CheckEvent(t, started[0], "encounter_started", cid("a"), cid("b"))
	combattest.CheckEvent(t, started[1], "tracker_round_started")
	combattest.CheckEvent(t, started[2], "tracker_turn_started", cid("a"))
	combattest.CheckEvent(t, started[3], "dying_combatant_turn_started", cid("a"))
	tr, err := e.ApplyDamage(cid("b"), encSlash(20), nil, combat.OtherSource, combat.NormalHit)
	_, hit := encOK(t, tr, err)
	combattest.CheckEvent(t, hit[0], "damage_taken", cid("b"))
	combattest.CheckEvent(t, hit[4], "condition_applied", cid("b"))
	g := grappleEnc(t, nil)
	rel, err := g.ReleaseGrapple(cid("b"))
	_, ge := encOK(t, rel, err)
	combattest.CheckEvent(t, ge[1], "grapple_ended", cid("b"), cid("a"))
	end := runEnd(t, e, combat.EndReasonMutualAgreement{}).(combat.EndEncounterApplied)
	combattest.CheckEvent(t, end.Events()[0], "encounter_ended", cid("a"), cid("b"))
	combattest.CheckEventsCopy(t, func() []combat.Event { return end.Events() })
	own := []combat.Event{started[0], started[3], end.Events()[0]}
	combattest.CheckEventKinds(t, own)
	for _, ev := range slices.Concat(started, hit, ge, end.Events()) {
		ee, ok := ev.(combat.EncounterEvent)
		if !ok || ee.EncounterID() != eid() || ee.Round() < 1 {
			t.Errorf("event %T is not an EncounterEvent with an id and a round [FR-051]", ev)
		}
	}
	// A returned event slice is a copy.
	tr, err = e.ApplyHealing(cid("b"), 0)
	if _, evs := encOK(t, tr, err); len(evs) != 0 {
		t.Errorf("no events expected")
	}
	adv, err := e.AdvanceTurn(combat.TurnAdvance{})
	encOK(t, adv, err)
	combattest.CheckEventsCopy(t, func() []combat.Event { return adv.Events() })
	if r := adv.Resolutions(); r != nil {
		t.Errorf("resolutions of an advance = %v, want none", r)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-051 - the round of an event is the new round for TrackerRoundStarted, TrackerTurnStarted and what follows at a wrap, and the old round for the preceding TrackerTurnEnded; a Dying event at the wrap carries the new round.
// Path: unhappy (boundary).
func TestRoundsOfEventsAtAWrap(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "b", combat.SideB, dyingStatus(t, 20, 0, 0), combat.Unconscious{}))
	e, _ = advance(t, e)
	_, events := advance(t, e)
	var rounds []int
	for _, ev := range events {
		rounds = append(rounds, ev.(combat.EncounterEvent).Round())
	}
	if want := []int{1, 2, 2}; !slices.Equal(rounds, want) {
		t.Errorf("rounds = %v for %v, want %v", rounds, encKinds(events), want)
	}
}
