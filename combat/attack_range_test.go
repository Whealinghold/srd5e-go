package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attacks.md FR-033, FR-107 - a range is positive and a long range is larger than the normal one;
// the error joins every violated field, in order, and matches srderr.ErrInvalidInput.
// Path: per case.
func TestNewAttackRange(t *testing.T) {
	t.Parallel()
	single, normal, long := combat.AttackRangeFieldSingle, combat.AttackRangeFieldNormal, combat.AttackRangeFieldLong
	cases := []struct {
		name, requirement, path string
		normal, long            int
		wantFields              []combat.AttackRangeField
		notWanted               []combat.AttackRangeField
	}{
		{"single range", "FR-033", "happy", 60, 0, nil, nil},
		{"normal and long", "FR-033", "happy", 80, 320, nil, nil},
		{"long one above normal", "FR-033", "unhappy (boundary)", 80, 81, nil, nil},
		{"long equal to normal", "FR-033", "unhappy (boundary)", 80, 80, []combat.AttackRangeField{long}, []combat.AttackRangeField{normal, single}},
		{"long below normal", "FR-033", "unhappy (invalid input)", 80, 20, []combat.AttackRangeField{long}, nil},
		{"zero single range", "FR-033", "unhappy (boundary)", 0, 0, []combat.AttackRangeField{single}, []combat.AttackRangeField{normal}},
		{"negative single range", "FR-033", "unhappy (invalid input)", -5, 0, []combat.AttackRangeField{single}, nil},
		{"zero normal with long", "FR-033", "unhappy (invalid input)", 0, 30, []combat.AttackRangeField{normal}, []combat.AttackRangeField{single}},
		{"negative long", "FR-033", "unhappy (invalid input)", 30, -1, []combat.AttackRangeField{long}, nil},
		{"both fields wrong", "FR-107", "unhappy (invalid input)", -1, -2, []combat.AttackRangeField{normal, long}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := combat.NewAttackRange(c.normal, c.long)
			if len(c.wantFields) == 0 {
				if err != nil || r.Normal() != c.normal || r.Long() != c.long || r.IsZero() {
					t.Errorf("NewAttackRange(%d, %d) = %+v, %v, want the range [%s]", c.normal, c.long, r, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || !r.IsZero() {
				t.Fatalf("NewAttackRange(%d, %d) = %+v, %v, want the zero range and ErrInvalidInput [%s]", c.normal, c.long, r, err, c.requirement)
			}
			for _, f := range c.wantFields {
				if !errors.Is(err, combat.AttackRangeProblem{Field: f}) {
					t.Errorf("error lacks the problem %v [%s]", f, c.requirement)
				}
			}
			for _, f := range c.notWanted {
				if errors.Is(err, combat.AttackRangeProblem{Field: f}) {
					t.Errorf("error has the problem %v it should not [%s]", f, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-107 - MustAttackRange, MustReach and MustNearbyEnemy panic with the error of their
// constructor, and only then.
// Path: unhappy (invalid input).
func TestAttackInputMustHelpers(t *testing.T) {
	t.Parallel()
	panics := func(f func()) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err, _ = r.(error)
			}
		}()
		f()
		return nil
	}
	if err := panics(func() { combat.MustAttackRange(-1, 0) }); !errors.Is(err, combat.AttackRangeProblem{Field: combat.AttackRangeFieldSingle}) {
		t.Errorf("MustAttackRange(-1, 0) panic = %v", err)
	}
	if err := panics(func() { combat.MustReach(0) }); !errors.Is(err, combat.ReachProblem{Field: combat.ReachFieldFeet}) {
		t.Errorf("MustReach(0) panic = %v", err)
	}
	if err := panics(func() { combat.MustNearbyEnemy(-1, true, false) }); !errors.Is(err, combat.NearbyEnemyProblem{Field: combat.NearbyEnemyFieldDistance}) {
		t.Errorf("MustNearbyEnemy(-1) panic = %v", err)
	}
	if err := panics(func() { combat.MustAttackRange(5, 0); combat.MustReach(1); combat.MustNearbyEnemy(0, false, false) }); err != nil {
		t.Errorf("valid values panicked: %v", err)
	}
}

// Requirement: doc/specs/attacks.md FR-032, FR-033 - a supplied reach larger than 5 feet replaces the default, a smaller
// one is read as 5, and a reach that is not positive is a validation error (SRD 5.2.1, "Reach", p. 15).
// Path: per case.
func TestAttackReach(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		feet                    int
		want                    int
		invalid                 bool
	}{
		{"ten feet replaces the default", "FR-032", "happy", 10, 10, false},
		{"six feet", "FR-032", "unhappy (boundary)", 6, 6, false},
		{"five feet", "FR-032", "unhappy (boundary)", 5, 5, false},
		{"three feet reads as five", "FR-032", "unhappy (boundary)", 3, 5, false},
		{"zero", "FR-033", "unhappy (boundary)", 0, 0, true},
		{"negative", "FR-033", "unhappy (invalid input)", -5, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := combat.NewReach(c.feet)
			if c.invalid {
				if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, combat.ReachProblem{Field: combat.ReachFieldFeet}) {
					t.Errorf("NewReach(%d) error = %v, want a ReachProblem [%s]", c.feet, err, c.requirement)
				}
				return
			}
			if err != nil || r.Feet() != c.want {
				t.Errorf("NewReach(%d) = %d, %v, want %d [%s]", c.feet, r.Feet(), err, c.want, c.requirement)
			}
		})
	}
	if got := (combat.Reach{}).Feet(); got != 5 {
		t.Errorf("zero Reach = %d feet, want the default 5 [FR-030]", got)
	}
}

// Requirement: doc/specs/attacks.md FR-033, FR-107 - a nearby enemy has a distance of 0 or more.
// Path: per case.
func TestNewNearbyEnemy(t *testing.T) {
	t.Parallel()
	e, err := combat.NewNearbyEnemy(0, true, false)
	if err != nil || e.Distance() != 0 || !e.CanSeeAttacker() || e.Incapacitated() {
		t.Errorf("NewNearbyEnemy(0, true, false) = %+v, %v", e, err)
	}
	e, err = combat.NewNearbyEnemy(5, false, true)
	if err != nil || e.Distance() != 5 || e.CanSeeAttacker() || !e.Incapacitated() {
		t.Errorf("NewNearbyEnemy(5, false, true) = %+v, %v", e, err)
	}
	_, err = combat.NewNearbyEnemy(-1, true, false)
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, combat.NearbyEnemyProblem{Field: combat.NearbyEnemyFieldDistance}) {
		t.Errorf("NewNearbyEnemy(-1) error = %v, want a NearbyEnemyProblem", err)
	}
}

// Requirement: doc/specs/attacks.md FR-107 - the problem field names are the snake_case words of their constants.
// Path: unhappy (invalid input).
func TestAttackProblemFieldNames(t *testing.T) {
	t.Parallel()
	got := []string{combat.AttackRangeFieldSingle.String(), combat.AttackRangeFieldNormal.String(), combat.AttackRangeFieldLong.String(),
		combat.AttackRangeField(0).String(), combat.ReachFieldFeet.String(), combat.ReachField(0).String(),
		combat.NearbyEnemyFieldDistance.String(), combat.NearbyEnemyField(7).String(),
		combat.AttackRangeProblem{Field: combat.AttackRangeFieldLong}.Error() + combat.ReachProblem{Field: combat.ReachFieldFeet}.Error() +
			combat.NearbyEnemyProblem{Field: combat.NearbyEnemyFieldDistance}.Error()}
	want := []string{"single", "normal", "long", "unknown", "feet", "unknown", "distance", "unknown",
		"combat: invalid attack range: longcombat: invalid reach: feetcombat: invalid nearby enemy: distance"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// Requirement: doc/specs/attacks.md FR-033, FR-107 and testing.md §10 - NewAttackRange, NewReach and NewNearbyEnemy never
// panic on any numbers, and an accepted range is positive with a long range larger than the normal one.
// Path: unhappy (invalid input).
func FuzzNewAttackRange(f *testing.F) {
	for _, s := range [][3]int{{80, 320, 5}, {0, 0, 0}, {-1, 1, -1}, {math.MaxInt, 1, 7}} {
		f.Add(s[0], s[1], s[2])
	}
	f.Fuzz(func(t *testing.T, normal, long, feet int) {
		if r, err := combat.NewAttackRange(normal, long); err == nil && (r.Normal() <= 0 || (r.Long() != 0 && r.Long() <= r.Normal())) {
			t.Errorf("accepted range %+v", r)
		}
		if r, err := combat.NewReach(feet); err == nil && r.Feet() < 5 {
			t.Errorf("accepted reach %d reads as %d", feet, r.Feet())
		}
		if _, err := combat.NewNearbyEnemy(feet, true, true); err != nil && feet >= 0 {
			t.Errorf("rejected distance %d", feet)
		}
	})
}
