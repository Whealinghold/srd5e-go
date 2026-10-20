package content

import (
	"errors"
	"strconv"
	"unicode/utf8"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// ProblemKind classifies one [Problem] found in a content document. The zero
// ProblemKind is invalid. Safe for concurrent use.
type ProblemKind int

// The problem kinds, in the order the schema lists them. Adding one is a
// specification change.
const (
	// ProblemSyntax: the document is not valid JSON.
	ProblemSyntax ProblemKind = iota + 1
	// ProblemUnsupportedVersion: the schema version is not supported.
	ProblemUnsupportedVersion
	// ProblemMissingField: a required field is absent.
	ProblemMissingField
	// ProblemUnknownField: a field the schema does not define is present.
	ProblemUnknownField
	// ProblemWrongType: a value has the wrong JSON type.
	ProblemWrongType
	// ProblemOutOfRange: a number is outside its allowed range.
	ProblemOutOfRange
	// ProblemInvalidValue: a value of the right type is not acceptable.
	ProblemInvalidValue
	// ProblemDuplicate: an id or key appears more than once.
	ProblemDuplicate
	// ProblemUnresolvedReference: a reference names no entry.
	ProblemUnresolvedReference
	// ProblemConstraintViolation: a reference target breaks a constraint.
	ProblemConstraintViolation
)

// String returns the wire spelling of k, or "unknown" for an undefined value.
func (k ProblemKind) String() string {
	switch k {
	case ProblemSyntax:
		return "syntax"
	case ProblemUnsupportedVersion:
		return "unsupportedVersion"
	case ProblemMissingField:
		return "missingField"
	case ProblemUnknownField:
		return "unknownField"
	case ProblemWrongType:
		return "wrongType"
	case ProblemOutOfRange:
		return "outOfRange"
	case ProblemInvalidValue:
		return "invalidValue"
	case ProblemDuplicate:
		return "duplicate"
	case ProblemUnresolvedReference:
		return "unresolvedReference"
	case ProblemConstraintViolation:
		return "constraintViolation"
	}
	return "unknown"
}

// Problem is one thing wrong with a content document: a JSON Pointer path
// (RFC 6901, "" for the root) and a [ProblemKind]. It carries no display text;
// do not branch on the text of [Problem.Error]. Problem is comparable with ==
// and safe for concurrent use. It matches srderr.ErrInvalidInput with
// errors.Is.
//
//nolint:errname // the name is fixed by schema FR-014.
type Problem struct {
	path string
	kind ProblemKind
}

// NewProblem returns the problem of kind k at the JSON Pointer path. It does
// not validate its arguments.
func NewProblem(path string, k ProblemKind) Problem { return Problem{path: path, kind: k} }

// Path returns the JSON Pointer path of the problem.
func (p Problem) Path() string { return p.path }

// Kind returns the kind of the problem.
func (p Problem) Kind() ProblemKind { return p.kind }

// MaxRenderedPath is the most bytes of a path that [Problem.Error] renders; a
// longer path is cut at a character boundary and marked with "...".
const MaxRenderedPath = 256

// Error renders the kind and the path as an ASCII-only quoted string, so a
// path that holds an attacker-chosen key (newlines, escapes, bidirectional
// characters) cannot forge a log line. The path is cut after
// [MaxRenderedPath] bytes, so the text is at most four times that (an escaped
// byte is four characters) plus the kind. The text is not a contract; read [Problem.Path]
// for the exact path.
func (p Problem) Error() string {
	path, more := p.path, ""
	if len(path) > MaxRenderedPath {
		cut := MaxRenderedPath
		for cut > 0 && !utf8.RuneStart(path[cut]) {
			cut--
		}
		path, more = path[:cut], "..."
	}
	return "content: " + p.kind.String() + " at " + strconv.QuoteToASCII(path) + more
}

// Unwrap returns srderr.ErrInvalidInput.
func (p Problem) Unwrap() error { return srderr.ErrInvalidInput }

// Problems returns every [Problem] held in err, flattened through errors.Join
// and wrapping, in order, as a new slice. It returns nil for a nil error and
// for an error holding none.
func Problems(err error) []Problem {
	var out []Problem
	collectProblems(err, &out)
	return out
}

func collectProblems(err error, out *[]Problem) {
	//nolint:errorlint // flattening must visit every branch of a join, which errors.As cannot do.
	switch e := err.(type) {
	case nil:
	case Problem:
		*out = append(*out, e)
	case interface{ Unwrap() []error }:
		for _, inner := range e.Unwrap() {
			collectProblems(inner, out)
		}
	default:
		collectProblems(errors.Unwrap(err), out)
	}
}
