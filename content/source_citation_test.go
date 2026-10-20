package content_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/content-schema-and-decoder.md FR-115 - NewSourceCitation
// needs a non-blank document and section and a page of at least SourceMinPage,
// checked in that order, else a *SourceCitationError wrapping ErrInvalidInput;
// MustSourceCitation panics with it.
// Path: per case.
func TestNewSourceCitation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path        string
		document, section string
		page              int
		want              content.SourceCitationProblem // 0 means accepted
	}{
		{"typical", "happy", "SRD 5.2.1", "Damage Rolls", 16, 0},
		{"minimum page", "unhappy (boundary)", "d", "s", content.SourceMinPage, 0},
		{"page 1", "unhappy (boundary)", "d", "s", 1, 0},
		{"blank document", "unhappy (invalid input)", "", "s", 1, content.SourceBlankDocument},
		{"whitespace document", "unhappy (invalid input)", " \t", "s", 1, content.SourceBlankDocument},
		{"blank section", "unhappy (invalid input)", "d", "  ", 1, content.SourceBlankSection},
		{"page 0", "unhappy (boundary)", "d", "s", 0, content.SourcePageBelowMinimum},
		{"negative page", "unhappy (boundary)", "d", "s", -1, content.SourcePageBelowMinimum},
		{"everything wrong reports the document first", "unhappy (rule interaction)", " ", " ", 0, content.SourceBlankDocument},
		{"section and page wrong reports the section first", "unhappy (rule interaction)", "d", " ", 0, content.SourceBlankSection},
		{"control character in the document", "unhappy (invalid input)", "d\x00", "s", 1, content.SourceInvalidCharacter},
		{"line feed in the section", "unhappy (invalid input)", "d", "s\nx", 1, content.SourceInvalidCharacter},
		{"bidi override in the section", "unhappy (invalid input)", "d", "s\u202Ex", 1, content.SourceInvalidCharacter},
		{"invalid UTF-8 in the document", "unhappy (invalid input)", "d\xff", "s", 1, content.SourceInvalidCharacter},
		{"maximum page", "unhappy (boundary)", "d", "s", content.SourceMaxPage, 0},
		{"page below minimum outranks a bad character", "unhappy (rule interaction)", "d\x00", "s", 0, content.SourcePageBelowMinimum},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.NewSourceCitation(c.document, c.section, c.page)
			panicked := panicValue(func() { content.MustSourceCitation(c.document, c.section, c.page) })
			if c.want == 0 {
				if err != nil || panicked != nil || got.Document() != c.document || got.Section() != c.section || got.Page() != c.page || got.IsZero() {
					t.Fatalf("got (%+v, %v), panic %v; want accepted [%s]", got, err, panicked, c.path)
				}
				return
			}
			var ce *content.SourceCitationError
			if !errors.As(err, &ce) || ce.Problem != c.want || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error = %v, want *SourceCitationError %v [%s]", err, c.want, c.path)
			}
			if !errors.Is(err, &content.SourceCitationError{Problem: c.want}) || errors.Is(err, &content.SourceCitationError{Problem: c.want + 1}) || errors.Is(err, &content.SourceCitationError{Problem: c.want - 1}) {
				t.Errorf("SourceCitationError.Is must match by Problem only")
			}
			pe, ok := panicked.(error)
			if !got.IsZero() || !ok || !errors.Is(pe, &content.SourceCitationError{Problem: c.want}) {
				t.Errorf("MustSourceCitation panic = %v, want the constructor error; zero citation %v", panicked, got.IsZero())
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-138 (SRD 5.2.1,
// "Damage Rolls", p. 16) - CitesSRD521 is true iff the document is exactly
// SRDDocument and the page is at most SRDLastPage; Text is `<document>,
// "<section>", p. <page>` character for character.
// Path: per case.
func TestSourceCitationFacts(t *testing.T) {
	t.Parallel()
	if content.SourceMinPage != 1 || content.SRDLastPage != 364 || content.SRDDocument != "SRD 5.2.1" {
		t.Fatalf("source constants changed")
	}
	cases := []struct {
		name, path        string
		document, section string
		page              int
		cites             bool
		text              string
	}{
		{"typical", "happy", "SRD 5.2.1", "Damage Rolls", 16, true, `SRD 5.2.1, "Damage Rolls", p. 16`},
		{"last page", "unhappy (boundary)", "SRD 5.2.1", "s", 364, true, `SRD 5.2.1, "s", p. 364`},
		{"one past the last page", "unhappy (boundary)", "SRD 5.2.1", "s", 365, false, `SRD 5.2.1, "s", p. 365`},
		{"first page", "unhappy (boundary)", "SRD 5.2.1", "s", 1, true, `SRD 5.2.1, "s", p. 1`},
		{"other document", "unhappy (boundary)", "Homebrew", "s", 1, false, `Homebrew, "s", p. 1`},
		{"case differs", "unhappy (boundary)", "srd 5.2.1", "s", 1, false, `srd 5.2.1, "s", p. 1`},
		{"parts kept verbatim", "unhappy (boundary)", " d ", " s ", 2, false, ` d , " s ", p. 2`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cit := content.MustSourceCitation(c.document, c.section, c.page)
			if cit.CitesSRD521() != c.cites || cit.Text() != c.text {
				t.Errorf("CitesSRD521 = %v, Text = %q; want %v, %q [%s]", cit.CitesSRD521(), cit.Text(), c.cites, c.text, c.path)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-097, FR-115 - a
// SourceCitation is comparable; the zero value means no citation.
// Path: unhappy (empty case).
func TestSourceCitationZeroAndEquality(t *testing.T) {
	t.Parallel()
	var zero content.SourceCitation
	a := content.MustSourceCitation("d", "s", 1)
	if !zero.IsZero() || a.IsZero() || zero.CitesSRD521() {
		t.Errorf("IsZero wrong")
	}
	if a != content.MustSourceCitation("d", "s", 1) || a == content.MustSourceCitation("d", "s", 2) ||
		a == content.MustSourceCitation("d", "t", 1) || a == content.MustSourceCitation("e", "s", 1) {
		t.Errorf("SourceCitation must compare by all three parts")
	}
	if got := content.SourceCitationProblem(0).String(); got != "unknown" {
		t.Errorf("zero problem String = %q", got)
	}
	for i, w := range []string{"blankDocument", "blankSection", "pageBelowMinimum", "pageAboveMaximum", "invalidCharacter"} {
		if got := content.SourceCitationProblem(i + 1).String(); got != w {
			t.Errorf("SourceCitationProblem(%d).String() = %q, want %q", i+1, got, w)
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-021 - a page above SourceMaxPage is rejected, so a
// decoded page is the same on every integer width.
// Path: unhappy (boundary).
func TestNewSourceCitationPageAboveMaximum(t *testing.T) {
	t.Parallel()
	if strconv.IntSize < 64 {
		return // no int exceeds SourceMaxPage here; the check is a no-op on a 32-bit platform
	}
	above := int64(content.SourceMaxPage)
	above++
	_, err := content.NewSourceCitation("d", "s", int(above))
	if !errors.Is(err, &content.SourceCitationError{Problem: content.SourcePageAboveMaximum}) {
		t.Errorf("error = %v, want SourcePageAboveMaximum", err)
	}
}
