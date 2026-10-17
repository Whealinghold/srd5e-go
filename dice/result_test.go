package dice_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
)

func rolls(die dice.Die, faces ...int) []dice.DieRoll {
	out := make([]dice.DieRoll, len(faces))
	for i, f := range faces {
		out[i] = dice.MustDieRoll(die, f)
	}
	return out
}

func TestResult(t *testing.T) {
	t.Parallel()
	e := dice.MustParseExpression("4d6+2")
	in := rolls(dice.D6(), 3, 6, 1, 4)
	r, err := dice.NewResult(e, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.DiceTotal() != 14 || r.Modifier() != 2 || r.Total() != 16 || r.Expression() != e || r.IsZero() {
		t.Errorf("result %v", r)
	}
	if r.String() != "4d6+2 [d6=3 d6=6 d6=1 d6=4]" {
		t.Errorf("String = %q", r)
	}
	in[0] = dice.MustDieRoll(dice.D6(), 6) // caller mutation must not leak in
	out := r.Rolls()
	out[1] = dice.MustDieRoll(dice.D6(), 1) // nor out
	if r.DiceTotal() != 14 || r.Rolls()[0].Value() != 3 || r.Rolls()[1].Value() != 6 {
		t.Error("Result is not isolated from caller slices")
	}

	for _, tc := range []struct{ n, high, low int }{{1, 6, 1}, {2, 10, 4}, {3, 13, 8}, {4, 14, 14}} {
		if h, err := r.SumOfHighest(tc.n); err != nil || h != tc.high {
			t.Errorf("SumOfHighest(%d) = %d, %v", tc.n, h, err)
		}
		if l, err := r.SumOfLowest(tc.n); err != nil || l != tc.low {
			t.Errorf("SumOfLowest(%d) = %d, %v", tc.n, l, err)
		}
	}
	for _, n := range []int{0, 5, -1} {
		if _, err := r.SumOfHighest(n); reasonOf(t, err) != dice.ReasonSelectionOutOfRange {
			t.Errorf("SumOfHighest(%d): %v", n, err)
		}
		if _, err := r.SumOfLowest(n); reasonOf(t, err) != dice.ReasonSelectionOutOfRange {
			t.Errorf("SumOfLowest(%d): %v", n, err)
		}
	}
}

func TestNewResultRejects(t *testing.T) {
	t.Parallel()
	e := dice.MustParseExpression("2d6")
	for name, tc := range map[string]struct {
		e    dice.Expression
		r    []dice.DieRoll
		want dice.Reason
	}{
		"zero expression": {dice.Expression{}, nil, dice.ReasonCountOutOfRange},
		"too few":         {e, rolls(dice.D6(), 1), dice.ReasonRollCountMismatch},
		"too many":        {e, rolls(dice.D6(), 1, 2, 3), dice.ReasonRollCountMismatch},
		"wrong die":       {e, rolls(dice.D8(), 1, 2), dice.ReasonDieMismatch},
		"zero roll":       {e, make([]dice.DieRoll, 2), dice.ReasonDieMismatch},
	} {
		r, err := dice.NewResult(tc.e, tc.r)
		if err == nil || reasonOf(t, err) != tc.want || !r.IsZero() {
			t.Errorf("%s: %v, %v", name, r, err)
		}
	}
}

func TestResultEqual(t *testing.T) {
	t.Parallel()
	e := dice.MustParseExpression("2d6")
	a, _ := dice.NewResult(e, rolls(dice.D6(), 1, 2))
	b, _ := dice.NewResult(e, rolls(dice.D6(), 1, 2))
	c, _ := dice.NewResult(e, rolls(dice.D6(), 2, 1))
	d, _ := dice.NewResult(dice.MustParseExpression("2d6+1"), rolls(dice.D6(), 1, 2))
	if !a.Equal(b) || a.Equal(c) || a.Equal(d) {
		t.Error("Equal wrong")
	}
}
