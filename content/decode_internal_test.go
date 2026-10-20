package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// The decoder core is exercised here with a synthetic registry because no kind
// row is registered yet (plan step 4): the framework (envelope, field rules,
// problem order, caps) cannot be reached through the public API.

// widget is the synthetic entry of the test registry.
type widget struct {
	key    Key
	name   string
	notes  string
	source SourceCitation
}

func (w widget) Key() Key { return w.key }

// widgetRow returns the row of a synthetic kind, whose fields exercise every
// field rule in a fixed order: id, name, source?, level, school, tags?,
// notes?, slots?.
func widgetRow(kind Kind) func(*object) decodedEntry {
	return func(o *object) decodedEntry {
		before := o.d.probs.count
		h := o.header()
		var notes string
		if n, p, ok := o.field("level", true); ok {
			o.d.integer(n, p, 1, 20)
		}
		if n, p, ok := o.field("school", true); ok {
			enumField(o.d, n, p, ParseDamageType)
		}
		if n, p, ok := o.field("tags", false); ok {
			enumSet(o.d, n, p, 1, 3, ParseDamageType)
		}
		if n, p, ok := o.field("notes", false); ok {
			notes, _ = o.d.text(n, p)
		}
		if n, p, ok := o.field("slots", false); ok {
			if _, ok := o.d.arrayLen(n, p, 0, 3); ok {
				for i, el := range n.elements() {
					o.d.integer(el, indexPath(p, i), 0, 9)
				}
			}
		}
		res := decodedEntry{id: h.id, idOK: h.idOK}
		if o.d.probs.count == before {
			res.entry = widget{key: NewKey(kind, h.id), name: h.name, notes: notes, source: h.source}
		}
		return res
	}
}

// registry mirrors the revision-table order of the arrays it uses.
func registry() []rootArray {
	return []rootArray{
		{key: "species", kind: KindSpecies, decode: widgetRow(KindSpecies)},
		{key: "classes", kind: KindCharacterClass, decode: widgetRow(KindCharacterClass)},
		{key: "feats", kind: KindFeat, decode: widgetRow(KindFeat)},
		{key: "spells", kind: KindSpell, decode: widgetRow(KindSpell)},
		{key: "weapons", kind: KindEquipment, decode: widgetRow(KindEquipment)},
		{key: "armor", kind: KindEquipment, decode: widgetRow(KindEquipment)},
	}
}

// good is a valid widget with id and optional extra members after "school".
func good(id string, extra ...string) string {
	s := `{"id":"` + id + `","name":"N","level":1,"school":"Acid"`
	for _, e := range extra {
		s += "," + e
	}
	return s + "}"
}

func doc(members ...string) string {
	return `{"schemaVersion":1` + strings.Join(append([]string{""}, members...), ",") + "}"
}

func arrayOf(key string, elems ...string) string {
	return `"` + key + `":[` + strings.Join(elems, ",") + `]`
}

func render(err error) []string {
	var out []string
	for _, p := range Problems(err) {
		out = append(out, p.Path()+" "+p.Kind().String())
	}
	return out
}

func decodeText(t *testing.T, text string) (Set, []string) {
	t.Helper()
	set, err := decodeDocument([]byte(text), registry())
	if err == nil && !set.Equal(Set{}) && len(Problems(err)) != 0 {
		t.Fatalf("a set came with problems")
	}
	if err != nil && (!set.Equal(Set{}) || !errors.Is(err, srderr.ErrInvalidInput) || len(Problems(err)) == 0) {
		t.Fatalf("a rejection must be the zero Set and an error of problems, got (%v, %v)", set, err)
	}
	return set, render(err)
}

type decodeCase struct {
	name, requirement, path string
	doc                     string
	want                    []string // "path kind"; nil means accepted
}

