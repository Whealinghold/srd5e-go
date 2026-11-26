package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// wantLimit fails unless err is a combat.LimitExceededError for collection with cap max and matches srderr.ErrInvalidInput.
func wantLimit(tb testing.TB, err error, collection string, max int) {
	tb.Helper()
	var le combat.LimitExceededError
	if !errors.As(err, &le) || le.Collection != collection || le.Max != max {
		tb.Errorf("error = %v, want LimitExceededError{%q, %d}", err, collection, max)
		return
	}
	if !errors.Is(err, srderr.ErrInvalidInput) {
		tb.Errorf("error %v does not match srderr.ErrInvalidInput", err)
	}
}

// Requirement: doc/plans/plan-phase-4.md section 4 P10 - the shared cap error renders the shared limit_exceeded value and matches srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestLimitExceededErrorIsTheSharedValue(t *testing.T) {
	t.Parallel()
	e := combat.LimitExceededError{Collection: "x", Max: 3}
	if got := e.String(); got != "limit_exceeded" {
		t.Errorf("String() = %q, want limit_exceeded", got)
	}
	wantLimit(t, e, "x", 3)
	if e.Error() == "" {
		t.Error("Error() is empty")
	}
}

// runCap runs the two boundary rows of one plan P10 cap: build(max) must return no error and build(max+1) a
// LimitExceededError for collection. The caller's comment names the requirement.
func runCap(t *testing.T, collection string, max int, build func(n int) error) {
	t.Helper()
	t.Run("at the cap", func(t *testing.T) {
		t.Parallel()
		if err := build(max); err != nil {
			t.Errorf("%d %s: error = %v, want none", max, collection, err)
		}
	})
	t.Run("cap plus one", func(t *testing.T) {
		t.Parallel()
		wantLimit(t, build(max+1), collection, max)
	})
}
