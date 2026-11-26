package combat

import "testing"

// Requirement: plan-phase-4.md P10 (cascade step budget 1000 per call), doc/specs/combat-action-resolution.md section 1 (Limits) - a call may take 1000 steps and no more: at the budget the call is within it, at budget plus one it is over, so the use case answers its Invalid result with the shared limit_exceeded value (as for the 4096 events). A step is a committed transition or a Concentration check; the wiring of this package takes a handful of them in one call, so only the accumulator can be taken to the budget.
// Path: per case.
func TestResolutionStepBudget(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		steps                   int
		over                    bool
	}{
		{"none", "P10 step budget", "unhappy (empty and minimal)", 0, false},
		{"one below the budget", "P10 step budget", "unhappy (boundary)", maxStepsPerCall - 1, false},
		{"at the budget", "P10 step budget", "unhappy (boundary)", maxStepsPerCall, false},
		{"budget plus one", "P10 step budget", "unhappy (boundary)", maxStepsPerCall + 1, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := &resolution{}
			for range c.steps {
				r.concentrated("a", concentration{outcome: ConcentrationHeld{}})
			}
			if r.steps != c.steps || r.overLimit() != c.over {
				t.Errorf("after %d steps: counted %d, over = %v, want over = %v [%s]", c.steps, r.steps, r.overLimit(), c.over, c.requirement)
			}
		})
	}
	if maxStepsPerCall != 1000 {
		t.Errorf("maxStepsPerCall = %d, want 1000 (plan P10)", maxStepsPerCall)
	}
	if (&resolution{events: make([]Event, maxEventsPerCall)}).overLimit() || !(&resolution{events: make([]Event, maxEventsPerCall+1)}).overLimit() {
		t.Errorf("the event cap is not 4096 and not strict [P10]")
	}
}
