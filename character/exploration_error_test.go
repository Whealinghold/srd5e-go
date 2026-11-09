package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/exploration.md section 5 (Failures) - the error matches
// srderr.ErrInvalidInput, matches by Input only, and every input has a name.
// Path: unhappy (invalid input).
func TestExplorationError(t *testing.T) {
	t.Parallel()
	a := &character.ExplorationError{Input: character.ExplorationNoLight}
	if !errors.Is(a, srderr.ErrInvalidInput) || !errors.Is(a, &character.ExplorationError{Input: character.ExplorationNoLight}) ||
		errors.Is(a, &character.ExplorationError{Input: character.ExplorationLightRadius}) {
		t.Error("ExplorationError must match by Input and wrap ErrInvalidInput")
	}
	if a.Error() == "" {
		t.Error("Error() is empty")
	}
	var nilErr *character.ExplorationError
	if a.Is(nilErr) || nilErr.Is(a) || a.Is(srderr.ErrInvalidInput) {
		t.Error("Is must reject nil and foreign targets")
	}
	want := []string{"vehicle_speed", "light_radius", "no_light", "light_shape", "light_distance", "fuel_kind", "fuel_remaining", "burn_duration", "character_id"}
	for i, w := range want {
		if got := character.ExplorationInput(i + 1).String(); got != w {
			t.Errorf("input %d = %q, want %q", i+1, got, w)
		}
	}
	if character.ExplorationInput(0).String() != "unknown" || (character.ExplorationCharacterID+1).String() != "unknown" {
		t.Error("undefined inputs must be unknown")
	}
}
