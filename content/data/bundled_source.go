// Embeds the JSON files in this directory. Contains SRD 5.2.1 content.
// Attribution: see NOTICE.md in the package root.

// Package data serves the bundled SRD 5.2.1 content: the JSON documents in
// this directory, embedded with go:embed and decoded at construction by the
// same content.Decoder as host JSON. It is the only package that imports
// embed. The documents are never exported; read them through [BundledSource],
// or through [Files] to feed content.Loader.LoadSet.
package data

import (
	"context"
	"embed"
	"io/fs"
	"maps"
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// documents is the embedded index of documents: the flat <stem>.json files of
// this directory (content-schema-and-decoder.md FR-120, FR-122). Nothing else
// indexes them; the index is read from here with fs.ReadDir.
//
//go:embed *.json
var documents embed.FS

// Files returns the embedded documents as a read-only file system in the flat
// layout of the directory, in lexical order (fs.ReadDir order). It is the one
// access to the embedded texts; a caller feeds each file to
// content.Loader.LoadSet or content.JSONDecoder.
func Files() fs.FS { return documents }

var _ content.Source = (*BundledSource)(nil)

// BundledSource is a [content.Source] over the bundled documents. It decodes
// every document with the decoder, validates the merged set with
// content.ValidateSet and then answers exactly like a content.InMemorySource
// over the validated entries. It reads no file at run time, starts no
// goroutine and is immutable, so it is safe for concurrent use.
type BundledSource struct{ memory *content.InMemorySource }

// Option configures [NewBundledSource].
type Option func(*config)

type config struct {
	decoder   content.Decoder
	documents map[string]string
	replaced  bool
}

// WithDocuments replaces the embedded index of documents with texts keyed by
// document name; the name only orders the documents (lexically) and labels no
// problem. An empty or nil map is an empty index, which builds an empty
// source. It is for tests and hosts that bring their own documents.
func WithDocuments(texts map[string]string) Option {
	return func(c *config) { c.documents, c.replaced = texts, true }
}

// WithDecoder replaces the default content.JSONDecoder, so a test injects a
// scripted decoder. A nil decoder is ignored.
func WithDecoder(d content.Decoder) Option {
	return func(c *config) {
		if d != nil {
			c.decoder = d
		}
	}
}

// NewBundledSource decodes each document of the index (by default the embedded
// one) in lexical name order, merges the sets with content.ValidateSet and
// returns the source. It decodes and validates on every call and caches
// nothing, so build it once per process and share it.
//
// A document that does not decode, or a merged set with a duplicate or
// unresolved reference, returns a nil source and a *[BundledInvalidError]; no
// partial source is ever returned. It emits no signal and never panics.
func NewBundledSource(opts ...Option) (*BundledSource, error) {
	c := config{decoder: content.JSONDecoder{}}
	for _, o := range opts {
		if o != nil {
			o(&c)
		}
	}
	texts := c.documents
	if !c.replaced {
		texts = readDocuments(documents)
	}
	var sets []content.Set
	var decodeProblems []content.Problem
	for _, name := range slices.Sorted(maps.Keys(texts)) {
		set, err := c.decoder.Decode([]byte(texts[name]))
		if err == nil {
			sets = append(sets, set)
			continue
		}
		problems := content.Problems(err)
		if len(problems) == 0 { // a decoder that breaks its contract still must not yield a source
			problems = []content.Problem{content.NewProblem("", content.ProblemSyntax)}
		}
		decodeProblems = append(decodeProblems, problems...)
	}
	if len(decodeProblems) > 0 { // validation runs only when every document decoded
		return nil, &BundledInvalidError{DecodeProblems: decodeProblems}
	}
	merged, err := content.ValidateSet(sets...)
	if err != nil {
		return nil, &BundledInvalidError{ReferenceProblems: content.ReferenceProblems(err)}
	}
	memory, err := content.NewInMemorySource(merged.Entries())
	if err != nil { // unreachable: a validated set has distinct valid keys
		return nil, err
	}
	return &BundledSource{memory: memory}, nil
}

// MustBundledSource is like [NewBundledSource] but panics with its error. It is
// for build-time-fixed embedded data, where a bad bundle is a packaging defect
// that no caller could handle.
func MustBundledSource(opts ...Option) *BundledSource {
	s, err := NewBundledSource(opts...)
	if err != nil {
		panic(err)
	}
	return s
}

// Lookup answers per content.Source from the validated entries; the error is
// always nil.
func (s *BundledSource) Lookup(ctx context.Context, key content.Key) (content.LookupResult, error) {
	return s.memory.Lookup(ctx, key)
}

// ListKind answers per content.Source from the validated entries; the error is
// always nil.
func (s *BundledSource) ListKind(ctx context.Context, kind content.Kind) ([]content.Entry, error) {
	return s.memory.ListKind(ctx, kind)
}

// readDocuments reads every file at the root of fsys into a map keyed by file
// name. A file or directory that cannot be read becomes a document with empty
// text (named "." for the directory), which the decoder rejects as a syntax
// problem, so a broken index is a bad bundle and never a silently smaller one.
// An embed.FS never fails; the fallback exists for the tests' fixture file
// systems.
func readDocuments(fsys fs.FS) map[string]string {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return map[string]string{".": ""}
	}
	texts := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, _ := fs.ReadFile(fsys, e.Name()) // nil on failure: empty text, see above
		texts[e.Name()] = string(b)
	}
	return texts
}

// BundledInvalidError reports a bundle that [NewBundledSource] rejected: a
// packaging error, not an infrastructure failure. DecodeProblems holds the
// problems of every document that failed to decode, in the decoder's order,
// document by document in index order. ReferenceProblems holds the problems of
// the whole-set validation, in its order; validation runs only when every
// document decoded, so at most one of the two is non-empty, and at least one
// is. Both are copies. It matches srderr.ErrInvalidInput with errors.Is.
type BundledInvalidError struct {
	DecodeProblems    []content.Problem
	ReferenceProblems []content.ReferenceProblem
}

// Error renders the problem counts. The text is not a contract.
func (e *BundledInvalidError) Error() string {
	return "data: invalid bundle: " + strconv.Itoa(len(e.DecodeProblems)) + " decode problems, " +
		strconv.Itoa(len(e.ReferenceProblems)) + " reference problems"
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *BundledInvalidError) Unwrap() error { return srderr.ErrInvalidInput }
