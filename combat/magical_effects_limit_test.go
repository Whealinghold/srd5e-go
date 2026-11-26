package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// mePendingN returns n distinct pending effect ids.
func mePendingN(n int) []combat.PendingEffectID {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("p%d", i)
	}
	return mePending(ids...)
}

// Requirement: doc/specs/magical-effects.md Limits (plan P10) - a teleport command holds at most 256 touched creatures.
// Path: unhappy (boundary).
func TestNewTeleportCommandTouchedCap(t *testing.T) {
	t.Parallel()
	runCap(t, "touched creatures", 256, func(n int) error {
		_, err := combat.NewTeleportCommand(combat.TeleportFields{Teleporter: gid("mage"), Destination: combat.Unoccupied,
			Sight: combat.DestinationSightNotRequired, Touched: capIDs(n)})
		return err
	})
}

// Requirement: doc/specs/magical-effects.md Limits (plan P10) - a creature holds at most 64 ongoing effects.
// Path: unhappy (boundary).
func TestNewOngoingEffectsCap(t *testing.T) {
	t.Parallel()
	runCap(t, "ongoing effects", 64, func(n int) error {
		es := make([]combat.OngoingEffect, n)
		for i := range es {
			es[i] = meCurse(fmt.Sprintf("c%d", i))
		}
		_, err := combat.NewOngoingEffects(es)
		return err
	})
}

// Requirement: doc/specs/magical-effects.md Limits (plan P10) - a line list of location facts holds at most 256 lines.
// Path: unhappy (boundary).
func TestNewOtherLocationFactsLineCap(t *testing.T) {
	t.Parallel()
	runCap(t, "area lines", 256, func(n int) error {
		_, err := combat.NewOtherLocationFacts(combat.InsideShape, make([]combat.Cover, n))
		return err
	})
}

// Requirement: doc/specs/magical-effects.md Limits (plan P10) - SimultaneousEffects.Order takes at most 256 pending and 256 chosen ids; one more of either is Rejected with the one problem OrderProblemLimitExceeded and the lists are not altered.
// Path: per case.
func TestOrderListCaps(t *testing.T) {
	t.Parallel()
	se := combat.SimultaneousEffects{}
	for _, c := range []struct {
		name            string
		pending, chosen int
		over            bool
	}{
		{"256 pending and chosen", 256, 256, false},
		{"257 pending", 257, 0, true},
		{"257 chosen", 2, 257, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pending, chosen := mePendingN(c.pending), mePendingN(c.chosen)
			got := se.Order(combat.TurnHolderPlayer, pending, chosen)
			rej, isRej := got.(combat.OrderSimultaneousEffectsRejected)
			if !c.over {
				if _, ok := got.(combat.OrderSimultaneousEffectsOrdered); !ok {
					t.Errorf("result = %#v, want Ordered", got)
				}
				return
			}
			if p := rej.Problems(); !isRej || len(p) != 1 || p[0].Kind != combat.OrderProblemLimitExceeded {
				t.Errorf("result = %#v, want the one problem LimitExceeded", got)
			}
			if len(pending) != c.pending || len(chosen) != c.chosen {
				t.Error("the lists changed")
			}
		})
	}
}
