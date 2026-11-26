package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/grappling.md FR-012 - the empty set holds nothing: every part is not holding, no creature is
// grappled, HeldBy is empty (and not nil).
// Path: unhappy (empty and minimal).
func TestEmptyGrapples(t *testing.T) {
	t.Parallel()
	for name, s := range map[string]combat.Grapples{"zero": {}, "from nil": grapplesOf(t), "from empty": grapplesOf(t, []combat.Grapple{}...)} {
		if s.Len() != 0 || len(s.All()) != 0 || !s.Equal(combat.Grapples{}) {
			t.Errorf("%s: not empty", name)
		}
		if s.IsHolding(gid("a"), hand(0)) != combat.PartStateNotHoldingGrapple {
			t.Errorf("%s: a part is holding", name)
		}
		if _, ok := s.GrappleOf(gid("b")).(combat.NotInGrapple); !ok {
			t.Errorf("%s: a creature is grappled", name)
		}
		if h := s.HeldBy(gid("a")); h == nil || len(h) != 0 {
			t.Errorf("%s: HeldBy = %#v, want an empty non-nil slice", name, h)
		}
	}
}

// Requirement: doc/specs/grappling.md FR-013, FR-014, FR-015 - two grapples with the same grappler and part, or with the
// same target, are rejected with ErrInvalidInput; one grappler on different parts, and different grapplers on equal parts,
// are accepted.
// Path: per case.
func TestNewGrapplesInvariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		gs                      []combat.Grapple
		bad                     bool
	}{
		{"one grappler, two parts, two targets", `FR-015; "One Grapple per Hand", p. 182`, "unhappy (boundary)",
			[]combat.Grapple{held("a", "b", hand(0)), held("a", "c", hand(1))}, false},
		{"two grapplers, equal parts", "FR-015", "unhappy (boundary)",
			[]combat.Grapple{held("a", "b", hand(0)), held("d", "c", hand(0))}, false},
		{"same grappler and part", "FR-013", "unhappy (invalid input)",
			[]combat.Grapple{held("a", "b", hand(0)), held("a", "c", hand(0))}, true},
		{"same target, other grappler", `FR-014; "Condition", p. 179`, "unhappy (invalid input)",
			[]combat.Grapple{held("a", "b", hand(0)), held("d", "b", hand(0))}, true},
		{"same target, other part", "FR-014", "unhappy (invalid input)",
			[]combat.Grapple{held("a", "b", hand(0)), held("a", "b", hand(1))}, true},
		{"the same grapple twice", "FR-013", "unhappy (invalid input)",
			[]combat.Grapple{held("a", "b", hand(0)), held("a", "b", hand(0))}, true},
		{"an unbuilt grapple", "package design", "unhappy (invalid input)", []combat.Grapple{{}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.NewGrapples(c.gs)
			if c.bad {
				if !errors.Is(err, srderr.ErrInvalidInput) || s.Len() != 0 {
					t.Errorf("= %v, %v, want ErrInvalidInput [%s]", s, err, c.requirement)
				}
				return
			}
			if err != nil || s.Len() != len(c.gs) {
				t.Errorf("= %v, %v, want a set of %d [%s]", s, err, len(c.gs), c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-083 - MustGrapples panics with the NewGrapples error, and only then.
// Path: unhappy (invalid input).
func TestMustGrapples(t *testing.T) {
	t.Parallel()
	if combat.MustGrapples([]combat.Grapple{held("a", "b", hand(0))}).Len() != 1 {
		t.Error("MustGrapples dropped a grapple")
	}
	defer func() {
		if err, _ := recover().(error); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an ErrInvalidInput error", err)
		}
	}()
	combat.MustGrapples([]combat.Grapple{{}})
}

// Requirement: doc/specs/grappling.md FR-016 - the set copies its input and returns copies: changing the input slice or a
// returned slice does not change the value.
// Path: unhappy (immutability).
func TestGrapplesCopiesInAndOut(t *testing.T) {
	t.Parallel()
	in := []combat.Grapple{held("a", "b", hand(0)), held("a", "c", hand(1))}
	s := grapplesOf(t, in...)
	in[0] = held("x", "y", hand(2))
	s.All()[0] = held("x", "y", hand(2))
	s.HeldBy(gid("a"))[1] = held("x", "y", hand(2))
	want := grapplesOf(t, held("a", "b", hand(0)), held("a", "c", hand(1)))
	if !s.Equal(want) || !slices.Equal(s.All(), want.All()) {
		t.Errorf("set = %v after outside changes, want %v", s.All(), want.All())
	}
}

// Requirement: doc/specs/grappling.md FR-017, FR-021 - Equal ignores insertion order and notices any difference; iteration
// and HeldBy follow the order the grapples began.
// Path: per case.
func TestGrapplesEqualAndOrder(t *testing.T) {
	t.Parallel()
	ab, ac, db := held("a", "b", hand(0)), held("a", "c", hand(1)), held("d", "e", hand(0))
	s := grapplesOf(t, ab, ac, db)
	if !s.Equal(grapplesOf(t, db, ac, ab)) {
		t.Error("the same grapples in another order are unequal [FR-017]")
	}
	for name, o := range map[string]combat.Grapples{
		"fewer":      grapplesOf(t, ab, ac),
		"more":       grapplesOf(t, ab, ac, db, held("f", "g", hand(0))),
		"one differ": grapplesOf(t, ab, ac, held("d", "e", hand(1))),
	} {
		if s.Equal(o) || o.Equal(s) {
			t.Errorf("%s: sets are equal [FR-017]", name)
		}
	}
	if !slices.Equal(s.All(), []combat.Grapple{ab, ac, db}) || !slices.Equal(s.HeldBy(gid("a")), []combat.Grapple{ab, ac}) {
		t.Errorf("order = %v / %v, want the order begun [FR-017, FR-021]", s.All(), s.HeldBy(gid("a")))
	}
	if h := s.HeldBy(gid("zed")); len(h) != 0 {
		t.Errorf("HeldBy of a grappler holding none = %v", h)
	}
}

// Requirement: doc/specs/grappling.md FR-018, FR-019 - IsHolding is per grappler and part; CanTarget forbids using a
// part that holds another creature and allows an idle part or the held target ("Grappling", p. 182).
// Path: per case.
func TestIsHoldingAndCanTarget(t *testing.T) {
	t.Parallel()
	s := grapplesOf(t, held("a", "b", hand(0)))
	cases := []struct {
		name, requirement, path string
		grappler                string
		part                    combat.GraspingPart
		target                  string
		holding                 combat.PartState
		permission              combat.ConditionPermission
	}{
		{"the part holding b", "FR-018", "happy", "a", hand(0), "b", combat.PartStateHolding, combat.ConditionPermissionAllowed},
		{"the part holding b, another target", "FR-019", "unhappy (rule interaction)", "a", hand(0), "c", combat.PartStateHolding, combat.ConditionPermissionForbidden},
		{"another part of a", "FR-019", "unhappy (rule interaction)", "a", hand(1), "c", combat.PartStateNotHoldingGrapple, combat.ConditionPermissionAllowed},
		{"hand 0 of another grappler", "FR-018", "unhappy (boundary)", "d", hand(0), "c", combat.PartStateNotHoldingGrapple, combat.ConditionPermissionAllowed},
		{"the held target compares below the asked one", "FR-019", "unhappy (rule interaction)", "a", hand(0), "a0", combat.PartStateHolding, combat.ConditionPermissionForbidden},
		{"a grappler that sorts before the holder", "FR-019", "unhappy (boundary)", "9", hand(0), "c", combat.PartStateNotHoldingGrapple, combat.ConditionPermissionAllowed},
		{"a body part with hand 0's index", "FR-005", "unhappy (boundary)", "a", combat.MustGraspingBodyPart("0"), "c", combat.PartStateNotHoldingGrapple, combat.ConditionPermissionAllowed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := s.IsHolding(gid(c.grappler), c.part); got != c.holding {
				t.Errorf("IsHolding = %v, want %v [%s]", got, c.holding, c.requirement)
			}
			if got := s.CanTarget(gid(c.grappler), c.part, gid(c.target)); got != c.permission {
				t.Errorf("CanTarget = %v, want %v [%s]", got, c.permission, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-020 - GrappleOf returns the grapple holding the target, or NotInGrapple.
// Path: happy.
func TestGrappleOf(t *testing.T) {
	t.Parallel()
	g := held("a", "b", hand(0))
	s := grapplesOf(t, g)
	if in, ok := s.GrappleOf(gid("b")).(combat.InGrapple); !ok || in.Grapple() != g {
		t.Errorf("GrappleOf(b) = %#v, want %v", s.GrappleOf(gid("b")), g)
	}
	if _, ok := s.GrappleOf(gid("a")).(combat.NotInGrapple); !ok {
		t.Error("the grappler itself is not a held target")
	}
}

// Requirement: doc/specs/grappling.md FR-062 to FR-065, table 4.10.3 - the extra movement cost per foot is the sum of the
// cost of each carried creature ("Grappled [Condition]", "Movable", p. 182); nothing carried costs 0; a creature the
// grappler does not hold, a missing or invalid size, a repeated creature and an invalid grappler size are ErrInvalidInput.
// Path: per case.
func TestExtraMovementCostPerFoot(t *testing.T) {
	t.Parallel()
	const p182 = `table 4.10.3; "Movable", p. 182`
	two := grapplesOf(t, held("a", "b", hand(0)), held("a", "c", hand(1)))
	one := grapplesOf(t, held("a", "b", hand(0)))
	sz := func(b, c content.CreatureSize) map[combat.CombatantID]content.CreatureSize {
		return map[combat.CombatantID]content.CreatureSize{gid("b"): b, gid("c"): c}
	}
	cases := []struct {
		name, requirement, path string
		set                     combat.Grapples
		size                    content.CreatureSize
		carried                 []combat.CombatantID
		sizes                   map[combat.CombatantID]content.CreatureSize
		want                    int
		bad                     bool
	}{
		{"carries same size", "FR-093, " + p182, "happy", one, content.SizeMedium, []combat.CombatantID{gid("b")}, sz(content.SizeMedium, 0), 1, false},
		{"carries one size smaller", p182, "unhappy (boundary)", one, content.SizeLarge, []combat.CombatantID{gid("b")}, sz(content.SizeMedium, 0), 1, false},
		{"carries two sizes smaller", p182, "unhappy (boundary)", one, content.SizeLarge, []combat.CombatantID{gid("b")}, sz(content.SizeSmall, 0), 0, false},
		{"carries Tiny", p182, "unhappy (boundary)", one, content.SizeMedium, []combat.CombatantID{gid("b")}, sz(content.SizeTiny, 0), 0, false},
		{"Tiny carries Tiny", p182, "unhappy (boundary)", one, content.SizeTiny, []combat.CombatantID{gid("b")}, sz(content.SizeTiny, 0), 0, false},
		{"carries larger", p182, "unhappy (boundary)", one, content.SizeMedium, []combat.CombatantID{gid("b")}, sz(content.SizeLarge, 0), 1, false},
		{"two carried, both costly", "FR-062, Q-5", "unhappy (rule interaction)", two, content.SizeLarge, []combat.CombatantID{gid("b"), gid("c")}, sz(content.SizeMedium, content.SizeMedium), 2, false},
		{"two carried, one free", "FR-062, Q-5", "unhappy (rule interaction)", two, content.SizeLarge, []combat.CombatantID{gid("b"), gid("c")}, sz(content.SizeMedium, content.SizeSmall), 1, false},
		{"holds two, carries one", "FR-062", "unhappy (rule interaction)", two, content.SizeLarge, []combat.CombatantID{gid("b")}, sz(content.SizeMedium, content.SizeMedium), 1, false},
		{"holds one, carries none", "FR-063", "unhappy (empty and minimal)", one, content.SizeMedium, nil, nil, 0, false},
		{"carries an unheld creature", "FR-064", "unhappy (invalid input)", one, content.SizeMedium, []combat.CombatantID{gid("c")}, sz(content.SizeMedium, content.SizeMedium), 0, true},
		{"carries a creature another grappler holds", "FR-064", "unhappy (invalid input)", grapplesOf(t, held("d", "b", hand(0))), content.SizeMedium, []combat.CombatantID{gid("b")}, sz(content.SizeMedium, 0), 0, true},
		{"held by a grappler that sorts before the asked one", "FR-064", "unhappy (invalid input)", grapplesOf(t, held("0", "b", hand(0))), content.SizeMedium, []combat.CombatantID{gid("b")}, sz(content.SizeMedium, 0), 0, true},
		{"size missing", "FR-065", "unhappy (invalid input)", one, content.SizeMedium, []combat.CombatantID{gid("b")}, map[combat.CombatantID]content.CreatureSize{}, 0, true},
		{"size nil map", "FR-065", "unhappy (invalid input)", one, content.SizeMedium, []combat.CombatantID{gid("b")}, nil, 0, true},
		{"size invalid", "FR-065", "unhappy (invalid input)", one, content.SizeMedium, []combat.CombatantID{gid("b")}, sz(0, 0), 0, true},
		{"grappler size invalid", "package design", "unhappy (invalid input)", one, 0, []combat.CombatantID{gid("b")}, sz(content.SizeMedium, 0), 0, true},
		{"same creature twice", "package design", "unhappy (invalid input)", one, content.SizeMedium, []combat.CombatantID{gid("b"), gid("b")}, sz(content.SizeMedium, 0), 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.set.ExtraMovementCostPerFoot(gid("a"), c.size, c.carried, c.sizes)
			if c.bad != errors.Is(err, srderr.ErrInvalidInput) || (err != nil) != c.bad || got != c.want {
				t.Errorf("= %d, %v, want %d (invalid %v) [%s]", got, err, c.want, c.bad, c.requirement)
			}
		})
	}
}
