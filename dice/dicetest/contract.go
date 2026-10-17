package dicetest

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// reporter is the part of testing.TB the checks use, so the checks can be
// tested against a recording double.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
}

// contractExpressions are the requests every roller must answer.
func contractExpressions() map[string]dice.Expression {
	return map[string]dice.Expression{
		"one d20":            dice.MustExpression(1, dice.D20(), 0),
		"two d8 plus 2":      dice.MustExpression(2, dice.D8(), 2),
		"four d6":            dice.MustExpression(4, dice.D6(), 0),
		"one d3":             dice.MustExpression(1, dice.D3(), 0),
		"one d100":           dice.MustExpression(1, dice.D100(), 0),
		"negative total":     dice.MustExpression(1, dice.D4(), -5),
		"smallest die":       dice.MustExpression(3, dice.MustDie(2), 0),
		"largest expression": dice.MustExpression(dice.MaxCount, dice.MustDie(dice.MaxSides), dice.MaxModifier),
	}
}

// RunRollerContract runs the dice.Roller success contract against rollers made
// by newRoller, one fresh roller per subtest. newRoller must return a roller
// that can answer any valid expression (so not a spent FixedRoller).
//
// It checks, for a range of expressions including the extremes, that Roll
// returns no error, answers the same expression, returns Count() rolls of the
// requested die with faces in 1..Sides(), and keeps the total between Minimum
// and Maximum; and that successive calls each answer their own expression.
func RunRollerContract(t *testing.T, newRoller func(t *testing.T) dice.Roller) {
	t.Helper()
	for name, e := range contractExpressions() {
		t.Run(name, func(t *testing.T) {
			checkAnswers(t, newRoller(t), e)
		})
	}
	t.Run("successive calls", func(t *testing.T) {
		r := newRoller(t)
		for _, e := range []dice.Expression{
			dice.MustExpression(2, dice.D6(), 0),
			dice.MustExpression(1, dice.D20(), 3),
			dice.MustExpression(2, dice.D6(), 0),
		} {
			checkAnswers(t, r, e)
		}
	})
}

func checkAnswers(t reporter, r dice.Roller, e dice.Expression) {
	t.Helper()
	res, err := r.Roll(e)
	if err != nil {
		t.Errorf("Roll(%s): unexpected error: %v", e, err)
		return
	}
	if res.Expression() != e {
		t.Errorf("Roll(%s) answered %s", e, res.Expression())
		return
	}
	rolls := res.Rolls()
	if len(rolls) != e.Count() {
		t.Errorf("Roll(%s) returned %d rolls", e, len(rolls))
	}
	for _, roll := range rolls {
		if roll.Die() != e.Die() || roll.Value() < 1 || roll.Value() > e.Die().Sides() {
			t.Errorf("Roll(%s) returned bad roll %s", e, roll)
		}
	}
	if total := res.Total(); total < e.Minimum() || total > e.Maximum() {
		t.Errorf("Roll(%s) total %d outside %d..%d", e, total, e.Minimum(), e.Maximum())
	}
}

// RunFailureContract runs the dice.Roller failure contract against rollers made
// by newFailing, which must return a roller whose backend fails with exactly
// the given cause (by returning it or panicking with it).
//
// It checks, for every contract expression, that Roll returns the zero Result
// and an error that is a *srderr.PortError for dice.Port, still matches the
// cause with errors.Is, and has srderr.ErrorType "dice_roller_error".
func RunFailureContract(t *testing.T, newFailing func(cause error) dice.Roller) {
	t.Helper()
	for name, e := range contractExpressions() {
		t.Run(name, func(t *testing.T) {
			checkWrapsFailure(t, newFailing, e)
		})
	}
}

func checkWrapsFailure(t reporter, newFailing func(cause error) dice.Roller, e dice.Expression) {
	t.Helper()
	cause := errors.New("backend failure")
	res, err := newFailing(cause).Roll(e)
	if !res.IsZero() {
		t.Errorf("Roll(%s) returned partial result %s with failure", e, res)
	}
	if err == nil {
		t.Errorf("Roll(%s) returned no error", e)
		return
	}
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port {
		t.Errorf("Roll(%s) error %v is not a PortError for %q", e, err, dice.Port)
	}
	if !errors.Is(err, cause) {
		t.Errorf("Roll(%s) error %v lost the cause", e, err)
	}
	if got := srderr.ErrorType(err); got != "dice_roller_error" {
		t.Errorf("ErrorType = %q, want dice_roller_error", got)
	}
}
