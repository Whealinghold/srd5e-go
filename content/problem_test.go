package content_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/content-schema-and-decoder.md FR-015 - ProblemKind is
// a closed enum with exactly ten members in schema order, zero invalid, String
// the wire spelling.
// Path: happy.
func TestProblemKindMembers(t *testing.T) {
	t.Parallel()
	want := []string{"syntax", "unsupportedVersion", "missingField", "unknownField", "wrongType", "outOfRange", "invalidValue", "duplicate", "unresolvedReference", "constraintViolation"}
	for i, w := range want {
		if got := content.ProblemKind(i + 1).String(); got != w {
			t.Errorf("ProblemKind(%d).String() = %q, want %q", i+1, got, w)
		}
	}
	for _, k := range []content.ProblemKind{0, -1, content.ProblemConstraintViolation + 1} {
		if k.String() != "unknown" {
			t.Errorf("ProblemKind(%d).String() = %q, want unknown [unhappy (boundary)]", k, k.String())
		}
	}
	if content.ProblemSyntax != 1 || content.ProblemConstraintViolation != 10 {
		t.Errorf("ProblemKind ordinals changed")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-014, FR-097 - Problem
// carries a path and a kind only, is comparable with ==, implements error with a
// value receiver and unwraps to ErrInvalidInput.
// Path: happy.
func TestProblem(t *testing.T) {
	t.Parallel()
	p := content.NewProblem("/spells/0/id", content.ProblemDuplicate)
	if p.Path() != "/spells/0/id" || p.Kind() != content.ProblemDuplicate {
		t.Errorf("accessors wrong: %v", p)
	}
	if p != content.NewProblem("/spells/0/id", content.ProblemDuplicate) ||
		p == content.NewProblem("/spells/1/id", content.ProblemDuplicate) ||
		p == content.NewProblem("/spells/0/id", content.ProblemOutOfRange) {
		t.Errorf("Problem must compare by path and kind")
	}
	var err error = p
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, content.NewProblem("/spells/0/id", content.ProblemDuplicate)) {
		t.Errorf("Problem must wrap ErrInvalidInput and match an equal Problem")
	}
	if p.Error() == "" {
		t.Errorf("Error() is empty")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-017 - Problems
// flattens a joined error in order, returns nil for nil and for a foreign
// error, and returns a new slice.
// Path: per case.
func TestProblems(t *testing.T) {
	t.Parallel()
	a := content.NewProblem("/a", content.ProblemMissingField)
	b := content.NewProblem("/b", content.ProblemWrongType)
	c := content.NewProblem("", content.ProblemSyntax)
	cases := []struct {
		name, path string
		err        error
		want       []content.Problem
	}{
		{"nil error", "unhappy (empty case)", nil, nil},
		{"foreign error", "unhappy (invalid input)", errors.New("boom"), nil},
		{"sentinel only", "unhappy (invalid input)", srderr.ErrInvalidInput, nil},
		{"one problem", "happy", a, []content.Problem{a}},
		{"joined in order", "happy", errors.Join(a, b, c), []content.Problem{a, b, c}},
		{"nested join", "happy", errors.Join(a, errors.Join(b, errors.New("x")), c), []content.Problem{a, b, c}},
		{"wrapped problem", "happy", fmt.Errorf("ctx: %w", b), []content.Problem{b}},
		{"wrapped join", "happy", fmt.Errorf("ctx: %w", errors.Join(a, b)), []content.Problem{a, b}},
		{"duplicates kept", "happy", errors.Join(a, a), []content.Problem{a, a}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := content.Problems(tc.err)
			if !slices.Equal(got, tc.want) {
				t.Errorf("Problems() = %v, want %v [%s]", got, tc.want, tc.path)
			}
		})
	}
	joined := errors.Join(a, b)
	first := content.Problems(joined)
	first[0] = c
	if !slices.Equal(content.Problems(joined), []content.Problem{a, b}) {
		t.Errorf("mutating the returned slice changed a later result")
	}
}
