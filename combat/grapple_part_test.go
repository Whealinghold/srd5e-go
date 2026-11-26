package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// problemOf returns the GrappleProblem of err or fails the test.
func problemOf(tb testing.TB, err error) combat.GrappleProblem {
	tb.Helper()
	var pe *combat.GrappleProblemError
	if !errors.As(err, &pe) || !errors.Is(err, srderr.ErrInvalidInput) {
		tb.Fatalf("error = %v, want a *GrappleProblemError matching ErrInvalidInput", err)
	}
	return pe.Problem
}

// Requirement: doc/specs/grappling.md FR-001, FR-002 - a hand index of 0 or more is accepted, a negative one is a
// GrappleProblemNegativeHandIndex error matching ErrInvalidInput.
// Path: per case.
func TestNewGraspingHand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		index                   int
		bad                     bool
	}{
		{"hand 0", `FR-001; "One Grapple per Hand", p. 182`, "happy", 0, false},
		{"hand 1", "FR-001", "happy", 1, false},
		{"a creature with many hands", "FR-001", "unhappy (boundary)", 1000, false},
		{"hand -1", "FR-002", "unhappy (boundary)", -1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			h, err := combat.NewGraspingHand(c.index)
			if c.bad {
				if problemOf(t, err) != combat.GrappleProblemNegativeHandIndex {
					t.Errorf("problem = %v [%s]", err, c.requirement)
				}
				return
			}
			if err != nil || h.Index() != c.index {
				t.Errorf("NewGraspingHand(%d) = %v, %v", c.index, h, err)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-003, FR-004 - a body part name is non-empty with no leading or trailing
// whitespace; anything else is a GrappleProblemBlankPartName error.
// Path: per case.
func TestNewGraspingBodyPart(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		in                      string
		bad                     bool
	}{
		{"tentacle", `FR-003; "Grappling", p. 182`, "happy", "tentacle", false},
		{"inner space", "FR-003", "unhappy (boundary)", "left maw", false},
		{"empty", "FR-004", "unhappy (invalid input)", "", true},
		{"only whitespace", "FR-004", "unhappy (invalid input)", " \t", true},
		{"leading space", "FR-004", "unhappy (invalid input)", " maw", true},
		{"trailing newline", "FR-004", "unhappy (invalid input)", "maw\n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := combat.NewGraspingBodyPart(c.in)
			if c.bad {
				if problemOf(t, err) != combat.GrappleProblemBlankPartName {
					t.Errorf("problem = %v [%s]", err, c.requirement)
				}
				return
			}
			if err != nil || p.Name() != c.in {
				t.Errorf("NewGraspingBodyPart(%q) = %v, %v", c.in, p, err)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-005 - parts compare with ==: equal when the same variant with the same index
// or name; hand 0 differs from hand 1 and a hand never equals a body part.
// Path: unhappy (boundary).
func TestGraspingPartsCompareByValue(t *testing.T) {
	t.Parallel()
	var a, b combat.GraspingPart = hand(0), hand(0)
	if a != b {
		t.Error("hand 0 != hand 0")
	}
	if a == combat.GraspingPart(hand(1)) {
		t.Error("hand 0 == hand 1")
	}
	if combat.GraspingPart(combat.MustGraspingBodyPart("maw")) != combat.GraspingPart(combat.MustGraspingBodyPart("maw")) {
		t.Error("maw != maw")
	}
	if combat.GraspingPart(combat.MustGraspingBodyPart("0")) == a {
		t.Error("a body part named 0 equals hand 0")
	}
}

// Requirement: doc/specs/grappling.md FR-083 - MustX returns what NewX returns and panics with its error, and only then.
// Path: unhappy (invalid input).
func TestMustPartsPanicOnlyOnInvalidInput(t *testing.T) {
	t.Parallel()
	if combat.MustGraspingHand(2).Index() != 2 || combat.MustGraspingBodyPart("maw").Name() != "maw" {
		t.Error("Must returned the wrong part")
	}
	for name, f := range map[string]func(){
		"hand": func() { combat.MustGraspingHand(-1) },
		"part": func() { combat.MustGraspingBodyPart("") },
	} {
		func() {
			defer func() {
				err, _ := recover().(error)
				if !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("%s: panic value = %v, want an ErrInvalidInput error", name, err)
				}
			}()
			f()
		}()
	}
}

// Requirement: doc/specs/grappling.md FR-083, testing.md section 10 - the part constructors never panic and return an
// error exactly for invalid input.
// Path: unhappy (invalid input).
func FuzzNewGraspingBodyPart(f *testing.F) {
	for _, s := range []string{"", " ", "maw", " maw", "maw ", "\xff", "tentacle "} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		p, err := combat.NewGraspingBodyPart(name)
		if err != nil {
			if problemOf(t, err) != combat.GrappleProblemBlankPartName {
				t.Fatalf("unexpected problem for %q", name)
			}
			return
		}
		if p.Name() != name || name == "" {
			t.Fatalf("accepted %q but holds %q", name, p.Name())
		}
	})
}

// Requirement: doc/specs/grappling.md FR-083 - NewGraspingHand never panics.
// Path: unhappy (invalid input).
func FuzzNewGraspingHand(f *testing.F) {
	for _, n := range []int{0, 1, -1, 1 << 30, -1 << 30} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		h, err := combat.NewGraspingHand(n)
		if (err == nil) != (n >= 0) || err == nil && h.Index() != n {
			t.Fatalf("NewGraspingHand(%d) = %v, %v", n, h, err)
		}
	})
}
