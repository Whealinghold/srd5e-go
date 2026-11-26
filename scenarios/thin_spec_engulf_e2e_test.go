package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-022, FR-043 (SRD 5.2.1, "Behir", p. 263) - a Behir that has swallowed one creature (capacity 1) cannot swallow a second grappled creature: the entry is rejected for no room and the held set is unchanged; a creature whose Dexterity save reaches DC 18 resists and is not contained.
// Path: unhappy (capacity boundary, failure).
func TestBehirCapacityAndResistedSwallow(t *testing.T) {
	t.Parallel()
	terms := containmentTerms(t, "behir")
	held := swallow(t, terms, "behir", "first", 3)
	enter := func(containments combat.Containments, victim string, face int) combat.EnterContainerResult {
		t.Helper()
		grapple := combat.MustGrapple(combat.MustCombatantID("behir"), combat.MustCombatantID(victim), combat.MustGraspingHand(0), 15, 5)
		cmd, err := combat.NewEnterContainerCommand(combat.EnterContainerInput{Containments: containments,
			Grapples: combat.MustGrapples([]combat.Grapple{grapple}), Container: combat.MustCombatantID("behir"), Occupant: combat.MustCombatantID(victim),
			OccupantSize: content.SizeMedium, Terms: terms, OccupantConditions: combat.MustConditionSet([]combat.Condition{grapple.Condition()}, nil),
			Circumstance: combat.EntryCircumstanceNone, Attempt: containmentRoll(0)})
		if err != nil {
			t.Fatal(err)
		}
		uc, err := combat.NewEnterContainer(dicetest.NewFixedRoller(face))
		if err != nil {
			t.Fatal(err)
		}
		res, err := uc.Run(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	full, ok := enter(held.Containments(), "second", 3).(combat.EnterContainerRejected)
	if !ok || full.Rejection != combat.ContainmentRejectionNoRoom || held.Containments().Len() != 1 {
		t.Errorf("a second swallow = %#v (held %d), want rejected, no room, and the held set unchanged", full, held.Containments().Len())
	}
	if _, ok := enter(combat.Containments{}, "third", 18).(combat.EnterContainerResisted); !ok { // 18 + 0 reaches DC 18
		t.Errorf("a save equal to the DC = %#v, want resisted", enter(combat.Containments{}, "third", 18))
	}
}
