package combat_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-053, FR-060 - the reasons of the three closed rejection sets have the snake_case names of the srd5e.combat.result values, in the order the rules check them.
// Path: happy.
func TestMountingRefusalReasonNames(t *testing.T) {
	t.Parallel()
	// FR-060: the reason values of each use case, as the snake_case of its rejection types.
	reasons := map[string][]string{
		"mount_creature": {"rider_not_active", "rider_already_mounted", "mount_unwilling", "mount_not_larger", "mount_anatomy", "exotic_saddle_required",
			"out_of_reach", "mount_cannot_be_controlled", "no_movement_remaining", "insufficient_movement"},
	}
	got := []string{}
	for _, r := range []combat.MountRejection{combat.MountingRiderNotActiveRejection{}, combat.MountingRiderAlreadyMountedRejection{}, combat.MountingMountUnwillingRejection{},
		combat.MountingMountNotLargerRejection{}, combat.MountingMountAnatomyRejection{}, combat.MountingExoticSaddleRequiredRejection{}, combat.MountingOutOfReachRejection{},
		combat.MountingMountCannotBeControlledRejection{}, combat.MountingNoMovementRemainingRejection{}, combat.MountingInsufficientMovementRejection{}} {
		got = append(got, r.String())
	}
	if !slices.Equal(got, reasons["mount_creature"]) {
		t.Errorf("mount reasons = %v, want %v [FR-060]", got, reasons["mount_creature"])
	}
	got = nil
	for _, r := range []combat.DismountRejection{combat.MountingRiderNotActiveRejection{}, combat.MountingRiderNotMountedRejection{}, combat.MountingNoMovementRemainingRejection{}, combat.MountingInsufficientMovementRejection{}} {
		got = append(got, r.String())
	}
	if want := []string{"rider_not_active", "rider_not_mounted", "no_movement_remaining", "insufficient_movement"}; !slices.Equal(got, want) {
		t.Errorf("dismount reasons = %v, want %v [FR-060]", got, want)
	}
	if got := (combat.FallingOffRejection)(combat.MountingRiderNotMountedRejection{}).String(); got != "rider_not_mounted" {
		t.Errorf("falling off reason = %q, want rider_not_mounted [FR-060]", got)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-053 - each rejection type is a comparable value carrying its data: two OutOfReach values differ by distance, two InsufficientMovement values by shortfall.
// Path: unhappy (boundary).
func TestMountingRejectionsCarryTheirData(t *testing.T) {
	t.Parallel()
	in := mntInput(t)
	in.DistanceFeet = 6
	far, _ := combat.NewMountCreature().Run(context.Background(), mntCommand(t, in))
	in.DistanceFeet = 9
	farther, _ := combat.NewMountCreature().Run(context.Background(), mntCommand(t, in))
	a, b := far.(combat.MountCreatureRejected).Reason(), farther.(combat.MountCreatureRejected).Reason()
	if a == b || a.(combat.MountingOutOfReachRejection).DistanceFeet() != 6 || b.(combat.MountingOutOfReachRejection).DistanceFeet() != 9 {
		t.Errorf("reasons = %v and %v, want out_of_reach carrying 6 and 9 feet", a, b)
	}
}
