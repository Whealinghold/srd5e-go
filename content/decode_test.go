package content_test

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/content-schema-and-decoder.md FR-007 - the supported versions are {1}, and the slice is
// a copy.
// Path: unhappy (immutability).
func TestSupportedSchemaVersions(t *testing.T) {
	t.Parallel()
	if content.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", content.SchemaVersion)
	}
	v := content.SupportedSchemaVersions()
	if !slices.Equal(v, []int{1}) {
		t.Fatalf("SupportedSchemaVersions() = %v, want [1]", v)
	}
	v[0] = 99
	if got := content.SupportedSchemaVersions(); !slices.Equal(got, []int{1}) {
		t.Errorf("mutating the result changed the next call: %v", got)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-001, FR-009, FR-078 - the zero JSONDecoder works, a
// minimal document decodes to an empty Set with a nil error, and a rejection
// is the zero Set with problems that match ErrInvalidInput.
// Path: happy.
func TestJSONDecoderZeroValue(t *testing.T) {
	t.Parallel()
	var d content.JSONDecoder
	set, err := d.Decode([]byte(`{"schemaVersion":1}`))
	if err != nil || len(set.Entries()) != 0 {
		t.Fatalf("Decode(minimal) = (%v, %v), want an empty Set and nil", set, err)
	}
	set, err = d.Decode([]byte(`{}`))
	ps := content.Problems(err)
	if len(set.Entries()) != 0 || len(ps) != 1 || ps[0] != content.NewProblem("/schemaVersion", content.ProblemMissingField) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("Decode({}) = (%v, %v), want one missingField at /schemaVersion", set, err)
	}
}

