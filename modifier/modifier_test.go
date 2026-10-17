package modifier_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func ints(vs ...int) []modifier.Modifier {
	out := make([]modifier.Modifier, len(vs))
	for i, v := range vs {
		out[i] = modifier.New(v)
	}
	return out
}

func TestSum(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []int
		want int
	}{
		{"empty", nil, 0},
		{"one bonus", []int{3}, 3},
		{"two bonuses", []int{2, 3}, 5},
		{"penalty", []int{-2}, -2},
		{"net positive", []int{3, -2}, 1},
		{"net zero", []int{2, -2}, 0},
		{"net negative", []int{1, -4}, -3},
		{"reordered", []int{-4, 1}, -3},
		{"zero among others", []int{0, 2}, 2},
		{"large", []int{1_000_000, 1_000_000}, 2_000_000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := ints(tc.in...)
			before := slices.Clone(in)
			if got := modifier.Sum(in...).Value(); got != tc.want {
				t.Errorf("Sum = %d, want %d", got, tc.want)
			}
			if !slices.Equal(in, before) {
				t.Error("Sum altered its input")
			}
		})
	}
}

func TestApplyToAndAdd(t *testing.T) {
	t.Parallel()
	tests := []struct{ d20, mod, want int }{
		{1, -5, -4}, {20, 10, 30}, {10, 0, 10}, {0, 2, 2}, {25, -3, 22},
		{20, 1_000_000, 1_000_020}, {1, -1_000_000, -999_999},
	}
	for _, tc := range tests {
		if got := modifier.New(tc.mod).ApplyTo(tc.d20); got != tc.want {
			t.Errorf("%d with %d = %d, want %d", tc.d20, tc.mod, got, tc.want)
		}
	}
	var zero modifier.Modifier
	m := modifier.New(4)
	if zero.Value() != 0 || m.Add(zero) != m || m.Add(modifier.New(-1)) != modifier.New(3) {
		t.Error("Add / zero value")
	}
}

func TestModifierString(t *testing.T) {
	t.Parallel()
	for v, w := range map[int]string{3: "+3", 0: "+0", -2: "-2"} {
		if g := modifier.New(v).String(); g != w {
			t.Errorf("%d -> %q, want %q", v, g, w)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	a, d := modifier.Advantage{}, modifier.Disadvantage{}
	rep := func(c modifier.RollCircumstance, n int) []modifier.RollCircumstance {
		return slices.Repeat([]modifier.RollCircumstance{c}, n)
	}
	tests := []struct {
		adv, dis int
		want     modifier.RollMode
	}{
		{0, 0, modifier.Normal}, {1, 0, modifier.WithAdvantage}, {0, 1, modifier.WithDisadvantage},
		{2, 0, modifier.WithAdvantage}, {5, 0, modifier.WithAdvantage},
		{0, 2, modifier.WithDisadvantage}, {0, 5, modifier.WithDisadvantage},
		{1, 1, modifier.Normal}, {3, 1, modifier.Normal}, {1, 3, modifier.Normal}, {4, 4, modifier.Normal},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%da%dd", tc.adv, tc.dis), func(t *testing.T) {
			t.Parallel()
			cs := append(rep(a, tc.adv), rep(d, tc.dis)...)
			if got := modifier.Resolve(cs...); got != tc.want {
				t.Errorf("Resolve = %v, want %v", got, tc.want)
			}
			slices.Reverse(cs)
			if got := modifier.Resolve(cs...); got != tc.want {
				t.Errorf("reversed Resolve = %v, want %v", got, tc.want)
			}
		})
	}
	if modifier.Resolve(nil, a) != modifier.WithAdvantage || modifier.Resolve() != modifier.Normal {
		t.Error("nil entries must be ignored")
	}
	if modifier.Resolve(&modifier.Advantage{}, &modifier.Disadvantage{}) != modifier.Normal ||
		modifier.Resolve(&modifier.Advantage{}) != modifier.Normal {
		t.Error("pointer circumstances must be ignored")
	}
	if a != (modifier.Advantage{}) || modifier.RollCircumstance(a) == modifier.RollCircumstance(d) {
		t.Error("circumstance equality")
	}
	if a.String() != "Advantage" || d.String() != "Disadvantage" {
		t.Error("circumstance strings")
	}
}

func TestRollModeBasics(t *testing.T) {
	t.Parallel()
	var zero modifier.RollMode
	if zero != modifier.Normal {
		t.Error("zero RollMode must be Normal")
	}
	for m, w := range map[modifier.RollMode][2]any{
		modifier.Normal: {"normal", 1}, modifier.WithAdvantage: {"advantage", 2}, modifier.WithDisadvantage: {"disadvantage", 2},
	} {
		if m.String() != w[0] || m.DiceCount() != w[1] {
			t.Errorf("%v: %s %d", m, m.String(), m.DiceCount())
		}
		got, err := modifier.ParseRollMode(strings.ToUpper(m.String()))
		if err != nil || got != m {
			t.Errorf("ParseRollMode(%v) = %v, %v", m, got, err)
		}
	}
	bad := modifier.RollMode(9)
	if bad.String() != "unknown" || bad.DiceCount() != 1 {
		t.Error("undefined mode")
	}
	_, err := modifier.ParseRollMode(strings.Repeat("z", 60))
	if !errors.Is(err, &modifier.InvalidError{Reason: modifier.ReasonUnknownRollMode}) || !strings.Contains(err.Error(), "...") {
		t.Errorf("err = %v", err)
	}
}

