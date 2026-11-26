package combat

import "testing"

// Requirement: doc/specs/combat-foundations.md FR-031, FR-039 - applied is a
// success without span error; refused is a failure without span error; invalid
// is a failure with span error; the zero value is not a class.
// Path: per case.
func TestCombatResultClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path, outcome, str string
		class                    combatResultClass
		spanError                bool
	}{
		{"applied", "happy", "success", "applied", resultApplied, false},
		{"refused", "unhappy (boundary)", "failure", "refused", resultRefused, false},
		{"invalid", "unhappy (boundary)", "failure", "invalid", resultInvalid, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.class.Outcome(); got != c.outcome {
				t.Errorf("Outcome() = %q, want %q", got, c.outcome)
			}
			if got := c.class.IsSpanError(); got != c.spanError {
				t.Errorf("IsSpanError() = %v, want %v", got, c.spanError)
			}
			if got := c.class.String(); got != c.str {
				t.Errorf("String() = %q, want %q", got, c.str)
			}
		})
	}
	if combatResultClass(99).IsSpanError() || combatResultClass(99).String() != "unknown" {
		t.Error("a value above the classes is a class")
	}
	var zero combatResultClass
	if zero.String() != "unknown" || zero.IsSpanError() {
		t.Errorf("zero class: String() = %q, IsSpanError() = %v", zero.String(), zero.IsSpanError())
	}
	func() {
		defer func() {
			if _, ok := recover().(error); !ok {
				t.Error("Outcome() of the zero class did not panic with an error")
			}
		}()
		zero.Outcome()
	}()
}
