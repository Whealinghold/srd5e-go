package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// assertInvalidInputCounted checks that a use case call that returned an error
// matching srderr.ErrInvalidInput incremented srd5e.combat.errors exactly once,
// with error.type invalid_input and no other error type.
//
// Requirement: doc/specs/combat-foundations.md FR-032 - an invalid-input error is counted once under errors with error.type invalid_input.
// Path: unhappy (invalid input).
func assertInvalidInputCounted(tb testing.TB, rec *telemetrytest.Recorder, op string) {
	tb.Helper()
	errs := measured(rec, "srd5e.combat.errors")
	if len(errs) != 1 || errs[0]["error.type"] != "invalid_input" || errs[0]["srd5e.combat.operation"] != op {
		tb.Errorf("%s: errors = %v, want one sample with error.type invalid_input", op, errs)
	}
}