func runDecodeCases(t *testing.T, cases []decodeCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, got := decodeText(t, c.doc)
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %q, want %q [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestDecodeEnvelope(t *testing.T) {
	t.Parallel()
	const (
		fr2   = "doc/specs/content-schema-and-decoder.md FR-002"
		fr3   = "doc/specs/content-schema-and-decoder.md FR-003"
		fr456 = "doc/specs/content-schema-and-decoder.md FR-004 to FR-006, FR-021"
		fr8   = "doc/specs/content-schema-and-decoder.md FR-008 to FR-010"
	)
	runDecodeCases(t, []decodeCase{
		{"empty text", fr2, "unhappy (invalid input)", ``, []string{" syntax"}},
		{"whitespace only", fr2, "unhappy (invalid input)", " \n\t", []string{" syntax"}},
		{"truncated", fr2, "unhappy (invalid input)", `{"schemaVersion": 1`, []string{" syntax"}},
		{"trailing data", fr2, "unhappy (invalid input)", `{"schemaVersion":1} 2`, []string{" syntax"}},
		{"byte order mark", fr2, "unhappy (invalid input)", "\xef\xbb\xbf{\"schemaVersion\":1}", []string{" syntax"}},
		{"array root", fr3, "unhappy (invalid input)", `[]`, []string{" wrongType"}},
		{"null root", fr3, "unhappy (invalid input)", `null`, []string{" wrongType"}},
		{"number root", fr3, "unhappy (invalid input)", `1`, []string{" wrongType"}},
		{"string root", fr3, "unhappy (invalid input)", `"x"`, []string{" wrongType"}},
		{"no version", fr456, "unhappy (missing content)", `{}`, []string{"/schemaVersion missingField"}},
		{"version as string", fr456, "unhappy (invalid input)", `{"schemaVersion":"1"}`, []string{"/schemaVersion wrongType"}},
		{"version fractional", fr456, "unhappy (invalid input)", `{"schemaVersion":1.5}`, []string{"/schemaVersion wrongType"}},
		{"version as a string hides the rest", fr456, "unhappy (rule interaction)", `{"schemaVersion":"1","bogus":1,"spells":3}`, []string{"/schemaVersion wrongType"}},
		{"fractional version hides the rest", fr456, "unhappy (rule interaction)", `{"schemaVersion":1.5,"bogus":1}`, []string{"/schemaVersion wrongType"}},
		{"version null", fr456, "unhappy (invalid input)", `{"schemaVersion":null}`, []string{"/schemaVersion wrongType"}},
		{"version boolean", fr456, "unhappy (invalid input)", `{"schemaVersion":true}`, []string{"/schemaVersion wrongType"}},
		{"version array", fr456, "unhappy (invalid input)", `{"schemaVersion":[1]}`, []string{"/schemaVersion wrongType"}},
		{"version 1.0", fr456, "unhappy (boundary)", `{"schemaVersion":1.0}`, nil},
		{"version 10e-1", fr456, "unhappy (boundary)", `{"schemaVersion":10e-1}`, nil},
		{"version 0", fr456, "unhappy (boundary)", `{"schemaVersion":0}`, []string{"/schemaVersion unsupportedVersion"}},
		{"version minus zero", fr456, "unhappy (boundary)", `{"schemaVersion":-0}`, []string{"/schemaVersion unsupportedVersion"}},
		{"version 2", fr456, "unhappy (boundary)", `{"schemaVersion":2}`, []string{"/schemaVersion unsupportedVersion"}},
		{"version -1", fr456, "unhappy (boundary)", `{"schemaVersion":-1}`, []string{"/schemaVersion unsupportedVersion"}},
		{"version 20 digits", fr456, "unhappy (boundary)", `{"schemaVersion":12345678901234567890}`, []string{"/schemaVersion unsupportedVersion"}},
		{"version huge exponent", fr456, "unhappy (boundary)", `{"schemaVersion":1e999999999}`, []string{"/schemaVersion unsupportedVersion"}},
		{"version 2^53", fr456, "unhappy (boundary)", `{"schemaVersion":9007199254740992}`, []string{"/schemaVersion unsupportedVersion"}},
		{"version plus other errors", fr456, "unhappy (rule interaction)", `{"schemaVersion":2,"bogus":1}`, []string{"/schemaVersion unsupportedVersion"}},
		{"missing version hides the rest", fr456, "unhappy (rule interaction)", `{"bogus":1,"spells":3}`, []string{"/schemaVersion missingField"}},
		{"minimal", fr8, "unhappy (empty case)", `{"schemaVersion":1}`, nil},
		{"every array may be empty", fr8, "unhappy (empty case)", doc(arrayOf("species"), arrayOf("classes"), arrayOf("feats"), arrayOf("spells"), arrayOf("weapons"), arrayOf("armor")), nil},
		{"legacy key", fr8, "unhappy (invalid input)", `{"schemaVersion":1,"races":[]}`, []string{"/races unknownField"}},
		{"unregistered array", fr8, "unhappy (invalid input)", doc(arrayOf("tools")), []string{"/tools unknownField"}},
		{"slash in key is escaped", fr8, "unhappy (boundary)", `{"schemaVersion":1,"a/b":1}`, []string{"/a~1b unknownField"}},
		{"tilde in key is escaped", fr8, "unhappy (boundary)", `{"schemaVersion":1,"a~b":1}`, []string{"/a~0b unknownField"}},
		{"unknown keys in ordinal order", fr8, "happy", `{"schemaVersion":1,"b":1,"a":2,"B":3}`, []string{"/B unknownField", "/a unknownField", "/b unknownField"}},
		{"array field not an array", "doc/specs/content-schema-and-decoder.md FR-011", "unhappy (invalid input)", `{"schemaVersion":1,"spells":{}}`, []string{"/spells wrongType"}},
		{"array field null", "doc/specs/content-schema-and-decoder.md FR-011, FR-020", "unhappy (invalid input)", `{"schemaVersion":1,"spells":null}`, []string{"/spells wrongType"}},
		{"element not an object", "doc/specs/content-schema-and-decoder.md FR-011", "unhappy (invalid input)", doc(arrayOf("spells", "1")), []string{"/spells/0 wrongType"}},
		{"elements not objects, none cascade", "doc/specs/content-schema-and-decoder.md FR-011, FR-013", "unhappy (rule interaction)", doc(arrayOf("spells", "[]", "null", `"x"`)), []string{"/spells/0 wrongType", "/spells/1 wrongType", "/spells/2 wrongType"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeFieldRules(t *testing.T) {
	t.Parallel()
	const (
		fr12   = "doc/specs/content-schema-and-decoder.md FR-012, FR-016"
		fr13   = "doc/specs/content-schema-and-decoder.md FR-013"
		fr19   = "doc/specs/content-schema-and-decoder.md FR-019 to FR-025"
		fr2628 = "doc/specs/content-schema-and-decoder.md FR-026 to FR-030"
		fr34   = "doc/specs/content-schema-and-decoder.md FR-022, FR-034, FR-115"
	)
	spell := func(extra ...string) string { return doc(arrayOf("spells", good("a", extra...))) }
	runDecodeCases(t, []decodeCase{
		{"valid entry", fr12, "happy", spell(), nil},
		{"two entries, two problems", fr12, "unhappy (invalid input)",
			doc(arrayOf("spells", `{"id":"a","name":"N","level":1}`, good("b"), good("c"), `{"id":"d","name":"N","level":1}`)),
			[]string{"/spells/0/school missingField", "/spells/3/school missingField"}},
		{"two fields of one entry", fr12, "unhappy (invalid input)",
			doc(arrayOf("spells", `{"id":"a","name":"N","level":"x","school":"Sound"}`)),
			[]string{"/spells/0/level wrongType", "/spells/0/school invalidValue"}},
		{"fields in declared order, not document order", fr12, "happy",
			doc(arrayOf("spells", `{"school":"Sound","level":0,"name":" ","id":"A"}`)),
			[]string{"/spells/0/id invalidValue", "/spells/0/name invalidValue", "/spells/0/level outOfRange", "/spells/0/school invalidValue"}},
		{"no cascade: array field as a string", fr13, "unhappy (rule interaction)", spell(`"tags":"Acid"`), []string{"/spells/0/tags wrongType"}},
		{"no cascade: bounded array of bad children", fr13, "unhappy (rule interaction)", spell(`"slots":[10,10,10,10]`), []string{"/spells/0/slots outOfRange"}},
		{"missing id", fr19, "unhappy (missing content)", doc(arrayOf("spells", `{"name":"N","level":1,"school":"Acid"}`)), []string{"/spells/0/id missingField"}},
		{"null required field", fr19, "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":null,"level":1,"school":"Acid"}`)), []string{"/spells/0/name wrongType"}},
		{"null optional field", fr19, "unhappy (invalid input)", spell(`"notes":null`), []string{"/spells/0/notes wrongType"}},
		{"unknown key in an entry", fr19, "unhappy (invalid input)", spell(`"zeta":1`), []string{"/spells/0/zeta unknownField"}},
		{"unknown key at depth", fr34, "unhappy (invalid input)", spell(`"source":{"document":"d","section":"s","page":1,"url":"x"}`), []string{"/spells/0/source/url unknownField"}},
		{"unknown keys sorted after the known problems", fr19, "happy", doc(arrayOf("spells", `{"zz":1,"id":"a","name":"N","level":0,"school":"Acid","aa":1}`)), []string{"/spells/0/level outOfRange", "/spells/0/aa unknownField", "/spells/0/zz unknownField"}},
		{"legacy key in an entry", "doc/specs/content-schema-and-decoder.md FR-074", "unhappy (invalid input)", spell(`"race":"x"`), []string{"/spells/0/race unknownField"}},
		{"integer 2.0", "doc/specs/content-schema-and-decoder.md FR-021", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":2.0,"school":"Acid"}`)), nil},
		{"integer 2e0", "doc/specs/content-schema-and-decoder.md FR-021", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":2e0,"school":"Acid"}`)), nil},
		{"integer 2.5", "doc/specs/content-schema-and-decoder.md FR-021", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":2.5,"school":"Acid"}`)), []string{"/spells/0/level wrongType"}},
		{"integer as string", "doc/specs/content-schema-and-decoder.md FR-021", "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N","level":"2","school":"Acid"}`)), []string{"/spells/0/level wrongType"}},
		{"integer as boolean", "doc/specs/content-schema-and-decoder.md FR-021", "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N","level":true,"school":"Acid"}`)), []string{"/spells/0/level wrongType"}},
		{"integer of 20 digits", "doc/specs/content-schema-and-decoder.md FR-018, FR-021", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":12345678901234567890,"school":"Acid"}`)), []string{"/spells/0/level outOfRange"}},
		{"integer with a huge exponent", "doc/specs/content-schema-and-decoder.md FR-018, FR-021", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":1e999999999,"school":"Acid"}`)), []string{"/spells/0/level outOfRange"}},
		{"integer 2^53 - 1 is above the range", "doc/specs/content-schema-and-decoder.md FR-023", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":9007199254740991,"school":"Acid"}`)), []string{"/spells/0/level outOfRange"}},
		{"range low end", "doc/specs/content-schema-and-decoder.md FR-023", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":1,"school":"Acid"}`)), nil},
		{"one below the range", "doc/specs/content-schema-and-decoder.md FR-023", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":0,"school":"Acid"}`)), []string{"/spells/0/level outOfRange"}},
		{"range high end", "doc/specs/content-schema-and-decoder.md FR-023", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":20,"school":"Acid"}`)), nil},
		{"one above the range", "doc/specs/content-schema-and-decoder.md FR-023", "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a","name":"N","level":21,"school":"Acid"}`)), []string{"/spells/0/level outOfRange"}},
		{"array of one", "doc/specs/content-schema-and-decoder.md FR-024", "unhappy (boundary)", spell(`"tags":["Acid"]`), nil},
		{"array of three", "doc/specs/content-schema-and-decoder.md FR-024", "unhappy (boundary)", spell(`"tags":["Acid","Cold","Fire"]`), nil},
		{"array too short", "doc/specs/content-schema-and-decoder.md FR-024", "unhappy (boundary)", spell(`"tags":[]`), []string{"/spells/0/tags outOfRange"}},
		{"array too long", "doc/specs/content-schema-and-decoder.md FR-024", "unhappy (boundary)", spell(`"tags":["Acid","Cold","Fire","Force"]`), []string{"/spells/0/tags outOfRange"}},
		{"bounded array at its bound", "doc/specs/content-schema-and-decoder.md FR-024", "unhappy (boundary)", spell(`"slots":[0,9,5]`), nil},
		{"bounded array element out of range", "doc/specs/content-schema-and-decoder.md FR-023, FR-024", "unhappy (boundary)", spell(`"slots":[0,10]`), []string{"/spells/0/slots/1 outOfRange"}},
		{"enum wrong case", "doc/specs/content-schema-and-decoder.md FR-025", "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N","level":1,"school":"acid"}`)), []string{"/spells/0/school invalidValue"}},
		{"enum padded", "doc/specs/content-schema-and-decoder.md FR-025", "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N","level":1,"school":" Acid"}`)), []string{"/spells/0/school invalidValue"}},
		{"enum as number", "doc/specs/content-schema-and-decoder.md FR-025, FR-021", "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N","level":1,"school":1}`)), []string{"/spells/0/school wrongType"}},
		{"set with a repeat", fr2628, "unhappy (invalid input)", spell(`"tags":["Acid","Cold","Acid"]`), []string{"/spells/0/tags/2 duplicate"}},
		{"set element wrong type and invalid", fr2628, "unhappy (invalid input)", spell(`"tags":[1,"nope"]`), []string{"/spells/0/tags/0 wrongType", "/spells/0/tags/1 invalidValue"}},
		{"name with leading space", fr2628, "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":" N","level":1,"school":"Acid"}`)), []string{"/spells/0/name invalidValue"}},
		{"name with trailing space", fr2628, "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N ","level":1,"school":"Acid"}`)), []string{"/spells/0/name invalidValue"}},
		{"empty name", fr2628, "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"","level":1,"school":"Acid"}`)), []string{"/spells/0/name invalidValue"}},
		{"name with an interior space", fr2628, "happy", doc(arrayOf("spells", `{"id":"a","name":"Acid Arrow","level":1,"school":"Acid"}`)), nil},
		{"blank text", fr2628, "unhappy (empty case)", spell(`"notes":" \n "`), []string{"/spells/0/notes invalidValue"}},
		{"id of 64 characters", fr2628, "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a`+strings.Repeat("b", 63)+`","name":"N","level":1,"school":"Acid"}`)), nil},
		{"id of 65 characters", fr2628, "unhappy (boundary)", doc(arrayOf("spells", `{"id":"a`+strings.Repeat("b", 64)+`","name":"N","level":1,"school":"Acid"}`)), []string{"/spells/0/id invalidValue"}},
		{"id with a digit first", fr2628, "unhappy (boundary)", doc(arrayOf("spells", `{"id":"2a","name":"N","level":1,"school":"Acid"}`)), []string{"/spells/0/id invalidValue"}},
		{"id with a hyphen", fr2628, "unhappy (boundary)", doc(arrayOf("spells", `{"id":"acid-arrow","name":"N","level":1,"school":"Acid"}`)), []string{"/spells/0/id invalidValue"}},
		{"id with an underscore and digit", fr2628, "unhappy (boundary)", doc(arrayOf("spells", `{"id":"acid_arrow_2","name":"N","level":1,"school":"Acid"}`)), nil},
		{"empty id", fr2628, "unhappy (boundary)", doc(arrayOf("spells", `{"id":"","name":"N","level":1,"school":"Acid"}`)), []string{"/spells/0/id invalidValue"}},
		{"id as a number", fr2628, "unhappy (invalid input)", doc(arrayOf("spells", `{"id":1,"name":"N","level":1,"school":"Acid"}`)), []string{"/spells/0/id wrongType"}},
		{"source accepted", fr34, "happy", spell(`"source":{"document":"SRD 5.2.1","section":"Damage Rolls","page":16}`), nil},
		{"source page 1", fr34, "unhappy (boundary)", spell(`"source":{"document":"d","section":"s","page":1}`), nil},
		{"source page 0", fr34, "unhappy (boundary)", spell(`"source":{"document":"d","section":"s","page":0}`), []string{"/spells/0/source/page outOfRange"}},
		{"source page above the maximum", fr34, "unhappy (boundary)", spell(`"source":{"document":"d","section":"s","page":2147483648}`), []string{"/spells/0/source/page outOfRange"}},
		{"source maximum page", fr34, "unhappy (boundary)", spell(`"source":{"document":"d","section":"s","page":2147483647}`), nil},
		{"source without a section", fr34, "unhappy (missing content)", spell(`"source":{"document":"d","page":1}`), []string{"/spells/0/source/section missingField"}},
		{"source without a document and a page", fr34, "unhappy (missing content)", spell(`"source":{"section":"s"}`), []string{"/spells/0/source/document missingField", "/spells/0/source/page missingField"}},
		{"source with a blank section", fr34, "unhappy (invalid input)", spell(`"source":{"document":"d","section":"  ","page":1}`), []string{"/spells/0/source/section invalidValue"}},
		{"source with a blank document", fr34, "unhappy (invalid input)", spell(`"source":{"document":"  ","section":"s","page":1}`), []string{"/spells/0/source/document invalidValue"}},
		{"source not an object", fr34, "unhappy (invalid input)", spell(`"source":"SRD"`), []string{"/spells/0/source wrongType"}},
		{"source fields of the wrong type", fr34, "unhappy (invalid input)", spell(`"source":{"document":1,"section":true,"page":"1"}`), []string{"/spells/0/source/document wrongType", "/spells/0/source/section wrongType", "/spells/0/source/page wrongType"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeDuplicateIDs(t *testing.T) {
	t.Parallel()
	const fr31 = "doc/specs/content-schema-and-decoder.md FR-016, FR-031, FR-032, FR-095, FR-098"
	bad := func(id string) string { return `{"id":"` + id + `","name":"N","level":1}` } // no school
	runDecodeCases(t, []decodeCase{
		{"both clean", fr31, "unhappy (invalid input)", doc(arrayOf("spells", good("a"), good("a"))), []string{"/spells/1/id duplicate"}},
		{"three of one id", fr31, "unhappy (invalid input)", doc(arrayOf("spells", good("a"), good("a"), good("a"))), []string{"/spells/1/id duplicate", "/spells/2/id duplicate"}},
		{"earlier entry has another problem", fr31, "unhappy (rule interaction)", doc(arrayOf("spells", bad("a"), good("a"))), []string{"/spells/0/school missingField", "/spells/1/id duplicate"}},
		{"later entry has another problem", fr31, "unhappy (rule interaction)", doc(arrayOf("spells", good("a"), bad("a"))), []string{"/spells/1/school missingField", "/spells/1/id duplicate"}},
		{"both have another problem", fr31, "unhappy (rule interaction)", doc(arrayOf("spells", bad("a"), bad("a"))), []string{"/spells/0/school missingField", "/spells/1/school missingField", "/spells/1/id duplicate"}},
		{"invalid ids never duplicate", fr31, "unhappy (rule interaction)", doc(arrayOf("spells", good("A"), good("A"))), []string{"/spells/0/id invalidValue", "/spells/1/id invalidValue"}},
		{"same id in two kinds", fr31, "unhappy (rule interaction)", doc(arrayOf("feats", good("a")), arrayOf("spells", good("a"))), nil},
		{"weapon and armor share an id", fr31, "unhappy (invalid input)", doc(arrayOf("weapons", good("a")), arrayOf("armor", good("a"))), []string{"/armor/0/id duplicate"}},
		{"entry problems come before duplicates", fr31, "happy", doc(arrayOf("spells", good("a"), good("a")), arrayOf("armor", `{"id":"b","name":"N","level":1}`)), []string{"/armor/0/school missingField", "/spells/1/id duplicate"}},
		{"duplicate groups by kind, not by root order", fr31, "unhappy (rule interaction)",
			doc(arrayOf("species", good("s"), good("s")), arrayOf("classes", good("c"), good("c")), arrayOf("feats", good("f"), good("f")),
				arrayOf("spells", good("p"), good("p")), arrayOf("weapons", good("w"), good("w"))),
			[]string{"/species/1/id duplicate", "/classes/1/id duplicate", "/feats/1/id duplicate", "/weapons/1/id duplicate", "/spells/1/id duplicate"}},
		{"within equipment weapons come before armor", fr31, "unhappy (rule interaction)",
			doc(arrayOf("weapons", good("w"), good("w")), arrayOf("armor", good("a"), good("a"), good("w"))),
			[]string{"/weapons/1/id duplicate", "/armor/1/id duplicate", "/armor/2/id duplicate"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeRepeatedKeys(t *testing.T) {
	t.Parallel()
	const p14 = "doc/plans/plan-phase-2.md P14 and doc/specs/content-schema-and-decoder.md FR-155 - a repeated key in one object is a duplicate problem at the later key"
	runDecodeCases(t, []decodeCase{
		{"repeated schemaVersion", p14, "unhappy (invalid input)", `{"schemaVersion":1,"schemaVersion":1}`, []string{"/schemaVersion duplicate"}},
		{"repeated known key in an entry", p14, "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N","id":"b","level":1,"school":"Acid"}`)), []string{"/spells/0/id duplicate"}},
		{"repeated unknown key is also unknown", p14, "unhappy (invalid input)", `{"schemaVersion":1,"x":1,"x":2}`, []string{"/x duplicate", "/x unknownField"}},
		{"duplicates before unknown keys of the object", p14, "happy", doc(arrayOf("spells", `{"zz":1,"id":"a","name":"N","level":1,"school":"Acid","id":"a"}`)), []string{"/spells/0/id duplicate", "/spells/0/zz unknownField"}},
		{"repeated key in the source object", p14, "unhappy (invalid input)", doc(arrayOf("spells", good("a", `"source":{"document":"d","document":"e","section":"s","page":1}`))), []string{"/spells/0/source/document duplicate"}},
		{"repeated key with an escaped spelling", p14, "unhappy (invalid input)", `{"schemaVersion":1,"\u0078":1,"x":2}`, []string{"/x duplicate", "/x unknownField"}},
		{"repeated key past an unread subtree", p14, "unhappy (invalid input)", `{"schemaVersion":1,"x":{"a":1,"a":2},"y":1}`, []string{"/x unknownField", "/y unknownField"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeStringHygiene(t *testing.T) {
	t.Parallel()
	const p14 = "doc/plans/plan-phase-2.md P14 and doc/specs/content-schema-and-decoder.md FR-027 to FR-029, FR-156 - control and bidirectional characters, invalid UTF-8 and lone surrogates are invalidValue"
	named := func(name string) string {
		return doc(arrayOf("spells", `{"id":"a","name":"`+name+`","level":1,"school":"Acid"}`))
	}
	noted := func(text string) string { return doc(arrayOf("spells", good("a", `"notes":"`+text+`"`))) }
	const bad = "/spells/0/name invalidValue"
	const badNote = "/spells/0/notes invalidValue"
	runDecodeCases(t, []decodeCase{
		{"typographic apostrophe", p14, "happy", named("Thieves\u2019 Cant"), nil},
		{"astral character as an escaped pair", p14, "happy", named(`\ud83d\ude00`), nil},
		{"astral character raw", p14, "happy", named("\U0001F600"), nil},
		{"escaped solidus and quote", p14, "happy", named(`a\/b\"c`), nil},
		{"lone high surrogate", p14, "unhappy (invalid input)", named(`\ud800`), []string{bad}},
		{"lone low surrogate", p14, "unhappy (invalid input)", named(`\udc00`), []string{bad}},
		{"high surrogate then a plain escape", p14, "unhappy (invalid input)", named(`\ud800\u0041`), []string{bad}},
		{"two high surrogates", p14, "unhappy (invalid input)", named(`\ud800\ud800`), []string{bad}},
		{"invalid UTF-8 byte", p14, "unhappy (invalid input)", named("a\xffb"), []string{bad}},
		{"truncated UTF-8 sequence", p14, "unhappy (invalid input)", named("a\xe2\x82"), []string{bad}},
		{"invalid UTF-8 next to an escape", p14, "unhappy (invalid input)", named("\\n\xff"), []string{bad}},
		{"NUL as an escape", p14, "unhappy (invalid input)", named(`a\u0000b`), []string{bad}},
		{"bell as an escape", p14, "unhappy (invalid input)", named(`a\u0007b`), []string{bad}},
		{"DEL", p14, "unhappy (invalid input)", named(`a\u007fb`), []string{bad}},
		{"C1 control", p14, "unhappy (invalid input)", named(`a\u0085b`), []string{bad}},
		{"tab in a name", p14, "unhappy (invalid input)", named(`a\tb`), []string{bad}},
		{"line feed in a name", p14, "unhappy (invalid input)", named(`a\nb`), []string{bad}},
		{"carriage return in a name", p14, "unhappy (invalid input)", named(`a\rb`), []string{bad}},
		{"line separator", p14, "unhappy (invalid input)", named(`a\u2028b`), []string{bad}},
		{"paragraph separator", p14, "unhappy (invalid input)", named(`a\u2029b`), []string{bad}},
		{"right-to-left override", p14, "unhappy (invalid input)", named(`a\u202eb`), []string{bad}},
		{"left-to-right embedding", p14, "unhappy (invalid input)", named(`a\u202ab`), []string{bad}},
		{"first strong isolate", p14, "unhappy (invalid input)", named(`a\u2068b`), []string{bad}},
		{"pop directional isolate", p14, "unhappy (invalid input)", named(`a\u2069b`), []string{bad}},
		{"arabic letter mark", p14, "unhappy (invalid input)", named(`a\u061cb`), []string{bad}},
		{"left-to-right mark", p14, "unhappy (invalid input)", named(`a\u200eb`), []string{bad}},
		{"right-to-left mark", p14, "unhappy (invalid input)", named(`a\u200fb`), []string{bad}},
		{"zero width space is allowed", p14, "happy", named(`a\u200bb`), nil},
		{"text keeps its line feeds", "doc/specs/content-schema-and-decoder.md FR-029", "happy", noted(`line one\nline two`), nil},
		{"text rejects a tab", p14, "unhappy (invalid input)", noted(`a\tb`), []string{badNote}},
		{"text rejects a carriage return", p14, "unhappy (invalid input)", noted(`a\r\nb`), []string{badNote}},
		{"text rejects a bidi override", p14, "unhappy (invalid input)", noted(`a\u202eb`), []string{badNote}},
		{"text rejects invalid UTF-8", p14, "unhappy (invalid input)", noted("a\xffb"), []string{badNote}},
		{"enum with a bad character is not a member", p14, "unhappy (invalid input)", doc(arrayOf("spells", `{"id":"a","name":"N","level":1,"school":"Ac\ud800id"}`)), []string{"/spells/0/school invalidValue"}},
		{"id with invalid UTF-8", p14, "unhappy (invalid input)", doc(arrayOf("spells", "{\"id\":\"a\xff\",\"name\":\"N\",\"level\":1,\"school\":\"Acid\"}")), []string{"/spells/0/id invalidValue"}},
		{"source section with a line feed", p14, "unhappy (invalid input)", doc(arrayOf("spells", good("a", `"source":{"document":"d","section":"a\nb","page":1}`))), []string{"/spells/0/source/section invalidValue"}},
		{"source document with invalid UTF-8", p14, "unhappy (invalid input)", doc(arrayOf("spells", good("a", "\"source\":{\"document\":\"d\xff\",\"section\":\"s\",\"page\":1}"))), []string{"/spells/0/source/document invalidValue"}},
		{"key with a lone surrogate is an unknown key", p14, "unhappy (invalid input)", `{"schemaVersion":1,"a\ud800":1}`, []string{"/a\uFFFD unknownField"}},
		{"key with invalid UTF-8 is an unknown key", p14, "unhappy (invalid input)", "{\"schemaVersion\":1,\"a\xff\":1}", []string{"/a\uFFFD unknownField"}},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-029 - a name and a text that pass reach the entry
// character for character.
// Path: happy.
func TestDecodeKeepsTextVerbatim(t *testing.T) {
	t.Parallel()
	set, got := decodeText(t, doc(arrayOf("spells", `{"id":"a","name":"Thieves\u2019 Cant","level":1,"school":"Acid","notes":"  one\ntwo \u00e9","source":{"document":"SRD 5.2.1","section":"S","page":16}}`)))
	if len(got) != 0 {
		t.Fatalf("problems %v", got)
	}
	w, ok := set.Entries()[0].(widget)
	if !ok || w.name != "Thieves\u2019 Cant" || w.notes != "  one\ntwo \u00e9" || w.source.Text() != `SRD 5.2.1, "S", p. 16` {
		t.Errorf("entry = %+v, want the text unchanged", set.Entries()[0])
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-012, FR-016 - a clean document decodes to a Set
// ordered by kind then id whatever the document order, and equal text gives equal sets.
// Path: happy.
func TestDecodeSetOrder(t *testing.T) {
	t.Parallel()
	a, _ := decodeText(t, doc(arrayOf("spells", good("b"), good("a")), arrayOf("feats", good("z")), arrayOf("species", good("m"))))
	b, _ := decodeText(t, doc(arrayOf("species", good("m")), arrayOf("feats", good("z")), arrayOf("spells", good("a"), good("b"))))
	if !a.Equal(b) {
		t.Errorf("documents differing only in order decode to different sets")
	}
	var keys []string
	for _, e := range a.Entries() {
		keys = append(keys, e.Key().Kind().String()+"/"+e.Key().ID().String())
	}
	if want := []string{"species/m", "feat/z", "spell/a", "spell/b"}; !slices.Equal(keys, want) {
		t.Errorf("order = %v, want %v", keys, want)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-018, FR-024 - a bounded array is checked by
// length first and skips its children, so a huge array costs no per-element
// work or problems.
// Path: unhappy (boundary).
func TestDecodeBoundedArrayLengthFirst(t *testing.T) {
	t.Parallel()
	huge := "[" + strings.Repeat("99,", 100_000) + "99]" // 100001 elements, every one out of range
	_, got := decodeText(t, doc(arrayOf("spells", good("a", `"slots":`+huge))))
	if want := []string{"/spells/0/slots outOfRange"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

// Requirement: doc/plans/plan-phase-2.md P13 and doc/specs/content-schema-and-decoder.md FR-154 - at most MaxProblems problems are reported, then one
// truncation problem (outOfRange at the path ""); a document over MaxDocumentBytes is one outOfRange.
// Path: unhappy (boundary).
func TestDecodeCaps(t *testing.T) {
	t.Parallel()
	many := func(n int) string {
		elems := make([]string, n)
		for i := range elems {
			elems[i] = fmt.Sprintf(`{"id":"e%d","name":"N","level":1}`, i) // one problem: no school
		}
		return doc(arrayOf("spells", elems...))
	}
	t.Run("exactly MaxProblems", func(t *testing.T) {
		t.Parallel()
		_, got := decodeText(t, many(MaxProblems))
		if len(got) != MaxProblems || got[MaxProblems-1] != fmt.Sprintf("/spells/%d/school missingField", MaxProblems-1) {
			t.Errorf("got %d problems, last %q, want %d and no truncation", len(got), got[len(got)-1], MaxProblems)
		}
	})
	t.Run("one more than MaxProblems", func(t *testing.T) {
		t.Parallel()
		_, got := decodeText(t, many(MaxProblems+1))
		if len(got) != MaxProblems+1 || got[MaxProblems] != " outOfRange" || got[MaxProblems-1] != fmt.Sprintf("/spells/%d/school missingField", MaxProblems-1) {
			t.Errorf("got %d problems, last two %q, want the first %d then the truncation problem", len(got), got[len(got)-2:], MaxProblems)
		}
	})
	t.Run("a hundred thousand bad entries", func(t *testing.T) {
		t.Parallel()
		_, got := decodeText(t, many(100_000))
		if len(got) != MaxProblems+1 {
			t.Errorf("got %d problems, want %d", len(got), MaxProblems+1)
		}
	})
	t.Run("many unknown keys", func(t *testing.T) {
		t.Parallel()
		var b strings.Builder
		b.WriteString(`{"schemaVersion":1`)
		for i := range 2 * MaxProblems {
			fmt.Fprintf(&b, `,"k%05d":1`, i)
		}
		b.WriteString("}")
		_, got := decodeText(t, b.String())
		if len(got) != MaxProblems+1 || got[0] != "/k00000 unknownField" || got[MaxProblems] != " outOfRange" {
			t.Errorf("got %d problems (first %q), want %d ending with the truncation problem", len(got), got[0], MaxProblems+1)
		}
	})
	t.Run("document at the size cap", func(t *testing.T) {
		t.Parallel()
		text := `{"schemaVersion":1}` + strings.Repeat(" ", MaxDocumentBytes-len(`{"schemaVersion":1}`))
		if _, got := decodeText(t, text); got != nil {
			t.Errorf("a document of exactly MaxDocumentBytes gave %q, want accepted", got)
		}
	})
	t.Run("document one byte over the size cap", func(t *testing.T) {
		t.Parallel()
		text := `{"schemaVersion":1}` + strings.Repeat(" ", MaxDocumentBytes-len(`{"schemaVersion":1}`)+1)
		if _, got := decodeText(t, text); !slices.Equal(got, []string{" outOfRange"}) {
			t.Errorf("got %q, want one outOfRange at the root", got)
		}
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-018 and doc/plans/plan-phase-2.md section 7 - the
// nesting limit of encoding/json (10000) is a syntax problem, never a crash.
// Path: unhappy (boundary).
func TestDecodeNestingDepth(t *testing.T) {
	t.Parallel()
	nest := func(depth int) string { return strings.Repeat("[", depth) + strings.Repeat("]", depth) }
	if _, got := decodeText(t, nest(10000)); !slices.Equal(got, []string{" wrongType"}) {
		t.Errorf("depth 10000 = %q, want a valid document that is not an object", got)
	}
	if _, got := decodeText(t, nest(10001)); !slices.Equal(got, []string{" syntax"}) {
		t.Errorf("depth 10001 = %q, want syntax", got)
	}
	deep := `{"schemaVersion":1,"x":` + nest(9999) + `}`
	if _, got := decodeText(t, deep); !slices.Equal(got, []string{"/x unknownField"}) {
		t.Errorf("a deep unknown key = %q, want unknownField only", got)
	}
}

// Requirement: doc/plans/plan-phase-2.md section 5.4 - a row that drops an element without a problem, or
// returns an entry with an invalid key, is reported rather than hidden or panicked on.
// Path: unhappy (failure).
func TestDecodeRowBugsAreReported(t *testing.T) {
	t.Parallel()
	dropping := []rootArray{{key: "spells", kind: KindSpell, decode: func(*object) decodedEntry { return decodedEntry{} }}}
	if _, err := decodeDocument([]byte(doc(arrayOf("spells", "{}"))), dropping); !slices.Equal(render(err), []string{"/spells/0 invalidValue"}) {
		t.Errorf("dropped element: %v", render(err))
	}
	badKey := []rootArray{{key: "spells", kind: KindSpell, decode: func(*object) decodedEntry {
		return decodedEntry{entry: widget{key: NewKey(KindSpell, ID{})}}
	}}}
	if _, err := decodeDocument([]byte(doc(arrayOf("spells", "{}"))), badKey); !slices.Equal(render(err), []string{" invalidValue"}) {
		t.Errorf("invalid key: %v", render(err))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-016 - the registration is a function in
// revision-table order; this pins the order for every row that registers.
// Path: happy.
func TestRootArraysAreInRevisionTableOrder(t *testing.T) {
	t.Parallel()
	canonical := []string{
		"species", "backgrounds", "classes", "feats", "spells", "weapons", "armor",
		"tools", "adventuringGear", "mounts", "tackHarnessAndDrawnVehicles", "airborneAndWaterborneVehicles",
		"lifestyleExpenses", "foodDrinkAndLodging", "hirelings", "spellcastingServices", "spellScrollCosts",
		"monsters", "magicItems",
	}
	last := -1
	seen := map[string]bool{}
	for _, a := range rootArrays() {
		i := slices.Index(canonical, a.key)
		if i < 0 || i <= last || seen[a.key] || !a.kind.Valid() || a.decode == nil || a.key == "schemaVersion" {
			t.Errorf("root array %q is unknown, repeated, out of revision-table order or incomplete", a.key)
		}
		seen[a.key], last = true, i
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-073, FR-076 - the decoder does not modify its
// input, and nodes expand on demand from it without copies surviving.
// Path: unhappy (immutability).
func TestDecodeDoesNotModifyInput(t *testing.T) {
	t.Parallel()
	input := []byte(doc(arrayOf("spells", good("a", `"notes":"x\ny"`), good("b"))))
	before := bytes.Clone(input)
	set1, err1 := decodeDocument(input, registry())
	clear(input)
	set2, err2 := decodeDocument(before, registry())
	if err1 != nil || err2 != nil || !set1.Equal(set2) {
		t.Errorf("results differ after the input buffer was cleared: %v, %v", err1, err2)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-021 - integer wholeness is decided on the
// decimal text with a bounded exponent, never on a rounded float64.
// Path: per case.
func TestClassifyNumber(t *testing.T) {
	t.Parallel()
	zeros := strings.Repeat("0", 1000)
	cases := []struct {
		lit   string
		want  int64
		class numberClass
	}{
		{"0", 0, numberWhole},
		{"-0", 0, numberWhole},
		{"0.0", 0, numberWhole},
		{"0e999999999", 0, numberWhole},
		{"0.000e-5", 0, numberWhole},
		{"1", 1, numberWhole},
		{"-1", -1, numberWhole},
		{"2.0", 2, numberWhole},
		{"2." + zeros, 2, numberWhole},
		{"2." + zeros + "1", 0, numberFractional},
		{"20e-1", 2, numberWhole},
		{"100e-2", 1, numberWhole},
		{"120e-2", 0, numberFractional},
		{"0.5e1", 5, numberWhole},
		{"5e-1", 0, numberFractional},
		{"2.5", 0, numberFractional},
		{"1E2", 100, numberWhole},
		{"1e+2", 100, numberWhole},
		{"0.1e1", 1, numberWhole},
		{"1e15", 1_000_000_000_000_000, numberWhole},
		{"1e16", 0, numberTooLarge},
		{"10000000000000000", 0, numberTooLarge},
		{"9007199254740990", 9007199254740990, numberWhole},
		{"9007199254740991", 9007199254740991, numberWhole},
		{"9007199254740992", 0, numberTooLarge},
		{"9007199254740993", 0, numberTooLarge},
		{"-9007199254740991", -9007199254740991, numberWhole},
		{"-9007199254740992", 0, numberTooLarge},
		{"9.007199254740991e15", 9007199254740991, numberWhole},
		{"9.007199254740992e15", 0, numberTooLarge},
		{"90071992547409910e-1", 9007199254740991, numberWhole},
		{"12345678901234567890", 0, numberTooLarge},
		{"1e999999999", 0, numberTooLarge},
		{"-1e999999999", 0, numberTooLarge},
		{"1e-999999999", 0, numberFractional},
		{"1e99999999999999999999", 0, numberTooLarge},
		{"1e15000000000000000000", 0, numberTooLarge}, // an exponent that wraps negative as an int64
		{"1e-15000000000000000000", 0, numberFractional},
		{"18446744073709551616", 0, numberTooLarge}, // 2^64 wraps to 0 as a uint64
		{"18446744073709551621", 0, numberTooLarge}, // 2^64 + 5 wraps to 5
		{"0.12e2", 12, numberWhole},
		{"0.120e3", 120, numberWhole},
		{"0.0012e4", 12, numberWhole},
		{"0.9007199254740991e16", 9007199254740991, numberWhole},
		{"0.9007199254740992e16", 0, numberTooLarge},
		{"1e-99999999999999999999", 0, numberFractional},
		{strings.Repeat("1", 1_000_000), 0, numberTooLarge},
		{"0." + strings.Repeat("0", 1_000_000) + "1", 0, numberFractional},
		{"1" + strings.Repeat("0", 1_000_000) + "e-1000000", 1, numberWhole},
	}
	for _, c := range cases {
		name := c.lit
		if len(name) > 40 {
			name = fmt.Sprintf("%s...(%d bytes)", name[:20], len(name))
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, class := classifyNumber(json.Number(c.lit))
			if class != c.class || (class == numberWhole && got != c.want) {
				t.Errorf("classifyNumber(%q) = (%d, %d), want (%d, %d)", name, got, class, c.want, c.class)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-021 - the decoder reads a number's text the way
// classifyNumber does, through a json.Number.
// Path: happy.
func TestNodeNumber(t *testing.T) {
	t.Parallel()
	n := node("1.50")
	if n.kind() != nodeNumber || n.number().String() != "1.50" {
		t.Errorf("number node = %v %q", n.kind(), n.number())
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-018 - the node layer reads every valid JSON
// shape without error and in order.
// Path: per case.
func TestNodeExpansion(t *testing.T) {
	t.Parallel()
	root, ok := parseDocument([]byte(" \n{\"a\": [1, {\"b\":null}, \"x\\ny\", true, false],\t\"a\":2,\n\"c\":{}} \r\n"))
	if !ok {
		t.Fatal("parseDocument rejected valid JSON")
	}
	ms := root.members()
	var keys []string
	for _, m := range ms {
		keys = append(keys, m.key)
	}
	if !slices.Equal(keys, []string{"a", "a", "c"}) || ms[2].val.kind() != nodeObject || string(ms[1].val) != "2" {
		t.Fatalf("members = %v", keys)
	}
	var kinds []nodeKind
	var texts []string
	for _, el := range ms[0].val.elements() {
		kinds = append(kinds, el.kind())
		if s, ok := el.text(); ok {
			texts = append(texts, s)
		}
	}
	if !slices.Equal(kinds, []nodeKind{nodeNumber, nodeObject, nodeString, nodeBool, nodeBool}) || !slices.Equal(texts, []string{"x\ny"}) {
		t.Errorf("elements = %v %v", kinds, texts)
	}
	count := 0
	for range ms[0].val.elements() {
		count++
		break // stopping early is allowed
	}
	if count != 1 || node(nil).kind() != nodeInvalid || node("null").kind() != nodeNull {
		t.Errorf("early stop or kinds wrong")
	}
	for _, text := range []string{"1", `"s"`, "true", "null", "[1,2]", "[]"} {
		if got := node(text).members(); got != nil {
			t.Errorf("members of %s = %v", text, got)
		}
	}
	for _, text := range []string{"{}", `{"a":1}`, "1", `"s"`, "null", "true"} {
		for i, el := range node(text).elements() {
			t.Errorf("elements of %s yielded %d %s", text, i, el)
		}
	}
	if s, ok := node("1").text(); ok || s != "" {
		t.Errorf("text of a number = %q, %v", s, ok)
	}
	if s, ok := node(`"`).text(); ok || s != "" {
		t.Errorf("text of a lone quote = %q, %v", s, ok)
	}
	if s, ok := node(`""`).text(); !ok || s != "" {
		t.Errorf("text of an empty string = %q, %v, want empty and ok", s, ok)
	}
}

// Requirement: doc/plans/plan-phase-2.md P14 and doc/specs/content-schema-and-decoder.md FR-156 - a string literal is decoded exactly as encoding/json would,
// and reported when json would have replaced bytes.
// Path: per case.
func TestUnquote(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{`""`, "", true},
		{`"abc"`, "abc", true},
		{`"a\"b\\c\/d\be\ff\ng\rh\ti"`, "a\"b\\c/d\be\ff\ng\rh\ti", true},
		{`"\u0041\u00e9\u20ac"`, "A\u00e9\u20ac", true},
		{`"\uD83D\uDE00"`, "\U0001F600", true},
		{`"\uDBFF\uDFFF"`, "\U0010FFFF", true}, // a high surrogate in the upper part of its range still pairs
		{`"\ud83d\ude00x"`, "\U0001F600x", true},
		{`"\uFFFD"`, "\uFFFD", true},
		{"\"\uFFFD\"", "\uFFFD", true},
		{`"\ud800"`, "\uFFFD", false},
		{`"\ud800x"`, "\uFFFDx", false},
		{`"\udc00"`, "\uFFFD", false},
		{`"\udc00\ud800"`, "\uFFFD\uFFFD", false},
		{`"\ud800\u0041"`, "\uFFFDA", false},
		{`"\ud800\ud800"`, "\uFFFD\uFFFD", false},
		{"\"a\xffb\"", "a\uFFFDb", false},
		{"\"\\n\xff\"", "\n\uFFFD", false},
		{"\"\\n\uFFFD\"", "\n\uFFFD", true}, // a real U+FFFD next to an escape is valid text
		{`"\u00AF\u00af\u00A9\u00Fa"`, "\u00AF\u00af\u00A9\u00FA", true},
		{`"\ud83dxuDE00"`, "\uFFFDxuDE00", false}, // not a pair: the next characters are plain text
		{`"\ud83d\\uDE00"`, "\uFFFD\\uDE00", false},
		{`"\ud83d\\DE00"`, "\uFFFD\\DE00", false}, // after the high surrogate a backslash, then four hex digits that would be a low surrogate
		{`"\ud83d\uDE00\ud83d\uDE00"`, "\U0001F600\U0001F600", true},
		{"\"\xe2\x82\"", "\uFFFD\uFFFD", false},
	}
	for _, c := range cases {
		got, ok := unquote([]byte(c.raw))
		if got != c.want || ok != c.ok {
			t.Errorf("unquote(%q) = (%q, %v), want (%q, %v)", c.raw, got, ok, c.want, c.ok)
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-014, FR-157 - the path of a problem is escaped and length
// bounded in Error(), and Path() keeps the exact JSON Pointer.
// Path: unhappy (boundary).
func TestProblemErrorIsEscapedAndBounded(t *testing.T) {
	t.Parallel()
	hostile := NewProblem("/a\nb\u202e\"c\x00/\u00e9", ProblemUnknownField)
	text := hostile.Error()
	for _, r := range text {
		if r < 0x20 || r > 0x7e {
			t.Errorf("Error() holds the unescaped rune %U: %q", r, text)
		}
	}
	if hostile.Path() != "/a\nb\u202e\"c\x00/\u00e9" || !strings.Contains(text, `\n`) || !strings.Contains(text, `\u202e`) {
		t.Errorf("Path() or Error() wrong: %q", text)
	}
	long := NewProblem("/"+strings.Repeat("x", 10_000), ProblemUnknownField)
	if got, want := long.Error(), `content: unknownField at "/`+strings.Repeat("x", MaxRenderedPath-1)+`"...`; got != want {
		t.Errorf("a long path renders %d bytes (%q...), want exactly MaxRenderedPath bytes of path marked with ...", len(got), got[:30])
	}
	if len(long.Path()) != 10_001 {
		t.Errorf("Path() was cut")
	}
	cut := NewProblem(strings.Repeat("x", MaxRenderedPath-1)+"\u00e9\u00e9", ProblemWrongType) // a multibyte rune straddles the limit
	if got, want := cut.Error(), `content: wrongType at "`+strings.Repeat("x", MaxRenderedPath-1)+`"...`; got != want {
		t.Errorf("a cut through a rune renders %q, want the cut moved back to the character boundary", got)
	}
	exact := NewProblem(strings.Repeat("x", MaxRenderedPath), ProblemWrongType)
	if strings.HasSuffix(exact.Error(), "...") {
		t.Errorf("a path of exactly MaxRenderedPath bytes was marked cut")
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-094 - each rule exists once: the decoder files use the
// exported limits and predicates and repeat none of their literals.
// Path: unhappy (rule interaction).
func TestDecoderFilesRepeatNoLimitLiteral(t *testing.T) {
	t.Parallel()
	banned := map[string]string{
		"64": "MaxIDLength", "364": "SRDLastPage", "2147483647": "SourceMaxPage", "16777216": "MaxDocumentBytes",
		"9007199254740991": "MaxSafeInteger", `"SRD 5.2.1"`: "SRDDocument",
	}
	files, err := filepath.Glob("decode*.go")
	if err != nil || len(files) < 4 {
		t.Fatalf("glob decode*.go = %v, %v", files, err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok {
				if use, bad := banned[lit.Value]; bad {
					t.Errorf("%s: literal %s repeats the limit %s; use the constant", fset.Position(lit.Pos()), lit.Value, use)
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-004 and FR-154 - a supported version lets decoding go on and
// return the entries; once MaxProblems are held the loop stops calling the row (work past the cap cannot be reported).
// Path: happy (version) and unhappy (boundary, cap).
func TestDecodeDocumentReturnsRowsAndStopsAtCap(t *testing.T) {
	t.Parallel()
	ok := []rootArray{{key: "spells", kind: KindSpell, decode: func(*object) decodedEntry {
		return decodedEntry{entry: widget{key: NewKey(KindSpell, MustID("a"))}, id: MustID("a"), idOK: true}
	}}}
	set, err := decodeDocument([]byte(doc(arrayOf("spells", "{}"))), ok)
	if err != nil || len(set.Entries()) != 1 {
		t.Errorf("supported version: entries = %d, err = %v, want 1 entry", len(set.Entries()), err)
	}
	calls := 0
	dropping := []rootArray{{key: "spells", kind: KindSpell, decode: func(*object) decodedEntry { calls++; return decodedEntry{} }}}
	elems := make([]string, MaxProblems+50)
	for i := range elems {
		elems[i] = "{}"
	}
	if _, err := decodeDocument([]byte(doc(arrayOf("spells", elems...))), dropping); len(Problems(err)) != MaxProblems+1 {
		t.Errorf("problems = %d, want %d", len(Problems(err)), MaxProblems+1)
	}
	if calls != MaxProblems {
		t.Errorf("row called %d times, want %d: decoding must stop at the cap", calls, MaxProblems)
	}
}
