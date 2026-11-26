package combat

import (
	"errors"
	"testing"
)

// Requirement: doc/specs/monster-combatant-setup.md FR-048 - the result name of an Invalid result is that of its first problem for the three problems found before a roll, and a problem that cannot arise there is an unreachable branch that panics with an error value.
// Path: unhappy (invalid input).
func TestInvalidResultName(t *testing.T) {
	t.Parallel()
	for p, want := range map[MonsterSetupProblem]string{SizeChoiceRequired: "size_choice_required", SizeNotOffered: "size_not_offered", ContentNumberOutOfRange: "content_number_out_of_range"} {
		if got := invalidResultName(p); got != want {
			t.Errorf("invalidResultName(%v) = %q, want %q", p, got, want)
		}
	}
	for _, p := range []MonsterSetupProblem{RolledExpressionMismatch, SetupArgumentInvalid, 0} {
		func() {
			defer func() {
				if err, ok := recover().(error); !ok || err == nil || errors.Unwrap(err) != nil {
					t.Errorf("invalidResultName(%v) did not panic with an error value", p)
				}
			}()
			invalidResultName(p)
		}()
	}
}
