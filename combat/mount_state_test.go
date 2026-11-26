package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-014, FR-027, section 3 (RiderStatus) - the zero RiderStatus is NotMounted; MountedOn holds the mount and the control, is mounted on that mount only and compares by value.
// Path: happy.
func TestRiderStatusHoldsTheRide(t *testing.T) {
	t.Parallel()
	s := mntMounted(t, combat.MountControlIndependent)
	c, ok := s.Control()
	if !s.IsMounted() || !s.IsOn(gid("horse")) || s.IsOn(gid("pony")) || s.IsOn(gid("ant")) || !ok || c != combat.MountControlIndependent {
		t.Errorf("MountedOn status = %+v, want mounted on horse, independent", s)
	}
	if s != mntMounted(t, combat.MountControlIndependent) || s == mntMounted(t, combat.MountControlControlled) {
		t.Error("RiderStatus must compare by mount and control")
	}
	var zero combat.RiderStatus
	if _, ok := zero.Control(); zero != combat.NotMounted() || zero.IsMounted() || zero.IsOn(gid("horse")) || ok {
		t.Errorf("zero RiderStatus = %+v, want NotMounted", zero)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-052 (malformed value) - MountedOn rejects a blank mount and a control that is not valid with ErrInvalidInput.
// Path: unhappy (invalid input).
func TestMountedOnRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for name, err := range map[string]error{
		"blank mount":     second(combat.MountedOn("", combat.MountControlControlled)),
		"zero control":    second(combat.MountedOn(gid("horse"), 0)),
		"unknown control": second(combat.MountedOn(gid("horse"), 3)),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-021, FR-022 and section 7 (attribute values) - the controls and turns have their snake_case names and the zero is unknown.
// Path: unhappy (boundary).
func TestMountControlAndTurnNames(t *testing.T) {
	t.Parallel()
	for _, c := range [][2]string{
		{combat.MountControlControlled.String(), "controlled"}, {combat.MountControlIndependent.String(), "independent"}, {combat.MountControl(0).String(), "unknown"},
		{combat.MountTurnOnRidersTurn.String(), "on_riders_turn"}, {combat.MountTurnOnOwnTurn.String(), "on_own_turn"}, {combat.MountTurn(0).String(), "unknown"},
	} {
		if c[0] != c[1] {
			t.Errorf("name = %q, want %q", c[0], c[1])
		}
	}
	if combat.MountControl(0).Valid() || combat.MountControl(3).Valid() || !combat.MountControlControlled.Valid() || !combat.MountControlIndependent.Valid() {
		t.Error("Valid must hold for the two controls only")
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md section 5 (MountFacts: five enums) - NewMountFacts accepts every combination of the constants and rejects a zero or unknown value of each of the five fields.
// Path: unhappy (invalid input).
func TestNewMountFactsRejectsUnknownValues(t *testing.T) {
	t.Parallel()
	w, a, tr, h, s := combat.MountWillingnessUnwilling, combat.MountAnatomyNotAppropriate, combat.MountTrainingNotTrained, combat.MountHabitatAquaticOrFlying, combat.ExoticSaddleNotFitted
	if _, err := combat.NewMountFacts(w, a, tr, h, s); err != nil {
		t.Fatalf("the second constant of each field: %v", err)
	}
	cases := map[string]error{
		"willingness 0": second(combat.NewMountFacts(0, a, tr, h, s)), "willingness 3": second(combat.NewMountFacts(3, a, tr, h, s)),
		"anatomy 0": second(combat.NewMountFacts(w, 0, tr, h, s)), "anatomy 3": second(combat.NewMountFacts(w, 3, tr, h, s)),
		"training 0": second(combat.NewMountFacts(w, a, 0, h, s)), "training 3": second(combat.NewMountFacts(w, a, 3, h, s)),
		"habitat 0": second(combat.NewMountFacts(w, a, tr, 0, s)), "habitat 3": second(combat.NewMountFacts(w, a, tr, 3, s)),
		"saddle 0": second(combat.NewMountFacts(w, a, tr, h, 0)), "saddle 3": second(combat.NewMountFacts(w, a, tr, h, 3)),
	}
	for name, err := range cases {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-052 (negative WalkingSpeedFeet), table 4.2a row "negative Speed" - NewMountingTurnFacts rejects a negative Speed and an unknown turn state, and accepts movement left of 0 or less, Speed 0 and the maximum.
// Path: unhappy (boundary).
func TestNewMountingTurnFacts(t *testing.T) {
	t.Parallel()
	for name, err := range map[string]error{
		"negative Speed": second(combat.NewMountingTurnFacts(combat.RiderTurnActive, -1, 0)),
		"turn 0":         second(combat.NewMountingTurnFacts(0, 30, 30)),
		"turn 3":         second(combat.NewMountingTurnFacts(3, 30, 30)),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	for name, err := range map[string]error{
		"Speed 0":               second(combat.NewMountingTurnFacts(combat.RiderTurnActive, 0, 0)),
		"movement left -5":      second(combat.NewMountingTurnFacts(combat.RiderTurnNotActive, 30, -5)),
		"extreme movement left": second(combat.NewMountingTurnFacts(combat.RiderTurnActive, 30, math.MinInt)),
	} {
		if err != nil {
			t.Errorf("%s: error = %v, want none", name, err)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-021, FR-022 - MountInitiativeChange has the two variants MountInitiativeMatchesRider (carrying the rider) and MountInitiativeUnchanged, comparable by value.
// Path: happy.
func TestMountInitiativeChangeVariants(t *testing.T) {
	t.Parallel()
	var a, b combat.MountInitiativeChange = combat.MountInitiativeMatchesRider{Rider: gid("rider")}, combat.MountInitiativeMatchesRider{Rider: gid("rider")}
	var c combat.MountInitiativeChange = combat.MountInitiativeUnchanged{}
	if a != b || a == c {
		t.Error("MountInitiativeChange must compare by value")
	}
}
