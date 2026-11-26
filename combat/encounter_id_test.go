package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestParseEncounterID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		in                      string
		ok                      bool
	}{
		{"empty", encSpec + "FR-001", "unhappy (invalid input)", "", false},
		{"spaces", encSpec + "FR-001", "unhappy (invalid input)", "   ", false},
		{"tab and newline", encSpec + "FR-001", "unhappy (invalid input)", "\t\n", false},
		{"plain", encSpec + "FR-002", "happy", "enc-1", true},
		{"not trimmed", encSpec + "FR-002", "happy", " enc ", true},
		{"non-ascii", encSpec + "FR-002", "happy", "竞技", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			id, err := combat.ParseEncounterID(c.in)
			if c.ok {
				if err != nil || string(id) != c.in || id.IsZero() {
					t.Errorf("ParseEncounterID(%q) = %q, %v, want it unchanged [%s]", c.in, id, err, c.requirement)
				}
				return
			}
			var ie *combat.EncounterIDError
			if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ie) || ie.Reason != combat.EncounterIDEmpty || !id.IsZero() {
				t.Errorf("ParseEncounterID(%q) = %q, %v, want EncounterIDEmpty matching ErrInvalidInput [%s]", c.in, id, err, c.requirement)
			}
			if !errors.Is(err, &combat.EncounterIDError{Reason: combat.EncounterIDEmpty}) || errors.Is(err, &combat.EncounterIDError{Reason: 9}) {
				t.Errorf("Is must match on the Reason")
			}
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-002 - MustEncounterID panics with the error of ParseEncounterID if and only if it returns one; two ids from one string are equal and different strings differ.
// Path: unhappy (invalid input).
func TestMustEncounterID(t *testing.T) {
	t.Parallel()
	if got := combat.MustEncounterID("x"); got != combat.MustEncounterID("x") || got == combat.MustEncounterID("y") {
		t.Errorf("equality of encounter ids is not by value [FR-005]")
	}
	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("MustEncounterID(\"\") panicked with %v, want an ErrInvalidInput error", err)
		}
	}()
	combat.MustEncounterID("")
}

func TestEncounterIDRejectionStrings(t *testing.T) {
	t.Parallel()
	if combat.EncounterIDEmpty.String() != "empty" || combat.EncounterIDRejection(0).String() != "unknown" {
		t.Errorf("rejection names are wrong")
	}
	if (&combat.EncounterIDError{Reason: combat.EncounterIDEmpty}).Error() == "" {
		t.Errorf("Error is empty")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-002 - ParseEncounterID never panics for any string and accepts exactly the strings that are not blank.
// Path: unhappy (invalid input).
func FuzzParseEncounterID(f *testing.F) {
	for _, s := range []string{"", " ", "enc", " enc ", " ", "\x00", "very long " + string(make([]byte, 300))} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id, err := combat.ParseEncounterID(s)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !id.IsZero() {
				t.Fatalf("ParseEncounterID(%q) = %q, %v", s, id, err)
			}
			return
		}
		if string(id) != s || id.IsZero() {
			t.Fatalf("ParseEncounterID(%q) = %q", s, id)
		}
	})
}

// Requirement: doc/specs/encounter-aggregate.md FR-002 - an unknown rejection value is named unknown, and Is matches only an *EncounterIDError with the same reason, never nil or another error.
// Path: unhappy (invalid input).
func TestEncounterIDErrorIsMatchesOnlyItsOwnKind(t *testing.T) {
	t.Parallel()
	e := &combat.EncounterIDError{Reason: combat.EncounterIDEmpty}
	var nilErr *combat.EncounterIDError
	for name, target := range map[string]error{"nil pointer": nilErr, "other error": errors.New("x"), "other reason": &combat.EncounterIDError{Reason: 2}} {
		if e.Is(target) {
			t.Errorf("Is matched %s", name)
		}
	}
	if nilErr.Is(e) {
		t.Errorf("a nil receiver matched")
	}
	if combat.EncounterIDRejection(9).String() != "unknown" || combat.EncounterIDRejection(2).String() != "unknown" {
		t.Errorf("an unknown rejection must be named unknown")
	}
}
