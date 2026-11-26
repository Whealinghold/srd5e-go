package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-001, FR-002, FR-105 - NewHitPoints accepts 0 <= current <=
// maximum with temporary >= 0 and otherwise returns a *HitPointsError listing every violated problem in the fixed
// order (SRD 5.2.1, "Hit Points", p. 16).
// Path: per case.
func TestNewHitPoints(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path             string
		current, maximum, temp int
		problems               []combat.HitPointsProblem
	}{
		{"valid", "happy", 5, 10, 2, nil},
		{"full", "unhappy (boundary)", 10, 10, 0, nil},
		{"empty at zero maximum", "unhappy (empty and minimal)", 0, 0, 0, nil},
		{"current below zero", "unhappy (invalid input)", -1, 10, 0, []combat.HitPointsProblem{combat.CurrentBelowZero}},
		{"current above maximum", "unhappy (boundary)", 11, 10, 0, []combat.HitPointsProblem{combat.CurrentAboveMaximum}},
		{"maximum below zero", "unhappy (invalid input)", 0, -1, 0, []combat.HitPointsProblem{combat.CurrentAboveMaximum, combat.MaximumBelowZero}},
		{"temporary below zero", "unhappy (invalid input)", 1, 2, -1, []combat.HitPointsProblem{combat.TemporaryBelowZero}},
		{"every problem", "unhappy (invalid input)", -5, -9, -3, []combat.HitPointsProblem{combat.CurrentBelowZero, combat.CurrentAboveMaximum, combat.MaximumBelowZero, combat.TemporaryBelowZero}},
		{"extremes never panic", "unhappy (boundary)", math.MinInt, math.MaxInt, math.MinInt, []combat.HitPointsProblem{combat.CurrentBelowZero, combat.TemporaryBelowZero}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			hp, err := combat.NewHitPoints(c.current, c.maximum, c.temp)
			if c.problems == nil {
				if err != nil || hp.Current() != c.current || hp.Maximum() != c.maximum || hp.Temporary() != c.temp {
					t.Fatalf("= %+v, %v", hp, err)
				}
				if combat.MustHitPoints(c.current, c.maximum, c.temp) != hp {
					t.Errorf("MustHitPoints differs")
				}
				return
			}
			var he *combat.HitPointsError
			if !errors.As(err, &he) || !errors.Is(err, srderr.ErrInvalidInput) || !slices.Equal(he.Problems(), c.problems) || hp != (combat.HitPoints{}) {
				t.Fatalf("= %+v, %v, want problems %v", hp, err, c.problems)
			}
			got := he.Problems()
			got[0] = 0
			if he.Problems()[0] == 0 {
				t.Errorf("Problems is not a copy")
			}
			func() {
				defer func() {
					if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
						t.Errorf("MustHitPoints panic = %v, want the parser error", r)
					}
				}()
				combat.MustHitPoints(c.current, c.maximum, c.temp)
			}()
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-105 - problems and errors carry names, not display text; two
// errors with the same problems are equal under errors.Is and different problems are not.
// Path: unhappy (boundary).
func TestHitPointsErrorIdentity(t *testing.T) {
	t.Parallel()
	_, a := combat.NewHitPoints(-1, 5, 0)
	_, b := combat.NewHitPoints(-2, 9, 0)
	_, c := combat.NewHitPoints(1, 0, 0)
	if !errors.Is(a, b) || errors.Is(a, c) || a.Error() == "" || errors.Is(a, srderr.ErrContractViolation) {
		t.Errorf("error identity wrong")
	}
	names := map[combat.HitPointsProblem]string{combat.CurrentBelowZero: "current_below_zero", combat.CurrentAboveMaximum: "current_above_maximum",
		combat.MaximumBelowZero: "maximum_below_zero", combat.TemporaryBelowZero: "temporary_below_zero", 0: "unknown"}
	for p, want := range names {
		if p.String() != want {
			t.Errorf("%d.String() = %q, want %q", p, p.String(), want)
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-105, testing.md section 10 - NewHitPoints never panics for
// any ints and MustHitPoints panics if and only if the parser errors.
// Path: unhappy (invalid input).
func FuzzNewHitPoints(f *testing.F) {
	for _, s := range [][3]int{{0, 0, 0}, {5, 10, 2}, {-1, 5, 0}, {11, 10, 0}, {math.MinInt, math.MaxInt, math.MinInt}} {
		f.Add(s[0], s[1], s[2])
	}
	f.Fuzz(func(t *testing.T, current, maximum, temp int) {
		hp, err := combat.NewHitPoints(current, maximum, temp)
		defer func() {
			if (recover() != nil) != (err != nil) {
				t.Errorf("MustHitPoints panics iff the parser fails: err = %v", err)
			}
		}()
		combat.MustHitPoints(current, maximum, temp)
		if err == nil && (hp.Current() < 0 || hp.Current() > hp.Maximum() || hp.Temporary() < 0) {
			t.Errorf("accepted %+v", hp)
		}
	})
}
