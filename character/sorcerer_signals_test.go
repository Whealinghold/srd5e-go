package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-sorcerer.md section 7, FR-124 - the Sorcerer span names and the srd5e.class and srd5e.feature values are a public contract: their strings are pinned here, each span appears in doc/observability/signals.md
// and each value in signals.md or doc/rules/observability.md; every SorcererRejectionKind has its snake_case name in doc/rules/observability.md.
// Path: happy.
func TestSorcererSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanChooseMetamagicOptions, "srd5e.character.choose_metamagic_options"},
		{character.SpanApplySorcererLevelUp, "srd5e.character.apply_sorcerer_level_up"},
		{character.SpanCreateSpellSlot, "srd5e.character.create_spell_slot"},
		{character.SpanUseMetamagic, "srd5e.character.use_metamagic"},
		{character.ValueClassSorcerer, "sorcerer"},
		{character.ValueFeatureMetamagic, "metamagic"},
		{character.ValueFeatureFontOfMagic, "font_of_magic"},
		{character.ValueFeatureSorcererLevelUp, "sorcerer_level_up"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for k := character.SorcererNotASorcerer; k.Valid(); k++ {
		if !strings.Contains(string(rules), "`"+k.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", k.String())
		}
	}
}
