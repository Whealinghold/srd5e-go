package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/initiative-and-actions.md FR-119 - every turn command constructor rejects the zero tracker and a blank combatant id with an error matching srderr.ErrInvalidInput, even when every other input is valid.
// Path: unhappy (invalid input).
func TestTurnCommandsRejectABadTarget(t *testing.T) {
	t.Parallel()
	tracker := trkABC(t)
	builds := map[string]func(tr combat.TurnTracker, id combat.CombatantID) error{
		"take action": func(tr combat.TurnTracker, id combat.CombatantID) error {
			_, err := combat.NewTakeActionCommand(tr, id, trkAvail, combat.DashChoice{Speed: combat.SpeedNameWalk})
			return err
		},
		"take bonus action": func(tr combat.TurnTracker, id combat.CombatantID) error {
			_, err := combat.NewTakeBonusActionCommand(tr, id, trkAvail, combat.NotEntitledToBonusAction{})
			return err
		},
		"take reaction": func(tr combat.TurnTracker, id combat.CombatantID) error {
			_, err := combat.NewTakeReactionCommand(tr, id, trkAvail, combat.ReadiedActionReaction{Trigger: "go"})
			return err
		},
		"move": func(tr combat.TurnTracker, id combat.CombatantID) error {
			_, err := combat.NewMoveCommand(tr, id, []combat.MoveSegment{trkWalk(t, 5)})
			return err
		},
		"change posture": func(tr combat.TurnTracker, id combat.CombatantID) error {
			_, err := combat.NewChangePostureCommand(tr, id, combat.DropProneChange{})
			return err
		},
	}
	for name, build := range builds {
		if err := build(combat.TurnTracker{}, trkA); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s with the zero tracker: err = %v, want ErrInvalidInput", name, err)
		}
		if err := build(tracker, ""); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s with a blank id: err = %v, want ErrInvalidInput", name, err)
		}
		if err := build(tracker, trkA); err != nil {
			t.Errorf("%s with valid input: err = %v, want none", name, err)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-163 - a Ready that holds a spell is a valid command only for its own caster, whichever way the caster's id sorts against the actor's; a Ready that holds no spell is valid for any actor.
// Path: unhappy (boundary).
func TestReadyCommandChecksTheHeldSpellCaster(t *testing.T) {
	t.Parallel()
	ready := func(caster combat.CombatantID) combat.ReadyChoice {
		held := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: caster, Trigger: "door"}}
		action, err := combat.NewReadiedAction("door", combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}, held)
		if err != nil {
			t.Fatal(err)
		}
		return combat.ReadyChoice{Action: action}
	}
	tracker := trkABC(t)
	if _, err := combat.NewTakeActionCommand(tracker, trkA, trkAvail, ready(trkA)); err != nil {
		t.Errorf("a Ready holding the actor's own spell: err = %v, want none", err)
	}
	for _, c := range []struct{ caster, actor combat.CombatantID }{{trkB, trkA}, {trkA, trkB}} {
		if _, err := combat.NewTakeActionCommand(tracker, c.actor, trkAvail, ready(c.caster)); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("caster %s, actor %s: err = %v, want ErrInvalidInput", c.caster, c.actor, err)
		}
	}
	move, err := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 5)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := combat.NewTakeActionCommand(tracker, trkA, trkAvail, combat.ReadyChoice{Action: trkReadied(t, "t", move)}); err != nil {
		t.Errorf("a Ready holding no spell: err = %v, want none", err)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-119 - a valid Move command carries its tracker, combatant and segments, so the Move use case applies it; a stand-up that costs 0 feet is valid and a negative cost is not.
// Path: happy.
func TestMoveCommandRunsAndStandUpCostBoundary(t *testing.T) {
	t.Parallel()
	cmd, err := combat.NewMoveCommand(trkABC(t), trkA, []combat.MoveSegment{trkWalk(t, 5)})
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewMove().Run(context.Background(), cmd)
	if _, ok := res.(combat.MoveApplied); err != nil || !ok {
		t.Errorf("Move.Run = %#v, %v, want MoveApplied", res, err)
	}
	for cost, wantErr := range map[int]bool{-1: true, 0: false, 1: false} {
		_, err := combat.NewChangePostureCommand(trkABC(t), trkA, combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: cost}})
		if (err != nil) != wantErr {
			t.Errorf("stand-up cost %d: err = %v, want an error: %v", cost, err, wantErr)
		}
	}
}
