package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Synthetic combatants A, B and C act in that order in every turn fixture.
var (
	trkA = combat.MustCombatantID("a")
	trkB = combat.MustCombatantID("b")
	trkC = combat.MustCombatantID("c")
)

const (
	trkAvail = combat.ConditionActionAvailabilityAvailable
	trkNone  = combat.ConditionActionAvailabilityNone
)

func trkSpeeds(tb testing.TB, walk int, specials ...combat.MovementSpeedEntry) combat.MovementSpeeds {
	tb.Helper()
	s, err := combat.NewMovementSpeeds(walk, specials...)
	if err != nil {
		tb.Fatalf("NewMovementSpeeds: %v", err)
	}
	return s
}

// trkWalkFly is walk 30, fly 40.
func trkWalkFly(tb testing.TB) combat.MovementSpeeds {
	tb.Helper()
	return trkSpeeds(tb, 30, combat.MovementSpeedEntry{Name: combat.SpeedNameFly, Feet: 40})
}

func trkProfile(tb testing.TB, id combat.CombatantID, size content.CreatureSize, s combat.MovementSpeeds) combat.CombatantProfile {
	tb.Helper()
	return combat.CombatantProfile{ID: id, Size: size, Speeds: s}
}

// trkTracker establishes a tracker over the profiles in the order listed.
func trkTracker(tb testing.TB, profiles ...combat.CombatantProfile) combat.TurnTracker {
	tb.Helper()
	standings := make([]combat.InitiativeStanding, len(profiles))
	for i, p := range profiles {
		standings[i] = combat.InitiativeStanding{Combatant: p.ID, Count: combat.InitiativeCount(100 - 10*i)}
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		tb.Fatalf("NewEstablishInitiativeOrderCommand: %v", err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	est, ok := res.(combat.InitiativeOrderEstablished)
	if !ok {
		tb.Fatalf("result = %#v, want InitiativeOrderEstablished", res)
	}
	return est.Tracker()
}

// trkABC is a tracker of A, B and C, Medium, walk 30 and fly 40, A active in round 1.
func trkABC(tb testing.TB) combat.TurnTracker {
	tb.Helper()
	p := func(id combat.CombatantID) combat.CombatantProfile {
		return trkProfile(tb, id, content.SizeMedium, trkWalkFly(tb))
	}
	return trkTracker(tb, p(trkA), p(trkB), p(trkC))
}

// trkApplied returns a function that takes the two results of a tracker operation and returns
// the tracker and events of an applied step, failing the test otherwise: trkApplied(tb)(t.Move(...)).
func trkApplied(tb testing.TB) func(combat.TrackerStep, error) (combat.TurnTracker, []combat.TurnEvent) {
	tb.Helper()
	return func(s combat.TrackerStep, err error) (combat.TurnTracker, []combat.TurnEvent) {
		tb.Helper()
		if err != nil {
			tb.Fatalf("operation error: %v", err)
		}
		a, ok := s.(combat.TrackerStepApplied)
		if !ok {
			tb.Fatalf("step = %#v, want TrackerStepApplied", s)
		}
		return a.Tracker(), a.Events()
	}
}

// trkReason is trkApplied for a rejected step: it returns the reason.
func trkReason(tb testing.TB) func(combat.TrackerStep, error) combat.TurnRejection {
	tb.Helper()
	return func(s combat.TrackerStep, err error) combat.TurnRejection {
		tb.Helper()
		if err != nil {
			tb.Fatalf("operation error: %v", err)
		}
		r, ok := s.(combat.TrackerStepRejected)
		if !ok {
			tb.Fatalf("step = %#v, want TrackerStepRejected", s)
		}
		return r.Reason
	}
}

// trkKinds lists the Kind of each event.
func trkKinds(events []combat.TurnEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

func trkSegment(tb testing.TB, in combat.MoveSegmentInput) combat.MoveSegment {
	tb.Helper()
	s, err := combat.NewMoveSegment(in)
	if err != nil {
		tb.Fatalf("NewMoveSegment: %v", err)
	}
	return s
}

// trkWalk is an open-ground walking segment of feet.
func trkWalk(tb testing.TB, feet int) combat.MoveSegment {
	tb.Helper()
	return trkSegment(tb, combat.MoveSegmentInput{DistanceFeet: feet, Mode: combat.MovementModeWalk})
}

func trkMode(tb testing.TB, feet int, mode combat.MovementMode) combat.MoveSegment {
	tb.Helper()
	return trkSegment(tb, combat.MoveSegmentInput{DistanceFeet: feet, Mode: mode})
}

// trkTake takes the choice for id as the action, applying it to t.
func trkTake(tb testing.TB, t combat.TurnTracker, id combat.CombatantID, c combat.ActionChoice) combat.TurnTracker {
	tb.Helper()
	n, _ := trkApplied(tb)(t.TakeAction(id, trkAvail, c))
	return n
}

// trkMove moves id along the segments, applying it to t.
func trkMove(tb testing.TB, t combat.TurnTracker, id combat.CombatantID, segs ...combat.MoveSegment) combat.TurnTracker {
	tb.Helper()
	n, _ := trkApplied(tb)(t.Move(id, segs))
	return n
}

func contentSpell(s string) content.SpellID { return content.SpellID(s) }
