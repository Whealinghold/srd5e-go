package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzMovementNumbers feeds the movement constructors host numbers (a kind, a mode, a Strength
// score, feet and a hover ability). It asserts properties: nothing panics, a rejection matches
// srderr.ErrInvalidInput and returns the zero value, and an accepted value assesses without
// refusing a request of 0 feet or exceeding the movement cost of its distance.
func FuzzMovementNumbers(f *testing.F) {
	f.Add(1, 1, 10, 10, 5, 30, 0)
	f.Add(2, 6, 1, 0, 0, 0, 1)
	f.Add(0, 0, 0, -1, -1, -1, 2)
	f.Add(math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt)
	f.Add(math.MinInt, math.MinInt, math.MinInt, math.MinInt, math.MinInt, math.MinInt, math.MinInt)
	f.Fuzz(func(t *testing.T, kind, mode, strength, runUp, requested, left, hover int) {
		bad := func(err error) bool { return err != nil && !errors.Is(err, srderr.ErrInvalidInput) }
		score, _ := ability.NewScore(strength)
		a, err := combat.NewJumpAttempt(combat.JumpKind(kind), score, combat.ConditionSet{}, runUp, combat.MovementMode(mode), requested, left)
		switch {
		case bad(err) || (err != nil && !a.IsZero()):
			t.Fatalf("NewJumpAttempt = %#v, %v", a, err)
		case err == nil:
			if p, ok := combat.AssessJump(a).(combat.JumpPermitted); ok && (p.MovementCostFeet != p.DistanceFeet || p.DistanceFeet > p.MaximumFeet) {
				t.Fatalf("permitted jump %#v costs more than it jumps or passes its maximum", p)
			}
		}
		if _, err := combat.ObstacleForLongJump(runUp, requested); bad(err) {
			t.Fatalf("ObstacleForLongJump: %v", err)
		}
		if reach, err := combat.HighJumpReachFeet(runUp, requested); bad(err) || (err == nil && reach < runUp) {
			t.Fatalf("HighJumpReachFeet = %d, %v", reach, err)
		}
		if got, err := combat.NewFlightFacts(combat.ConditionSet{}, left, combat.HoverAbility(hover)); bad(err) || (err != nil && got != (combat.FlightFacts{})) {
			t.Fatalf("NewFlightFacts = %#v, %v", got, err)
		} else if err == nil {
			combat.AssessFlight(got)
		}
	})
}
