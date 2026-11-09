package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-warlock.md section 7, FR-148 - the Warlock span names and the srd5e.class and srd5e.feature values are a public contract: their strings are pinned here, each span appears in doc/observability/signals.md and each value in signals.md or doc/rules/observability.md; every WarlockRejectionKind has its snake_case name in doc/rules/observability.md.
// Path: happy.
func TestWarlockSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanChooseWarlockInvocations, "srd5e.character.choose_warlock_invocations"},
		{character.SpanUseDarkOnesOwnLuck, "srd5e.character.use_dark_ones_own_luck"},
		{character.SpanChooseWarlockMysticArcanum, "srd5e.character.choose_mystic_arcanum"},
		{character.SpanChooseWarlockFamiliarForm, "srd5e.character.choose_warlock_familiar_form"},
		{character.SpanConjureBookOfShadows, "srd5e.character.conjure_book_of_shadows"},
		{character.SpanBondPactWeapon, "srd5e.character.bond_pact_weapon"},
		{character.SpanCastInvocationSpell, "srd5e.character.cast_invocation_spell"},
		{character.SpanUseLifedrinker, "srd5e.character.use_lifedrinker"},
		{character.ValueClassWarlock, "warlock"},
		{character.ValueFeatureEldritchInvocations, "eldritch_invocations"},
		{character.ValueFeatureDarkOnesOwnLuck, "dark_ones_own_luck"},
		{character.ValueFeatureMysticArcanum, "mystic_arcanum"},
		{character.ValueFeatureFamiliarForm, "familiar_form"},
		{character.ValueFeatureBookOfShadows, "book_of_shadows"},
		{character.ValueFeaturePactWeapon, "pact_weapon"},
		{character.ValueFeatureInvocationSpell, "invocation_spell"},
		{character.ValueFeatureLifedrinker, "lifedrinker"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for k := character.WarlockNotAWarlock; k.Valid(); k++ {
		if !strings.Contains(string(rules), "`"+k.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", k.String())
		}
	}
}
