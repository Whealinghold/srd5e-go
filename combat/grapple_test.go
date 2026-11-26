package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/grappling.md FR-006 - a Grapple compares by value on all five fields: equal when all are
// equal, unequal when any one differs.
// Path: unhappy (boundary).
func TestGrappleComparesOnEveryField(t *testing.T) {
	t.Parallel()
	base := combat.MustGrapple(gid("a"), gid("b"), hand(0), 13, 5)
	same := combat.MustGrapple(gid("a"), gid("b"), hand(0), 13, 5)
	if base != same {
		t.Error("equal grapples differ")
	}
	for name, other := range map[string]combat.Grapple{
		"grappler": combat.MustGrapple(gid("c"), gid("b"), hand(0), 13, 5),
		"target":   combat.MustGrapple(gid("a"), gid("c"), hand(0), 13, 5),
		"part":     combat.MustGrapple(gid("a"), gid("b"), hand(1), 13, 5),
		"DC":       combat.MustGrapple(gid("a"), gid("b"), hand(0), 14, 5),
		"range":    combat.MustGrapple(gid("a"), gid("b"), hand(0), 13, 6),
	} {
		if base == other {
			t.Errorf("grapples differing in %s are equal", name)
		}
	}
}

// Requirement: doc/specs/grappling.md FR-007, FR-008, FR-009, FR-099, FR-102, table 4.10.7 - NewGrapple rejects (in this
// order) a blank id or part, equal ids and a negative range with a GrappleProblemError; a range of 0 and any escape DC are
// held unchanged.
// Path: per case.
func TestNewGrapple(t *testing.T) {
	t.Parallel()
	var none combat.GraspingPart
	cases := []struct {
		name, requirement, path string
		grappler, target        string
		part                    combat.GraspingPart
		dc                      d20.DifficultyClass
		rng                     int
		problem                 combat.GrappleProblem
	}{
		{"valid", "FR-006", "happy", "a", "b", hand(0), 13, 5, 0},
		{"range 0", "FR-008", "unhappy (boundary)", "a", "b", hand(0), 13, 0, 0},
		{"DC far below any ability check", "FR-009", "unhappy (boundary)", "a", "b", hand(0), d20.DifficultyClass(math.MinInt), 5, 0},
		{"DC at the maximum", "FR-009", "unhappy (boundary)", "a", "b", hand(0), d20.DifficultyClass(math.MaxInt), 5, 0},
		{"same grappler and target", "FR-007", "unhappy (invalid input)", "a", "a", hand(0), 13, 5, combat.GrappleProblemSameGrapplerAndTarget},
		{"range -1", "FR-008, FR-099, FR-102", "unhappy (boundary)", "a", "b", hand(0), 13, -1, combat.GrappleProblemNegativeRange},
		{"both problems", "FR-099, table 4.10.7", "unhappy (rule interaction)", "a", "a", hand(0), 13, -1, combat.GrappleProblemSameGrapplerAndTarget},
		{"blank grappler", "package design", "unhappy (invalid input)", "", "b", hand(0), 13, 5, combat.GrappleProblemMissingValue},
		{"blank target", "package design", "unhappy (invalid input)", "a", "", hand(0), 13, 5, combat.GrappleProblemMissingValue},
		{"no part", "package design", "unhappy (invalid input)", "a", "b", none, 13, 5, combat.GrappleProblemMissingValue},
		{"the zero hand is hand 0", "FR-001", "unhappy (boundary)", "a", "b", combat.GraspingHand{}, 13, 5, 0},
		{"blank body part literal", "package design", "unhappy (invalid input)", "a", "b", combat.GraspingBodyPart{}, 13, 5, combat.GrappleProblemMissingValue},
		{"missing value before same ids", "package design", "unhappy (rule interaction)", "", "", hand(0), 13, -1, combat.GrappleProblemMissingValue},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			g, err := combat.NewGrapple(combat.CombatantID(c.grappler), combat.CombatantID(c.target), c.part, c.dc, c.rng)
			if c.problem != 0 {
				if g != (combat.Grapple{}) || problemOf(t, err) != c.problem {
					t.Errorf("= %v, %v, want problem %v [%s]", g, err, c.problem, c.requirement)
				}
				return
			}
			if err != nil || g.EscapeDC != c.dc || g.RangeFeet != c.rng {
				t.Errorf("= %v, %v, want the values held unchanged [%s]", g, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-010, FR-011 - Condition is Grappled with the grappler as its source, and the
// Unarmed Strike range is 5 feet.
// Path: happy.
func TestGrappleCondition(t *testing.T) {
	t.Parallel()
	g := held("a", "b", hand(0))
	if g.Condition() != (combat.Grappled{Grappler: gid("a")}) || g.Condition().Kind() != content.ConditionGrappled {
		t.Errorf("Condition() = %v", g.Condition())
	}
	if combat.GrappleUnarmedStrikeRangeFeet != 5 {
		t.Error("the Unarmed Strike range is not 5")
	}
}

// Requirement: doc/specs/grappling.md FR-083 - MustGrapple panics with the NewGrapple error, and only then.
// Path: unhappy (invalid input).
func TestMustGrapple(t *testing.T) {
	t.Parallel()
	if g := combat.MustGrapple(gid("a"), gid("b"), hand(0), 13, 5); g.Grappler != gid("a") {
		t.Error("MustGrapple changed a value")
	}
	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an ErrInvalidInput error", err)
		}
	}()
	combat.MustGrapple(gid("a"), gid("a"), hand(0), 13, 5)
}

// Requirement: doc/specs/grappling.md FR-083, testing.md section 10 - NewGrapple never panics and returns an error
// exactly when the input is invalid.
// Path: unhappy (invalid input).
func FuzzNewGrapple(f *testing.F) {
	f.Add("a", "b", 0, 13, 5)
	f.Add("a", "a", 0, 0, -1)
	f.Add("", "", -1, math.MinInt, math.MinInt)
	f.Add(" ", "b", 3, math.MaxInt, math.MaxInt)
	f.Fuzz(func(t *testing.T, grappler, target string, index, dc, rng int) {
		var part combat.GraspingPart
		if h, err := combat.NewGraspingHand(index); err == nil {
			part = h
		}
		g, err := combat.NewGrapple(combat.CombatantID(grappler), combat.CombatantID(target), part, d20.DifficultyClass(dc), rng)
		bad := grappler == "" || target == "" || part == nil || grappler == target || rng < 0
		if (err != nil) != bad {
			t.Fatalf("NewGrapple(%q, %q, %v, %d, %d) = %v, %v; invalid = %v", grappler, target, part, dc, rng, g, err, bad)
		}
		if err != nil {
			problemOf(t, err)
		}
	})
}
