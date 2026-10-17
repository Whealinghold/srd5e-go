package dicetest_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func TestFixedRoller(t *testing.T) {
	t.Parallel()
	r := dicetest.NewFixedRoller(1, 2, 3, 20)
	if n, ok := r.Remaining(); n != 4 || !ok {
		t.Errorf("Remaining = %d, %v", n, ok)
	}
	a, err := r.Roll(dice.MustParseExpression("3d6"))
	if err != nil || a.DiceTotal() != 6 {
		t.Fatalf("first = %v, %v", a, err)
	}
	// A bad face fails and consumes nothing.
	if _, err := r.Roll(dice.MustParseExpression("1d6")); !errors.Is(err, dicetest.ErrValueOutOfRange) {
		t.Errorf("bad face: %v", err)
	}
	if _, err := r.Roll(dice.MustParseExpression("2d6")); !errors.Is(err, dicetest.ErrExhausted) {
		t.Errorf("exhausted: %v", err)
	}
	b, err := r.Roll(dice.MustParseExpression("1d20"))
	if err != nil || b.Total() != 20 {
		t.Fatalf("second = %v, %v", b, err)
	}
	if n, _ := r.Remaining(); n != 0 {
		t.Errorf("Remaining = %d", n)
	}
	reqs := r.Requests()
	if len(reqs) != 2 || reqs[0].Notation() != "3d6" || reqs[1].Notation() != "1d20" {
		t.Errorf("Requests = %v", reqs)
	}
	reqs[0] = dice.Expression{}
	if r.Requests()[0].IsZero() {
		t.Error("Requests leaks internal slice")
	}
	if _, err := r.Roll(dice.Expression{}); err == nil {
		t.Error("zero expression accepted")
	}
}

func TestFixedRollerCopiesValues(t *testing.T) {
	t.Parallel()
	vals := []int{4}
	r := dicetest.NewFixedRoller(vals...)
	vals[0] = 1
	if res, _ := r.Roll(dice.MustParseExpression("1d6")); res.Total() != 4 {
		t.Errorf("total = %d", res.Total())
	}
}

func TestExtremeRollers(t *testing.T) {
	t.Parallel()
	e := dice.MustParseExpression("3d8+1")
	lo, _ := dicetest.NewMinimumRoller().Roll(e)
	hi, _ := dicetest.NewMaximumRoller().Roll(e)
	if lo.Total() != e.Minimum() || hi.Total() != e.Maximum() {
		t.Errorf("lo %d hi %d", lo.Total(), hi.Total())
	}
	m := dicetest.NewMaximumRoller()
	m.Roll(e)
	if n, ok := m.Remaining(); ok || n != 0 || len(m.Requests()) != 1 {
		t.Errorf("Remaining = %d, %v; requests %v", n, ok, m.Requests())
	}
}

func TestContractsPassForDoubles(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func() dice.Roller{
		"minimum": func() dice.Roller { return dicetest.NewMinimumRoller() },
		"maximum": func() dice.Roller { return dicetest.NewMaximumRoller() },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dicetest.RunRollerContract(t, func(*testing.T) dice.Roller { return f() })
		})
	}
	dicetest.RunFailureContract(t, func(cause error) dice.Roller {
		return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			return dice.Result{}, srderr.WrapPort(dice.Port, "roll", fmt.Errorf("wrapped: %w", cause))
		})
	})
}
