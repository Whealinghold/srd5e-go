package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// capIDs returns n distinct combatant ids.
func capIDs(n int) []combat.CombatantID {
	ids := make([]combat.CombatantID, n)
	for i := range ids {
		ids[i] = gid(fmt.Sprintf("c%d", i))
	}
	return ids
}

// Requirement: doc/specs/initiative-and-actions.md Limits (plan P10) - a roll-initiative command holds at most 256 participants.
// Path: unhappy (boundary).
func TestNewRollInitiativeCommandCap(t *testing.T) {
	t.Parallel()
	runCap(t, "initiative participants", 256, func(n int) error {
		ps := make([]combat.InitiativeParticipant, n)
		for i, id := range capIDs(n) {
			ps[i] = trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{id}, Modifier: modifier.New(0)})
		}
		_, err := combat.NewRollInitiativeCommand(combat.InitiativeMethodRolled, ps)
		return err
	})
}

// Requirement: doc/specs/initiative-and-actions.md Limits (plan P10) - an establish-order command holds at most 256 standings, 256 profiles and 256 tie-breaks, each checked before an element is read.
// Path: per case.
func TestNewEstablishInitiativeOrderCommandCaps(t *testing.T) {
	t.Parallel()
	build := func(n int) ([]combat.InitiativeStanding, []combat.CombatantProfile) {
		var ss []combat.InitiativeStanding
		var ps []combat.CombatantProfile
		for i, id := range capIDs(n) {
			ss = append(ss, combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(i)})
			ps = append(ps, trkProfile(t, id, content.SizeMedium, trkSpeeds(t, 30)))
		}
		return ss, ps
	}
	t.Run("standings and profiles", func(t *testing.T) {
		t.Parallel()
		runCap(t, "standings", 256, func(n int) error {
			ss, ps := build(n)
			_, err := combat.NewEstablishInitiativeOrderCommand(ss, ps, nil)
			return err
		})
	})
	t.Run("profiles over the cap alone", func(t *testing.T) {
		t.Parallel()
		ss, ps := build(257)
		_, err := combat.NewEstablishInitiativeOrderCommand(ss[:1], ps, nil)
		wantLimit(t, err, "profiles", 256)
	})
	t.Run("tie-breaks", func(t *testing.T) {
		t.Parallel()
		runCap(t, "tie-breaks", 256, func(n int) error {
			ss, ps := build(2)
			tb, err := combat.NewTieBreak(capIDs(2))
			if err != nil {
				t.Fatal(err)
			}
			ties := make([]combat.TieBreak, n)
			for i := range ties {
				ties[i] = tb
			}
			_, err = combat.NewEstablishInitiativeOrderCommand(ss, ps, ties)
			return err
		})
	})
}

// Requirement: doc/specs/initiative-and-actions.md Limits (plan P10) - an advance holds at most 256 speed updates and 256 shared-space occupants, in the command and in TurnTracker.Advance, and the tracker is not changed by a rejected call.
// Path: per case.
func TestAdvanceCaps(t *testing.T) {
	t.Parallel()
	tracker := trkABC(t)
	updates := func(n int) []combat.SpeedUpdate {
		out := make([]combat.SpeedUpdate, n)
		for i, id := range capIDs(n) {
			out[i] = combat.SpeedUpdate{Combatant: id, Speeds: trkSpeeds(t, 30)}
		}
		return out
	}
	occupants := func(n int) []content.CreatureSize {
		out := make([]content.CreatureSize, n)
		for i := range out {
			out[i] = content.SizeSmall
		}
		return out
	}
	for _, c := range []struct {
		name, collection string
		advance          func(n int) combat.TurnAdvance
	}{
		{"speed updates", "speed updates", func(n int) combat.TurnAdvance { return combat.TurnAdvance{Updates: updates(n)} }},
		{"shared-space occupants", "shared-space occupants", func(n int) combat.TurnAdvance {
			return combat.TurnAdvance{SharedSpaceOccupants: occupants(n)}
		}},
	} {
		t.Run(c.name+" command", func(t *testing.T) {
			t.Parallel()
			runCap(t, c.collection, 256, func(n int) error {
				_, err := combat.NewAdvanceTrackerTurnCommand(tracker, c.advance(n))
				return err
			})
		})
		t.Run(c.name+" operation", func(t *testing.T) {
			t.Parallel()
			runCap(t, c.collection, 256, func(n int) error {
				_, err := tracker.Advance(c.advance(n))
				return err
			})
			if !tracker.Equal(trkABC(t)) {
				t.Error("the tracker changed")
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md Limits (plan P10) - a move, a move command and a readied move response hold at most 256 segments.
// Path: per case.
func TestMoveSegmentCaps(t *testing.T) {
	t.Parallel()
	segs := func(n int) []combat.MoveSegment {
		out := make([]combat.MoveSegment, n)
		for i := range out {
			out[i] = trkWalk(t, 1)
		}
		return out
	}
	tracker := trkABC(t)
	t.Run("tracker move", func(t *testing.T) {
		t.Parallel()
		runCap(t, "move segments", 256, func(n int) error {
			_, err := tracker.Move(trkA, segs(n))
			return err
		})
	})
	t.Run("move command", func(t *testing.T) {
		t.Parallel()
		runCap(t, "move segments", 256, func(n int) error {
			_, err := combat.NewMoveCommand(tracker, trkA, segs(n))
			return err
		})
	})
	t.Run("readied move response", func(t *testing.T) {
		t.Parallel()
		runCap(t, "move segments", 256, func(n int) error {
			_, err := combat.NewReadiedMoveResponse(segs(n))
			return err
		})
	})
}
