package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/species-trait-behaviour.md section 5 (Failures), section 7 (srd5e.character.rejection) - each rejection variant is a comparable value whose String is the snake_case of its name, the attribute value, and the variants are all different.
// Path: per case.
func TestSpeciesTraitRejectionStrings(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for want, r := range map[string]character.SpeciesTraitRejection{
		"species_behaviour_unknown":       character.SpeciesBehaviourUnknown{},
		"species_choices_invalid":         character.SpeciesChoicesInvalid{},
		"species_use_not_offered":         character.SpeciesUseNotOffered{},
		"species_use_has_own_use_case":    character.SpeciesUseHasOwnUseCase{},
		"species_use_not_yet_available":   character.SpeciesUseNotYetAvailable{MinimumLevel: 5},
		"not_enough_space_for_large_form": character.NotEnoughSpaceForLargeForm{},
		"species_use_exhausted":           character.SpeciesUseExhausted{},
		"cantrip_window_closed":           character.CantripWindowClosed{},
		"die_already_rerolled":            character.DieAlreadyRerolled{Index: 1},
		"character_mismatch":              character.RestNotForCharacter{},
	} {
		if r.String() != want || seen[want] {
			t.Errorf("%T.String() = %q, want %q", r, r.String(), want)
		}
		seen[want] = true
	}
	a, b := character.SpeciesUseNotYetAvailable{MinimumLevel: 3}, character.SpeciesUseNotYetAvailable{MinimumLevel: 5}
	if a == b {
		t.Error("the minimum level is not part of the value")
	}
}
