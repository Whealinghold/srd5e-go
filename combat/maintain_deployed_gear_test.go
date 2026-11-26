package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func maintainRun(tb testing.TB, set combat.DeployedGears, holds combat.GearHolds, m combat.GearMaintenance, opts ...combat.Option) (combat.MaintainDeployedGearResult, error) {
	tb.Helper()
	cmd, err := combat.NewMaintainDeployedGearCommand(set, holds, m)
	if err != nil {
		tb.Fatal(err)
	}
	return combat.NewMaintainDeployedGear(opts...).Run(context.Background(), cmd)
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-010 to FR-013, FR-059, FR-060, table 4.9.2 - MaintainDeployedGear lights Oil, recovers Caltrops, resets a sprung Hunting Trap with no creature in it and expires burning Oil, returning MaintainDeployedGearApplied with the new set and events, or MaintainDeployedGearRejected with the reason; Expire with nothing to burn out is an applied result with no event (SRD 5.2.1, pp. 96-98).
// Path: per case.
func TestMaintainDeployedGearRun(t *testing.T) {
	t.Parallel()
	caught := gearHolds(t, gearHold(t, combat.GearHoldInput{ID: "h", Kind: combat.HoldKindHuntingTrap, Target: gid("walker"), Restraint: combat.HoldRestraintDoesNotRestrain, GearID: "g"}))
	lit := gearBurning(t, 1, "a")
	cases := []struct {
		name, path string
		set        combat.DeployedGears
		holds      combat.GearHolds
		m          combat.GearMaintenance
		want       combat.GearRejection
		events     []string
		left       int
	}{
		{"light oil", "happy", gearOne(t, combat.DeployedGearKindPouredOil), combat.GearHolds{}, combat.LightOil{GearID: "g", Stamp: gearStamp(1, "a")}, 0, []string{"poured_oil_lit"}, 1},
		{"light oil twice", "unhappy (state error)", lit, combat.GearHolds{}, combat.LightOil{GearID: "g", Stamp: gearStamp(1, "a")}, combat.GearRejectionAlreadyLit, nil, 1},
		{"recover caltrops", "happy", gearOne(t, combat.DeployedGearKindCaltrops), combat.GearHolds{}, combat.RecoverGear{GearID: "g"}, 0, []string{"deployed_gear_recovered"}, 0},
		{"recover a trap", "unhappy (state error)", gearOne(t, combat.DeployedGearKindHuntingTrap), combat.GearHolds{}, combat.RecoverGear{GearID: "g"}, combat.GearRejectionNotRecoverable, nil, 1},
		{"reset a free trap", "happy", gearSprung(t), combat.GearHolds{}, combat.ResetHuntingTrap{GearID: "g"}, 0, []string{"hunting_trap_reset"}, 1},
		{"reset an occupied trap", "unhappy (state error)", gearSprung(t), caught, combat.ResetHuntingTrap{GearID: "g"}, combat.GearRejectionTrapOccupied, nil, 1},
		{"reset an unknown trap", "unhappy (state error)", gearSprung(t), caught, combat.ResetHuntingTrap{GearID: "zz"}, combat.GearRejectionUnknownGear, nil, 1},
		{"expire at the lighter's last turn", "unhappy (boundary)", lit, combat.GearHolds{}, combat.ExpireBurningOil{EndedTurn: gearStamp(3, "a")}, 0, []string{"poured_oil_burned_out"}, 0},
		{"expire at another creature's turn", "unhappy (boundary)", lit, combat.GearHolds{}, combat.ExpireBurningOil{EndedTurn: gearStamp(3, "b")}, 0, nil, 1},
		{"expire with no gear", "unhappy (empty)", combat.DeployedGears{}, combat.GearHolds{}, combat.ExpireBurningOil{EndedTurn: gearStamp(3, "b")}, 0, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := maintainRun(t, c.set, c.holds, c.m)
			if err != nil {
				t.Fatal(err)
			}
			if c.want != 0 {
				if r, ok := res.(combat.MaintainDeployedGearRejected); !ok || r.Reason() != c.want {
					t.Errorf("Run = %#v, want rejected %v [%s]", res, c.want, gearSpec)
				}
				return
			}
			a, ok := res.(combat.MaintainDeployedGearApplied)
			if !ok || a.Gears().Len() != c.left || len(gearKinds(a.Events())) != len(c.events) {
				t.Fatalf("Run = %#v, want applied with %d gears and events %v", res, c.left, c.events)
			}
			for i, k := range gearKinds(a.Events()) {
				if k != c.events[i] {
					t.Errorf("event %d = %s, want %s", i, k, c.events[i])
				}
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-071 - NewMaintainDeployedGearCommand returns an invalid_value GearError for a maintenance that is nil, of another type or holds a blank id or the zero stamp; Run returns ErrInvalidInput for the zero command and ctx.Err() for a cancelled ctx.
// Path: unhappy (invalid input, cancellation).
func TestNewMaintainDeployedGearCommand(t *testing.T) {
	t.Parallel()
	for name, m := range map[string]combat.GearMaintenance{"nil": nil, "blank light": combat.LightOil{Stamp: gearStamp(1, "a")}, "light, zero stamp": combat.LightOil{GearID: "g"},
		"blank recover": combat.RecoverGear{}, "blank reset": combat.ResetHuntingTrap{GearID: " "}, "zero expire": combat.ExpireBurningOil{}} {
		if _, err := combat.NewMaintainDeployedGearCommand(combat.DeployedGears{}, combat.GearHolds{}, m); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
			t.Errorf("%s: err = %v, want invalid_value", name, err)
		}
	}
	if _, err := combat.NewMaintainDeployedGear().Run(context.Background(), combat.MaintainDeployedGearCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: err = %v, want ErrInvalidInput", err)
	}
	cmd, _ := combat.NewMaintainDeployedGearCommand(combat.DeployedGears{}, combat.GearHolds{}, combat.RecoverGear{GearID: "g"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := combat.NewMaintainDeployedGear().Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled ctx: %v, %v; want context.Canceled", res, err)
	}
}
