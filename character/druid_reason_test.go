package character_test

import (
	"regexp"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-druid.md FR-161, FR-173, D-5 - every DruidRejection has a distinct snake_case name, the members follow the
// total order of D-5 (availability, state, casting, timing, arguments, resources, limits, the expended slot, then the level reasons),
// the zero value and values past the last are unknown and invalid.
// Path: per case (unhappy: boundary, rule interaction).
func TestDruidRejectionNamesAndOrder(t *testing.T) {
	t.Parallel()
	snake := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	seen := map[string]bool{}
	n := 0
	for r := character.DruidFeatureNotGranted; r.Valid(); r++ {
		n++
		if s := r.String(); !snake.MatchString(s) || seen[s] || s == "unknown" {
			t.Errorf("reason %d name %q is not unique snake_case", r, s)
		} else {
			seen[s] = true
		}
	}
	if n != int(character.DruidMaximumLevelReached) {
		t.Errorf("%d reasons, last is %d", n, character.DruidMaximumLevelReached)
	}
	for _, r := range []character.DruidRejection{0, character.DruidMaximumLevelReached + 1, -1} {
		if r.Valid() || r.String() != "unknown" {
			t.Errorf("reason %d = %q, valid %v; want unknown", r, r, r.Valid())
		}
	}
	order := []character.DruidRejection{
		character.DruidFeatureNotGranted, character.DruidFeatureNotChosen, character.DruidLandTypeNotChosen, // 1 availability
		character.DruidNotInBeastForm, character.DruidNoActiveSanctuary, character.DruidSanctuaryAlreadyActive, character.DruidPrimalOrderAlreadyChosen, character.DruidElementalFuryAlreadyChosen, // 2 state
		character.DruidCannotCastInBeastForm, character.DruidMaterialComponentRestricted, // 3 casting
		character.DruidChangeNotAllowedNow, character.DruidRecoveryNotAllowedNow, character.DruidAlreadyUsedThisTurn, character.DruidAlreadyUsedSinceLongRest, // 4 timing
		character.DruidInvalidConversion, character.DruidAttackNotEligible, character.DruidNotACircleSpell, character.DruidNotADruidCantrip, character.DruidFormNotKnown, // 5 arguments
		character.DruidNotABeast, character.DruidChallengeRatingTooHigh, character.DruidFlySpeedNotAllowed, character.DruidDuplicateForm, character.DruidTooManyKnownForms,
		character.DruidItemUnassigned, character.DruidItemCannotBeWorn, character.DruidMoveTooFar,
		character.DruidNoUsesRemaining, character.DruidWildShapeUsesRemain, character.DruidNoSlotAvailable, // 6 resources
		character.DruidSlotLevelTooHigh, character.DruidRecoveryBudgetExceeded, character.DruidTooManyReplacements, // 7 limits
		character.DruidNoExpendedSlot, // 8
		character.DruidLevelNotConsecutive, character.DruidMaximumLevelReached,
	}
	if len(order) != n {
		t.Fatalf("the order lists %d reasons, the enum has %d", len(order), n)
	}
	for i, r := range order {
		if int(r) != i+1 {
			t.Errorf("reason %s is number %d, want %d (the D-5 order)", r, r, i+1)
		}
	}
	if character.DruidFeatureNotGranted.String() != "feature_not_granted" || character.DruidNoExpendedSlot.String() != "no_expended_slot" {
		t.Error("first and slot reasons misnamed")
	}
}
