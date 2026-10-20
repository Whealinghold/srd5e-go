package content_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func problemErr(ps ...content.Problem) error {
	errs := make([]error, len(ps))
	for i, p := range ps {
		errs[i] = p
	}
	return errors.Join(errs...)
}

func mustLoader(t *testing.T, d content.Decoder, opts ...content.Option) content.Loader {
	t.Helper()
	l, err := content.NewLoader(d, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-077 - LoadSet calls the Decoder once and returns
// LoadSetLoaded with the port's Set for a nil error.
// Path: happy.
func TestLoadSetLoaded(t *testing.T) {
	t.Parallel()
	set := content.MustSet(plain(content.KindSpell, "a"), plain(content.KindFeat, "b"))
	fake := contenttest.NewFakeDecoder().Return("doc", set, nil)
	res, err := mustLoader(t, fake).LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("doc")})
	loaded, ok := res.(content.LoadSetLoaded)
	if err != nil || !ok || !loaded.Set().Equal(set) {
		t.Fatalf("LoadSet = (%T %v, %v), want LoadSetLoaded holding the port's Set", res, res, err)
	}
	if got := fake.Inputs(); len(got) != 1 || string(got[0]) != "doc" {
		t.Errorf("decoder inputs = %q, want one call with the document", got)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-077, FR-097 - an empty set with a nil error is a
// valid, loaded document (FR-009), and the result types have Equal.
// Path: unhappy (empty case).
func TestLoadSetEmptySetIsLoaded(t *testing.T) {
	t.Parallel()
	fake := contenttest.NewFakeDecoder().Return("empty", content.Set{}, nil)
	res, err := mustLoader(t, fake).LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("empty")})
	loaded, ok := res.(content.LoadSetLoaded)
	if err != nil || !ok || len(loaded.Set().Entries()) != 0 || !loaded.Equal(content.LoadSetLoaded{}) {
		t.Fatalf("LoadSet = (%T %v, %v), want an empty LoadSetLoaded", res, res, err)
	}
	other := content.MustSet(plain(content.KindSpell, "a"))
	res2, _ := mustLoader(t, contenttest.NewFakeDecoder().Return("x", other, nil)).LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("x")})
	if res2.(content.LoadSetLoaded).Equal(loaded) || !res2.(content.LoadSetLoaded).Equal(res2.(content.LoadSetLoaded)) {
		t.Errorf("LoadSetLoaded.Equal wrong")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-077, FR-017, FR-097 - LoadSet returns
// LoadSetRejected with an equal, non-empty list of the port's problems; Problems() is a copy.
// Path: unhappy (failure).
func TestLoadSetRejected(t *testing.T) {
	t.Parallel()
	ps := []content.Problem{content.NewProblem("/spells/0/school", content.ProblemMissingField), content.NewProblem("/x", content.ProblemUnknownField)}
	fake := contenttest.NewFakeDecoder().Return("bad", content.Set{}, problemErr(ps...))
	l := mustLoader(t, fake)
	res, err := l.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("bad")})
	rejected, ok := res.(content.LoadSetRejected)
	if err != nil || !ok || !slices.Equal(rejected.Problems(), ps) {
		t.Fatalf("LoadSet = (%T %v, %v), want LoadSetRejected holding %v", res, res, err, ps)
	}
	got := rejected.Problems()
	got[0] = content.NewProblem("/mutated", content.ProblemSyntax)
	if !slices.Equal(rejected.Problems(), ps) {
		t.Errorf("mutating Problems() changed the result")
	}
	again, _ := l.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("bad")})
	if !again.(content.LoadSetRejected).Equal(rejected) || rejected.Equal(content.LoadSetRejected{}) {
		t.Errorf("LoadSetRejected.Equal wrong")
	}
	if !(content.LoadSetRejected{}).Equal(content.LoadSetRejected{}) {
		t.Errorf("zero values must be equal")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-017 - mutating the slice the decoder returned
// leaves the result unchanged (the Loader keeps its own list).
// Path: unhappy (immutability).
func TestLoadSetCopiesDecoderProblems(t *testing.T) {
	t.Parallel()
	ps := []content.Problem{content.NewProblem("/a", content.ProblemWrongType)}
	err := problemErr(ps...)
	res, _ := mustLoader(t, contenttest.NewFakeDecoder().Return("d", content.Set{}, err)).LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("d")})
	content.Problems(err)[0] = content.NewProblem("/b", content.ProblemSyntax)
	if got := res.(content.LoadSetRejected).Problems(); !slices.Equal(got, ps) {
		t.Errorf("problems = %v, want %v", got, ps)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-086 - a decoder that fails as a port is a
// *srderr.PortError (content_error) wrapped once, and the result is nil.
// Path: per case.
func TestLoadSetPortFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	cases := []struct {
		name   string
		script func(*contenttest.FakeDecoder) *contenttest.FakeDecoder
		is     []error
		op     string // the Op of the outermost PortError
	}{
		{"a panic", func(f *contenttest.FakeDecoder) *contenttest.FakeDecoder {
			return f.Panic("d", "secret document bytes")
		}, []error{srderr.ErrPortPanic}, "decode"},
		{"a panic with an error value", func(f *contenttest.FakeDecoder) *contenttest.FakeDecoder { return f.Panic("d", boom) }, []error{srderr.ErrPortPanic, boom}, "decode"},
		{"an error that holds no problem", func(f *contenttest.FakeDecoder) *contenttest.FakeDecoder { return f.Return("d", content.Set{}, boom) }, []error{srderr.ErrContractViolation, boom}, "decode"},
		{"an error that wraps a PortError of the port", func(f *contenttest.FakeDecoder) *contenttest.FakeDecoder {
			return f.Return("d", content.Set{}, srderr.WrapPort(content.Port, "io", boom))
		}, []error{boom}, "io"},
		{"an error that wraps a PortError of another port", func(f *contenttest.FakeDecoder) *contenttest.FakeDecoder {
			return f.Return("d", content.Set{}, srderr.WrapPort("other", "io", boom))
		}, []error{boom}, "decode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := mustLoader(t, c.script(contenttest.NewFakeDecoder())).LoadSet(context.Background(), content.LoadSetCommand{Document: []byte("d")})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != content.Port || srderr.ErrorType(err) != "content_error" {
				t.Fatalf("LoadSet = (%v, %v), want a nil result and a content *PortError", res, err)
			}
			for _, want := range c.is {
				if !errors.Is(err, want) {
					t.Errorf("error %v does not match %v", err, want)
				}
			}
			if pe.Op != c.op {
				t.Errorf("outermost PortError op = %q, want %q (a failure is wrapped once)", pe.Op, c.op)
			}
			if _, ok := res.(content.LoadSetRejected); ok {
				t.Errorf("a port failure must not be a rejection")
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-086 - cancellation of ctx before the call returns
// ctx.Err() unwrapped and never reaches the decoder.
// Path: unhappy (failure).
func TestLoadSetCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fake := contenttest.NewFakeDecoder()
	res, err := mustLoader(t, fake).LoadSet(ctx, content.LoadSetCommand{Document: []byte("d")})
	if res != nil || !errors.Is(err, context.Canceled) || len(fake.Inputs()) != 0 {
		t.Errorf("LoadSet = (%v, %v) with %d decoder calls, want nil, context.Canceled (unwrapped) and none", res, err, len(fake.Inputs()))
	}
	var pe *srderr.PortError
	if errors.As(err, &pe) {
		t.Errorf("cancellation was wrapped as a port failure")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-077, FR-086 - NewLoader needs a decoder and the zero
// Loader says so before doing anything.
// Path: unhappy (invalid input).
func TestNewLoaderRequiresDecoder(t *testing.T) {
	t.Parallel()
	_, err := content.NewLoader(nil)
	var le *content.LoaderError
	if !errors.As(err, &le) || le.Problem != content.LoaderMissingDecoder || !errors.Is(err, srderr.ErrInvalidInput) || le.Error() == "" {
		t.Fatalf("NewLoader(nil) error = %v, want *LoaderError missingDecoder", err)
	}
	if !errors.Is(err, &content.LoaderError{Problem: content.LoaderMissingDecoder}) || errors.Is(err, &content.LoaderError{Problem: 2}) || errors.Is(err, &content.LoaderError{Problem: 0}) || (&content.LoaderError{Problem: 1}).Is((*content.LoaderError)(nil)) || (&content.LoaderError{}).Is(errors.New("x")) {
		t.Errorf("LoaderError.Is must match by Problem only")
	}
	var zero content.Loader
	res, err := zero.LoadSet(context.Background(), content.LoadSetCommand{})
	if res != nil || !errors.Is(err, &content.LoaderError{Problem: content.LoaderMissingDecoder}) {
		t.Errorf("zero Loader.LoadSet = (%v, %v), want the missing-decoder error", res, err)
	}
	if content.LoaderMissingDecoder.String() != "missingDecoder" || content.LoaderProblem(0).String() != "unknown" || content.LoaderProblem(2).String() != "unknown" {
		t.Errorf("LoaderProblem.String wrong")
	}
	// A nil clock and a zero Telemetry are ignored or defaulted.
	if _, err := content.NewLoader(contenttest.NewFakeDecoder(), content.WithClock(nil), content.WithTelemetry(telemetry.Telemetry{})); err != nil {
		t.Errorf("NewLoader with nil clock = %v", err)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-076, FR-078 - the Loader neither retains nor
// modifies the document, and concurrent loads share nothing.
// Path: unhappy (immutability).
func TestLoadSetConcurrentAndInputUntouched(t *testing.T) {
	t.Parallel()
	good := content.MustSet(plain(content.KindSpell, "a"))
	fake := contenttest.NewFakeDecoder().Return("ok", good, nil).Return("bad", content.Set{}, problemErr(content.NewProblem("/x", content.ProblemUnknownField)))
	l := mustLoader(t, fake)
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc := []byte("ok")
			want := "ok"
			if i%2 == 1 {
				doc, want = []byte("bad"), "bad"
			}
			res, err := l.LoadSet(context.Background(), content.LoadSetCommand{Document: doc})
			if err != nil || string(doc) != want {
				t.Errorf("LoadSet = (%v, %v), document now %q", res, err, doc)
			}
			switch res.(type) {
			case content.LoadSetLoaded:
				if want != "ok" {
					t.Errorf("wrong variant")
				}
			case content.LoadSetRejected:
				if want != "bad" {
					t.Errorf("wrong variant")
				}
			}
		}()
	}
	wg.Wait()
	if len(fake.Inputs()) != 16 {
		t.Errorf("decoder calls = %d, want 16", len(fake.Inputs()))
	}
}
