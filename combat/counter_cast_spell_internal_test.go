package combat

import "testing"

// Requirement: doc/specs/dispel-and-counterspell.md section 7 and FR-056 - a conclusion that is CounterspellInvalid is classed invalid with counterspell_invalid_save, and a Dispel Magic invalid with dispel_invalid_checks. The use cases build their own save and checks, so no public call reaches these branches; they are pinned here through the classifiers.
// Path: per case.
func TestInvalidConclusionsAreClassedInvalid(t *testing.T) {
	t.Parallel()
	counter := describeCounterCast(CounterCastSpellConcluded{result: CounterspellInvalid{problem: CounterspellSaveTargetMismatch}})
	if counter.Class != resultInvalid || counter.Name != valueResultCounterspellInvalidSave || len(counter.Events) != 0 {
		t.Errorf("Counterspell invalid = %+v, want class invalid, result %s and no event [FR-056]", counter, valueResultCounterspellInvalidSave)
	}
	dispel := describeDispel(DispelOngoingSpellsInvalid{problem: DispelPlanMismatch})
	if dispel.Class != resultInvalid || dispel.Name != valueResultDispelInvalidChecks {
		t.Errorf("Dispel Magic invalid = %+v, want class invalid and result %s [FR-056]", dispel, valueResultDispelInvalidChecks)
	}
	ineligible := describeCounterCast(CounterCastSpellIneligible{})
	if ineligible.Class != resultRefused || ineligible.Name != valueResultCounterspellIneligible {
		t.Errorf("Counterspell ineligible = %+v, want class refused [FR-056]", ineligible)
	}
}
