package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-barbarian.md section 7, FR-089, FR-092 - the Barbarian span names and attribute values are a public contract: their strings are pinned here, each span and value appears in doc/observability/signals.md,
// and the srd5e.class and srd5e.feature values and every rejection value in doc/rules/observability.md.
// Path: happy.
func TestBarbarianSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanUseRelentlessRage, "srd5e.character.use_relentless_rage"},
		{character.SpanApplyBarbarianOption, "srd5e.character.apply_barbarian_option"},
		{character.ValueClassBarbarian, "barbarian"},
		{character.ValueFeatureRelentlessRage, "relentless_rage"},
		{character.ValueFeatureBarbarianWeaponMastery, "weapon_mastery"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for r := character.BarbarianNotABarbarian; r <= character.BarbarianAlreadyChosen; r++ {
		if !strings.Contains(string(rules), "`"+r.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", r.String())
		}
	}
}
