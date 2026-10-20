package content_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/content-schema-and-decoder.md FR-149 (SRD 5.2.1,
// "Duration", p. 106) - four members in ascending order; String and
// ParseDurationUnit round-trip; wrong case, singular, years and the empty
// string return a *ParseError wrapping ErrInvalidInput.
// Path: per case.
func TestDurationUnit(t *testing.T) {
	t.Parallel()
	for i, name := range []string{"rounds", "minutes", "hours", "days"} {
		u, err := content.ParseDurationUnit(name)
		if err != nil || int(u) != i+1 || u.String() != name {
			t.Errorf("ParseDurationUnit(%q) = (%d, %v), want ordinal %d [happy]", name, u, err, i+1)
		}
	}
	for _, in := range []string{"Rounds", "hour", "years", "weeks", "", " days", "1"} {
		u, err := content.ParseDurationUnit(in)
		var pe *content.ParseError
		if !errors.As(err, &pe) || pe.Value != in || pe.Vocabulary != "DurationUnit" || !errors.Is(err, srderr.ErrInvalidInput) || u != 0 {
			t.Errorf("ParseDurationUnit(%q) = (%d, %v), want *ParseError [unhappy (invalid input)]", in, u, err)
		}
	}
	for _, u := range []content.DurationUnit{0, 5, -1} {
		if u.String() != "unknown" {
			t.Errorf("DurationUnit(%d).String() = %q, want unknown [unhappy (boundary)]", u, u.String())
		}
	}
}
