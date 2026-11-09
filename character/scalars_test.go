package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/character-aggregate.md FR-001, FR-002, FR-003, FR-073,
// FR-101 - ids compare by value; an empty or whitespace id is rejected with a
// typed reason and never panics; MustIdentifier panics iff NewIdentifier errs.
// Path: per case.
func TestIdentifier(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, in, requirement, path string
		bad                         bool
	}{
		{"plain", "pc-1", "FR-001", "happy", false},
		{"empty", "", "FR-002", "unhappy (invalid input)", true},
		{"whitespace", " \t\n", "FR-002", "unhappy (invalid input)", true},
		{"padded is kept", " a ", "FR-003", "happy", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			id, err := character.NewIdentifier(c.in)
			panicked := recovered(t, func() { character.MustIdentifier(c.in) })
			if c.bad {
				var ie *character.IdentifierError
				if !errors.As(err, &ie) || ie.Reason != character.IdentifierEmpty || !errors.Is(err, srderr.ErrInvalidInput) || !id.IsZero() {
					t.Errorf("%s: err = %v", c.requirement, err)
				}
				if !errors.Is(panicked, err) {
					t.Errorf("MustIdentifier panic = %v, want %v", panicked, err)
				}
				return
			}
			if err != nil || panicked != nil || id.String() != c.in || id.IsZero() {
				t.Errorf("%s: id %q err %v panic %v", c.requirement, id.String(), err, panicked)
			}
		})
	}
	a1, a2, b := character.MustIdentifier("a"), character.MustIdentifier("a"), character.MustIdentifier("b")
	if a1 != a2 || a1 == b {
		t.Error("FR-001: ids must compare by value")
	}
	if got := character.IdentifierEmpty.String() + character.IdentifierReason(0).String(); got != "emptyunknown" {
		t.Errorf("reason strings = %q", got)
	}
	if (&character.IdentifierError{Reason: character.IdentifierEmpty}).Error() == "" || errors.Is(&character.IdentifierError{}, errors.New("x")) {
		t.Error("error text or Is mismatch")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-008, FR-101 - Experience
// Points reject a negative value, accept 0, never panic; Must agrees.
// Path: per case.
func TestExperiencePoints(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		in  int
		bad bool
	}{{-1, true}, {0, false}, {1, false}, {355000, false}} {
		xp, err := character.NewExperiencePoints(c.in)
		panicked := recovered(t, func() { character.MustExperiencePoints(c.in) })
		if c.bad {
			var e *character.ExperiencePointsError
			if !errors.As(err, &e) || e.Reason != character.ExperiencePointsNegative || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(panicked, err) {
				t.Errorf("%d: err %v panic %v", c.in, err, panicked)
			}
			continue
		}
		if err != nil || panicked != nil || xp.Value() != c.in {
			t.Errorf("%d: xp %d err %v panic %v", c.in, xp.Value(), err, panicked)
		}
	}
	if character.ExperiencePointsNegative.String() != "negative" || character.ExperiencePointsReason(0).String() != "unknown" {
		t.Error("reason strings")
	}
	e := &character.ExperiencePointsError{Reason: character.ExperiencePointsNegative}
	if e.Error() == "" || !e.Is(&character.ExperiencePointsError{Reason: character.ExperiencePointsNegative}) || e.Is(&character.ExperiencePointsError{}) {
		t.Error("error text or Is")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-022, FR-081, FR-101 - an award
// is positive; 0 and negative are rejected with their own reason; Must agrees.
// Path: per case.
func TestExperienceAmount(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		in     int
		reason character.ExperienceAmountReason
	}{{-1, character.ExperienceAmountNegative}, {0, character.ExperienceAmountZero}, {1, 0}} {
		a, err := character.NewExperienceAmount(c.in)
		panicked := recovered(t, func() { character.MustExperienceAmount(c.in) })
		if c.reason == 0 {
			if err != nil || panicked != nil || a.Value() != 1 || a.IsZero() {
				t.Errorf("%d: %v %v", c.in, err, panicked)
			}
			continue
		}
		var e *character.ExperienceAmountError
		if !errors.As(err, &e) || e.Reason != c.reason || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(panicked, err) || !a.IsZero() {
			t.Errorf("%d: err %v panic %v", c.in, err, panicked)
		}
	}
	if character.ExperienceAmountZero.String() != "zero" || character.ExperienceAmountNegative.String() != "negative" || character.ExperienceAmountReason(0).String() != "unknown" {
		t.Error("reason strings")
	}
	e := &character.ExperienceAmountError{Reason: character.ExperienceAmountZero}
	if e.Error() == "" || !e.Is(&character.ExperienceAmountError{Reason: character.ExperienceAmountZero}) || e.Is(&character.ExperienceAmountError{Reason: character.ExperienceAmountNegative}) {
		t.Error("error text or Is")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-073, FR-081 - the scalar
// constructors never panic for any input and Must panics iff New errs.
// Path: unhappy (invalid input).
func FuzzScalarConstructors(f *testing.F) {
	f.Add("pc", 0)
	f.Add(" ", -1)
	f.Fuzz(func(t *testing.T, s string, n int) {
		_, e1 := character.NewIdentifier(s)
		_, e2 := character.NewExperiencePoints(n)
		_, e3 := character.NewExperienceAmount(n)
		for i, f := range []func(){
			func() { character.MustIdentifier(s) }, func() { character.MustExperiencePoints(n) }, func() { character.MustExperienceAmount(n) },
		} {
			if p := recovered(t, f); (p != nil) != (([]error{e1, e2, e3})[i] != nil) {
				t.Errorf("Must %d panic %v, New error %v", i, p, []error{e1, e2, e3}[i])
			}
		}
	})
}
