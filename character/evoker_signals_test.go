package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-wizard-evoker.md section 7, class-wizard.md FR-098 - the six Evoker span names and the srd5e.feature and rejection values are a public contract: their strings are pinned here, each span appears
// in doc/observability/signals.md, each feature value in signals.md or doc/rules/observability.md, and every Evoker rejection name in doc/rules/observability.md.
// Path: happy.
func TestEvokerSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ got, want string }{
		{character.SpanAddEvocationSavantSpells, "srd5e.character.add_evocation_savant_spells"},
		{character.SpanAddEvokerFreeSpell, "srd5e.character.add_evoker_free_spell"},
		{character.SpanResolvePotentCantrip, "srd5e.character.resolve_potent_cantrip"},
		{character.SpanResolveSculptSpells, "srd5e.character.resolve_sculpt_spells"},
		{character.SpanResolveEmpoweredEvocation, "srd5e.character.resolve_empowered_evocation"},
		{character.SpanUseOverchannel, "srd5e.character.use_overchannel"},
		{character.ValueFeatureEvocationSavant, "evocation_savant"},
		{character.ValueFeatureEvokerFreeSpell, "evoker_free_spell"},
		{character.ValueFeaturePotentCantrip, "potent_cantrip"},
		{character.ValueFeatureSculptSpells, "sculpt_spells"},
		{character.ValueFeatureEmpoweredEvocation, "empowered_evocation"},
		{character.ValueFeatureOverchannel, "overchannel"},
		{character.ValueRejectionSubclassUnknown, "subclass_unknown"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for _, kinds := range evkKindNames() {
		for _, name := range kinds {
			if !strings.Contains(string(rules), "`"+name+"`") {
				t.Errorf("rejection value %q is not in observability.md", name)
			}
		}
	}
}
