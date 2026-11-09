package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-054 to FR-058 -
// Table EA, one row each: the combined count is the largest per-class count (the
// features do not stack), unchanged for one class or classes with no Extra Attack,
// independent of order, with the input list unaltered; an empty list or a count
// below 1 is invalid input naming the parameter and value.
// Path: per case.
func TestExtraAttackCombination(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		counts                  []int
		want                    int
		badValue                int // the InvalidValue of an invalid input
		invalid                 bool
	}{
		{"single class, below level 5", aiSpec + " FR-056 (SRD 5.2.1, Multiclassing, Extra Attack, p. 25)", "happy", []int{1}, 1, 0, false},
		{"single class, level 5", aiSpec + " FR-056", "happy", []int{2}, 2, 0, false},
		{"no class gives Extra Attack", aiSpec + " FR-056", "happy", []int{1, 1}, 1, 0, false},
		{"two classes give 2, no stacking", aiSpec + " FR-054, FR-055", "unhappy (rule interaction)", []int{2, 2}, 2, 0, false},
		{"three classes give 2", aiSpec + " FR-055", "unhappy (boundary)", []int{2, 2, 2}, 2, 0, false},
		{"Fighter 11 and Paladin 5", aiSpec + " FR-055", "unhappy (rule interaction)", []int{3, 2}, 3, 0, false},
		{"Fighter 20 and Ranger 5", aiSpec + " FR-055", "unhappy (rule interaction)", []int{4, 2}, 4, 0, false},
		{"order does not matter, descending", aiSpec + " FR-058", "unhappy (determinism)", []int{3, 2}, 3, 0, false},
		{"order does not matter, ascending", aiSpec + " FR-058", "unhappy (determinism)", []int{2, 3}, 3, 0, false},
		{"one class has it, the other does not", aiSpec + " FR-054", "unhappy (rule interaction)", []int{2, 1}, 2, 0, false},
		{"empty", aiSpec + " FR-057", "unhappy (invalid input)", nil, 0, 0, true},
		{"empty, non-nil", aiSpec + " FR-057", "unhappy (invalid input)", []int{}, 0, 0, true},
		{"count zero", aiSpec + " FR-057", "unhappy (boundary)", []int{0, 2}, 0, 0, true},
		{"negative count", aiSpec + " FR-057", "unhappy (invalid input)", []int{-1}, 0, -1, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(c.counts)
			got, err := character.ExtraAttackCombination(c.counts)
			if !slices.Equal(in, c.counts) {
				t.Error("FR-058: the input must not be altered")
			}
			if c.invalid {
				var e *character.ExtraAttackCountError
				if !isInvalid(err) || !errors.As(err, &e) || e.Parameter != "counts" || e.InvalidValue != c.badValue || got != 0 {
					t.Errorf("= %d, %v, want invalid input counts/%d [%s]", got, err, c.badValue, c.requirement)
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("= %d, %v, want %d [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	if (&character.ExtraAttackCountError{Parameter: "counts"}).Error() == "" {
		t.Error("Error must render")
	}
}
