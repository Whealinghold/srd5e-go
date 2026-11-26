package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/movement-lifting-and-flight.md FR-023 to FR-025 - only a Long Jump into Difficult Terrain needs the DC 10
// Dexterity (Acrobatics) landing check (SRD 5.2.1, "Long Jump", p. 185; "High Jump", p. 183).
// Path: per case.
func TestJumpLandingFor(t *testing.T) {
	t.Parallel()
	check := combat.JumpLandingCheckRequired{Requirement: combat.RequirementFor(combat.MovementCheckKindLongJumpLanding)}
	cases := []struct {
		name    string
		kind    combat.JumpKind
		terrain combat.Terrain
		want    combat.JumpLanding
	}{
		{"Long Jump, Difficult Terrain", longJump, combat.TerrainDifficult, check},
		{"Long Jump, open ground", longJump, combat.TerrainOpen, combat.JumpLandsClear{}},
		{"High Jump, Difficult Terrain", highJump, combat.TerrainDifficult, combat.JumpLandsClear{}},
		{"High Jump, open ground", highJump, combat.TerrainOpen, combat.JumpLandsClear{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := combat.JumpLandingFor(c.kind, c.terrain); got != c.want {
				t.Errorf("JumpLandingFor(%v, %v) = %#v, want %#v", c.kind, c.terrain, got, c.want)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestObstacleForLongJump(t *testing.T) {
	t.Parallel()
	const r = `SRD 5.2.1, "Long Jump", p. 185; "Round Down", p. 187 (doc/specs/movement-lifting-and-flight.md cases 4.3a)`
	check := combat.JumpObstacleCheckRequired{Requirement: combat.RequirementFor(combat.MovementCheckKindLongJumpObstacle)}
	open := combat.JumpObstacleNotCovered{}
	cases := []struct {
		name, path string
		distance   int
		height     int
		want       combat.JumpObstacle
	}{
		{"exactly a quarter", "unhappy (boundary)", 20, 5, check},
		{"one over a quarter", "unhappy (boundary)", 20, 6, open},
		{"well under", "happy", 20, 4, check},
		{"flat", "unhappy (boundary)", 20, 0, check},
		{"quarter rounds down, at it", "unhappy (boundary)", 7, 1, check},
		{"quarter rounds down, over it", "unhappy (boundary)", 7, 2, open},
		{"short jump, quarter is 0", "unhappy (boundary)", 3, 1, open},
		{"one-foot jump, flat", "unhappy (boundary)", 1, 0, check},
		{"one-foot jump, one foot", "unhappy (boundary)", 1, 1, open},
		{"no jump", "unhappy (empty case)", 0, 0, open},
		{"no jump, any height", "unhappy (empty case)", 0, 1, open},
		{"the longest jump, the largest obstacle", "unhappy (boundary)", math.MaxInt, math.MaxInt, open},
		{"the longest jump, a quarter high", "unhappy (boundary)", math.MaxInt, math.MaxInt / 4, check},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.ObstacleForLongJump(c.distance, c.height)
			if err != nil || got != c.want {
				t.Errorf("ObstacleForLongJump(%d, %d) = %#v, %v; want %#v [%s, %s]", c.distance, c.height, got, err, c.want, r, c.path)
			}
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-028, FR-030 - a negative distance, obstacle height or jump height, and a creature height below 1 foot, are invalid-input errors.
// Path: unhappy (invalid input).
func TestObstacleAndReachRejectInvalid(t *testing.T) {
	t.Parallel()
	_, e1 := combat.ObstacleForLongJump(-1, 1)
	_, e2 := combat.ObstacleForLongJump(20, -1)
	_, e3 := combat.ObstacleForLongJump(math.MinInt, math.MinInt)
	_, e4 := combat.HighJumpReachFeet(3, 0)
	_, e5 := combat.HighJumpReachFeet(3, -1)
	_, e6 := combat.HighJumpReachFeet(-1, 6)
	_, e7 := combat.HighJumpReachFeet(math.MinInt, math.MinInt)
	for i, err := range []error{e1, e2, e3, e4, e5, e6, e7} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("case %d: err = %v, want invalid input", i+1, err)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestHighJumpReachFeet(t *testing.T) {
	t.Parallel()
	const r = `SRD 5.2.1, "High Jump", p. 183; "Round Down", p. 187 (doc/specs/movement-lifting-and-flight.md FR-029, cases 4.3b)`
	cases := []struct {
		name, path string
		jump, tall int
		want       int
	}{
		{"ordinary", "happy", 3, 6, 12},
		{"no jump", "unhappy (boundary)", 0, 6, 9},
		{"half foot rounds down", "unhappy (boundary)", 0, 5, 7},
		{"tall jump", "happy", 13, 6, 22},
		{"smallest creature height", "unhappy (boundary)", 0, 1, 1},
		{"the largest height saturates", "unhappy (boundary)", 0, math.MaxInt, math.MaxInt},
		{"the largest jump saturates", "unhappy (boundary)", math.MaxInt, 6, math.MaxInt},
		{"both largest saturate", "unhappy (boundary)", math.MaxInt, math.MaxInt, math.MaxInt},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got, err := combat.HighJumpReachFeet(c.jump, c.tall); err != nil || got != c.want {
				t.Errorf("HighJumpReachFeet(%d, %d) = %d, %v; want %d [%s, %s]", c.jump, c.tall, got, err, c.want, r, c.path)
			}
		})
	}
}
