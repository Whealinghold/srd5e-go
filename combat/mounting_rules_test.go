package combat_test

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

type mountRow struct {
	name, requirement, path string
	change                  func(t *testing.T, in *combat.MountCreatureInput)
	cost                    int
	refusal                 combat.MountRejection // nil: accepted
}

func sizes(rider, mount content.CreatureSize) func(*testing.T, *combat.MountCreatureInput) {
	return func(_ *testing.T, in *combat.MountCreatureInput) { in.RiderSize, in.MountSize = rider, mount }
}

func turn(state combat.RiderTurnState, speed, left int) func(*testing.T, *combat.MountCreatureInput) {
	return func(t *testing.T, in *combat.MountCreatureInput) {
		t.Helper()
		in.Turn = mntTurn(t, state, speed, left)
	}
}

func facts(w combat.MountWillingness, a combat.MountAnatomy, tr combat.MountTraining, h combat.MountHabitat, s combat.ExoticSaddle) func(*testing.T, *combat.MountCreatureInput) {
	return func(t *testing.T, in *combat.MountCreatureInput) { t.Helper(); in.Facts = mntFacts(t, w, a, tr, h, s) }
}

func both(fs ...func(*testing.T, *combat.MountCreatureInput)) func(*testing.T, *combat.MountCreatureInput) {
	return func(t *testing.T, in *combat.MountCreatureInput) {
		t.Helper()
		for _, f := range fs {
			f(t, in)
		}
	}
}

func alreadyMounted(c combat.MountControl) func(*testing.T, *combat.MountCreatureInput) {
	return func(t *testing.T, in *combat.MountCreatureInput) {
		t.Helper()
		in.Status = mntMounted(t, c)
	}
}

func distance(feet int) func(*testing.T, *combat.MountCreatureInput) {
	return func(_ *testing.T, in *combat.MountCreatureInput) { in.DistanceFeet = feet }
}

func control(c combat.MountControl) func(*testing.T, *combat.MountCreatureInput) {
	return func(_ *testing.T, in *combat.MountCreatureInput) { in.Control = c }
}

