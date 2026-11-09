package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-bard.md FR-022 (SRD 5.2.1, "Bard Features", p. 31) - a Bard level below 1 or above 20 is refused with an error matching ErrInvalidInput; levels 0, 1, 20 and 21 are tested; MustBardLevel panics with that error and
// only then; the zero BardLevel is invalid.
// Path: per case (unhappy: boundary).
func TestNewBardLevelBoundaries(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n  int
		ok bool
	}{{0, false}, {1, true}, {20, true}, {21, false}, {-1, false}} {
		l, err := character.NewBardLevel(c.n)
		if (err == nil) != c.ok || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || (c.ok && (l.Value() != c.n || l.IsZero())) || (!c.ok && !l.IsZero()) {
			t.Errorf("NewBardLevel(%d) = %v, %v", c.n, l, err)
		}
	}
	if character.MustBardLevel(7).Value() != 7 {
		t.Error("MustBardLevel(7) is not 7")
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
			t.Errorf("MustBardLevel(21) recovered %v, want an ErrInvalidInput error", r)
		}
	}()
	character.MustBardLevel(21)
}

// Requirement: doc/specs/class-bard.md FR-125 (SRD 5.2.1, "Bardic Inspiration", "At Higher Levels", p. 32) - the Bardic Die is a d6 at levels 1 to 4, a d8 at 5 to 9, a d10 at 10 to 14 and a d12 at 15 to 20, and maps to the kernel Die
// with the same number of sides; the bundled Features table is compared in scenarios/class_bard_test.go.
// Path: per case (the 20 levels): happy / unhappy (boundary at 4/5, 9/10, 14/15).
func TestBardicDieAt(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 20; n++ {
		want := character.BardicD6
		switch {
		case n >= 15:
			want = character.BardicD12
		case n >= 10:
			want = character.BardicD10
		case n >= 5:
			want = character.BardicD8
		}
		got := character.BardicDieAt(character.MustBardLevel(n))
		if got != want || got.Die().Sides() != want.Sides() || !got.Valid() {
			t.Errorf("BardicDieAt(%d) = %v, want %v", n, got, want)
		}
	}
	if character.BardicDieAt(character.BardLevel{}) != character.BardicD6 {
		t.Error("the zero level does not give the d6")
	}
}

// Requirement: doc/specs/class-bard.md FR-125 - a BardicDie that is not one of the four is invalid, has 0 sides and renders "unknown"; the four render d6, d8, d10 and d12.
// Path: per case (unhappy: invalid input).
func TestBardicDieValidity(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		d     character.BardicDie
		valid bool
		sides int
		str   string
	}{{character.BardicD6, true, 6, "d6"}, {character.BardicD8, true, 8, "d8"}, {character.BardicD10, true, 10, "d10"}, {character.BardicD12, true, 12, "d12"}, {0, false, 0, "unknown"}, {7, false, 0, "unknown"}} {
		if c.d.Valid() != c.valid || c.d.Sides() != c.sides || c.d.String() != c.str {
			t.Errorf("BardicDie(%d) = valid %v, sides %d, %q", int(c.d), c.d.Valid(), c.d.Sides(), c.d.String())
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-120 - the Bard and the College of Lore ids are the SRD's.
// Path: happy.
func TestBardIDs(t *testing.T) {
	t.Parallel()
	if character.BardClassID != "bard" || character.CollegeOfLoreSubclassID != "college_of_lore" {
		t.Errorf("ids = %q, %q", character.BardClassID, character.CollegeOfLoreSubclassID)
	}
}
