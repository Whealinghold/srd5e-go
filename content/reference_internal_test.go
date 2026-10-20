package content

import "testing"

// zeroCategorised is an entry whose feat category is the zero value.
type zeroCategorised struct{}

func (zeroCategorised) Key() Key                   { return Key{} }
func (zeroCategorised) FeatCategory() FeatCategory { return 0 }

// Requirement: doc/specs/content-schema-and-decoder.md FR-126 - a zero Category is never satisfied, even by a
// target whose own category is the zero value, so a hand-built constraint cannot match it.
// Path: the zero-category constraint against a zero-category target.
func TestTargetHasFeatCategoryZeroNeverSatisfied(t *testing.T) {
	t.Parallel()
	if (TargetHasFeatCategory{}).satisfiedBy(zeroCategorised{}) {
		t.Fatal("zero category satisfied by a zero-category target")
	}
}
