package content

import (
	"slices"
	"strings"
	"testing"
)

// These tests call the field helpers directly so that both the value and the
// ok flag are asserted: the flags decide whether a row builds its entry, and no
// registered kind exercises them yet.

// valueNode parses text as a JSON value and returns its node.
func valueNode(t *testing.T, text string) node {
	t.Helper()
	n, ok := parseDocument([]byte(text))
	if !ok {
		t.Fatalf("%q is not JSON", text)
	}
	return n
}

// kinds renders the problems as "path kind" strings.
func kinds(ps []Problem) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Path()+" "+p.Kind().String())
	}
	return out
}

// Requirement: doc/plans/plan-phase-2.md P13 and doc/specs/content-schema-and-decoder.md FR-154 - the problem list counts every problem, keeps the first
// MaxProblems, remembers truncation, and ends with the truncation problem only then.
// Path: unhappy (boundary).
func TestProblemList(t *testing.T) {
	t.Parallel()
	var l problemList
	if l.err() != nil || l.full() || l.truncated {
		t.Fatalf("a new list must be empty, not full and not truncated")
	}
	for range MaxProblems - 1 {
		l.add("/a", ProblemWrongType)
	}
	if l.full() || l.truncated || l.count != MaxProblems-1 || len(l.items) != MaxProblems-1 {
		t.Errorf("below the cap: full=%v truncated=%v count=%d items=%d", l.full(), l.truncated, l.count, len(l.items))
	}
	l.add("/b", ProblemDuplicate)
	if l.truncated || l.count != MaxProblems || len(l.items) != MaxProblems {
		t.Errorf("at the cap: truncated=%v count=%d items=%d, want no truncation until one is dropped", l.truncated, l.count, len(l.items))
	}
	if got := Problems(l.err()); len(got) != MaxProblems || got[MaxProblems-1] != NewProblem("/b", ProblemDuplicate) {
		t.Errorf("err() at the cap holds %d problems, last %v", len(got), got[len(got)-1])
	}
	if !l.full() || !l.truncated {
		t.Errorf("full() at the cap = false or did not mark the list truncated")
	}
	l.add("/c", ProblemSyntax)
	if l.count != MaxProblems+1 || len(l.items) != MaxProblems {
		t.Errorf("past the cap: count=%d items=%d, want the extra counted and dropped", l.count, len(l.items))
	}
	got := Problems(l.err())
	if len(got) != MaxProblems+1 || got[MaxProblems] != NewProblem("", ProblemOutOfRange) || got[0] != NewProblem("/a", ProblemWrongType) {
		t.Errorf("truncated err() holds %d problems, last %v, want the cap then the truncation problem", len(got), got[len(got)-1])
	}
	var dropped problemList
	for range MaxProblems + 1 {
		dropped.add("/x", ProblemWrongType)
	}
	if !dropped.truncated {
		t.Errorf("a dropped problem must mark the list truncated")
	}
	var one problemList
	one.add("/x", ProblemSyntax)
	if got := Problems(one.err()); len(got) != 1 {
		t.Errorf("one problem gave %d, want no truncation problem", len(got))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-011, FR-013, FR-020 - object expands an object and
// reports anything else, null included, as wrongType without examining it.
// Path: per case.
func TestObjectExpansion(t *testing.T) {
	t.Parallel()
	for _, text := range []string{`[]`, `null`, `1`, `"x"`, `true`} {
		d := &decoder{}
		o, ok := d.object(valueNode(t, text), "/p")
		if o != nil || ok || !slices.Equal(kinds(d.probs.items), []string{"/p wrongType"}) {
			t.Errorf("object(%s) = (%v, %v) %v, want nil, false and one wrongType", text, o, ok, kinds(d.probs.items))
		}
	}
	d := &decoder{}
	o, ok := d.object(valueNode(t, `{"a":1,"b":2,"a":3}`), "/p")
	if !ok || len(o.members) != 3 || !slices.Equal(o.dups, []int{2}) || o.index["a"] != 2 || o.index["b"] != 1 {
		t.Fatalf("object = %+v, want three members, the repeat at 2 and the last value indexed", o)
	}
	n, p, present := o.field("a", true)
	if !present || p != "/p/a" || string(n) != "3" {
		t.Errorf("field(a) = (%s, %s, %v), want the last value", n, p, present)
	}
	if _, p, present := o.field("zz", false); present || p != "/p/zz" || len(d.probs.items) != 0 {
		t.Errorf("an absent optional field = (%s, %v) with problems %v", p, present, kinds(d.probs.items))
	}
	if _, _, present := o.field("yy/~", true); present || !slices.Equal(kinds(d.probs.items), []string{"/p/yy~1~0 missingField"}) {
		t.Errorf("an absent required field: %v", kinds(d.probs.items))
	}
	o.finish()
	if got := kinds(d.probs.items); !slices.Equal(got, []string{"/p/yy~1~0 missingField", "/p/a duplicate", "/p/b unknownField"}) {
		t.Errorf("finish = %v, want the duplicate then the unknown key (a, zz and yy/~ were read)", got)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-021, FR-025, FR-027, FR-028, FR-029, FR-030 - the string
// helpers return the text with ok, or an empty value with a single problem.
// Path: per case.
func TestStringHelpers(t *testing.T) {
	t.Parallel()
	type check func(d *decoder, n node) (string, bool)
	asID := func(d *decoder, n node) (string, bool) { id, ok := d.id(n, "/v"); return id.String(), ok }
	cases := []struct {
		name  string
		f     check
		json  string
		want  string
		ok    bool
		probs []string
	}{
		{"str", func(d *decoder, n node) (string, bool) { return d.str(n, "/v") }, `"a\nb"`, "a\nb", true, nil},
		{"str of a number", func(d *decoder, n node) (string, bool) { return d.str(n, "/v") }, `1`, "", false, []string{"/v wrongType"}},
		{"str of null", func(d *decoder, n node) (string, bool) { return d.str(n, "/v") }, `null`, "", false, []string{"/v wrongType"}},
		{"str with a lone surrogate", func(d *decoder, n node) (string, bool) { return d.str(n, "/v") }, `"\ud800"`, "", false, []string{"/v invalidValue"}},
		{"name", func(d *decoder, n node) (string, bool) { return d.name(n, "/v") }, `"Acid Arrow"`, "Acid Arrow", true, nil},
		{"name with a space around", func(d *decoder, n node) (string, bool) { return d.name(n, "/v") }, `" x"`, "", false, []string{"/v invalidValue"}},
		{"name of an array", func(d *decoder, n node) (string, bool) { return d.name(n, "/v") }, `[]`, "", false, []string{"/v wrongType"}},
		{"text", func(d *decoder, n node) (string, bool) { return d.text(n, "/v") }, `" a\nb "`, " a\nb ", true, nil},
		{"blank text", func(d *decoder, n node) (string, bool) { return d.text(n, "/v") }, `" "`, "", false, []string{"/v invalidValue"}},
		{"line", func(d *decoder, n node) (string, bool) { return d.line(n, "/v") }, `" SRD "`, " SRD ", true, nil},
		{"line with a line feed", func(d *decoder, n node) (string, bool) { return d.line(n, "/v") }, `"a\nb"`, "", false, []string{"/v invalidValue"}},
		{"blank line", func(d *decoder, n node) (string, bool) { return d.line(n, "/v") }, `"\t "`, "", false, []string{"/v invalidValue"}},
		{"id", asID, `"a_1"`, "a_1", true, nil},
		{"id with a capital", asID, `"A"`, "", false, []string{"/v invalidValue"}},
		{"id of a number", asID, `1`, "", false, []string{"/v wrongType"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d := &decoder{}
			got, ok := c.f(d, valueNode(t, c.json))
			if got != c.want || ok != c.ok || !slices.Equal(kinds(d.probs.items), c.probs) {
				t.Errorf("= (%q, %v) %v, want (%q, %v) %v", got, ok, kinds(d.probs.items), c.want, c.ok, c.probs)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-021, FR-023 - integer returns the value with ok when
// whole and in range, and zero with a single problem otherwise.
// Path: per case.
func TestIntegerHelper(t *testing.T) {
	t.Parallel()
	cases := []struct {
		json  string
		lo    int64
		hi    int64
		want  int64
		ok    bool
		probs []string
	}{
		{`5`, 1, 20, 5, true, nil},
		{`5.0`, 1, 20, 5, true, nil},
		{`1`, 1, 20, 1, true, nil},
		{`20`, 1, 20, 20, true, nil},
		{`-3`, -5, 5, -3, true, nil},
		{`0`, 1, 20, 0, false, []string{"/v outOfRange"}},
		{`21`, 1, 20, 0, false, []string{"/v outOfRange"}},
		{`2.5`, 1, 20, 0, false, []string{"/v wrongType"}},
		{`"5"`, 1, 20, 0, false, []string{"/v wrongType"}},
		{`null`, 1, 20, 0, false, []string{"/v wrongType"}},
		{`1e999999999`, 1, 20, 0, false, []string{"/v outOfRange"}},
		{`9007199254740991`, 1, MaxSafeInteger, MaxSafeInteger, true, nil},
		{`9007199254740992`, 1, MaxSafeInteger, 0, false, []string{"/v outOfRange"}},
	}
	for _, c := range cases {
		d := &decoder{}
		got, ok := d.integer(valueNode(t, c.json), "/v", c.lo, c.hi)
		if got != c.want || ok != c.ok || !slices.Equal(kinds(d.probs.items), c.probs) {
			t.Errorf("integer(%s, %d..%d) = (%d, %v) %v, want (%d, %v) %v", c.json, c.lo, c.hi, got, ok, kinds(d.probs.items), c.want, c.ok, c.probs)
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-013, FR-024 - arrayLen returns the count with ok when
// within bounds, and zero with one outOfRange (children skipped) otherwise; it stops counting past the bound.
// Path: per case.
func TestArrayLenHelper(t *testing.T) {
	t.Parallel()
	cases := []struct {
		json   string
		lo, hi int
		want   int
		ok     bool
		probs  []string
	}{
		{`[]`, 0, 3, 0, true, nil},
		{`[1]`, 1, 3, 1, true, nil},
		{`[1,2,3]`, 1, 3, 3, true, nil},
		{`[]`, 1, 3, 0, false, []string{"/v outOfRange"}},
		{`[1,2,3,4]`, 1, 3, 0, false, []string{"/v outOfRange"}},
		{`{}`, 0, 3, 0, false, []string{"/v wrongType"}},
		{`null`, 0, 3, 0, false, []string{"/v wrongType"}},
		{`"abc"`, 0, 3, 0, false, []string{"/v wrongType"}},
	}
	for _, c := range cases {
		d := &decoder{}
		got, ok := d.arrayLen(valueNode(t, c.json), "/v", c.lo, c.hi)
		if got != c.want || ok != c.ok || !slices.Equal(kinds(d.probs.items), c.probs) {
			t.Errorf("arrayLen(%s, %d..%d) = (%d, %v) %v, want (%d, %v) %v", c.json, c.lo, c.hi, got, ok, kinds(d.probs.items), c.want, c.ok, c.probs)
		}
	}
	// A hundred thousand elements against a bound of three is one problem, and
	// the count stops at four (the elements are not all visited).
	d := &decoder{}
	if _, ok := d.arrayLen(valueNode(t, "["+strings.Repeat("0,", 100_000)+"0]"), "/v", 0, 3); ok || len(d.probs.items) != 1 {
		t.Errorf("a huge array: ok=%v problems=%v", ok, kinds(d.probs.items))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-025, FR-026 - enumField and enumSet return the
// parsed members with ok, report each bad element once, and flag a repeat at the later index.
// Path: per case.
func TestEnumHelpers(t *testing.T) {
	t.Parallel()
	d := &decoder{}
	if v, ok := enumField(d, valueNode(t, `"Acid"`), "/v", ParseDamageType); !ok || v != DamageAcid || len(d.probs.items) != 0 {
		t.Errorf("enumField(Acid) = (%v, %v) %v", v, ok, kinds(d.probs.items))
	}
	for json, want := range map[string]string{`"acid"`: "/v invalidValue", `1`: "/v wrongType", `null`: "/v wrongType"} {
		d := &decoder{}
		if v, ok := enumField(d, valueNode(t, json), "/v", ParseDamageType); ok || v != 0 || !slices.Equal(kinds(d.probs.items), []string{want}) {
			t.Errorf("enumField(%s) = (%v, %v) %v, want zero, false, %s", json, v, ok, kinds(d.probs.items), want)
		}
	}
	cases := []struct {
		json   string
		lo, hi int
		want   []DamageType
		ok     bool
		probs  []string
	}{
		{`["Acid","Cold"]`, 1, 3, []DamageType{DamageAcid, DamageCold}, true, nil},
		{`["Acid"]`, 1, 3, []DamageType{DamageAcid}, true, nil},
		{`["Acid","Cold","Acid"]`, 1, 3, []DamageType{DamageAcid, DamageCold}, false, []string{"/v/2 duplicate"}},
		{`["Acid","Acid","Acid"]`, 1, 3, []DamageType{DamageAcid}, false, []string{"/v/1 duplicate", "/v/2 duplicate"}},
		{`[1,"nope","Cold"]`, 1, 3, []DamageType{DamageCold}, false, []string{"/v/0 wrongType", "/v/1 invalidValue"}},
		{`[]`, 1, 3, nil, false, []string{"/v outOfRange"}},
		{`["Acid","Cold","Fire","Force"]`, 1, 3, nil, false, []string{"/v outOfRange"}},
		{`"Acid"`, 1, 3, nil, false, []string{"/v wrongType"}},
	}
	for _, c := range cases {
		d := &decoder{}
		got, ok := enumSet(d, valueNode(t, c.json), "/v", c.lo, c.hi, ParseDamageType)
		if !slices.Equal(got, c.want) || ok != c.ok || !slices.Equal(kinds(d.probs.items), c.probs) {
			t.Errorf("enumSet(%s) = (%v, %v) %v, want (%v, %v) %v", c.json, got, ok, kinds(d.probs.items), c.want, c.ok, c.probs)
		}
	}
	// A full problem list stops the walk early.
	full := &decoder{}
	for range MaxProblems {
		full.probs.add("/x", ProblemSyntax)
	}
	if got, ok := enumSet(full, valueNode(t, `["Acid","Cold"]`), "/v", 0, 3, ParseDamageType); got != nil || ok || !full.probs.truncated {
		t.Errorf("enumSet on a full list = (%v, %v) truncated=%v, want it to stop", got, ok, full.probs.truncated)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-034, FR-115 - source returns the citation with ok when
// absent (zero) or valid, and ok false with the field problems otherwise.
// Path: per case.
func TestSourceHelper(t *testing.T) {
	t.Parallel()
	cites := func(document, section string, page int) SourceCitation {
		return MustSourceCitation(document, section, page)
	}
	cases := []struct {
		name  string
		json  string // the entry object
		want  SourceCitation
		ok    bool
		probs []string
	}{
		{"absent", `{}`, SourceCitation{}, true, nil},
		{"valid", `{"source":{"document":"SRD 5.2.1","section":"S","page":16}}`, cites("SRD 5.2.1", "S", 16), true, nil},
		{"no document", `{"source":{"section":"S","page":1}}`, SourceCitation{}, false, []string{"/source/document missingField"}},
		{"no section", `{"source":{"document":"d","page":1}}`, SourceCitation{}, false, []string{"/source/section missingField"}},
		{"no page", `{"source":{"document":"d","section":"S"}}`, SourceCitation{}, false, []string{"/source/page missingField"}},
		{"bad document", `{"source":{"document":" ","section":"S","page":1}}`, SourceCitation{}, false, []string{"/source/document invalidValue"}},
		{"bad section", `{"source":{"document":"d","section":" ","page":1}}`, SourceCitation{}, false, []string{"/source/section invalidValue"}},
		{"bad page", `{"source":{"document":"d","section":"S","page":0}}`, SourceCitation{}, false, []string{"/source/page outOfRange"}},
		{"not an object", `{"source":1}`, SourceCitation{}, false, []string{"/source wrongType"}},
		{"unknown key still builds the citation", `{"source":{"document":"d","section":"S","page":1,"url":"x"}}`, cites("d", "S", 1), true, []string{"/source/url unknownField"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d := &decoder{}
			o, _ := d.object(valueNode(t, c.json), "")
			got, ok := o.source()
			o.known["source"] = struct{}{}
			if got != c.want || ok != c.ok {
				t.Errorf("source() = (%v, %v), want (%v, %v)", got, ok, c.want, c.ok)
			}
			// the unknown-key problem comes from the source object's own finish
			if !slices.Equal(kinds(d.probs.items), c.probs) {
				t.Errorf("problems = %v, want %v", kinds(d.probs.items), c.probs)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-016, FR-095, FR-098 - header reads id, name and source
// in that order; the id counts for the duplicate check even when the rest has problems.
// Path: per case.
func TestHeaderHelper(t *testing.T) {
	t.Parallel()
	good := MustSourceCitation("d", "s", 3)
	cases := []struct {
		name  string
		json  string
		want  entryHeader
		probs []string
	}{
		{"all present", `{"id":"a","name":"N","source":{"document":"d","section":"s","page":3}}`, entryHeader{id: MustID("a"), idOK: true, name: "N", source: good, ok: true}, nil},
		{"no source", `{"id":"a","name":"N"}`, entryHeader{id: MustID("a"), idOK: true, name: "N", ok: true}, nil},
		{"bad name keeps the id", `{"id":"a","name":" N"}`, entryHeader{id: MustID("a"), idOK: true}, []string{"/name invalidValue"}},
		{"bad source keeps the id", `{"id":"a","name":"N","source":{}}`, entryHeader{id: MustID("a"), idOK: true, name: "N"}, []string{"/source/document missingField", "/source/section missingField", "/source/page missingField"}},
		{"bad id", `{"id":"A","name":"N"}`, entryHeader{name: "N"}, []string{"/id invalidValue"}},
		{"nothing", `{}`, entryHeader{}, []string{"/id missingField", "/name missingField"}},
	}
	for _, c := range cases {
		d := &decoder{}
		o, _ := d.object(valueNode(t, c.json), "")
		if got := o.header(); got != c.want || !slices.Equal(kinds(d.probs.items), c.probs) {
			t.Errorf("%s: header = %+v %v, want %+v %v", c.name, got, kinds(d.probs.items), c.want, c.probs)
		}
	}
}
