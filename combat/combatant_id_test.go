package combat_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-foundations.md FR-002 to FR-004 - a string that
// is not blank is accepted unchanged; empty and blank strings are rejected with
// a *CombatantIDError (CombatantIDEmpty) matching srderr.ErrInvalidInput.
// Path: per case.
func TestNewCombatantID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, path string
		ok             bool
	}{
		{"plain", "a", "happy", true},
		{"underscore", "goblin_1", "happy", true},
		{"leading space kept", " a", "unhappy (boundary)", true},
		{"trailing space kept", "a ", "unhappy (boundary)", true},
		{"leading no-break space kept", " a", "unhappy (boundary)", true},
		{"empty", "", "unhappy (invalid input)", false},
		{"space", " ", "unhappy (boundary)", false},
		{"tab", "\t", "unhappy (boundary)", false},
		{"newline", "\n", "unhappy (boundary)", false},
		{"carriage return", "\r", "unhappy (boundary)", false},
		{"no-break space", " ", "unhappy (boundary)", false},
		{"mixed blank", "  \n\r ", "unhappy (boundary)", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			id, err := combat.NewCombatantID(c.in)
			if c.ok {
				if err != nil || string(id) != c.in || id.IsZero() {
					t.Fatalf("NewCombatantID(%q) = %q, %v, want it unchanged and nil", c.in, id, err)
				}
				return
			}
			var ce *combat.CombatantIDError
			if !errors.As(err, &ce) || ce.Reason != combat.CombatantIDEmpty {
				t.Fatalf("NewCombatantID(%q) error = %v, want *CombatantIDError{CombatantIDEmpty}", c.in, err)
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &combat.CombatantIDError{Reason: combat.CombatantIDEmpty}) {
				t.Errorf("NewCombatantID(%q) error = %v, want it to match ErrInvalidInput and the same reason", c.in, err)
			}
			if !id.IsZero() {
				t.Errorf("NewCombatantID(%q) id = %q, want the zero id", c.in, id)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-003 - the rejection reason has
// a name, and an error with another reason or another type does not match.
// Path: unhappy (invalid input).
func TestCombatantIDErrorMatching(t *testing.T) {
	t.Parallel()
	e := &combat.CombatantIDError{Reason: combat.CombatantIDEmpty}
	if got := e.Reason.String(); got != "empty" {
		t.Errorf("CombatantIDEmpty.String() = %q, want empty", got)
	}
	if got := combat.CombatantIDRejection(99).String(); got != "unknown" {
		t.Errorf("rejection 99 String() = %q, want unknown", got)
	}
	if got := combat.CombatantIDRejection(0).String(); got != "unknown" {
		t.Errorf("zero rejection String() = %q, want unknown", got)
	}
	if e.Error() == "" {
		t.Error("Error() is empty")
	}
	if errors.Is(e, &combat.CombatantIDError{Reason: 99}) {
		t.Error("errors.Is matched a different reason")
	}
	if e.Is(errors.New("x")) || e.Is((*combat.CombatantIDError)(nil)) {
		t.Error("Is matched a foreign or nil target")
	}
	var nilErr *combat.CombatantIDError
	if nilErr.Is(e) {
		t.Error("nil receiver matched")
	}
}

// Requirement: doc/specs/combat-foundations.md FR-005 - MustCombatantID returns
// the id NewCombatantID returns and panics with that error otherwise.
// Path: unhappy (invalid input).
func TestMustCombatantID(t *testing.T) {
	t.Parallel()
	if got := combat.MustCombatantID("goblin_1"); got != "goblin_1" {
		t.Errorf("MustCombatantID = %q, want goblin_1", got)
	}
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic value = %#v, want an error", r)
		}
		var ce *combat.CombatantIDError
		if !errors.As(err, &ce) || ce.Reason != combat.CombatantIDEmpty {
			t.Errorf("panic error = %v, want *CombatantIDError{CombatantIDEmpty}", err)
		}
	}()
	combat.MustCombatantID(" ")
	t.Fatal("MustCombatantID(\" \") did not panic")
}

// Requirement: doc/specs/combat-foundations.md FR-006, FR-007 - ids are not
// normalised; equal strings give equal ids that work as the same map key.
// Path: unhappy (boundary).
func TestCombatantIDIsNotNormalised(t *testing.T) {
	t.Parallel()
	a, b, c := combat.MustCombatantID("Goblin 1"), combat.MustCombatantID("goblin 1"), combat.MustCombatantID(" Goblin 1")
	if a == b || a == c || b == c {
		t.Errorf("ids %q, %q, %q must be pairwise different", a, b, c)
	}
	m := map[combat.CombatantID]int{combat.MustCombatantID("x"): 1}
	if m[combat.MustCombatantID("x")] != 1 || m[combat.MustCombatantID("y")] != 0 {
		t.Error("equal ids are not the same map key, or different ids are")
	}
}

// Requirement: doc/specs/combat-foundations.md FR-008 - a CombatantID is not
// assignable to, or from, another id type (the go/types check of the spec,
// done with reflect's assignability rules, which are the same).
// Path: unhappy (invalid input).
func TestCombatantIDIsADistinctType(t *testing.T) {
	t.Parallel()
	id := reflect.TypeOf(combat.CombatantID(""))
	for _, other := range []reflect.Type{reflect.TypeOf(content.MonsterID("")), reflect.TypeOf(character.Identifier{}), reflect.TypeOf(combat.EncounterID(""))} {
		if id.AssignableTo(other) || other.AssignableTo(id) {
			t.Errorf("%v and %v are assignable", id, other)
		}
	}
}

// Requirement: doc/specs/combat-foundations.md FR-005 - NewCombatantID never
// panics for any string, and agrees with the blank predicate.
// Path: unhappy (invalid input).
func FuzzNewCombatantID(f *testing.F) {
	for _, s := range []string{"", " ", "a", " ", "\r\n", "goblin 1", "\xff\xfe", string(make([]byte, 100000))} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id, err := combat.NewCombatantID(s)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !id.IsZero() {
				t.Fatalf("NewCombatantID(%q) = %q, %v, want the zero id and ErrInvalidInput", s, id, err)
			}
			return
		}
		if string(id) != s {
			t.Fatalf("NewCombatantID(%q) = %q, want it unchanged", s, id)
		}
	})
}
