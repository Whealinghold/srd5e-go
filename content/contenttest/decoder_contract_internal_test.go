package contenttest

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Deliberately bad decoders. Each wraps a good scripted fake and breaks one
// rule of the contract.

func problems(ps ...content.Problem) error {
	errs := make([]error, len(ps))
	for i, p := range ps {
		errs[i] = p
	}
	return errors.Join(errs...)
}

func goodDecoder() *FakeDecoder {
	return NewFakeDecoder().
		Return("valid", content.MustSet(ent(content.KindSpell, "a")), nil).
		Return("invalid", content.Set{}, problems(content.NewProblem("/x", content.ProblemMissingField)))
}

// keeper retains the first buffer it is given and answers from it, so it
// changes its answer when the caller overwrites the buffer.
type keeper struct{ kept []byte }

func (k *keeper) Decode(b []byte) (content.Set, error) {
	if k.kept == nil {
		k.kept = b
	}
	if len(k.kept) > 0 && k.kept[0] == 0 {
		return content.Set{}, problems(content.NewProblem("/x", content.ProblemSyntax))
	}
	return content.Set{}, nil
}

type decoderFunc func([]byte) (content.Set, error)

func (f decoderFunc) Decode(d []byte) (content.Set, error) { return f(d) }

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - the decoder contract checks pass a conforming
// decoder and report each deliberately bad one through the reporter: an error
// for a valid document, success for an invalid one, an error that holds no
// problem, an error that is not ErrInvalidInput, a Set with an error, a panic,
// a result that changes between calls, an input that is modified and one that is
// retained.
// Path: per case.
func TestDecoderChecksReportBadDecoders(t *testing.T) {
	t.Parallel()
	valid, invalid := []byte("valid"), []byte("invalid")
	good := goodDecoder()
	setA := content.MustSet(ent(content.KindSpell, "a"))
	calls := 0
	cases := []struct {
		name  string
		d     content.Decoder
		check func(reporter, content.Decoder)
		bad   bool
	}{
		{"conforming valid", good, func(r reporter, d content.Decoder) { checkValidDocument(r, d, valid) }, false},
		{"conforming invalid", good, func(r reporter, d content.Decoder) { checkInvalidDocument(r, d, invalid) }, false},
		{"conforming corpus", good, checkHostileCorpus, false},
		{"conforming repeatable", good, func(r reporter, d content.Decoder) {
			checkRepeatableDecode(r, d, valid)
			checkRepeatableDecode(r, d, invalid)
		}, false},
		{"conforming input untouched", good, func(r reporter, d content.Decoder) {
			checkInputUntouched(r, d, valid)
			checkInputUntouched(r, d, invalid)
		}, false},
		{"error for a valid document", decoderFunc(func([]byte) (content.Set, error) {
			return content.Set{}, problems(content.NewProblem("", content.ProblemSyntax))
		}),
			func(r reporter, d content.Decoder) { checkValidDocument(r, d, valid) }, true},
		{"success for an invalid document", decoderFunc(func([]byte) (content.Set, error) { return setA, nil }),
			func(r reporter, d content.Decoder) { checkInvalidDocument(r, d, invalid) }, true},
		{"error holding no problem", decoderFunc(func([]byte) (content.Set, error) { return content.Set{}, errors.New("plain") }),
			func(r reporter, d content.Decoder) { checkInvalidDocument(r, d, invalid) }, true},
		{"a Set with its error", decoderFunc(func([]byte) (content.Set, error) {
			return setA, problems(content.NewProblem("", content.ProblemSyntax))
		}), func(r reporter, d content.Decoder) { checkInvalidDocument(r, d, invalid) }, true},
		{"a panic", decoderFunc(func([]byte) (content.Set, error) { panic(errors.New("boom")) }),
			func(r reporter, d content.Decoder) { checkInvalidDocument(r, d, invalid) }, true},
		{"a panic on the corpus", decoderFunc(func(b []byte) (content.Set, error) {
			if len(b) == 0 {
				panic(errors.New("boom"))
			}
			return content.Set{}, nil
		}), checkHostileCorpus, true},
		{"a plain error on the corpus", decoderFunc(func([]byte) (content.Set, error) { return content.Set{}, errors.New("plain") }), checkHostileCorpus, true},
		{"a result that changes", decoderFunc(func([]byte) (content.Set, error) {
			calls++
			if calls%2 == 0 {
				return content.Set{}, nil
			}
			return content.Set{}, problems(content.NewProblem("/x", content.ProblemSyntax))
		}), func(r reporter, d content.Decoder) { checkRepeatableDecode(r, d, invalid) }, true},
		{"a modified input", decoderFunc(func(b []byte) (content.Set, error) {
			if len(b) > 0 {
				b[0] = 'X'
			}
			return content.Set{}, nil
		}), func(r reporter, d content.Decoder) { checkInputUntouched(r, d, valid) }, true},
		{"a retained input", &keeper{}, func(r reporter, d content.Decoder) { checkInputUntouched(r, d, valid) }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			c.check(rec, c.d)
			if got := len(rec.failures) > 0; got != c.bad {
				t.Errorf("reported failures = %v, want failing: %v", rec.failures, c.bad)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - the check helpers have no effect on short
// documents beyond the report, and truncate long ones in messages.
// Path: unhappy (boundary).
func TestTruncate(t *testing.T) {
	t.Parallel()
	if got := truncate(make([]byte, 100)); len(got) != 40 {
		t.Errorf("truncate(100 bytes) = %d bytes, want 40", len(got))
	}
	if got := truncate([]byte("short")); string(got) != "short" {
		t.Errorf("truncate(short) = %q", got)
	}
}
