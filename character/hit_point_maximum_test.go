package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/derived-stats.md FR-060 - a maximum is at least 1
// (SRD 5.2.1, "Gaining a Level", p. 23); 0 is a *HitPointMaximumError.
// Path: unhappy (boundary).
func TestNewHitPointMaximum(t *testing.T) {
	t.Parallel()
	if m, err := character.NewHitPointMaximum(1); err != nil || m.Value() != 1 || m.IsZero() {
		t.Errorf("NewHitPointMaximum(1) = %v, %v", m, err)
	}
	_, err := character.NewHitPointMaximum(0)
	var he *character.HitPointMaximumError
	if !errors.As(err, &he) || he.Reason != character.HitPointMaximumBelowMinimum || he.Value != 0 || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewHitPointMaximum(0) error = %v", err)
	}
	if he.Error() == "" || he.Reason.String() != "belowMinimum" || character.HitPointMaximumRejection(0).String() != "unknown" {
		t.Error("error text or rejection name wrong")
	}
	if errors.Is(he, &character.HitPointMaximumError{Value: -1, Reason: character.HitPointMaximumBelowMinimum}) {
		t.Error("errors with different values match")
	}
}

// Requirement: doc/specs/derived-stats.md FR-060 - MustHitPointMaximum panics
// exactly when NewHitPointMaximum errors.
// Path: unhappy (invalid input).
func TestMustHitPointMaximum(t *testing.T) {
	t.Parallel()
	if got := recovered(t, func() { character.MustHitPointMaximum(7) }); got != nil {
		t.Errorf("MustHitPointMaximum(7) panicked: %v", got)
	}
	if got := recovered(t, func() { character.MustHitPointMaximum(0) }); !errors.Is(got, srderr.ErrInvalidInput) {
		t.Errorf("MustHitPointMaximum(0) panic = %v", got)
	}
}

// Requirement: doc/specs/derived-stats.md FR-060 - Plus returns a new value and
// never goes below 1.
// Path: unhappy (boundary).
func TestHitPointMaximumPlus(t *testing.T) {
	t.Parallel()
	m := character.MustHitPointMaximum(10)
	if got := m.Plus(3); got.Value() != 13 || m.Value() != 10 {
		t.Errorf("Plus(3) = %d, receiver %d", got.Value(), m.Value())
	}
	if got := m.Plus(-20); got.Value() != 1 {
		t.Errorf("Plus(-20) = %d, want 1", got.Value())
	}
	if got := (character.HitPointMaximum{}).Plus(5); !got.IsZero() {
		t.Errorf("zero.Plus(5) = %d, want zero", got.Value())
	}
}
