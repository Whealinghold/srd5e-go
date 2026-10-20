package content

import "fmt"

// This file lets the external agreement test (agreement_test.go, package
// content_test) reach the field rules of the decoder, which no registered
// kind exposes yet. It exports test-only functions, so it compiles into the
// test binary only.

// probe decodes {"<key>": value} and returns the object and the value node.
func probe(d *decoder, key, value string) (*object, node, string) {
	root, ok := parseDocument([]byte(`{"` + key + `":` + value + `}`))
	if !ok {
		d.probs.add("", ProblemSyntax)
		return nil, nil, ""
	}
	o, _ := d.object(root, "")
	n, p, _ := o.field(key, true)
	return o, n, p
}

// ProbeField decodes the JSON value text with the named field rule and
// returns the problems found at or below it. The rules are id, name, text and
// integer:LO:HI.
func ProbeField(rule, value string) []Problem {
	d := &decoder{}
	_, n, p := probe(d, "v", value)
	switch rule {
	case "id":
		d.id(n, p)
	case "name":
		d.name(n, p)
	case "text":
		d.text(n, p)
	default:
		var lo, hi int64
		if _, err := fmt.Sscanf(rule, "integer:%d:%d", &lo, &hi); err != nil {
			return []Problem{{path: rule, kind: ProblemSyntax}}
		}
		d.integer(n, p, lo, hi)
	}
	return d.probs.items
}

// ProbeSource decodes the JSON value text as the "source" field of an entry
// and returns the problems found, and whether a citation was built.
func ProbeSource(value string) ([]Problem, SourceCitation, bool) {
	d := &decoder{}
	o, _, _ := probe(d, "source", value)
	c, ok := o.source()
	o.finish()
	return d.probs.items, c, ok
}

// ProbeEnum decodes the JSON value text as a member of the vocabulary parse
// accepts, like an enum field does.
func ProbeEnum[T any](value string, parse func(string) (T, error)) []Problem {
	d := &decoder{}
	_, n, p := probe(d, "v", value)
	enumField(d, n, p, parse)
	return d.probs.items
}
