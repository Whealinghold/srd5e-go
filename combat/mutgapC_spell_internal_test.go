package combat

import "testing"

// Requirement: spell-effects.md FR-043 - a cast with no slot level has no slot steps, whatever level the scaling starts above.
// Path: unhappy (boundary).
func TestMutgapCSlotStepsOfACastWithNoSlot(t *testing.T) {
	t.Parallel()
	for _, above := range []int{-2, -1, 0, 1, 5} {
		if got := slotSteps(0, above); got != 0 {
			t.Errorf("slotSteps(0, %d) = %d, want 0", above, got)
		}
	}
}
