package combat

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/environment-hazards-and-objects.md FR-064, FR-059 - the rules behind RestoreBreathing and RemoveExhaustionLevels pass on the ErrInvalidInput error of ConditionSet.ReduceExhaustion when called with a zero target, which the use cases rule out first; they never panic.
// Path: unhappy (invalid input).
func TestExhaustionRulesPassOnTheReductionError(t *testing.T) {
	t.Parallel()
	set := MustConditionSet([]Condition{ExhaustionCondition{MustExhaustionLevel(3)}}, nil)
	rec := MustHazardRecord(0, 0, 2, 0)
	if _, err := restoreBreathing(BreathNormal{}, set, "", rec); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("restoreBreathing with a zero target = %v, want ErrInvalidInput", err)
	}
	if _, err := removeExhaustion(set, "", MustHazardRecord(0, 0, 0, 0), 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("removeExhaustion with a zero target = %v, want ErrInvalidInput", err)
	}
}
