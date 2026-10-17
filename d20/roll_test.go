package d20_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func TestNewRoll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		mode        modifier.RollMode
		dice        []int
		wantNatural int
		wantKept    int
	}{
		{"normal", modifier.Normal, []int{10}, 10, 0},
		{"advantage keeps higher", modifier.WithAdvantage, []int{3, 18}, 18, 1},
		{"advantage first higher", modifier.WithAdvantage, []int{18, 3}, 18, 0},
		{"disadvantage keeps lower", modifier.WithDisadvantage, []int{18, 3}, 3, 1},
		{"equal dice keep index 0", modifier.WithAdvantage, []int{7, 7}, 7, 0},
		{"lowest face", modifier.Normal, []int{1}, 1, 0},
		{"highest face", modifier.Normal, []int{20}, 20, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, err := d20.NewRoll(tc.mode, tc.dice)
			if err != nil {
				t.Fatal(err)
			}
			if r.Natural() != tc.wantNatural || r.KeptIndex() != tc.wantKept || r.Mode() != tc.mode || r.IsZero() {
				t.Errorf("roll = natural %d kept %d mode %v zero %v", r.Natural(), r.KeptIndex(), r.Mode(), r.IsZero())
			}
			if !slices.Equal(r.Dice(), tc.dice) {
				t.Errorf("Dice() = %v, want %v", r.Dice(), tc.dice)
			}
		})
	}
}

func TestNewRollRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mode modifier.RollMode
		dice []int
	}{
		{"no dice", modifier.Normal, nil},
		{"too many for normal", modifier.Normal, []int{1, 2}},
		{"too few for advantage", modifier.WithAdvantage, []int{5}},
		{"three dice", modifier.WithDisadvantage, []int{1, 2, 3}},
		{"face zero", modifier.Normal, []int{0}},
		{"face 21", modifier.WithAdvantage, []int{5, 21}},
		{"negative face", modifier.Normal, []int{-1}},
		{"undefined mode", modifier.RollMode(9), []int{5}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, err := d20.NewRoll(tc.mode, tc.dice)
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
			if !r.IsZero() {
				t.Error("roll is not zero on error")
			}
		})
	}
}

func TestRollIsAValue(t *testing.T) {
	t.Parallel()
	src := []int{4, 9}
	r := d20.MustRoll(modifier.WithAdvantage, src...)
	src[0] = 20
	r.Dice()[1] = 1
	if !slices.Equal(r.Dice(), []int{4, 9}) {
		t.Errorf("Dice() = %v after mutating inputs", r.Dice())
	}
	if r != d20.MustRoll(modifier.WithAdvantage, 4, 9) {
		t.Error("equal rolls differ under ==")
	}
	if r == d20.MustRoll(modifier.WithDisadvantage, 4, 9) {
		t.Error("rolls of different modes are equal")
	}
	var zero d20.Roll
	if !zero.IsZero() || len(zero.Dice()) != 0 || zero.KeptIndex() != 0 {
		t.Error("zero roll is not empty")
	}
}

func TestMustRollPanicsWithAnError(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("recovered %v, want an ErrInvalidInput error", r)
		}
	}()
	d20.MustRoll(modifier.Normal, 0)
}

func TestReplaceDie(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		roll        d20.Roll
		index, face int
		wantDice    []int
		wantNatural int
	}{
		{"normal", d20.MustRoll(modifier.Normal, 10), 0, 3, []int{3}, 3},
		{"advantage replaces the kept die", d20.MustRoll(modifier.WithAdvantage, 4, 18), 1, 2, []int{4, 2}, 4},
		{"advantage replaces the dropped die", d20.MustRoll(modifier.WithAdvantage, 4, 18), 0, 19, []int{19, 18}, 19},
		{"disadvantage", d20.MustRoll(modifier.WithDisadvantage, 4, 18), 0, 20, []int{20, 18}, 18},
		{"lower than the old value is kept", d20.MustRoll(modifier.Normal, 15), 0, 1, []int{1}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			before := tc.roll.Dice()
			got, err := tc.roll.ReplaceDie(tc.index, tc.face)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got.Dice(), tc.wantDice) || got.Natural() != tc.wantNatural || got.Mode() != tc.roll.Mode() {
				t.Errorf("got dice %v natural %d mode %v", got.Dice(), got.Natural(), got.Mode())
			}
			if !slices.Equal(tc.roll.Dice(), before) {
				t.Error("original roll changed")
			}
		})
	}
}

func TestReplaceDieRejects(t *testing.T) {
	t.Parallel()
	adv := d20.MustRoll(modifier.WithAdvantage, 4, 18)
	tests := []struct {
		name        string
		roll        d20.Roll
		index, face int
		want        d20.Reason
	}{
		{"negative index", adv, -1, 5, d20.ReasonIndexOutOfRange},
		{"index past end", adv, 2, 5, d20.ReasonIndexOutOfRange},
		{"index 1 on a normal roll", d20.MustRoll(modifier.Normal, 5), 1, 5, d20.ReasonIndexOutOfRange},
		{"zero roll", d20.Roll{}, 0, 5, d20.ReasonIndexOutOfRange},
		{"face 0", adv, 0, 0, d20.ReasonFaceOutOfRange},
		{"face 21", adv, 1, 21, d20.ReasonFaceOutOfRange},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.roll.ReplaceDie(tc.index, tc.face)
			if !errors.Is(err, &d20.InvalidError{Reason: tc.want}) {
				t.Errorf("err = %v, want reason %v", err, tc.want)
			}
		})
	}
}

func TestApplyReroll(t *testing.T) {
	t.Parallel()
	adv := d20.MustRoll(modifier.WithAdvantage, 4, 18)
	tests := []struct {
		name     string
		decision d20.RerollDecision
		want     []int
		wantErr  d20.Reason
	}{
		{"keep", d20.KeepRoll{}, []int{4, 18}, 0},
		{"reroll first", d20.RerollDie{Index: 0, Face: 19}, []int{19, 18}, 0},
		{"reroll second lower", d20.RerollDie{Index: 1, Face: 1}, []int{4, 1}, 0},
		{"bad index", d20.RerollDie{Index: 2, Face: 5}, nil, d20.ReasonIndexOutOfRange},
		{"bad face", d20.RerollDie{Index: 0, Face: 0}, nil, d20.ReasonFaceOutOfRange},
		{"nil decision", nil, nil, d20.ReasonUnknownDecision},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := adv.ApplyReroll(tc.decision)
			if tc.wantErr != 0 {
				if !errors.Is(err, &d20.InvalidError{Reason: tc.wantErr}) {
					t.Errorf("err = %v, want reason %v", err, tc.wantErr)
				}
				return
			}
			if err != nil || !slices.Equal(got.Dice(), tc.want) {
				t.Errorf("got %v, %v", got.Dice(), err)
			}
		})
	}
}
