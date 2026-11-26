package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-038 - FallingOffDifficultyClass is a DifficultyClass equal to 10 and equal by value to any d20.DifficultyClass(10).
// Path: happy.
func TestFallingOffDifficultyClass(t *testing.T) {
	t.Parallel()
	if combat.FallingOffDifficultyClass != d20.DifficultyClass(10) {
		t.Errorf("FallingOffDifficultyClass = %d, want 10", combat.FallingOffDifficultyClass)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-029, FR-033, section 7 - the three triggers and the landing spaces are valid, have their snake_case names, and 0 and out-of-range values are not valid.
// Path: unhappy (boundary).
func TestFallingOffTriggerAndLandingSpace(t *testing.T) {
	t.Parallel()
	for trigger, name := range map[combat.FallingOffTrigger]string{
		combat.FallingOffTriggerMountMovedAgainstItsWill: "mount_moved_against_its_will", combat.FallingOffTriggerRiderKnockedProne: "rider_knocked_prone",
		combat.FallingOffTriggerMountKnockedProne: "mount_knocked_prone", 0: "unknown", 4: "unknown",
	} {
		if trigger.String() != name || trigger.Valid() != (name != "unknown") {
			t.Errorf("trigger %d = %q valid %v, want %q", int(trigger), trigger.String(), trigger.Valid(), name)
		}
	}
	for landing, valid := range map[combat.LandingSpace]bool{
		combat.LandingSpaceUnoccupiedWithin5Feet: true, combat.LandingSpaceNoUnoccupiedSpaceReported: true, 0: false, 3: false,
	} {
		if landing.Valid() != valid {
			t.Errorf("landing %d valid = %v, want %v", int(landing), landing.Valid(), valid)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-073 - FallingOffSave has exactly the two variants; Outcome is the held result's outcome (a forced failure is always Failure) and the held result is equal by value to the one resolved.
// Path: happy.
func TestFallingOffSaveVariants(t *testing.T) {
	t.Parallel()
	roll, err := d20.NewRoll(modifier.Normal, []int{8})
	if err != nil {
		t.Fatal(err)
	}
	check, err := d20.ResolveCheck(d20.SavingThrow, roll, d20.NewRollModifiers(modifier.New(1)), combat.FallingOffDifficultyClass)
	if err != nil {
		t.Fatal(err)
	}
	forced, err := d20.FailWithoutRoll(d20.SavingThrow, d20.Automatic)
	if err != nil {
		t.Fatal(err)
	}
	res, err := mntFall(t, combat.FallingOffTriggerMountMovedAgainstItsWill, d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(1))}, 8)
	if err != nil {
		t.Fatal(err)
	}
	got := res.(combat.ResolveFallingOffFellOff).Save().(combat.FallingOffSaveRolled)
	if got.Outcome() != check.Outcome() || !got.Result().Equal(check) {
		t.Errorf("rolled save = %+v, want outcome %v and result %+v", got, check.Outcome(), check)
	}
	res, err = mntFall(t, combat.FallingOffTriggerMountMovedAgainstItsWill, d20.AutomaticFailure{})
	if err != nil {
		t.Fatal(err)
	}
	f := res.(combat.ResolveFallingOffFellOff).Save().(combat.FallingOffSaveForced)
	if f.Outcome() != d20.Failure || f.Result() != forced {
		t.Errorf("forced save = %+v, want Failure and %+v", f, forced)
	}
}
