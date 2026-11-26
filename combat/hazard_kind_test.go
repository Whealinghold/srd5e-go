package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/environment-hazards-and-objects.md FR-001 - HazardKind has exactly five values in alphabetical order, each with its lowercase name.
// Path: happy.
func TestHazardKindHasFiveValuesInOrder(t *testing.T) {
	t.Parallel()
	want := []string{"burning", "dehydration", "falling", "malnutrition", "suffocation"}
	for i, name := range want {
		k := combat.HazardKind(i + 1)
		if !k.Valid() || k.String() != name {
			t.Errorf("HazardKind(%d) = %q valid %v, want %q valid", i+1, k, k.Valid(), name)
		}
	}
	for _, k := range []combat.HazardKind{0, 6, -1} {
		if k.Valid() || k.String() != "unknown" {
			t.Errorf("HazardKind(%d) = %q valid %v, want unknown", int(k), k, k.Valid())
		}
	}
	if combat.HazardBurning != 1 || combat.HazardSuffocation != 5 {
		t.Error("the constants are not in the order of the spec")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md section 3 and 7 - the Exhaustion outcomes are named for the srd5e.combat.result values.
// Path: happy.
func TestHazardExhaustionOutcomeNames(t *testing.T) {
	t.Parallel()
	want := map[combat.HazardExhaustionOutcome]string{combat.HazardNoExhaustion: "no_exhaustion", combat.HazardExhaustionGained: "exhaustion_gained",
		combat.HazardExhaustionAtMaximum: "exhaustion_at_maximum", combat.HazardImmune: "immune", 0: "unknown", 5: "unknown"}
	for o, name := range want {
		if o.String() != name {
			t.Errorf("HazardExhaustionOutcome(%d) = %q, want %q", int(o), o, name)
		}
	}
}
