package combat_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-014, FR-021, FR-022, FR-055, FR-068 - an accepted mount returns MountedOn(mount, control), the cost and one CreatureMounted carrying rider, mount, control, cost, the Initiative change and the turn; a controlled ride matches the rider's Initiative on the rider's turn, an independent one changes nothing.
// Path: happy.
func TestMountCreatureAppliedFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		control combat.MountControl
		init    combat.MountInitiativeChange
		turn    combat.MountTurn
	}{
		{"controlled", combat.MountControlControlled, combat.MountInitiativeMatchesRider{Rider: gid("rider")}, combat.MountTurnOnRidersTurn},
		{"independent", combat.MountControlIndependent, combat.MountInitiativeUnchanged{}, combat.MountTurnOnOwnTurn},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mntInput(t)
			in.Control = c.control
			res, err := combat.NewMountCreature().Run(context.Background(), mntCommand(t, in))
			a, ok := res.(combat.MountCreatureApplied)
			if err != nil || !ok {
				t.Fatalf("Run = %#v, %v, want applied", res, err)
			}
			if a.RiderStatus() != mntMounted(t, c.control) || a.MovementCostFeet() != 15 || len(a.Events()) != 1 {
				t.Fatalf("applied = %+v, want mounted, cost 15, one event", a)
			}
			ev, ok := a.Events()[0].(combat.CreatureMounted)
			if !ok || ev.Control() != c.control || ev.MovementCostFeet() != 15 || ev.InitiativeChange() != c.init || ev.Turn() != c.turn {
				t.Errorf("event = %#v, want control %v, cost 15, %v, %v", a.Events()[0], c.control, c.init, c.turn)
			}
			combattest.CheckEvent(t, ev, "creature_mounted", gid("rider"), gid("horse"))
			combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{a.Events()[0]} })
			events := a.Events()
			events[0] = nil
			if a.Events()[0] == nil {
				t.Error("Events must return a copy [FR-051]")
			}
		})
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-005, FR-008, FR-052 and section 5 - NewMountCreatureCommand rejects equal ids, a blank id, a negative distance, a size, control, facts or turn facts that are not valid with ErrInvalidInput.
// Path: unhappy (invalid input).
func TestNewMountCreatureCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*combat.MountCreatureInput){
		"rider is the mount": func(in *combat.MountCreatureInput) { in.Mount = in.Rider },
		"blank rider":        func(in *combat.MountCreatureInput) { in.Rider = "" },
		"blank mount":        func(in *combat.MountCreatureInput) { in.Mount = "" },
		"negative distance":  func(in *combat.MountCreatureInput) { in.DistanceFeet = -1 },
		"the smallest int":   func(in *combat.MountCreatureInput) { in.DistanceFeet = math.MinInt },
		"zero rider size":    func(in *combat.MountCreatureInput) { in.RiderSize = 0 },
		"zero mount size":    func(in *combat.MountCreatureInput) { in.MountSize = 0 },
		"unknown size":       func(in *combat.MountCreatureInput) { in.MountSize = 7 },
		"zero control":       func(in *combat.MountCreatureInput) { in.Control = 0 },
		"zero facts":         func(in *combat.MountCreatureInput) { in.Facts = combat.MountFacts{} },
		"zero turn facts":    func(in *combat.MountCreatureInput) { in.Turn = combat.MountingTurnFacts{} },
	}
	for name, change := range cases {
		in := mntInput(t)
		change(&in)
		if _, err := combat.NewMountCreatureCommand(in); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-066 - ids are opaque: an arbitrary CombatantID that no tracker knows is accepted; the zero command is an ErrInvalidInput error and a cancelled ctx is returned unwrapped, each before any signal.
// Path: unhappy (invalid input).
func TestMountCreatureRunGuards(t *testing.T) {
	t.Parallel()
	in := mntInput(t)
	in.Rider, in.Mount = gid("nobody-knows-me"), gid("neither-me")
	if res, err := combat.NewMountCreature().Run(context.Background(), mntCommand(t, in)); err != nil || reflect.TypeOf(res) != reflect.TypeOf(combat.MountCreatureApplied{}) {
		t.Errorf("unknown ids: %#v, %v, want applied", res, err)
	}
	rec, opts := recording()
	if _, err := combat.NewMountCreature(opts...).Run(context.Background(), combat.MountCreatureCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command error = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := combat.NewMountCreature(opts...).Run(ctx, mntCommand(t, mntInput(t))); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled error = %v, want context.Canceled", err)
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("a guard error recorded %d signals, want none", n)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-011, FR-051 - a refusal returns no cost to charge and leaves the command and the RiderStatus unchanged: a second Run on the same command gives an equal result.
// Path: unhappy (immutability).
func TestMountCreatureRefusalLeavesInputsUnchanged(t *testing.T) {
	t.Parallel()
	in := mntInput(t)
	in.Status = mntMounted(t, combat.MountControlIndependent)
	cmd := mntCommand(t, in)
	before := cmd
	first, err1 := combat.NewMountCreature().Run(context.Background(), cmd)
	second, err2 := combat.NewMountCreature().Run(context.Background(), cmd)
	if err1 != nil || err2 != nil || !reflect.DeepEqual(first, second) || cmd != before || in.Status != mntMounted(t, combat.MountControlIndependent) {
		t.Errorf("a refusal changed an input or differs between calls: %#v vs %#v", first, second)
	}
}
