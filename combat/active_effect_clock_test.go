package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md section 4.4 (ClockTickFromTurnEvent) - TrackerTurnStarted gives TurnStartReached, TrackerTurnEnded gives TurnEndReached, TrackerRoundStarted gives RoundStartReached, and any other tracker event gives false.
// Path: per case.
func TestClockTickFromTurnEvent(t *testing.T) {
	t.Parallel()
	tr := trkABC(t)
	_, ev := trkAdvance(t, tr)
	end, ok := combat.ClockTickFromTurnEvent(ev[0])
	if !ok || end != aeEndTick(t, string(trkA)) {
		t.Errorf("TrackerTurnEnded = %#v, %v, want TurnEndReached(A)", end, ok)
	}
	start, ok := combat.ClockTickFromTurnEvent(ev[1])
	if !ok || start != aeStartTick(t, string(trkB)) {
		t.Errorf("TrackerTurnStarted = %#v, %v, want TurnStartReached(B)", start, ok)
	}
	solo := trkTracker(t, trkProfile(t, trkA, 3, trkWalkFly(t)))
	_, wrap := trkAdvance(t, solo)
	round := false
	for _, e := range wrap {
		if tick, ok := combat.ClockTickFromTurnEvent(e); ok && tick == (combat.RoundStartReached{}) {
			round = true
		}
	}
	if !round {
		t.Errorf("a wrap %v gave no RoundStartReached", trkKinds(wrap))
	}

	_, other := trkApplied(t)(tr.TakeAction(trkA, trkAvail, combat.SimpleActionChoice{Action: combat.SimpleStandardActionUtilize}))
	for _, e := range other {
		if tick, ok := combat.ClockTickFromTurnEvent(e); ok {
			t.Errorf("%s mapped to %#v", e.Kind(), tick)
		}
	}
	if tick, ok := combat.ClockTickFromTurnEvent(nil); ok || tick != nil {
		t.Errorf("a nil event gave %v, %v", tick, ok)
	}
}

// Requirement: doc/specs/active-effects.md FR-020, FR-062 - a turn tick, a departure and an elapsed time need a combatant or rounds.
// Path: unhappy (invalid input).
func TestClockTickConstructors(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewTurnStartReached(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewTurnStartReached(zero) err = %v", err)
	}
	if _, err := combat.NewTurnEndReached(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewTurnEndReached(zero) err = %v", err)
	}
	if aeStartTick(t, "a") == aeEndTick(t, "a") {
		t.Error("a start tick equals an end tick")
	}
}
