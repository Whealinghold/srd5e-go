package combat

import "github.com/Whealinghold/srd5e-go/d20"

// FallingOffDifficultyClass is the Difficulty Class of the Dexterity saving throw to stay
// mounted: 10 (SRD 5.2.1, "Falling Off", p. 16).
const FallingOffDifficultyClass d20.DifficultyClass = 10

// FallingOffTrigger is what calls for the Falling Off save (SRD 5.2.1, "Falling Off", p.
// 16). The trigger changes neither the Difficulty Class nor the ability, only what the
// result and event report. The zero value is invalid.
type FallingOffTrigger int

// The three triggers.
const (
	// FallingOffTriggerMountMovedAgainstItsWill: an effect is about to move the mount against its will.
	FallingOffTriggerMountMovedAgainstItsWill FallingOffTrigger = iota + 1
	// FallingOffTriggerRiderKnockedProne: the rider is knocked Prone.
	FallingOffTriggerRiderKnockedProne
	// FallingOffTriggerMountKnockedProne: the mount is knocked Prone.
	FallingOffTriggerMountKnockedProne
)

// Valid reports whether t is one of the three triggers.
func (t FallingOffTrigger) Valid() bool {
	return t >= FallingOffTriggerMountMovedAgainstItsWill && t <= FallingOffTriggerMountKnockedProne
}

// String returns "mount_moved_against_its_will", "rider_knocked_prone", "mount_knocked_prone"
// or "unknown".
func (t FallingOffTrigger) String() string {
	return enumName([]string{"mount_moved_against_its_will", "rider_knocked_prone", "mount_knocked_prone"}, int(t))
}

// LandingSpace says whether the host found an unoccupied space within 5 feet for a rider
// who falls off (SRD 5.2.1, "Falling Off", p. 16; "Unoccupied Space", p. 191). The host
// judges and places the rider. The zero value is invalid.
type LandingSpace int

// The two answers.
const (
	// LandingSpaceUnoccupiedWithin5Feet: there is an unoccupied space within 5 feet of the mount.
	LandingSpaceUnoccupiedWithin5Feet LandingSpace = iota + 1
	// LandingSpaceNoUnoccupiedSpaceReported: the host reports none; the save is still rolled.
	LandingSpaceNoUnoccupiedSpaceReported
)

// Valid reports whether s is one of the two answers.
func (s LandingSpace) Valid() bool {
	return s == LandingSpaceUnoccupiedWithin5Feet || s == LandingSpaceNoUnoccupiedSpaceReported
}

// FallingOffSave is the Falling Off saving throw as d20 resolved it. Outcome is the held
// result's outcome: the rider stays mounted exactly when it is d20.Success. Variants:
// FallingOffSaveRolled, FallingOffSaveForced.
type FallingOffSave interface {
	isFallingOffSave()
	Outcome() d20.Outcome
}

// FallingOffSaveRolled: the save was rolled.
type FallingOffSaveRolled struct{ result d20.CheckResult }

// Result returns the resolved check.
func (s FallingOffSaveRolled) Result() d20.CheckResult { return s.result }

// Outcome returns the outcome of the check.
func (s FallingOffSaveRolled) Outcome() d20.Outcome { return s.result.Outcome() }

// FallingOffSaveForced: the save failed without a die, because the rider chose to fail it or
// an effect made it fail automatically.
type FallingOffSaveForced struct{ result d20.ForcedFailureResult }

// Result returns the forced failure.
func (s FallingOffSaveForced) Result() d20.ForcedFailureResult { return s.result }

// Outcome returns d20.Failure.
func (FallingOffSaveForced) Outcome() d20.Outcome { return d20.Failure }

func (FallingOffSaveRolled) isFallingOffSave() {}
func (FallingOffSaveForced) isFallingOffSave() {}

// staysMounted reports whether the save keeps the rider on the mount.
func staysMounted(s FallingOffSave) bool { return s.Outcome() == d20.Success }
