package contenttest

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RunDecoderContract runs the format-independent contract of
// [content.Decoder] against decoders built by factory, one fresh decoder per
// subtest. valid must be a document the decoder accepts and invalid one it
// rejects; the suite does not know the format, so the caller supplies both.
//
// It checks that valid decodes with a nil error and invalid with a non-nil
// error that holds at least one [content.Problem], matches
// srderr.ErrInvalidInput and comes with the zero Set; that a small corpus of
// bytes (nil, empty, binary) never panics and is either accepted or rejected
// with problems, never with an error that holds none; that decoding the same
// bytes twice gives equal results; and that the decoder neither modifies nor
// retains its input (a result stays equal after the input buffer is
// overwritten). It covers schema FR-001 and FR-018 at port level only: what a
// document's text means is tested on the decoder itself.
//
// A consumer's own decoder runs the same suite from its own tests.
func RunDecoderContract(t *testing.T, factory func(t *testing.T) content.Decoder, valid, invalid []byte) {
	t.Helper()
	t.Run("valid document", func(t *testing.T) { checkValidDocument(t, factory(t), valid) })
	t.Run("invalid document", func(t *testing.T) { checkInvalidDocument(t, factory(t), invalid) })
	t.Run("hostile corpus", func(t *testing.T) { checkHostileCorpus(t, factory(t)) })
	t.Run("repeatable", func(t *testing.T) {
		d := factory(t)
		checkRepeatableDecode(t, d, valid)
		checkRepeatableDecode(t, d, invalid)
	})
	t.Run("input untouched", func(t *testing.T) {
		d := factory(t)
		checkInputUntouched(t, d, valid)
		checkInputUntouched(t, d, invalid)
	})
}

// decode calls d and reports a panic instead of crashing the suite.
func decode(r reporter, d content.Decoder, document []byte) (set content.Set, err error, panicked bool) {
	r.Helper()
	defer func() {
		if v := recover(); v != nil {
			r.Errorf("Decode(%q) panicked: %v", truncate(document), v)
			set, err, panicked = content.Set{}, nil, true
		}
	}()
	set, err = d.Decode(document)
	return set, err, false
}

func truncate(b []byte) []byte {
	if len(b) > 40 {
		return b[:40]
	}
	return b
}

func checkValidDocument(r reporter, d content.Decoder, valid []byte) {
	r.Helper()
	if _, err, panicked := decode(r, d, valid); err != nil && !panicked {
		r.Errorf("Decode(valid) error = %v, want nil", err)
	}
}

func checkInvalidDocument(r reporter, d content.Decoder, invalid []byte) {
	r.Helper()
	set, err, panicked := decode(r, d, invalid)
	switch {
	case panicked:
	case err == nil:
		r.Errorf("Decode(invalid) error = nil, want a rejection")
	case len(content.Problems(err)) == 0:
		r.Errorf("Decode(invalid) error holds no Problem: %v", err)
	case !errors.Is(err, srderr.ErrInvalidInput):
		r.Errorf("Decode(invalid) error does not match srderr.ErrInvalidInput: %v", err)
	case !set.Equal(content.Set{}):
		r.Errorf("Decode(invalid) returned a non-zero Set with its error")
	}
}

func checkHostileCorpus(r reporter, d content.Decoder) {
	r.Helper()
	corpus := [][]byte{nil, {}, {0x00}, {0xff, 0xfe, 0x00, 0x80}, []byte("\xef\xbb\xbf{}"), []byte(`{"a":`), bytes.Repeat([]byte("["), 64)}
	for _, doc := range corpus {
		_, err, panicked := decode(r, d, doc)
		if !panicked && err != nil && len(content.Problems(err)) == 0 {
			r.Errorf("Decode(%q) error holds no Problem: %v", truncate(doc), err)
		}
	}
}

func checkRepeatableDecode(r reporter, d content.Decoder, document []byte) {
	r.Helper()
	set1, err1, p1 := decode(r, d, document)
	set2, err2, p2 := decode(r, d, document)
	if p1 || p2 {
		return
	}
	if (err1 == nil) != (err2 == nil) || !slices.Equal(content.Problems(err1), content.Problems(err2)) || !set1.Equal(set2) {
		r.Errorf("Decode(%q) twice gave different results: (%v, %v) then (%v, %v)", truncate(document), set1, err1, set2, err2)
	}
}

func checkInputUntouched(r reporter, d content.Decoder, document []byte) {
	r.Helper()
	input := slices.Clone(document)
	set1, err1, panicked := decode(r, d, input)
	if panicked {
		return
	}
	if !bytes.Equal(input, document) {
		r.Errorf("Decode modified its input")
	}
	want := fmt.Sprint(content.Problems(err1))
	clear(input) // a decoder that kept the buffer would now see zeros
	set2, err2, _ := decode(r, d, slices.Clone(document))
	if fmt.Sprint(content.Problems(err1)) != want || !set1.Equal(set2) || (err1 == nil) != (err2 == nil) {
		r.Errorf("Decode retained its input: the result changed after the buffer was overwritten")
	}
}