// Requirement: doc/plans/plan-phase-2.md section 5.4 - while only the rows of steps 5a to 5g are registered (species, backgrounds, classes, feats, spells, the equipment arrays, monsters and
// magicItems) every other root array is an unknown field (the later rows flip the status).
// Path: unhappy (invalid input).
func TestJSONDecoderUnregisteredArraysAreUnknown(t *testing.T) {
	t.Parallel()
	_, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":1,"traps":[],"effects":[]}`))
	var got []string
	for _, p := range content.Problems(err) {
		got = append(got, p.Path()+" "+p.Kind().String())
	}
	if want := []string{"/effects unknownField", "/traps unknownField"}; !slices.Equal(got, want) {
		t.Errorf("problems = %v, want %v", got, want)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-018 - the decoder never panics for any bytes and
// every rejection is a typed list of problems, the same on every call.
// Path: unhappy (invalid input).
func TestJSONDecoderHostileCorpus(t *testing.T) {
	t.Parallel()
	corpus := [][]byte{
		nil, {}, {0}, {0xff, 0xfe}, []byte("\xef\xbb\xbf{}"), []byte("{"), []byte("}"), []byte(`{"schemaVersion":`),
		[]byte(`{"schemaVersion":1e999999999}`), []byte(`{"schemaVersion":12345678901234567890}`),
		[]byte(`{"schemaVersion":1,"a":"\ud800"}`), []byte("{\"schemaVersion\":1,\"a\xff\":1}"),
		[]byte(`{"schemaVersion":-0}`), []byte(`{"schemaVersion":1,"x":` + strings.Repeat("[", 5000) + strings.Repeat("]", 5000) + `}`),
		[]byte(`{"schemaVersion":1,"` + strings.Repeat("k", 1_000_000) + `":1}`),
		[]byte(`{"schemaVersion":1,"a":` + strings.Repeat("9", 1_000_000) + `}`),
		[]byte(`{"schemaVersion":` + strings.Repeat("9", 1_000_000) + `}`),
		[]byte(`{"schemaVersion":1.` + strings.Repeat("0", 1_000_000) + `}`),
		[]byte(`{"schemaVersion":1,"x":[` + strings.Repeat("1,", 1_000_000) + `1]}`),
	}
	for i, doc := range corpus {
		set1, err1 := content.JSONDecoder{}.Decode(doc)
		set2, err2 := content.JSONDecoder{}.Decode(doc)
		if (err1 == nil) != (err2 == nil) || !slices.Equal(content.Problems(err1), content.Problems(err2)) || !set1.Equal(set2) {
			t.Errorf("document %d decodes differently twice", i)
		}
		if err1 != nil && (len(content.Problems(err1)) == 0 || !errors.Is(err1, srderr.ErrInvalidInput) || len(content.Problems(err1)) > content.MaxProblems+1) {
			t.Errorf("document %d: error %v is not a bounded list of problems", i, err1)
		}
	}
	// A 1 MB number literal as the version is one unsupportedVersion, a long
	// fraction of zeros is the integer 1.
	if _, err := (content.JSONDecoder{}).Decode([]byte(`{"schemaVersion":1.` + strings.Repeat("0", 1_000_000) + `}`)); err != nil {
		t.Errorf("1.000...0 = %v, want the integer 1", err)
	}
	if _, err := (content.JSONDecoder{}).Decode([]byte(`{"schemaVersion":` + strings.Repeat("9", 1_000_000) + `}`)); !errors.Is(err, srderr.ErrInvalidInput) || content.Problems(err)[0].Kind() != content.ProblemUnsupportedVersion {
		t.Errorf("a megabyte of nines = %v, want unsupportedVersion", err)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-076 - concurrent Decode calls share nothing, and the
// input is neither retained nor modified.
// Path: unhappy (immutability).
func TestJSONDecoderConcurrent(t *testing.T) {
	t.Parallel()
	docs := [][]byte{[]byte(`{"schemaVersion":1}`), []byte(`{"schemaVersion":2}`), []byte(`{"schemaVersion":1,"x":1,"y":2}`), []byte(`not json`)}
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc := bytes.Clone(docs[i%len(docs)])
			set, err := content.JSONDecoder{}.Decode(doc)
			wantOK := i%len(docs) == 0
			if (err == nil) != wantOK || !bytes.Equal(doc, docs[i%len(docs)]) || (err != nil && !set.Equal(content.Set{})) {
				t.Errorf("document %d: Decode = (%v, %v) or the input changed", i%len(docs), set, err)
			}
		}()
	}
	wg.Wait()
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-087 - JSONDecoder passes the decoder contract suite
// on real valid and invalid documents.
// Path: happy.
func TestJSONDecoderPassesDecoderContract(t *testing.T) {
	t.Parallel()
	contenttest.RunDecoderContract(t, func(*testing.T) content.Decoder { return content.JSONDecoder{} },
		[]byte(`{"schemaVersion":1}`), []byte(`{"schemaVersion":2}`))
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-016 - the same text always gives an equal result,
// and documents that differ only in whitespace give equal results.
// Path: happy.
func TestJSONDecoderIsDeterministicAcrossWhitespace(t *testing.T) {
	t.Parallel()
	a, errA := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":1,"b":1,"a":2}`))
	b, errB := content.JSONDecoder{}.Decode([]byte(" {\n\t\"schemaVersion\" : 1 ,\r\n \"b\":1,\"a\":2 }\n"))
	if !slices.Equal(content.Problems(errA), content.Problems(errB)) || !a.Equal(b) || len(content.Problems(errA)) != 2 {
		t.Errorf("problems differ across whitespace: %v vs %v", content.Problems(errA), content.Problems(errB))
	}
}

// Requirement: doc/plans/plan-phase-2.md P13 and doc/specs/content-schema-and-decoder.md FR-153 - a document over MaxDocumentBytes is one outOfRange problem at
// the root, before any parsing.
// Path: unhappy (boundary).
func TestJSONDecoderRejectsOversizedDocument(t *testing.T) {
	t.Parallel()
	_, err := content.JSONDecoder{}.Decode(bytes.Repeat([]byte(" "), content.MaxDocumentBytes+1))
	if ps := content.Problems(err); !slices.Equal(ps, []content.Problem{content.NewProblem("", content.ProblemOutOfRange)}) {
		t.Errorf("problems = %v, want one outOfRange at the root", ps)
	}
	if content.MaxDocumentBytes != 16<<20 || content.MaxProblems != 1000 || content.MaxSafeInteger != 9007199254740991 {
		t.Errorf("limits changed: %d %d %d", content.MaxDocumentBytes, content.MaxProblems, int64(content.MaxSafeInteger))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-021 and FR-023 - a schemaVersion above 2^32 or
// below -2^32 that would truncate to 1 in a 32-bit int is unsupportedVersion on every platform (the
// test-386 job runs this).
// Path: unhappy (boundary).
func TestDecodeSchemaVersionDoesNotTruncate(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"4294967297", "-4294967295"} {
		_, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":` + v + `}`))
		want := []content.Problem{content.NewProblem("/schemaVersion", content.ProblemUnsupportedVersion)}
		if got := content.Problems(err); !slices.Equal(got, want) {
			t.Errorf("schemaVersion %s: problems = %v, want %v", v, got, want)
		}
	}
}
