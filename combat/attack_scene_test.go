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
func TestNewAttackScene(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		distance                int
		ok                      bool
	}{
		{"distance 0", resSpec + "FR-097", "unhappy (boundary)", 0, true},
		{"distance 5", resSpec + "FR-097", "happy", 5, true},
		{"negative distance", resSpec + "FR-097", "unhappy (boundary)", -1, false},
		{"most negative distance", resSpec + "FR-097", "unhappy (boundary)", math.MinInt, false},
		{"largest distance", resSpec + "FR-097", "unhappy (boundary)", math.MaxInt, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.NewAttackScene(c.distance, combat.SightRelationCanSee, combat.SightRelationCannotSee, combat.LineOfSightClear)
			if c.ok {
				if err != nil || s.DistanceFeet() != c.distance || s.AttackerSeesTarget() != combat.SightRelationCanSee ||
					s.TargetSeesAttacker() != combat.SightRelationCannotSee || s.AttackerFearSource() != combat.LineOfSightClear {
					t.Errorf("scene = %+v, %v, want the facts given [%s]", s, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || s != (combat.AttackScene{}) {
				t.Errorf("= %+v, %v, want the zero scene and ErrInvalidInput [%s]", s, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-097, testing.md section 10 - NewAttackScene never panics and rejects a distance if and only if it is negative, whatever the other values.
// Path: unhappy (invalid input).
func FuzzNewAttackScene(f *testing.F) {
	for _, d := range []int{0, 5, -1, math.MaxInt, math.MinInt} {
		f.Add(d, 1, 2, 1)
	}
	f.Fuzz(func(t *testing.T, distance, attackerSees, targetSees, fear int) {
		s, err := combat.NewAttackScene(distance, combat.SightRelation(attackerSees), combat.SightRelation(targetSees), combat.LineOfSight(fear))
		if (err != nil) != (distance < 0) || (err == nil && s.DistanceFeet() != distance) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("NewAttackScene(%d) = %+v, %v", distance, s, err)
		}
	})
}
