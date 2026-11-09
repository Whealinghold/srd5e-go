package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-ranger.md section 7, FR-105 - the Ranger span names and attribute values are a public contract: their strings are pinned here, each span and value appears in doc/observability/signals.md or doc/rules/observability.md,
// and every rejection value (the String of RangerReason) is in doc/rules/observability.md.
// Path: happy.
func TestRangerSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanChooseRangerOption, "srd5e.character.choose_ranger_option"},
		{character.SpanResolveRangerFeatures, "srd5e.character.resolve_ranger_features"},
		{character.SpanUseTireless, "srd5e.character.use_tireless"},
		{character.ValueClassRanger, "ranger"},
		{character.ValueFeatureTireless, "tireless"},
		{character.ValueFeatureRangerFeatures, "ranger_features"},
		{character.ValueFeatureDeftExplorer, "deft_explorer"},
		{character.ValueFeatureRangerExpertise, "expertise"},
		{character.ValueFeatureHuntersPrey, "hunters_prey"},
		{character.ValueFeatureDefensiveTactics, "defensive_tactics"},
		{character.ValueFeatureRangerFeatGrant, "feat_grant"},
		{character.ValueFeatureRangerWeaponMastery, "weapon_mastery"},
		{character.ValueFeatureRangerFightingStyle, "fighting_style"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for r := character.RangerNotARanger; r.Valid(); r++ {
		if !strings.Contains(string(rules), "`"+r.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", r.String())
		}
	}
}
