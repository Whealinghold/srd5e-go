package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// envExhaustion is a set holding Exhaustion at level n (no condition for 0).
func envExhaustion(tb testing.TB, n int, others ...combat.Condition) combat.ConditionSet {
	tb.Helper()
	cs := others
	if n > 0 {
		cs = append(cs, condLevel(n))
	}
	return condSet(tb, cs)
}

// envLevel is the Exhaustion level stored in set, 0 when there is none.
func envLevel(set combat.ConditionSet) int {
	for _, c := range set.Explicit() {
		if e, ok := c.(combat.ExhaustionCondition); ok {
			return e.Level.Value()
		}
	}
	return 0
}

// envRecord builds a record, failing the test on a bad one.
func envRecord(tb testing.TB, dehydration, malnutrition, suffocation, days, contagion int) combat.HazardRecord {
	tb.Helper()
	r, err := combat.NewHazardRecord(dehydration, malnutrition, suffocation, days)
	if err == nil {
		r, err = r.WithContagionLevels(contagion)
	}
	if err != nil {
		tb.Fatalf("hazard record: %v", err)
	}
	return r
}

// envKinds lists the Kind of each event.
func envKinds(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// envFailingRoller is a roller whose every roll fails with cause.
func envFailingRoller(cause error) combat.Roller {
	return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })
}

// envPanicRoller is a roller that panics with an error.
func envPanicRoller() combat.Roller {
	return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(errors.New("roller bug")) })
}

// envWrongRoller answers another expression than the one asked.
func envWrongRoller() combat.Roller {
	return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D20(), 0))
	})
}

// envPortFailure asserts err is a dice roller *srderr.PortError.
func envPortFailure(t *testing.T, err error) {
	t.Helper()
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port {
		t.Fatalf("err = %v, want a dice roller *srderr.PortError", err)
	}
}
