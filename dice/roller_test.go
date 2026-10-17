package dice_test

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type panicSource struct{ cause error }

func (p panicSource) Uint64() uint64 { panic(p.cause) }

func TestSeededRollerContract(t *testing.T) {
	t.Parallel()
	dicetest.RunRollerContract(t, func(*testing.T) dice.Roller {
		return dice.NewSeededRoller(rand.New(rand.NewPCG(1, 2)))
	})
}

func TestSeededRollerFailureContract(t *testing.T) {
	t.Parallel()
	dicetest.RunFailureContract(t, func(cause error) dice.Roller {
		return dice.NewSeededRoller(rand.New(panicSource{cause}))
	})
}

func TestSeededRollerDeterministic(t *testing.T) {
	t.Parallel()
	e := dice.MustParseExpression("10d20")
	a, _ := dice.NewSeededRoller(rand.New(rand.NewPCG(7, 7))).Roll(e)
	b, _ := dice.NewSeededRoller(rand.New(rand.NewPCG(7, 7))).Roll(e)
	if !a.Equal(b) {
		t.Error("same seed gave different rolls")
	}
}

func TestSeededRollerEdges(t *testing.T) {
	t.Parallel()
	res, err := dice.NewSeededRoller(nil).Roll(dice.MustParseExpression("1d6"))
	var pe *srderr.PortError
	if !errors.As(err, &pe) || !errors.Is(err, srderr.ErrInvalidInput) || !res.IsZero() {
		t.Errorf("nil rng: %v, %v", res, err)
	}
	_, err = dice.NewSeededRoller(rand.New(rand.NewPCG(1, 1))).Roll(dice.Expression{})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero expression: %v", err)
	}
}

func TestRoll(t *testing.T) {
	t.Parallel()
	e := dice.MustParseExpression("2d6")
	other := dice.MustParseExpression("1d6")
	cause := errors.New("boom")

	got, err := dice.Roll(dicetest.NewFixedRoller(3, 4), e)
	if err != nil || got.Total() != 7 {
		t.Fatalf("good roll = %v, %v", got, err)
	}

	for name, tc := range map[string]struct {
		roller dice.Roller
		is     error
	}{
		"error":     {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), cause},
		"panic":     {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(cause) }), srderr.ErrPortPanic},
		"wrong":     {dicetest.NewMaximumRoller(), srderr.ErrContractViolation},
		"zero":      {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, nil }), srderr.ErrContractViolation},
		"nil":       {nil, srderr.ErrInvalidInput},
		"exhausted": {dicetest.NewFixedRoller(), dicetest.ErrExhausted},
	} {
		req := e
		if name == "wrong" {
			// maximum roller answers faithfully; make a roller that answers another expression
			tc.roller = dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dicetest.NewMaximumRoller().Roll(other) })
		}
		res, err := dice.Roll(tc.roller, req)
		var pe *srderr.PortError
		if !res.IsZero() || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, tc.is) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
}

func TestSeededRollerDistribution(t *testing.T) {
	t.Parallel()
	for _, sides := range []int{6, 20} {
		e := dice.MustParseExpression("1000d" + map[int]string{6: "6", 20: "20"}[sides])
		res, err := dice.NewSeededRoller(rand.New(rand.NewPCG(3, 4))).Roll(e)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[int]bool{}
		for _, r := range res.Rolls() {
			if r.Value() < 1 || r.Value() > sides {
				t.Fatalf("d%d rolled %d", sides, r.Value())
			}
			seen[r.Value()] = true
		}
		if len(seen) != sides || !seen[1] || !seen[sides] {
			t.Errorf("d%d saw %d distinct faces, want all %d", sides, len(seen), sides)
		}
	}
}

func TestSeededRollerGolden(t *testing.T) {
	t.Parallel()
	res, err := dice.NewSeededRoller(rand.New(rand.NewPCG(1, 2))).Roll(dice.MustParseExpression("8d20"))
	if err != nil {
		t.Fatal(err)
	}
	var got []int
	for _, r := range res.Rolls() {
		got = append(got, r.Value())
	}
	want := []int{16, 13, 16, 16, 5, 1, 10, 9}
	if !slices.Equal(got, want) {
		t.Errorf("faces = %v, want %v", got, want)
	}
}
