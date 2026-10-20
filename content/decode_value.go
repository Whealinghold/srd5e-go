package content

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

// problemList collects the problems of one decode, to the cap MaxProblems.
type problemList struct {
	items     []Problem
	count     int // problems added, dropped ones included
	truncated bool
}

// add records a problem. Past MaxProblems it records nothing and remembers
// that the list is incomplete.
func (l *problemList) add(path string, k ProblemKind) {
	l.count++
	if len(l.items) >= MaxProblems { // equivalent mutant: the list never holds more than MaxProblems, so >= and == agree
		l.truncated = true
		return
	}
	l.items = append(l.items, Problem{path: path, kind: k})
}

// full reports whether the cap is reached, in which case the caller stops
// decoding (work past the cap cannot be reported) and the list is marked
// incomplete.
func (l *problemList) full() bool {
	if len(l.items) < MaxProblems { // equivalent mutant: the list never holds more than MaxProblems, so < and != agree
		return false
	}
	l.truncated = true
	return true
}

// err returns the joined problems, with the truncation problem last when the
// list is incomplete, or nil when there are none.
func (l *problemList) err() error {
	var errs []error // errors.Join of nothing is nil
	for _, p := range l.items {
		errs = append(errs, p)
	}
	if l.truncated {
		errs = append(errs, Problem{kind: ProblemOutOfRange})
	}
	return errors.Join(errs...)
}

// decoder is the state of one decode: the problems found so far.
type decoder struct{ probs problemList }

