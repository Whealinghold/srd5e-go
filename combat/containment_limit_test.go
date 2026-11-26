package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// ctApart is a set of n Small creatures o0, o1, ..., each held by a container c0, c1, ... of its own, with the
// terms of the frog, so no capacity is shared.
func ctApart(tb testing.TB, n int) []combat.Containment {
	tb.Helper()
	terms := ctTerms(tb, "frog")
	rs := make([]combat.Containment, n)
	for i := range rs {
		rs[i] = ctRecord(tb, fmt.Sprintf("c%d", i), fmt.Sprintf("o%d", i), content.SizeSmall, terms, 1)
	}
	return rs
}

// ctConditionsOf is a map of n zero condition sets keyed o0, o1, ....
func ctConditionsOf(n int) map[combat.CombatantID]combat.ConditionSet {
	m := make(map[combat.CombatantID]combat.ConditionSet, n)
	for i := range n {
		m[gid(fmt.Sprintf("o%d", i))] = combat.ConditionSet{}
	}
	return m
}

// Requirement: doc/specs/monster-engulf-and-swallow.md Limits (plan P10) - a Containments set holds at most 256 records, so a container holds at most 256 occupants.
// Path: unhappy (boundary).
func TestNewContainmentsCap(t *testing.T) {
	t.Parallel()
	runCap(t, "containments", 256, func(n int) error {
		_, err := combat.NewContainments(ctApart(t, n))
		return err
	})
}

// Requirement: doc/specs/monster-engulf-and-swallow.md Limits (plan P10) - the commands that carry OccupantConditions (advance, release, regurgitation) read at most 256 of them, and AdvanceContainerTurn at most 256 Survivors.
// Path: per case.
func TestContainmentCommandMapAndListCaps(t *testing.T) {
	t.Parallel()
	t.Run("advance, occupant conditions", func(t *testing.T) {
		t.Parallel()
		runCap(t, "occupant conditions", 256, func(n int) error {
			_, err := combat.NewAdvanceContainerTurnCommand(combat.AdvanceContainerTurnInput{Container: gid("m"),
				Boundary: combat.ContainerTurnBoundaryStart, OccupantConditions: ctConditionsOf(n)})
			return err
		})
	})
	t.Run("advance, survivors", func(t *testing.T) {
		t.Parallel()
		runCap(t, "survivors", 256, func(n int) error {
			ids := make([]combat.CombatantID, n)
			for i := range ids {
				ids[i] = gid(fmt.Sprintf("o%d", i))
			}
			_, err := combat.NewAdvanceContainerTurnCommand(combat.AdvanceContainerTurnInput{Container: gid("m"),
				Boundary: combat.ContainerTurnBoundaryEnd, Survivors: ids})
			return err
		})
	})
	t.Run("release, occupant conditions", func(t *testing.T) {
		t.Parallel()
		runCap(t, "occupant conditions", 256, func(n int) error {
			_, err := combat.NewReleaseOccupantsCommand(combat.ReleaseOccupantsInput{OccupantConditions: ctConditionsOf(n), Trigger: combat.GrappleReview{}})
			return err
		})
	})
	t.Run("regurgitation, occupant conditions", func(t *testing.T) {
		t.Parallel()
		runCap(t, "occupant conditions", 256, func(n int) error {
			_, err := combat.NewResolveRegurgitationCommand(combat.ResolveRegurgitationInput{Container: gid("m"),
				OccupantConditions: ctConditionsOf(n), Attempt: d20.AutomaticFailure{}})
			return err
		})
	})
}

// Requirement: doc/specs/monster-engulf-and-swallow.md Limits (plan P10) - an entry into a set that already holds 256 records is EnterContainerInvalid with ContainmentProblemLimitExceeded (the shared limit_exceeded value) before any die; with 255 the creature is held.
// Path: unhappy (boundary).
func TestEnterContainerCap(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		held int
		over bool
	}{{"255 held", 255, false}, {"256 held", 256, true}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := ctEnterInput(t, "frog")
			in.OccupantSize, in.Attempt = content.SizeSmall, nil
			in.Containments = ctSet(t, ctApart(t, c.held)...)
			res, roller := ctEnter(t, in)
			inv, isInvalid := res.(combat.EnterContainerInvalid)
			if isInvalid != c.over || len(roller.Requests()) != 0 {
				t.Fatalf("result = %#v, dice %d, want invalid = %v", res, len(roller.Requests()), c.over)
			}
			if c.over && (inv.Problem != combat.ContainmentProblemLimitExceeded || inv.Problem.String() != "limit_exceeded") {
				t.Errorf("problem = %v, want limit_exceeded", inv.Problem)
			}
			if !c.over {
				ctContained(t, res)
			}
		})
	}
}
