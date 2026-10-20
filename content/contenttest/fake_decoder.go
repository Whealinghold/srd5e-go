package contenttest

import (
	"errors"
	"slices"
	"sync"

	"github.com/Whealinghold/srd5e-go/content"
)

var _ content.Decoder = (*FakeDecoder)(nil)

// FakeDecoder is a [content.Decoder] double scripted per input document: each
// document text maps to a (Set, error) pair or to a panic value, and every
// input is recorded in call order. It is a double, not an adapter: it
// decodes nothing. A document that was not scripted is rejected with one
// content.ProblemSyntax problem at the path "", so an unscripted call is an
// ordinary invalid document and never a panic or a non-problem error.
//
// Build it with [NewFakeDecoder] and script it before sharing it; Decode and
// Inputs are safe for concurrent use, the script methods are not.
type FakeDecoder struct {
	script map[string]scripted

	mu     sync.Mutex
	inputs [][]byte
}

// scripted is one scripted outcome.
type scripted struct {
	set    content.Set
	err    error
	panics bool
	value  any
}

// NewFakeDecoder returns a fake with an empty script.
func NewFakeDecoder() *FakeDecoder { return &FakeDecoder{script: map[string]scripted{}} }

// Return scripts document to decode to set and err, as the real decoder
// would: a nil err with the Set, or a non-nil err (an errors.Join of
// [content.Problem] values) with the zero Set. A non-nil err that holds no
// problem is allowed, to script a decoder that breaks the port contract. It
// replaces any earlier script for document.
func (f *FakeDecoder) Return(document string, set content.Set, err error) *FakeDecoder {
	f.script[document] = scripted{set: set, err: err}
	return f
}

// Panic scripts document to make Decode panic with value, to script a decoder
// that panics.
func (f *FakeDecoder) Panic(document string, value any) *FakeDecoder {
	f.script[document] = scripted{panics: true, value: value}
	return f
}

// Inputs returns a copy of the documents seen so far, in call order,
// including those that panicked.
func (f *FakeDecoder) Inputs() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]byte, len(f.inputs))
	for i, in := range f.inputs {
		out[i] = slices.Clone(in)
	}
	return out
}

// Decode records document and returns the scripted outcome for its text. It
// does not retain or modify document, and returns the problems of a scripted
// error as a fresh error each call.
func (f *FakeDecoder) Decode(document []byte) (content.Set, error) {
	f.mu.Lock()
	f.inputs = append(f.inputs, slices.Clone(document))
	f.mu.Unlock()
	o, ok := f.script[string(document)]
	switch {
	case !ok:
		return content.Set{}, errors.Join(content.NewProblem("", content.ProblemSyntax))
	case !o.panics:
		return o.set, copyOf(o.err)
	default:
		panic(o.value)
	}
}

// copyOf returns an error that holds the same problems as err, rebuilt, or err
// itself when it holds none.
func copyOf(err error) error {
	problems := content.Problems(err)
	if len(problems) == 0 {
		return err
	}
	errs := make([]error, len(problems))
	for i, p := range problems {
		errs[i] = p
	}
	return errors.Join(errs...)
}