func TestSelect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mode   modifier.RollMode
		rolled []int
		want   int
	}{
		{"normal", modifier.Normal, []int{7}, 7},
		{"advantage higher", modifier.WithAdvantage, []int{4, 15}, 15},
		{"advantage swapped", modifier.WithAdvantage, []int{15, 4}, 15},
		{"disadvantage lower", modifier.WithDisadvantage, []int{4, 15}, 4},
		{"disadvantage swapped", modifier.WithDisadvantage, []int{15, 4}, 4},
		{"equal", modifier.WithAdvantage, []int{9, 9}, 9},
		{"extremes advantage", modifier.WithAdvantage, []int{1, 20}, 20},
		{"extremes disadvantage", modifier.WithDisadvantage, []int{1, 20}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(tc.rolled)
			got, err := tc.mode.Select(in)
			if err != nil || got != tc.want || !slices.Equal(in, tc.rolled) {
				t.Errorf("Select = %d, %v", got, err)
			}
		})
	}
}

func TestSelectRejects(t *testing.T) {
	t.Parallel()
	count := &modifier.InvalidError{Reason: modifier.ReasonRollCountMismatch}
	face := &modifier.InvalidError{Reason: modifier.ReasonFaceOutOfRange}
	tests := []struct {
		name         string
		mode         modifier.RollMode
		rolled       []int
		wantCount    bool
		wantFace     bool
		wantUnknown  bool
		wantProblems int
	}{
		{"normal empty", modifier.Normal, nil, true, false, false, 1},
		{"normal two", modifier.Normal, []int{1, 2}, true, false, false, 1},
		{"advantage one", modifier.WithAdvantage, []int{5}, true, false, false, 1},
		{"disadvantage three", modifier.WithDisadvantage, []int{1, 2, 3}, true, false, false, 1},
		{"zero face", modifier.Normal, []int{0}, false, true, false, 1},
		{"21 face", modifier.WithAdvantage, []int{21, 3}, false, true, false, 1},
		{"negative face", modifier.WithDisadvantage, []int{3, -1}, false, true, false, 1},
		{"both problems", modifier.WithAdvantage, []int{0}, true, true, false, 2},
		{"unknown mode", modifier.RollMode(7), []int{5}, false, false, true, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.mode.Select(tc.rolled)
			if got != 0 || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("Select = %d, %v", got, err)
			}
			if errors.Is(err, count) != tc.wantCount || errors.Is(err, face) != tc.wantFace ||
				errors.Is(err, &modifier.InvalidError{Reason: modifier.ReasonUnknownRollMode}) != tc.wantUnknown {
				t.Errorf("wrong reasons: %v", err)
			}
			if n := strings.Count(err.Error(), "\n") + 1; n != tc.wantProblems {
				t.Errorf("%d problems, want %d: %v", n, tc.wantProblems, err)
			}
		})
	}
}

func TestInvalidErrorAndReason(t *testing.T) {
	t.Parallel()
	if modifier.Reason(0).String() != "unknown" || modifier.Reason(50).String() != "unknown" {
		t.Error("undefined Reason")
	}
	e := &modifier.InvalidError{Reason: modifier.ReasonFaceOutOfRange}
	if e.Error() != "modifier: face out of range" || errors.Is(e, errors.New("x")) {
		t.Error("InvalidError")
	}
	if !errors.Is(e, &modifier.InvalidError{Reason: modifier.ReasonFaceOutOfRange, Detail: "ignored"}) {
		t.Error("Is should ignore Detail")
	}
}

func FuzzSelect(f *testing.F) {
	f.Add(0, 1, 2)
	f.Add(1, 20, 1)
	f.Add(2, 0, 21)
	f.Fuzz(func(t *testing.T, mode, x, y int) {
		m := modifier.RollMode(mode)
		rolled := []int{x, y}[:m.DiceCount()]
		got, err := m.Select(rolled)
		if err != nil {
			if got != 0 || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("bad failure %d, %v", got, err)
			}
			return
		}
		if got < 1 || got > 20 || !slices.Contains(rolled, got) {
			t.Fatalf("Select(%v) = %d", rolled, got)
		}
	})
}

func FuzzParseRollMode(f *testing.F) {
	f.Add("normal")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		m, err := modifier.ParseRollMode(s)
		if err == nil && !strings.EqualFold(m.String(), s) {
			t.Fatalf("ParseRollMode(%q) = %v", s, m)
		}
	})
}

func ExampleResolve() {
	mode := modifier.Resolve(modifier.Advantage{}, modifier.Advantage{}, modifier.Disadvantage{})
	fmt.Println(mode)
	fmt.Println(modifier.Resolve(modifier.Advantage{}, modifier.Advantage{}))
	// Output:
	// normal
	// advantage
}

func ExampleRollMode_Select() {
	used, _ := modifier.WithAdvantage.Select([]int{4, 17})
	total := modifier.Sum(modifier.New(3), modifier.New(2)).ApplyTo(used)
	fmt.Println(used, total)
	// Output: 17 22
}
