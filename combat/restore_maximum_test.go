package combat_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (table 4.6; Source: SRD 5.2.1, "Greater Restoration" p. 137)
// Path: per case.
func TestRestoreMaximumFunction(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		status                  combat.HitPointStatus
		amount                  int
		maximum, current        int
		events                  []string
	}{
		{"maximum 12, current 12, restore 8", "FR-031; table 4.6", "happy", pcStatus(t, 12, 12, 0), 8, 20, 12, []string{"hit_point_maximum_restored"}},
		{"a Dying creature stays Dying", "FR-031; table 4.6", "unhappy (state error)", dyingStatus(t, 12, 0, 1), 8, 20, 0, []string{"hit_point_maximum_restored"}},
		{"amount 0 is a no-op", "FR-031; table 4.6", "unhappy (boundary)", pcStatus(t, 12, 12, 3), 0, 12, 12, nil},
		{"amount 1", "FR-031", "unhappy (boundary)", pcStatus(t, 12, 12, 0), 1, 13, 12, []string{"hit_point_maximum_restored"}},
		{"Temporary Hit Points stay", "FR-031", "unhappy (rule interaction)", pcStatus(t, 5, 12, 4), 8, 20, 5, []string{"hit_point_maximum_restored"}},
		{"the largest amount stops at the largest int", "FR-031 (overflow)", "unhappy (boundary)", pcStatus(t, 5, 12, 0), math.MaxInt, math.MaxInt, 5, []string{"hit_point_maximum_restored"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			res, err := combat.RestoreMaximum(gid("hero"), c.status, c.amount)
			a, ok := res.(combat.RestoreMaximumApplied)
			if err != nil || !ok {
				t.Fatalf("RestoreMaximum = %v, %v, want applied", res, err)
			}
			hp, was := a.Status().HitPoints(), c.status.HitPoints()
			if hp.Maximum() != c.maximum || hp.Current() != c.current || hp.Temporary() != was.Temporary() || !reflect.DeepEqual(a.Status().LifeState(), c.status.LifeState()) {
				t.Errorf("status = %+v / %T, want %d of %d, the life state and Temporary Hit Points unchanged [%s; %s]", hp, a.Status().LifeState(), c.current, c.maximum, c.requirement, c.path)
			}
			if got := rrKinds(a.Events()); !slices.Equal(got, append([]string{}, c.events...)) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, c.requirement)
			}
			if len(c.events) > 0 {
				if e := a.Events()[0].(combat.HitPointMaximumRestored); e.Maximum() != c.maximum || e.PreviousMaximum() != was.Maximum() {
					t.Errorf("HitPointMaximumRestored = %d to %d, want %d to %d", e.PreviousMaximum(), e.Maximum(), was.Maximum(), c.maximum)
				}
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-031, table 4.6 - amount -1 is RestoreMaximumInvalidAmount carrying the amount; a Dead creature, including one that died
// of a maximum of 0, is RestoreMaximumRejected (CreatureIsDead) whatever the amount (Q-10); a zero subject or status is an error matching srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestRestoreMaximumFunctionRefusals(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 12, 12, 0)
	if res, err := combat.RestoreMaximum(gid("hero"), st, -1); err != nil || res != (combat.RestoreMaximumInvalidAmount{Amount: -1}) {
		t.Errorf("amount -1: %v, %v, want RestoreMaximumInvalidAmount{-1}", res, err)
	}
	if res, _ := combat.RestoreMaximum(gid("hero"), st, math.MinInt); res != (combat.RestoreMaximumInvalidAmount{Amount: math.MinInt}) {
		t.Errorf("the smallest amount: %v, want RestoreMaximumInvalidAmount", res)
	}
	zeroMax := status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities())
	for _, s := range []combat.HitPointStatus{deadStatus(t, combat.MassiveDamage), zeroMax} {
		for _, amount := range []int{8, 0, -1} {
			if res, err := combat.RestoreMaximum(gid("hero"), s, amount); err != nil || res != (combat.RestoreMaximumRejected{Rejection: combat.CreatureIsDead}) {
				t.Errorf("Dead creature, amount %d: %v, %v, want RestoreMaximumRejected (CreatureIsDead)", amount, res, err)
			}
		}
	}
	for name, call := range map[string]func() (combat.RestoreHitPointMaximumResult, error){
		"zero subject": func() (combat.RestoreHitPointMaximumResult, error) { return combat.RestoreMaximum("", st, 3) },
		"zero status": func() (combat.RestoreHitPointMaximumResult, error) {
			return combat.RestoreMaximum(gid("hero"), combat.HitPointStatus{}, 3)
		},
	} {
		if res, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: %v, %v, want a nil result and an invalid input error", name, res, err)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-045 - RestoreMaximum changes none of its inputs, gives equal results for equal inputs and returns a copy of its events.
// Path: unhappy (immutability).
func TestRestoreMaximumFunctionImmutable(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 12, 12, 0)
	before := st
	first, _ := combat.RestoreMaximum(gid("hero"), st, 8)
	second, _ := combat.RestoreMaximum(gid("hero"), st, 8)
	if !reflect.DeepEqual(st, before) || !reflect.DeepEqual(first, second) {
		t.Error("RestoreMaximum changed an input or gave unequal results for equal inputs")
	}
	ev := first.(combat.RestoreMaximumApplied).Events()
	ev[0] = nil
	if first.(combat.RestoreMaximumApplied).Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}
