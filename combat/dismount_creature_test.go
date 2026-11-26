package combat_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestDismountCreatureRules(t *testing.T) {
	t.Parallel()
	spec := "doc/specs/mounted-and-underwater-combat.md "
	mounted, none := mntMounted(t, combat.MountControlControlled), combat.NotMounted()
	active, idle := combat.RiderTurnActive, combat.RiderTurnNotActive
	cases := []struct {
		name, requirement, path string
		status                  combat.RiderStatus
		turn                    combat.MountingTurnFacts
		cost                    int
		refusal                 combat.DismountRejection
	}{
		{"dismount, same cost", spec + "FR-016 table 4.2a", "happy", mounted, mntTurn(t, active, 30, 30), 15, nil},
		{"odd Speed rounds down", spec + "FR-010 table 4.2a", "unhappy (boundary)", mounted, mntTurn(t, active, 25, 25), 12, nil},
		{"Speed 1 costs nothing", spec + "FR-010 table 4.2a", "unhappy (boundary)", mounted, mntTurn(t, active, 1, 1), 0, nil},
		{"after Dash, half of Speed only", spec + "FR-013 table 4.2a", "unhappy (rule interaction)", mounted, mntTurn(t, active, 30, 60), 15, nil},
		{"dismount, one short", spec + "FR-012 table 4.2a", "unhappy (boundary)", mounted, mntTurn(t, active, 30, 14), 15, combat.MountingInsufficientMovementRejection{}},
		{"nothing left", spec + "FR-012 table 4.2a", "unhappy (boundary)", mounted, mntTurn(t, active, 30, 0), 15, combat.MountingNoMovementRemainingRejection{}},
		{"rider not active, dismount", spec + "FR-009 table 4.2a", "unhappy (state error)", mounted, mntTurn(t, idle, 30, 30), 0, combat.MountingRiderNotActiveRejection{}},
		{"not mounted", spec + "FR-015", "unhappy (state error)", none, mntTurn(t, active, 30, 30), 0, combat.MountingRiderNotMountedRejection{}},
		{"not active beats not mounted", spec + "FR-009 FR-017 table 4.2b", "unhappy (rule interaction)", none, mntTurn(t, idle, 30, 30), 0, combat.MountingRiderNotActiveRejection{}},
		{"not active beats movement", spec + "FR-009 FR-017 table 4.2b", "unhappy (rule interaction)", mounted, mntTurn(t, idle, 30, 0), 0, combat.MountingRiderNotActiveRejection{}},
		{"not mounted beats movement", spec + "FR-015 FR-017 table 4.2b", "unhappy (rule interaction)", none, mntTurn(t, active, 30, 0), 0, combat.MountingRiderNotMountedRejection{}},
		{"movement last", spec + "FR-012 FR-017 table 4.2b", "unhappy (rule interaction)", mounted, mntTurn(t, active, 30, 14), 0, combat.MountingInsufficientMovementRejection{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewDismountCreature().Run(context.Background(), mntDismountCommand(t, c.status, c.turn))
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if c.refusal == nil {
				a, ok := got.(combat.DismountCreatureApplied)
				if !ok || a.MovementCostFeet() != c.cost || a.RiderStatus() != combat.NotMounted() || len(a.Events()) != 1 {
					t.Errorf("result = %#v, want applied, cost %d, NotMounted, one event [%s]", got, c.cost, c.requirement)
				}
				return
			}
			r, ok := got.(combat.DismountCreatureRejected)
			if !ok || reflect.TypeOf(r.Reason()) != reflect.TypeOf(c.refusal) {
				t.Errorf("result = %#v, want refused with %T [%s]", got, c.refusal, c.requirement)
			}
			if s, ok := r.Reason().(combat.MountingInsufficientMovementRejection); ok && s.ShortfallFeet() != 1 {
				t.Errorf("shortfall = %d, want 1 [FR-012]", s.ShortfallFeet())
			}
		})
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-016, FR-055, FR-068, FR-069 - an accepted dismount emits one CreatureDismounted carrying rider, the mount it was on, the control it had and the cost; Events is a copy.
// Path: happy.
func TestDismountCreatureEvent(t *testing.T) {
	t.Parallel()
	res, err := combat.NewDismountCreature().Run(context.Background(), mntDismountCommand(t, mntMounted(t, combat.MountControlIndependent), mntTurn(t, combat.RiderTurnActive, 30, 30)))
	a, ok := res.(combat.DismountCreatureApplied)
	if err != nil || !ok {
		t.Fatalf("Run = %#v, %v", res, err)
	}
	ev, ok := a.Events()[0].(combat.CreatureDismounted)
	if !ok || ev.Control() != combat.MountControlIndependent || ev.MovementCostFeet() != 15 {
		t.Errorf("event = %#v, want independent, cost 15", a.Events()[0])
	}
	combattest.CheckEvent(t, ev, "creature_dismounted", gid("rider"), gid("horse"))
	events := a.Events()
	events[0] = nil
	if a.Events()[0] == nil {
		t.Error("Events must return a copy [FR-051]")
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-074, FR-052 - NewDismountCreatureCommand rejects a blank rider, a rider that is the mount of its own MountedOn and turn facts that are not valid with ErrInvalidInput.
// Path: unhappy (invalid input).
func TestNewDismountCreatureCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	turn := mntTurn(t, combat.RiderTurnActive, 30, 30)
	selfMount, _ := combat.MountedOn(gid("rider"), combat.MountControlControlled)
	for name, err := range map[string]error{
		"rider is its own mount": second(combat.NewDismountCreatureCommand(gid("rider"), selfMount, turn)),
		"blank rider":            second(combat.NewDismountCreatureCommand("", combat.NotMounted(), turn)),
		"zero turn facts":        second(combat.NewDismountCreatureCommand(gid("rider"), combat.NotMounted(), combat.MountingTurnFacts{})),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-066, FR-051 - an arbitrary id is accepted; the zero command is ErrInvalidInput and a cancelled ctx is returned unwrapped, each before any signal; a refusal leaves the command unchanged.
// Path: unhappy (invalid input).
func TestDismountCreatureRunGuards(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	if _, err := combat.NewDismountCreature(opts...).Run(context.Background(), combat.DismountCreatureCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command error = %v, want ErrInvalidInput", err)
	}
	cmd := mntDismountCommand(t, combat.NotMounted(), mntTurn(t, combat.RiderTurnActive, 30, 30))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := combat.NewDismountCreature(opts...).Run(ctx, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled error = %v, want context.Canceled", err)
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("a guard error recorded %d signals, want none", n)
	}
	a, _ := combat.NewDismountCreature().Run(context.Background(), cmd)
	b, _ := combat.NewDismountCreature().Run(context.Background(), cmd)
	if !reflect.DeepEqual(a, b) {
		t.Error("a refusal must not change the command")
	}
}
