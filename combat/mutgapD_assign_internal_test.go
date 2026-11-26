package combat

import "testing"

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - the assignment search tries at most maxAssignNodes picks:
// the pick that is exactly the last one the budget allows still counts, and one more is refused.
// Path: unhappy (boundary).
func TestAssignNodeBudgetBoundary(t *testing.T) {
	t.Parallel()
	options := func(n int) func(int) []int {
		return func(int) []int {
			out := make([]int, n)
			for i := range out {
				out[i] = i
			}
			return out
		}
	}
	onlyLast := func(n int) func([]int) bool {
		return func(picks []int) bool { return picks[len(picks)-1] == n-1 }
	}
	if !assign(1, options(maxAssignNodes), onlyLast(maxAssignNodes)) {
		t.Errorf("assign with the answer on pick %d = false, want true", maxAssignNodes)
	}
	if assign(1, options(maxAssignNodes+1), onlyLast(maxAssignNodes+1)) {
		t.Errorf("assign with the answer on pick %d = true, want false (over the budget)", maxAssignNodes+1)
	}
}
