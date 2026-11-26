package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: per case
// Path: per case.
func TestUnarmedEffectsPlans(t *testing.T) {
	t.Parallel()
	a := cid("a")
	cases := []struct {
		name, requirement, path string
		plan                    combat.UnarmedEffectPlan
		condition               combat.Condition
		push                    bool
	}{
		{"grapple, save failed", resSpec + "FR-046, 4.7.1", "happy", (combat.UnarmedEffects{}).Grapple(a, true), combat.Grappled{Grappler: a}, false},
		{"grapple, save made", resSpec + "FR-048, 4.7.1", "happy", (combat.UnarmedEffects{}).Grapple(a, false), nil, false},
		{"shove to Prone, save failed", resSpec + "FR-047, 4.7.1", "happy", (combat.UnarmedEffects{}).Shove(combat.GiveProne, true), combat.Prone{}, false},
		{"shove 5 feet, save failed", resSpec + "FR-047, 4.7.1", "happy", (combat.UnarmedEffects{}).Shove(combat.PushFiveFeet, true), nil, true},
		{"shove, save made", resSpec + "FR-048", "happy", (combat.UnarmedEffects{}).Shove(combat.GiveProne, false), nil, false},
		{"shove with no effect", resSpec + "FR-047", "unhappy (invalid input)", (combat.UnarmedEffects{}).Shove(combat.ShoveEffect(0), true), nil, false},
		{"zero plan", resSpec + "FR-048", "unhappy (empty and minimal)", combat.UnarmedEffectPlan{}, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cond, has := c.plan.Condition()
			if cond != c.condition || has != (c.condition != nil) || c.plan.Push() != c.push {
				t.Errorf("plan = %v, %v, push %v, want %v, push %v [%s]", cond, has, c.plan.Push(), c.condition, c.push, c.requirement)
			}
		})
	}
}
