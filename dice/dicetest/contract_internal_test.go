package dicetest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type recorder struct{ failures []string }

func (r *recorder) Helper() {}
func (r *recorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

var e6 = dice.MustParseExpression("2d6")

func TestCheckAnswersDetectsViolations(t *testing.T) {
	t.Parallel()
	good, _ := NewMaximumRoller().Roll(e6)
	other, _ := NewMaximumRoller().Roll(dice.MustParseExpression("1d6"))
	d8, _ := dice.NewResult(dice.MustParseExpression("1d8"), []dice.DieRoll{dice.MustDieRoll(dice.D8(), 8)})
	for name, tc := range map[string]struct {
		roller dice.Roller
		fails  bool
	}{
		"good":       {CallbackRoller(func(dice.Expression) (dice.Result, error) { return good, nil }), false},
		"error":      {CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, errors.New("x") }), true},
		"other expr": {CallbackRoller(func(dice.Expression) (dice.Result, error) { return other, nil }), true},
		"other die":  {CallbackRoller(func(dice.Expression) (dice.Result, error) { return d8, nil }), true},
	} {
		rec := &recorder{}
		checkAnswers(rec, tc.roller, e6)
		if (len(rec.failures) > 0) != tc.fails {
			t.Errorf("%s: failures %v", name, rec.failures)
		}
	}
}

func TestCheckWrapsFailureDetectsViolations(t *testing.T) {
	t.Parallel()
	good, _ := NewMaximumRoller().Roll(e6)
	for name, tc := range map[string]struct {
		make  func(cause error) dice.Roller
		fails bool
	}{
		"ok": {func(c error) dice.Roller {
			return CallbackRoller(func(dice.Expression) (dice.Result, error) {
				return dice.Result{}, srderr.WrapPort(dice.Port, "roll", c)
			})
		}, false},
		"no error": {func(error) dice.Roller {
			return CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, nil })
		}, true},
		"raw leak": {func(c error) dice.Roller {
			return CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, c })
		}, true},
		"lost cause": {func(error) dice.Roller {
			return CallbackRoller(func(dice.Expression) (dice.Result, error) {
				return dice.Result{}, srderr.WrapPort(dice.Port, "roll", errors.New("other"))
			})
		}, true},
		"partial result": {func(c error) dice.Roller {
			return CallbackRoller(func(dice.Expression) (dice.Result, error) { return good, srderr.WrapPort(dice.Port, "roll", c) })
		}, true},
	} {
		rec := &recorder{}
		checkWrapsFailure(rec, tc.make, e6)
		if (len(rec.failures) > 0) != tc.fails {
			t.Errorf("%s: failures %v", name, rec.failures)
		}
	}
}
