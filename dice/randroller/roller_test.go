package randroller_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/dice/randroller"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func inBounds(t *testing.T, r dice.Roller, e dice.Expression) {
	t.Helper()
	res, err := r.Roll(e)
	if err != nil {
		t.Errorf("Roll(%v): %v", e, err)
		return
	}
	if res.Expression() != e || res.Total() < e.Minimum() || res.Total() > e.Maximum() {
		t.Errorf("Roll(%v) = %v, outside %d..%d", e, res, e.Minimum(), e.Maximum())
	}
}

// Requirement: doc/specs/dice-and-rolls.md FR-088, FR-089 - Roller honours the
//
//	dice.Roller contract suite (FR-045 to FR-047).
//
// Path: happy.
func TestContract(t *testing.T) {
	t.Parallel()
	dicetest.RunRollerContract(t, func(*testing.T) dice.Roller { return randroller.New() })
}

// Requirement: per case
// Path: per case.
func TestRollStaysWithinBounds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		roller                  dice.Roller
		expr                    string
	}{
		{"New, one d20", "doc/specs/dice-and-rolls.md FR-089 - total within Minimum..Maximum", "happy", randroller.New(), "1d20"},
		{"zero value, one d20", "doc/specs/dice-and-rolls.md FR-088 - the zero value is ready to use", "happy", randroller.Roller{}, "1d20"},
		{"modifier", "doc/specs/dice-and-rolls.md FR-089 - the modifier is included in the bounds", "happy", randroller.New(), "2d8+2"},
		{"several dice", "doc/specs/dice-and-rolls.md FR-089 - total within Minimum..Maximum", "happy", randroller.New(), "4d6"},
		{"many dice", "doc/specs/dice-and-rolls.md FR-089 - large expressions stay in bounds", "unhappy (boundary)", randroller.New(), "100d100"},
		{"smallest die", "doc/specs/dice-and-rolls.md FR-089 - the fewest-sided die (MinSides) stays in bounds", "unhappy (boundary)", randroller.New(), "3d2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := dice.MustParseExpression(c.expr)
			for range 500 {
				inBounds(t, c.roller, e)
			}
		})
	}
}

// Requirement: doc/specs/dice-and-rolls.md FR-090 - one Roller is safe to share
//
//	between goroutines (meaningful under -race).
//
// Path: happy.
func TestConcurrentUse(t *testing.T) {
	t.Parallel()
	r := randroller.New()
	e := dice.MustParseExpression("2d8+2")
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				inBounds(t, r, e)
			}
		}()
	}
	wg.Wait()
}

// Requirement: doc/specs/dice-and-rolls.md FR-091 - the zero expression yields
//
//	the zero Result and an invalid-input port error, never a partial result.
//
// Path: unhappy (invalid input).
func TestZeroExpression(t *testing.T) {
	t.Parallel()
	res, err := dice.Roll(randroller.New(), dice.Expression{})
	var pe *srderr.PortError
	if !res.IsZero() || !errors.As(err, &pe) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("Roll(zero) = %v, %v; want zero Result and invalid-input PortError", res, err)
	}
}
