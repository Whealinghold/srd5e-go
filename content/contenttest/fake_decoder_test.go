package contenttest_test

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func joined(ps ...content.Problem) error {
	errs := make([]error, len(ps))
	for i, p := range ps {
		errs[i] = p
	}
	return errors.Join(errs...)
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - FakeDecoder returns the scripted (Set, error)
// per document text, records every input in order as copies, and rejects an
// unscripted document with one syntax problem.
// Path: happy.
func TestFakeDecoderScriptingAndRecording(t *testing.T) {
	t.Parallel()
	set := content.MustSet(testEntry{content.NewKey(content.KindSpell, content.MustID("a"))})
	bad := joined(content.NewProblem("/x", content.ProblemUnknownField), content.NewProblem("/y", content.ProblemWrongType))
	f := contenttest.NewFakeDecoder().Return("ok", set, nil).Return("bad", content.Set{}, bad).Return("ok", set, nil)

	got, err := f.Decode([]byte("ok"))
	if err != nil || !got.Equal(set) {
		t.Errorf("Decode(ok) = (%v, %v), want the scripted set and nil", got, err)
	}
	got, err = f.Decode([]byte("bad"))
	if !slices.Equal(content.Problems(err), content.Problems(bad)) || !got.Equal(content.Set{}) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("Decode(bad) = (%v, %v), want the scripted problems", got, err)
	}
	if errors.Is(err, bad) && err == bad { //nolint:errorlint // identity is the point: a copy is not the same value.
		t.Errorf("the scripted error was returned, not a copy")
	}
	got, err = f.Decode(nil)
	if ps := content.Problems(err); !slices.Equal(ps, []content.Problem{content.NewProblem("", content.ProblemSyntax)}) || !got.Equal(content.Set{}) {
		t.Errorf("Decode(unscripted) = (%v, %v), want one syntax problem", got, err)
	}

	in := f.Inputs()
	if len(in) != 3 || string(in[0]) != "ok" || string(in[1]) != "bad" || len(in[2]) != 0 {
		t.Errorf("Inputs() = %q, want ok, bad, empty in order", in)
	}
	in[0][0] = 'X'
	if string(f.Inputs()[0]) != "ok" {
		t.Errorf("Inputs() returned the recorder's own storage")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - the document is recorded as a copy, so the
// caller may reuse its buffer; an error that holds no problem is returned as
// scripted (to script a broken port).
// Path: unhappy (immutability).
func TestFakeDecoderCopiesTheDocumentAndKeepsForeignErrors(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	f := contenttest.NewFakeDecoder().Return("d", content.Set{}, boom)
	doc := []byte("d")
	_, err := f.Decode(doc)
	doc[0] = 'z'
	if !errors.Is(err, boom) || string(f.Inputs()[0]) != "d" {
		t.Errorf("err = %v, inputs = %q, want the scripted error and the original text", err, f.Inputs())
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - a scripted panic panics with the value, and
// the call is still recorded.
// Path: unhappy (failure).
func TestFakeDecoderScriptedPanic(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	f := contenttest.NewFakeDecoder().Panic("p", boom).Panic("q", "text value")
	for doc, want := range map[string]any{"p": boom, "q": "text value"} {
		func() {
			defer func() {
				if got := recover(); got != want {
					t.Errorf("Decode(%s) panicked with %v, want %v", doc, got, want)
				}
			}()
			_, _ = f.Decode([]byte(doc))
		}()
	}
	if len(f.Inputs()) != 2 {
		t.Errorf("a panicking call was not recorded: %q", f.Inputs())
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - Decode and Inputs are safe for concurrent use.
// Path: unhappy (immutability).
func TestFakeDecoderConcurrent(t *testing.T) {
	t.Parallel()
	f := contenttest.NewFakeDecoder().Return("a", content.Set{}, nil)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = f.Decode([]byte("a"))
			_ = f.Inputs()
		}()
	}
	wg.Wait()
	if len(f.Inputs()) != 20 {
		t.Errorf("recorded %d calls, want 20", len(f.Inputs()))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - the fake runs the decoder contract suite, with
// the valid and invalid documents scripted.
// Path: happy.
func TestFakeDecoderPassesDecoderContract(t *testing.T) {
	t.Parallel()
	valid, invalid := []byte("valid"), []byte("invalid")
	contenttest.RunDecoderContract(t, func(*testing.T) content.Decoder {
		return contenttest.NewFakeDecoder().
			Return(string(valid), content.MustSet(testEntry{content.NewKey(content.KindFeat, content.MustID("f"))}), nil).
			Return(string(invalid), content.Set{}, joined(content.NewProblem("/x", content.ProblemMissingField)))
	}, valid, invalid)
}
