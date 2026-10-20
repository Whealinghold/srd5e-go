package content_test

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// panicValue runs f and returns the value it panicked with, or nil.
func panicValue(f func()) (v any) {
	defer func() { v = recover() }()
	f()
	return nil
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-143, FR-144 - ParseID
// accepts [a-z][a-z0-9_]* up to 64 bytes and otherwise returns an *IDError
// (empty, then tooLong, then invalidFormat) that wraps ErrInvalidInput; MustID
// panics with the same error exactly when ParseID errors.
// Path: per case.
func TestParseID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		in         string
		want       content.IDProblem // 0 means accepted
	}{
		{"single letter", "unhappy (boundary)", "a", 0},
		{"letter and digit", "unhappy (boundary)", "a1", 0},
		{"underscore inside", "unhappy (boundary)", "a_b", 0},
		{"digit zero after letter", "unhappy (boundary)", "a0", 0},
		{"digit nine after letter", "unhappy (boundary)", "a9", 0},
		{"trailing underscore", "unhappy (boundary)", "a_", 0},
		{"64 bytes", "unhappy (boundary)", strings.Repeat("a", 64), 0},
		{"65 bytes", "unhappy (boundary)", strings.Repeat("a", 65), content.IDTooLong},
		{"65 uppercase gives tooLong first", "unhappy (boundary)", strings.Repeat("A", 65), content.IDTooLong},
		{"empty", "unhappy (invalid input)", "", content.IDEmpty},
		{"only spaces", "unhappy (invalid input)", "   ", content.IDInvalidFormat},
		{"leading space", "unhappy (invalid input)", " a", content.IDInvalidFormat},
		{"trailing space", "unhappy (invalid input)", "a ", content.IDInvalidFormat},
		{"interior space", "unhappy (invalid input)", "a b", content.IDInvalidFormat},
		{"newline", "unhappy (invalid input)", "a\nb", content.IDInvalidFormat},
		{"NUL", "unhappy (invalid input)", "a\x00", content.IDInvalidFormat},
		{"non-ASCII", "unhappy (invalid input)", "é", content.IDInvalidFormat},
		{"uppercase", "unhappy (invalid input)", "aB", content.IDInvalidFormat},
		{"hyphen", "unhappy (invalid input)", "a-1", content.IDInvalidFormat},
		{"digit first", "unhappy (boundary)", "1a", content.IDInvalidFormat},
		{"underscore first", "unhappy (boundary)", "_a", content.IDInvalidFormat},
		{"invalid UTF-8", "unhappy (invalid input)", "a\xff", content.IDInvalidFormat},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			id, err := content.ParseID(c.in)
			panicked := panicValue(func() { content.MustID(c.in) })
			if c.want == 0 {
				if err != nil || id.String() != c.in || id.IsZero() {
					t.Fatalf("ParseID(%q) = (%q, %v), want accepted [%s]", c.in, id, err, c.path)
				}
				if panicked != nil {
					t.Fatalf("MustID(%q) panicked: %v", c.in, panicked)
				}
				return
			}
			var ie *content.IDError
			if !errors.As(err, &ie) || ie.Problem != c.want || ie.Value != c.in {
				t.Fatalf("ParseID(%q) error = %v, want *IDError %v with the input [%s]", c.in, err, c.want, c.path)
			}
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("error does not wrap ErrInvalidInput")
			}
			if !errors.Is(err, &content.IDError{Problem: c.want}) || errors.Is(err, &content.IDError{Problem: c.want + 1}) || errors.Is(err, &content.IDError{Problem: c.want - 1}) {
				t.Errorf("IDError.Is must match by Problem only")
			}
			if !id.IsZero() {
				t.Errorf("rejected ID = %q, want the zero ID", id)
			}
			pe, ok := panicked.(error)
			if !ok || !errors.Is(pe, &content.IDError{Problem: c.want}) {
				t.Errorf("MustID(%q) panic = %v, want the ParseID error", c.in, panicked)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-143 - the zero ID is
// invalid, an ID is comparable and usable as a map key.
// Path: unhappy (empty case).
func TestIDZeroAndComparable(t *testing.T) {
	t.Parallel()
	var zero content.ID
	if !zero.IsZero() || zero.String() != "" {
		t.Errorf("zero ID = %q, IsZero %v", zero, zero.IsZero())
	}
	m := map[content.ID]int{content.MustID("a"): 1}
	if m[content.MustID("a")] != 1 || content.MustID("a") == content.MustID("b") {
		t.Errorf("ID must compare by value")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-144 - IDProblem has
// exactly empty, tooLong, invalidFormat in that order.
// Path: happy.
func TestIDProblemString(t *testing.T) {
	t.Parallel()
	for p, want := range []string{"unknown", "empty", "tooLong", "invalidFormat", "unknown"} {
		if got := content.IDProblem(p).String(); got != want {
			t.Errorf("IDProblem(%d).String() = %q, want %q", p, got, want)
		}
	}
	if content.IDEmpty != 1 || content.IDTooLong != 2 || content.IDInvalidFormat != 3 {
		t.Errorf("IDProblem order changed")
	}
}

// FuzzParseID checks the FR-143 and FR-144 properties on arbitrary input.
//
// Requirement: doc/specs/content-schema-and-decoder.md FR-143, FR-144 - ParseID
// never panics, errors are typed, and accepted input matches the grammar and
// round-trips; MustID panics iff ParseID errors.
func FuzzParseID(f *testing.F) {
	for _, s := range []string{"", "a", "a_b1", "1a", "_a", "A", "a-b", " a", "é", "a\x00", "a\xff", strings.Repeat("a", 64), strings.Repeat("a", 65), strings.Repeat("A", 65)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id, err := content.ParseID(s)
		panicked := panicValue(func() { content.MustID(s) }) != nil
		if panicked != (err != nil) {
			t.Fatalf("MustID(%q) panicked = %v but ParseID error = %v", s, panicked, err)
		}
		if err != nil {
			var ie *content.IDError
			if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) || ie.Value != s || !id.IsZero() {
				t.Fatalf("ParseID(%q) error %v is not a well-formed *IDError", s, err)
			}
			return
		}
		if id.String() != s || len(s) > content.MaxIDLength || !utf8.ValidString(s) || s[0] < 'a' || s[0] > 'z' {
			t.Fatalf("ParseID accepted %q", s)
		}
		for _, r := range s {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
				t.Fatalf("ParseID accepted %q with %q", s, r)
			}
		}
	})
}

// FuzzParseClassIDs checks that the class id parsers agree with ParseID.
//
// Requirement: doc/specs/content-classes.md FR-143, FR-144 - ParseClassColumnID
// and ParseFeatureOptionID never panic, accept exactly what ParseID accepts,
// and round-trip through String.
// Path: unhappy (invalid input).
func FuzzParseClassIDs(f *testing.F) {
	for _, s := range []string{"", "a", "a_b1", "1a", "A", "a-b", "a\x00", "a\xff", strings.Repeat("a", 65)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, want := content.ParseID(s)
		col, cerr := content.ParseClassColumnID(s)
		opt, oerr := content.ParseFeatureOptionID(s)
		if (cerr != nil) != (want != nil) || (oerr != nil) != (want != nil) {
			t.Fatalf("%q: ParseID %v, column %v, option %v", s, want, cerr, oerr)
		}
		if want == nil && (col.String() != s || opt.String() != s || !col.Valid() || !opt.Valid()) {
			t.Fatalf("%q did not round-trip", s)
		}
	})
}