func mountRows() []mountRow {
	const (
		willing, unwilling = combat.MountWillingnessWilling, combat.MountWillingnessUnwilling
		fit, noFit         = combat.ExoticSaddleFitted, combat.ExoticSaddleNotFitted
		other, flying      = combat.MountHabitatOther, combat.MountHabitatAquaticOrFlying
		good, bad          = combat.MountAnatomyAppropriate, combat.MountAnatomyNotAppropriate
		trained, untrained = combat.MountTrainingTrainedToAcceptRider, combat.MountTrainingNotTrained
		active, idle       = combat.RiderTurnActive, combat.RiderTurnNotActive
		tiny, small, med   = content.SizeTiny, content.SizeSmall, content.SizeMedium
		large, huge, garg  = content.SizeLarge, content.SizeHuge, content.SizeGargantuan
	)
	notLarger := combat.MountingMountNotLargerRejection{}
	spec := "doc/specs/mounted-and-underwater-combat.md "
	return []mountRow{
		{"one size larger", spec + "FR-002 table 4.1a", "happy", sizes(med, large), 15, nil},
		{"several sizes larger", spec + "FR-002 table 4.1a", "unhappy (boundary)", sizes(small, huge), 15, nil},
		{"same size", spec + "FR-002 table 4.1a", "unhappy (boundary)", sizes(med, med), 0, notLarger},
		{"one size smaller", spec + "FR-002 table 4.1a", "unhappy (boundary)", sizes(large, med), 0, notLarger},
		{"bottom of the scale", spec + "FR-002 table 4.1a", "unhappy (boundary)", sizes(tiny, small), 15, nil},
		{"top of the scale, no larger exists", spec + "FR-002 table 4.1a", "unhappy (boundary)", sizes(garg, garg), 0, notLarger},
		{"one below the top", spec + "FR-002 table 4.1a", "unhappy (boundary)", sizes(huge, garg), 15, nil},
		{"Tiny rider, Tiny mount", spec + "FR-002 table 4.1a", "unhappy (empty and minimal)", sizes(tiny, tiny), 0, notLarger},
		{"adjacent", spec + "FR-008 table 4.1b", "unhappy (boundary)", distance(0), 15, nil},
		{"exactly 5 feet", spec + "FR-008 table 4.1b", "unhappy (boundary)", distance(5), 15, nil},
		{"6 feet", spec + "FR-008 table 4.1b", "unhappy (boundary)", distance(6), 0, combat.MountingOutOfReachRejection{}},
		{"unwilling", spec + "FR-001 table 4.1b", "unhappy (state error)", facts(unwilling, good, trained, other, noFit), 0, combat.MountingMountUnwillingRejection{}},
		{"anatomy", spec + "FR-003 table 4.1b", "unhappy (state error)", facts(willing, bad, trained, other, noFit), 0, combat.MountingMountAnatomyRejection{}},
		{"flying mount, no Exotic Saddle", spec + "FR-004 table 4.1b", "unhappy (rule interaction)", facts(willing, good, trained, flying, noFit), 0, combat.MountingExoticSaddleRequiredRejection{}},
		{"flying mount, Exotic Saddle", spec + "FR-004 table 4.1b", "happy", facts(willing, good, trained, flying, fit), 15, nil},
		{"ground mount, Exotic Saddle fitted", spec + "FR-004 table 4.1b", "unhappy (boundary)", facts(willing, good, trained, other, fit), 15, nil},
		{"the SRD's example", spec + "FR-010 table 4.2a", "happy", turn(active, 30, 30), 15, nil},
		{"odd Speed rounds down", spec + "FR-010 table 4.2a", "unhappy (boundary)", turn(active, 25, 25), 12, nil},
		{"Speed 1 costs nothing", spec + "FR-010 table 4.2a", "unhappy (boundary)", turn(active, 1, 1), 0, nil},
		{"Speed 5", spec + "FR-010 table 4.2a", "unhappy (boundary)", turn(active, 5, 5), 2, nil},
		{"exactly enough left", spec + "FR-012 table 4.2a", "unhappy (boundary)", turn(active, 30, 15), 15, nil},
		{"one short", spec + "FR-012 table 4.2a", "unhappy (boundary)", turn(active, 30, 14), 15, combat.MountingInsufficientMovementRejection{}},
		{"nothing left", spec + "FR-012 table 4.2a", "unhappy (boundary)", turn(active, 30, 0), 15, combat.MountingNoMovementRemainingRejection{}},
		{"less than nothing left", spec + "FR-012 table 4.2a", "unhappy (boundary)", turn(active, 30, -5), 15, combat.MountingNoMovementRemainingRejection{}},
		{"the smallest movement left", spec + "FR-012 overflow", "unhappy (boundary)", turn(active, 30, math.MinInt), 15, combat.MountingNoMovementRemainingRejection{}},
		{"after Dash, half of Speed only", spec + "FR-013 table 4.2a", "unhappy (rule interaction)", turn(active, 30, 60), 15, nil},
		{"after moving 20", spec + "FR-012 table 4.2a", "unhappy (rule interaction)", turn(active, 30, 10), 15, combat.MountingInsufficientMovementRejection{}},
		{"Speed reduced mid-turn to 20", spec + "FR-010 table 4.2a", "unhappy (rule interaction)", turn(active, 20, 20), 10, nil},
		{"Speed 0", spec + "FR-012 Q-7 table 4.2a", "unhappy (boundary)", turn(active, 0, 0), 0, combat.MountingNoMovementRemainingRejection{}},
		{"the largest Speed", spec + "FR-010 overflow", "unhappy (boundary)", turn(active, math.MaxInt, math.MaxInt), math.MaxInt / 2, nil},
		{"rider not active", spec + "FR-009 table 4.2a", "unhappy (state error)", turn(idle, 30, 30), 0, combat.MountingRiderNotActiveRejection{}},
		{"trained, controlled", spec + "FR-019 table 4.3a", "happy", both(control(combat.MountControlControlled), facts(willing, good, trained, other, noFit)), 15, nil},
		{"untrained, controlled", spec + "FR-019 table 4.3a", "unhappy (state error)", both(control(combat.MountControlControlled), facts(willing, good, untrained, other, noFit)), 0, combat.MountingMountCannotBeControlledRejection{}},
		{"untrained, independent", spec + "FR-020 table 4.3a", "unhappy (boundary)", both(control(combat.MountControlIndependent), facts(willing, good, untrained, other, noFit)), 15, nil},
		{"trained, independent", spec + "FR-020 table 4.3a", "unhappy (rule interaction)", control(combat.MountControlIndependent), 15, nil},
		{"already mounted", spec + "FR-006", "unhappy (state error)", alreadyMounted(combat.MountControlIndependent), 0, combat.MountingRiderAlreadyMountedRejection{}},
		{"not active beats the mount's facts", spec + "FR-009 FR-017 table 4.2b", "unhappy (rule interaction)", both(turn(idle, 30, 30), facts(unwilling, good, trained, other, noFit)), 0, combat.MountingRiderNotActiveRejection{}},
		{"not active beats already mounted", spec + "FR-009 FR-017 table 4.2b", "unhappy (rule interaction)", both(turn(idle, 30, 30), alreadyMounted(combat.MountControlControlled)), 0, combat.MountingRiderNotActiveRejection{}},
		{"already mounted beats unwilling", spec + "FR-006 FR-017 table 4.2b", "unhappy (rule interaction)", both(facts(unwilling, good, trained, other, noFit), alreadyMounted(combat.MountControlControlled)), 0, combat.MountingRiderAlreadyMountedRejection{}},
		{"unwilling beats size", spec + "FR-001 FR-017 table 4.2b", "unhappy (rule interaction)", both(facts(unwilling, good, trained, other, noFit), sizes(med, med)), 0, combat.MountingMountUnwillingRejection{}},
		{"size beats anatomy", spec + "FR-002 FR-017 table 4.2b", "unhappy (rule interaction)", both(sizes(med, med), facts(willing, bad, trained, other, noFit)), 0, notLarger},
		{"anatomy beats saddle", spec + "FR-003 FR-017 table 4.2b", "unhappy (rule interaction)", facts(willing, bad, trained, flying, noFit), 0, combat.MountingMountAnatomyRejection{}},
		{"saddle beats reach", spec + "FR-004 FR-017 table 4.2b", "unhappy (rule interaction)", both(facts(willing, good, trained, flying, noFit), distance(6)), 0, combat.MountingExoticSaddleRequiredRejection{}},
		{"reach beats control", spec + "FR-008 FR-017 table 4.2b", "unhappy (rule interaction)", both(distance(6), facts(willing, good, untrained, other, noFit)), 0, combat.MountingOutOfReachRejection{}},
		{"control beats movement", spec + "FR-019 FR-017 table 4.2b", "unhappy (rule interaction)", both(facts(willing, good, untrained, other, noFit), turn(active, 30, 0)), 0, combat.MountingMountCannotBeControlledRejection{}},
	}
}