// escapePointer escapes one JSON Pointer segment (RFC 6901): "~" as "~0" and
// "/" as "~1".
func escapePointer(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

// keyPath returns the path of the member key of the object at path.
func keyPath(path, key string) string { return path + "/" + escapePointer(key) }

// indexPath returns the path of element i of the array at path.
func indexPath(path string, i int) string { return path + "/" + strconv.Itoa(i) }

// object is an expanded JSON object being decoded by a kind row. A row reads
// its fields in the order its spec declares (the order problems are
// reported, schema FR-016) and the framework calls finish afterwards.
type object struct {
	d       *decoder
	path    string
	members []member
	index   map[string]int // key to the last member with it
	dups    []int          // members that repeat an earlier key, in order
	known   map[string]struct{}
}

// object expands n as an object at path. A value of another type, null
// included, is wrongType and its children are not examined (FR-011, FR-013,
// FR-020).
func (d *decoder) object(n node, path string) (*object, bool) {
	if n.kind() != nodeObject {
		d.probs.add(path, ProblemWrongType)
		return nil, false
	}
	o := &object{d: d, path: path, members: n.members(), known: map[string]struct{}{}}
	o.index = make(map[string]int, len(o.members))
	for i, m := range o.members {
		if _, repeated := o.index[m.key]; repeated {
			o.dups = append(o.dups, i)
		}
		o.index[m.key] = i // the last wins, as in encoding/json
	}
	return o, true
}

// field returns the value of key and its path, and marks the key known so
// that finish does not report it. An absent key is missingField when required
// (FR-019); the result then has ok false.
func (o *object) field(key string, required bool) (node, string, bool) {
	o.known[key] = struct{}{}
	p := keyPath(o.path, key)
	i, present := o.index[key]
	if !present {
		if required {
			o.d.probs.add(p, ProblemMissingField)
		}
		return nil, p, false
	}
	return o.members[i].val, p, true
}

// finish reports, after the fields of the object, each repeated key at its
// later occurrence as duplicate (plan item P14), then each key no field read
// as unknownField (FR-022), once each, in ordinal order.
func (o *object) finish() {
	for _, i := range o.dups {
		o.d.probs.add(keyPath(o.path, o.members[i].key), ProblemDuplicate)
	}
	var unknown []string
	seen := map[string]struct{}{}
	for _, m := range o.members {
		if _, ok := o.known[m.key]; ok {
			continue
		}
		if _, ok := seen[m.key]; !ok {
			seen[m.key] = struct{}{}
			unknown = append(unknown, m.key)
		}
	}
	slices.Sort(unknown)
	for _, k := range unknown {
		o.d.probs.add(keyPath(o.path, k), ProblemUnknownField)
	}
}

// str reads a string value. Another JSON type, null included, is wrongType; a
// literal with invalid UTF-8 or an unpaired surrogate escape is invalidValue,
// so the text never differs from the document's (FR-029).
func (d *decoder) str(n node, path string) (string, bool) {
	if n.kind() != nodeString {
		d.probs.add(path, ProblemWrongType)
		return "", false
	}
	s, ok := n.text()
	if !ok {
		d.probs.add(path, ProblemInvalidValue)
		return "", false
	}
	return s, true
}

// checked reads a string and applies valid; a failure is invalidValue.
func (d *decoder) checked(n node, path string, valid func(string) bool) (string, bool) {
	s, ok := d.str(n, path)
	if !ok {
		return "", false
	}
	if !valid(s) {
		d.probs.add(path, ProblemInvalidValue)
		return "", false
	}
	return s, true
}

// name reads a name (FR-027).
func (d *decoder) name(n node, path string) (string, bool) { return d.checked(n, path, validName) }

// text reads a text (FR-028).
func (d *decoder) text(n node, path string) (string, bool) { return d.checked(n, path, validText) }

// line reads a single-line, non-blank string (a source document or section).
func (d *decoder) line(n node, path string) (string, bool) {
	return d.checked(n, path, validLine)
}

// id reads an id (FR-030) with the one rule of ParseID.
func (d *decoder) id(n node, path string) (ID, bool) {
	s, ok := d.str(n, path)
	if !ok {
		return ID{}, false
	}
	id, err := ParseID(s)
	if err != nil {
		d.probs.add(path, ProblemInvalidValue)
		return ID{}, false
	}
	return id, true
}

// integer reads an integer in [lo, hi]. A whole number is accepted whatever
// its spelling (2, 2.0, 20e-1); a fraction, a string, a boolean and null are
// wrongType; a whole number beyond MaxSafeInteger, or outside the range, is
// outOfRange (FR-021, FR-023).
func (d *decoder) integer(n node, path string, lo, hi int64) (int64, bool) {
	if n.kind() != nodeNumber {
		d.probs.add(path, ProblemWrongType)
		return 0, false
	}
	v, class := classifyNumber(n.number())
	switch {
	case class == numberFractional:
		d.probs.add(path, ProblemWrongType)
	case class == numberTooLarge || v < lo || v > hi:
		d.probs.add(path, ProblemOutOfRange)
	default:
		return v, true
	}
	return 0, false
}

// arrayLen checks that n is an array holding between lo and hi elements and
// returns the count. A value of another type is wrongType; a length out of
// bounds is one outOfRange at the array, and the children are never examined
// (FR-013, FR-024). The count stops past hi, so a huge array costs no more
// than hi elements.
func (d *decoder) arrayLen(n node, path string, lo, hi int) (int, bool) {
	if n.kind() != nodeArray {
		d.probs.add(path, ProblemWrongType)
		return 0, false
	}
	count := 0
	for range n.elements() {
		count++
		if count > hi {
			break
		}
	}
	if count < lo || count > hi {
		d.probs.add(path, ProblemOutOfRange)
		return 0, false
	}
	return count, true
}

// enumField reads a string that must be a member of a closed vocabulary,
// exactly and case sensitively; anything else is invalidValue (FR-025).
func enumField[T any](d *decoder, n node, path string, parse func(string) (T, error)) (T, bool) {
	var zero T
	s, ok := d.str(n, path)
	if !ok {
		return zero, false
	}
	v, err := parse(s)
	if err != nil {
		d.probs.add(path, ProblemInvalidValue)
		return zero, false
	}
	return v, true
}

// enumSet reads an array of vocabulary members that is a set between lo and
// hi long. A repeat is duplicate at the index of the later occurrence
// (FR-026). The members come back only when no element had a problem.
func enumSet[T comparable](d *decoder, n node, path string, lo, hi int, parse func(string) (T, error)) ([]T, bool) {
	count, ok := d.arrayLen(n, path, lo, hi)
	if !ok {
		return nil, false
	}
	out := make([]T, 0, count)
	seen := make(map[T]struct{}, count)
	clean := true
	for i, el := range n.elements() {
		if d.probs.full() {
			return nil, false
		}
		p := indexPath(path, i)
		v, ok := enumField(d, el, p, parse)
		if !ok {
			clean = false
			continue
		}
		if _, repeated := seen[v]; repeated {
			d.probs.add(p, ProblemDuplicate)
			clean = false
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out, clean
}

// source reads the optional "source" field (FR-034): absent is the zero
// citation and ok; present, it needs document, section and page, each by its
// own rule, and decodes to a SourceCitation (FR-115).
func (o *object) source() (SourceCitation, bool) {
	n, p, present := o.field("source", false)
	if !present {
		return SourceCitation{}, true
	}
	so, ok := o.d.object(n, p)
	if !ok {
		return SourceCitation{}, false
	}
	defer so.finish()
	// A missing or bad part leaves its zero value, which NewSourceCitation
	// rejects (blank document or section, page below the minimum), so the
	// citation is built only when every part decoded cleanly.
	var document, section string
	var page int64
	if n, p, ok := so.field("document", true); ok {
		document, _ = o.d.line(n, p)
	}
	if n, p, ok := so.field("section", true); ok {
		section, _ = o.d.line(n, p)
	}
	if n, p, ok := so.field("page", true); ok {
		page, _ = o.d.integer(n, p, SourceMinPage, SourceMaxPage)
	}
	c, err := NewSourceCitation(document, section, int(page))
	return c, err == nil
}

// entryHeader is the start every entry shares: id, name and source? (schema
// section 5), read in that order.
type entryHeader struct {
	id     ID
	idOK   bool
	name   string
	source SourceCitation
	ok     bool // every part decoded with no problem
}

// header reads the id, name and source of an entry. idOK is set even when
// the name or source has a problem, because an entry with a valid id takes
// part in the duplicate check (FR-095, FR-098).
func (o *object) header() entryHeader {
	var h entryHeader
	nameOK := false
	if n, p, ok := o.field("id", true); ok {
		h.id, h.idOK = o.d.id(n, p)
	}
	if n, p, ok := o.field("name", true); ok {
		h.name, nameOK = o.d.name(n, p)
	}
	var sourceOK bool
	h.source, sourceOK = o.source()
	h.ok = h.idOK && nameOK && sourceOK
	return h
}

// boolean reads a JSON boolean. Another JSON type, null included, is
// wrongType (FR-021).
func (d *decoder) boolean(n node, path string) (bool, bool) {
	if n.kind() != nodeBool {
		d.probs.add(path, ProblemWrongType)
		return false, false
	}
	return n[0] == 't', true
}

// each calls f for every element of the array node n at path, with its index
// and path, and stops once the problem cap is reached (work past the cap
// cannot be reported).
func (d *decoder) each(n node, path string, f func(i int, el node, p string)) {
	for i, el := range n.elements() {
		if d.probs.full() {
			return
		}
		f(i, el, indexPath(path, i))
	}
}

// list checks that n is an array of lo to hi elements (arrayLen) and calls f
// for each; f is never called when the bounds failed.
func (d *decoder) list(n node, path string, lo, hi int, f func(i int, el node, p string)) {
	if _, ok := d.arrayLen(n, path, lo, hi); ok {
		d.each(n, path, f)
	}
}

// skip marks keys known without reading them, so finish does not report the
// fields of a variant whose kind could not be decided as unknown.
func (o *object) skip(keys ...string) {
	for _, k := range keys {
		o.known[k] = struct{}{}
	}
}

// skipAll marks every key of the object known, for an object whose fields
// cannot be read (its declaration had a problem), so none is reported unknown.
func (o *object) skipAll() {
	for _, m := range o.members {
		o.known[m.key] = struct{}{}
	}
}
