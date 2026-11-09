package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/species-trait-behaviour.md section 7, FR-065 - the species signal names and operation values are a public contract: their strings are pinned here, each span appears in doc/observability/signals.md, each operation and rejection value in doc/observability/signals.md and doc/rules/observability.md.
// Path: happy.
func TestSpeciesSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	pins := []struct{ got, want string }{
		{character.SpanExpendSpeciesTraitUse, "srd5e.character.expend_species_trait_use"},
		{character.SpanRenewSpeciesTraitUses, "srd5e.character.renew_species_trait_uses"},
		{character.SpanUseRelentlessEndurance, "srd5e.character.use_relentless_endurance"},
		{character.SpanRollHalflingLuck, "srd5e.character.roll_halfling_luck"},
		{character.SpanRollStonesEndurance, "srd5e.character.roll_stones_endurance"},
		{character.SpanReplaceHighElfCantrip, "srd5e.character.replace_high_elf_cantrip"},
		{character.ValueOperationExpendSpeciesTraitUse, "expend_species_trait_use"},
		{character.ValueOperationRenewSpeciesTraitUses, "renew_species_trait_uses"},
		{character.ValueOperationUseRelentlessEndurance, "use_relentless_endurance"},
		{character.ValueOperationRollHalflingLuck, "roll_halfling_luck"},
		{character.ValueOperationRollStonesEndurance, "roll_stones_endurance"},
		{character.ValueOperationReplaceHighElfCantrip, "replace_high_elf_cantrip"},
		{character.SpeciesBehaviourUnknown{}.String(), "species_behaviour_unknown"},
		{character.SpeciesChoicesInvalid{}.String(), "species_choices_invalid"},
		{character.SpeciesUseNotOffered{}.String(), "species_use_not_offered"},
		{character.SpeciesUseHasOwnUseCase{}.String(), "species_use_has_own_use_case"},
		{character.SpeciesUseNotYetAvailable{}.String(), "species_use_not_yet_available"},
		{character.NotEnoughSpaceForLargeForm{}.String(), "not_enough_space_for_large_form"},
		{character.SpeciesUseExhausted{}.String(), "species_use_exhausted"},
		{character.CantripWindowClosed{}.String(), "cantrip_window_closed"},
		{character.DieAlreadyRerolled{}.String(), "die_already_rerolled"},
		{character.RestNotForCharacter{}.String(), "character_mismatch"},
	}
	for _, p := range pins {
		if p.got != p.want {
			t.Errorf("constant = %q, want %q", p.got, p.want)
		}
		if !strings.Contains(string(catalogue), "`"+p.want+"`") {
			t.Errorf("signals.md does not list %q", p.want)
		}
		if !strings.HasPrefix(p.want, "srd5e.") && !strings.Contains(string(rules), "`"+p.want+"`") {
			t.Errorf("observability.md does not list %q", p.want)
		}
	}
}
