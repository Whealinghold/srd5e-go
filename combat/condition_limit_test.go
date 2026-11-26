package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/conditions.md Limits (plan P10) - the set is bounded by the closed kind list and Exhaustion 6: a set holding all
// fifteen kinds with Exhaustion level 6 is accepted, and Apply refuses every kind (Exhaustion at its maximum, Poisoned as immune because
// Petrified implies it, the rest already held) and leaves the set unchanged.
// Path: unhappy (boundary).
func TestFullyLoadedConditionSetRefusesEveryKind(t *testing.T) {
	t.Parallel()
	all := allFifteen()
	all[3] = condLevel(6)
	set, err := combat.NewConditionSet(all, nil)
	if err != nil {
		t.Fatalf("NewConditionSet(all fifteen, Exhaustion 6) = %v, want accepted", err)
	}
	if got := len(set.Explicit()); got != 15 {
		t.Fatalf("Explicit() holds %d conditions, want 15", got)
	}
	for _, c := range all {
		res := set.Apply(idA(), c)
		if _, added := res.(combat.ConditionAdded); added || !res.Set().Equal(set) {
			t.Errorf("Apply(%v) = %#v, want a refusal that leaves the set unchanged", c.Kind(), res)
		}
		_, atMax := res.(combat.ExhaustionAtMaximum)
		if atMax != (c.Kind() == content.ConditionExhaustion) {
			t.Errorf("Apply(%v) = %T, ExhaustionAtMaximum only for Exhaustion", c.Kind(), res)
		}
	}
}
