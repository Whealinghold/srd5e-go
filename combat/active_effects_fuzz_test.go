package combat_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-001, FR-003, FR-004 - the constructors that take untrusted text never panic, keep a valid value unchanged and refuse a blank one with an error matching ErrInvalidInput.
// Path: unhappy (invalid input).
func FuzzActiveEffectTextConstructors(f *testing.F) {
	for _, s := range []string{"", " ", "a", " a ", "\t", "é", "id with spaces", "\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		blank := strings.TrimSpace(s) == ""
		id, idErr := combat.NewActiveEffectID(s)
		place, placeErr := combat.NewEffectPlaceID(s)
		other, otherErr := combat.NewOtherEffectOrigin(s)
		for name, err := range map[string]error{"id": idErr, "place": placeErr, "origin": otherErr} {
			if blank != (err != nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Fatalf("%s(%q): err = %v, blank = %v", name, s, err, blank)
			}
		}
		if !blank && (string(id) != s || string(place) != s || other.Key() != s) {
			t.Fatalf("%q was changed", s)
		}
	})
}

// Requirement: doc/specs/active-effects.md FR-006, FR-005, FR-003 - converting a time span and building a duration, an elapsed time or an origin from untrusted numbers never panics; a result is positive, equals amount times the unit and never overflows.
// Path: unhappy (boundary).
func FuzzActiveEffectNumberConstructors(f *testing.F) {
	for _, n := range []int{0, 1, -1, 9, 10, 1 << 30, -1 << 30} {
		for u := 0; u <= 5; u++ {
			f.Add(n, u)
		}
	}
	f.Fuzz(func(t *testing.T, n, unit int) {
		rounds, err := combat.RoundsInTimeSpan(n, content.DurationUnit(unit))
		if err == nil && (rounds < n || rounds < 1) {
			t.Fatalf("RoundsInTimeSpan(%d, %d) = %d", n, unit, rounds)
		}
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("err = %v, want ErrInvalidInput", err)
		}
		_, e1 := combat.NewEffectLastsRounds(n)
		_, e2 := combat.NewTimeElapsed(n)
		_, e3 := combat.NewSpellEffectOrigin("bless", n)
		if (e1 == nil) != (n >= 1) || (e2 == nil) != (n >= 1) || (e3 == nil) != (n >= 0 && n <= 9) {
			t.Fatalf("n = %d: %v %v %v", n, e1, e2, e3)
		}
	})
}
