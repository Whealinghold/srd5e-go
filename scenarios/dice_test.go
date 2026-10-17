package scenarios_test

import (
	"errors"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// A scripted roller serves 18 and 3 for 2d20 through dice.Roll (SRD 5.2.1,
// "Roll Two D20s", p. 8).
func TestScriptedRollerServesTwoD20(t *testing.T) {
	t.Parallel()
	expr := dice.MustExpression(2, dice.D20(), 0)

	res, err := dice.Roll(dicetest.NewFixedRoller(18, 3), expr)
	if err != nil {
		t.Fatalf("Roll: %v", err)
	}
	if res.Expression() != expr {
		t.Errorf("expression = %v, want %v", res.Expression(), expr)
	}
	if got := []int{res.Rolls()[0].Value(), res.Rolls()[1].Value()}; got[0] != 18 || got[1] != 3 {
		t.Errorf("faces = %v, want [18 3]", got)
	}
	if res.Total() != 21 {
		t.Errorf("total = %d, want 21", res.Total())
	}
}

// Parsed text is rolled by the bundled seeded roller (SRD 5.2.1, "Hit Points",
// p. 255: "2d8 + 2").
func TestParsedTextIsRolledBySeededRoller(t *testing.T) {
	t.Parallel()
	expr, err := dice.ParseExpression("2d8 + 2")
	if err != nil {
		t.Fatalf("ParseExpression: %v", err)
	}
	roller := dice.NewSeededRoller(rand.New(rand.NewPCG(7, 7)))

	for range 50 {
		res, err := dice.Roll(roller, expr)
		if err != nil {
			t.Fatalf("Roll: %v", err)
		}
		if res.Total() < 4 || res.Total() > 18 || res.Total() != res.DiceTotal()+2 {
			t.Fatalf("total %d (dice %d) outside 4..18 or modifier not applied", res.Total(), res.DiceTotal())
		}
	}
}

// The same seed gives the same game.
func TestSeededRollerIsReproducible(t *testing.T) {
	t.Parallel()
	expr := dice.MustExpression(4, dice.D6(), 0)
	roll := func() int {
		res, err := dice.Roll(dice.NewSeededRoller(rand.New(rand.NewPCG(1, 2))), expr)
		if err != nil {
			t.Fatalf("Roll: %v", err)
		}
		return res.Total()
	}
	if a, b := roll(), roll(); a != b {
		t.Fatalf("same seed gave %d then %d", a, b)
	}
}

// A consumer branches on a roller failure with errors.Is and errors.As, using
// only public types.
func TestRollerFailureIsAPortError(t *testing.T) {
	t.Parallel()
	backendDown := errors.New("backend down")
	roller := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, backendDown })

	_, err := dice.Roll(roller, dice.MustExpression(1, dice.D20(), 0))

	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port {
		t.Fatalf("err = %v, want a *srderr.PortError for port %q", err, dice.Port)
	}
	if !errors.Is(err, backendDown) {
		t.Error("the cause is not reachable with errors.Is")
	}
	if got := srderr.ErrorType(err); got != "dice_roller_error" {
		t.Errorf("ErrorType = %q, want dice_roller_error", got)
	}
}

// A roller answering another expression breaks the port contract.
func TestRollerAnsweringAnotherExpressionIsAContractViolation(t *testing.T) {
	t.Parallel()
	other := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewFixedRoller(3).Roll(dice.MustExpression(1, dice.D20(), 1))
	})

	_, err := dice.Roll(other, dice.MustExpression(1, dice.D20(), 0))

	if !errors.Is(err, srderr.ErrContractViolation) {
		t.Fatalf("err = %v, want ErrContractViolation", err)
	}
}

// A roller that panics is recovered and reported, never propagated.
func TestRollerPanicIsRecovered(t *testing.T) {
	t.Parallel()
	roller := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(errors.New("kaboom")) })

	_, err := dice.Roll(roller, dice.MustExpression(1, dice.D20(), 0))

	if !errors.Is(err, srderr.ErrPortPanic) {
		t.Fatalf("err = %v, want ErrPortPanic", err)
	}
}

// Malformed text is rejected with a typed, errors.Is-able error and never
// panics.
func TestMalformedExpressionIsInvalidInput(t *testing.T) {
	t.Parallel()
	for _, text := range []string{"", "d", "2d", "0d6", "2d1", "9999d6", "2d6+", "abc"} {
		_, err := dice.ParseExpression(text)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("ParseExpression(%q) err = %v, want ErrInvalidInput", text, err)
		}
		var ie *dice.InvalidError
		if !errors.As(err, &ie) {
			t.Errorf("ParseExpression(%q) err is not a *dice.InvalidError", text)
		}
	}
}
