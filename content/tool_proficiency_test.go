package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: architecture.md §3 (a failed constructor returns the zero value), SRD 5.2.1, "Parts of a Background: Tool Proficiency", p. 83.
// Path: unhappy (invalid input).
func TestToolProficiencyFailureReturnsZeroValue(t *testing.T) {
	t.Parallel()
	if s, err := content.NewSpecificToolProficiency(" x"); err == nil || s != (content.SpecificToolProficiency{}) {
		t.Errorf("NewSpecificToolProficiency(invalid) = (%+v, %v), want zero value and an error", s, err)
	}
	if c, err := content.NewChoiceToolProficiency(" "); err == nil || c != (content.ChoiceToolProficiency{}) {
		t.Errorf("NewChoiceToolProficiency(blank) = (%+v, %v), want zero value and an error", c, err)
	}
}
