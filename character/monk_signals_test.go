package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-monk.md section 7, FR-130, FR-147 - the Monk span names and attribute values are a public contract: their strings are pinned here, each span and
// value appears in doc/observability/signals.md or doc/rules/observability.md, and every rejection value in doc/rules/observability.md.
// Path: happy.
func TestMonkSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanUseFocusFeature, "srd5e.character.use_focus_feature"},
		{character.SpanUseUncannyMetabolism, "srd5e.character.use_uncanny_metabolism"},
		{character.SpanResolveDeflectAttacks, "srd5e.character.resolve_deflect_attacks"},
		{character.SpanRedirectDeflectedAttack, "srd5e.character.redirect_deflected_attack"},
		{character.SpanUseWholenessOfBody, "srd5e.character.use_wholeness_of_body"},
		{character.SpanEndQuiveringPalm, "srd5e.character.end_quivering_palm"},
		{character.ValueClassMonk, "monk"},
		{character.ValueFeatureFlurryOfBlows, "flurry_of_blows"},
		{character.ValueFeaturePatientDefense, "patient_defense"},
		{character.ValueFeatureStepOfTheWind, "step_of_the_wind"},
		{character.ValueFeatureUncannyMetabolism, "uncanny_metabolism"},
		{character.ValueFeatureDeflectAttacks, "deflect_attacks"},
		{character.ValueFeatureDeflectRedirect, "deflect_redirect"},
		{character.ValueFeatureWholenessOfBody, "wholeness_of_body"},
		{character.ValueFeatureQuiveringPalmEnd, "quivering_palm_end"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for r := character.MonkNotAMonk; r <= character.MonkNotEnoughFocusPoints; r++ {
		if !strings.Contains(string(rules), "`"+r.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", r.String())
		}
	}
	for _, f := range []character.MonkFocusFeature{character.MonkFlurryOfBlows, character.MonkPatientDefense, character.MonkStepOfTheWind} {
		if f.String() != map[character.MonkFocusFeature]string{character.MonkFlurryOfBlows: character.ValueFeatureFlurryOfBlows, character.MonkPatientDefense: character.ValueFeaturePatientDefense, character.MonkStepOfTheWind: character.ValueFeatureStepOfTheWind}[f] {
			t.Errorf("feature %v does not match its srd5e.feature value", f)
		}
	}
}
