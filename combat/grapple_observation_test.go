package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewGrappleObservation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		grappler, target        string
		distance                int
		opts                    []combat.GrappleOption
		problem                 combat.GrappleProblem
	}{
		{"valid", `FR-048; "Ending a Grapple", p. 182`, "happy", "a", "b", 5, nil, 0},
		{"distance 0", "FR-048", "unhappy (boundary)", "a", "b", 0, nil, 0},
		{"distance -1", "FR-048, FR-102", "unhappy (boundary)", "a", "b", -1, nil, combat.GrappleProblemNegativeDistance},
		{"same ids", "FR-048", "unhappy (invalid input)", "a", "a", 5, nil, combat.GrappleProblemSameGrapplerAndTarget},
		{"both problems", "FR-100, table 4.10.7", "unhappy (rule interaction)", "a", "a", -1, nil, combat.GrappleProblemSameGrapplerAndTarget},
		{"blank grappler", "package design", "unhappy (invalid input)", "", "b", 5, nil, combat.GrappleProblemMissingValue},
		{"blank both", "package design", "unhappy (rule interaction)", "", "", -1, nil, combat.GrappleProblemMissingValue},
		{"dead option", "FR-104", "happy", "a", "b", 5, []combat.GrappleOption{combat.WithGrapplerDead(true)}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			conds := condSet(t, condConds(combat.Prone{}))
			o, err := combat.NewGrappleObservation(combat.CombatantID(c.grappler), combat.CombatantID(c.target), conds, c.distance, c.opts...)
			if c.problem != 0 {
				if o != (combat.GrappleObservation{}) || problemOf(t, err) != c.problem {
					t.Errorf("= %v, %v, want problem %v [%s]", o, err, c.problem, c.requirement)
				}
				return
			}
			if err != nil || o.DistanceFeet() != c.distance || !o.Conditions().Equal(conds) || o.GrapplerDead() != (len(c.opts) > 0) {
				t.Errorf("= %+v, %v, want the values held [%s]", o, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-083 - MustGrappleObservation panics with the constructor's error, and only then.
// Path: unhappy (invalid input).
func TestMustGrappleObservation(t *testing.T) {
	t.Parallel()
	if combat.MustGrappleObservation(gid("a"), gid("b"), combat.ConditionSet{}, 1).DistanceFeet() != 1 {
		t.Error("MustGrappleObservation changed a value")
	}
	defer func() {
		if err, _ := recover().(error); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an ErrInvalidInput error", err)
		}
	}()
	combat.MustGrappleObservation(gid("a"), gid("b"), combat.ConditionSet{}, -1)
}

// Requirement: doc/specs/grappling.md FR-083, testing.md section 10 - NewGrappleObservation never panics and returns an
// error exactly when the input is invalid.
// Path: unhappy (invalid input).
func FuzzNewGrappleObservation(f *testing.F) {
	f.Add("a", "b", 5)
	f.Add("a", "a", -1)
	f.Add("", "", math.MinInt)
	f.Add("a", "b", math.MaxInt)
	f.Fuzz(func(t *testing.T, grappler, target string, distance int) {
		_, err := combat.NewGrappleObservation(combat.CombatantID(grappler), combat.CombatantID(target), combat.ConditionSet{}, distance)
		if bad := grappler == "" || target == "" || grappler == target || distance < 0; (err != nil) != bad {
			t.Fatalf("NewGrappleObservation(%q, %q, %d) error = %v, invalid = %v", grappler, target, distance, err, bad)
		}
	})
}
