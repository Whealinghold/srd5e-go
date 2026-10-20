package content

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// The facts of the bundled source and the lowest page a citation may name.
const (
	// SourceMinPage is the lowest page a [SourceCitation] may name.
	SourceMinPage = 1
	// SourceMaxPage is the highest page a [SourceCitation] may name, 2^31 - 1,
	// so the decoded page is the same on every platform integer width
	// (schema FR-021). No document comes close.
	SourceMaxPage = 1<<31 - 1
	// SRDLastPage is the last PDF page of SRD 5.2.1.
	SRDLastPage = 364
	// SRDDocument is the document name that cites SRD 5.2.1.
	SRDDocument = "SRD 5.2.1"
)

// SourceCitationProblem says why [NewSourceCitation] rejected its arguments.
// The zero value is not a rejection.
type SourceCitationProblem int

// The citation rejection reasons, listed in the order they are checked.
const (
	// SourceBlankDocument: the document is empty or only white space.
	SourceBlankDocument SourceCitationProblem = iota + 1
	// SourceBlankSection: the section is empty or only white space.
	SourceBlankSection
	// SourcePageBelowMinimum: the page is below [SourceMinPage].
	SourcePageBelowMinimum
	// SourcePageAboveMaximum: the page is above [SourceMaxPage].
	SourcePageAboveMaximum
	// SourceInvalidCharacter: the document or the section holds invalid UTF-8
	// or a control or bidirectional character (see the hygiene rule of schema
	// FR-027).
	SourceInvalidCharacter
)

// String returns the wire spelling of p, or "unknown" for an undefined value.
func (p SourceCitationProblem) String() string {
	switch p {
	case SourceBlankDocument:
		return "blankDocument"
	case SourceBlankSection:
		return "blankSection"
	case SourcePageBelowMinimum:
		return "pageBelowMinimum"
	case SourcePageAboveMaximum:
		return "pageAboveMaximum"
	case SourceInvalidCharacter:
		return "invalidCharacter"
	}
	return "unknown"
}

// SourceCitationError reports arguments rejected by [NewSourceCitation]. It
// matches srderr.ErrInvalidInput with errors.Is.
type SourceCitationError struct {
	Problem SourceCitationProblem
}

// Error renders the problem. The text is not a contract.
func (e *SourceCitationError) Error() string {
	return "content: invalid source citation: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *SourceCitationError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *SourceCitationError with the same Problem.
func (e *SourceCitationError) Is(target error) bool {
	t, ok := target.(*SourceCitationError)
	return ok && t.Problem == e.Problem
}

// SourceCitation says where in a document an entry comes from: document,
// section and page. The zero SourceCitation means "no citation" (see
// [SourceCitation.IsZero]). It is immutable, comparable and safe for
// concurrent use.
type SourceCitation struct {
	document string
	section  string
	page     int
}

// NewSourceCitation returns a citation of the given document, section and
// page. The parts are kept character for character.
//
// It returns a *[SourceCitationError] wrapping srderr.ErrInvalidInput when
// document is blank (SourceBlankDocument), section is blank
// (SourceBlankSection), page is below [SourceMinPage]
// (SourcePageBelowMinimum), page is above [SourceMaxPage]
// (SourcePageAboveMaximum), or either text holds invalid UTF-8 or a control or
// bidirectional character (SourceInvalidCharacter), checked in that order. It
// never panics.
func NewSourceCitation(document, section string, page int) (SourceCitation, error) {
	switch {
	case blankText(document):
		return SourceCitation{}, &SourceCitationError{Problem: SourceBlankDocument}
	case blankText(section):
		return SourceCitation{}, &SourceCitationError{Problem: SourceBlankSection}
	case page < SourceMinPage:
		return SourceCitation{}, &SourceCitationError{Problem: SourcePageBelowMinimum}
	case page > SourceMaxPage:
		return SourceCitation{}, &SourceCitationError{Problem: SourcePageAboveMaximum}
	case !cleanCharacters(document, false) || !cleanCharacters(section, false):
		return SourceCitation{}, &SourceCitationError{Problem: SourceInvalidCharacter}
	}
	return SourceCitation{document: document, section: section, page: page}, nil
}

// MustSourceCitation is like [NewSourceCitation] but panics with its error
// when the arguments are invalid.
func MustSourceCitation(document, section string, page int) SourceCitation {
	c, err := NewSourceCitation(document, section, page)
	if err != nil {
		panic(err)
	}
	return c
}

// Document returns the document name.
func (c SourceCitation) Document() string { return c.document }

// Section returns the section heading.
func (c SourceCitation) Section() string { return c.section }

// Page returns the page number.
func (c SourceCitation) Page() int { return c.page }

// IsZero reports whether c is the zero citation (no citation).
func (c SourceCitation) IsZero() bool { return c == SourceCitation{} }

// CitesSRD521 reports whether c cites [SRDDocument] at a page no later than
// [SRDLastPage]. It is not a constructor rule: host content may cite its own
// document.
func (c SourceCitation) CitesSRD521() bool {
	return c.document == SRDDocument && c.page <= SRDLastPage
}

// Text renders the citation as `SRD 5.2.1, "Damage Rolls", p. 16`, parts
// character for character. It is for display only and is never parsed back.
func (c SourceCitation) Text() string {
	return c.document + `, "` + c.section + `", p. ` + strconv.Itoa(c.page)
}
