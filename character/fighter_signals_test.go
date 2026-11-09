package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-fighter.md section 7, FR-138 - the Fighter span names, the log event and the attribute values are a public contract: their strings are pinned
// here, each span and value appears in doc/observability/signals.md, and the srd5e.class and srd5e.feature values and the rejection values in doc/rules/observability.md.
// Path: happy.
func TestFighterSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanUseSecondWind, "srd5e.character.use_second_wind"},
		{character.SpanUseTacticalMind, "srd5e.character.use_tactical_mind"},
		{character.SpanUseIndomitable, "srd5e.character.use_indomitable"},
		{character.SpanUseActionSurge, "srd5e.character.use_action_surge"},
		{character.SpanApplyFighterOption, "srd5e.character.apply_fighter_option"},
		{character.LogClassFeatureResolved, "srd5e.character.class_feature_resolved"},
		{character.ValueClassFighter, "fighter"},
		{character.ValueFeatureSecondWind, "second_wind"},
		{character.ValueFeatureTacticalMind, "tactical_mind"},
		{character.ValueFeatureActionSurge, "action_surge"},
		{character.ValueFeatureIndomitable, "indomitable"},
		{character.ValueFeatureFightingStyle, "fighting_style"},
		{character.ValueFeatureWeaponMastery, "weapon_mastery"},
		{character.ValueRejectionContentMissing, "content_missing"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for r := character.FighterNotAFighter; r <= character.FighterNotBloodied; r++ {
		if !strings.Contains(string(rules), "`"+r.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", r.String())
		}
	}
}