// Requirement: per case
// Path: per case.
func TestMountCreatureRules(t *testing.T) {
	t.Parallel()
	for _, c := range mountRows() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mntInput(t)
			c.change(t, &in)
			got, err := combat.NewMountCreature().Run(context.Background(), mntCommand(t, in))
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if c.refusal == nil {
				a, ok := got.(combat.MountCreatureApplied)
				if !ok || a.MovementCostFeet() != c.cost || !a.RiderStatus().IsOn(gid("horse")) {
					t.Errorf("result = %#v, want applied with cost %d on the horse [%s]", got, c.cost, c.requirement)
				}
				return
			}
			r, ok := got.(combat.MountCreatureRejected)
			if !ok || reflect.TypeOf(r.Reason()) != reflect.TypeOf(c.refusal) {
				t.Fatalf("result = %#v, want refused with %T [%s]", got, c.refusal, c.requirement)
			}
			switch v := r.Reason().(type) {
			case combat.MountingOutOfReachRejection:
				if v.DistanceFeet() != 6 {
					t.Errorf("distance = %d, want 6 [FR-008]", v.DistanceFeet())
				}
			case combat.MountingInsufficientMovementRejection:
				if want := 15 - map[string]int{"one short": 14, "after moving 20": 10}[c.name]; v.ShortfallFeet() != want {
					t.Errorf("shortfall = %d, want %d [FR-012]", v.ShortfallFeet(), want)
				}
			}
		})
	}
}
