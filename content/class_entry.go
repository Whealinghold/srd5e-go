package content

import (
	"slices"
	"strconv"
)

// FeatureEntryFields holds the members of a [FeatureEntry] beyond its
// identity, for [NewFeatureEntry]. Construct it with a keyed literal.
type FeatureEntryFields struct {
	// Level is the class level the feature is gained at, 1 to [ClassLevels].
	Level int
	// Text is the faithful SRD text of the feature.
	Text string
	// Facts lists the structured facts, distinct; possibly none.
	Facts []FeatureFact
	// Options lists the named options, possibly none.
	Options []FeatureOption
	// Tables lists the embedded tables, possibly none.
	Tables []FeatureTable
}

// FeatureEntry is one class or subclass feature ("Level 1: Rage"): a stable
// id, the printed name and level, the faithful text, and the facts, options
// and tables the text does not carry (SRD 5.2.1, "Barbarian Class Features",
// pp. 28-30). The zero value is invalid; build it with [NewFeatureEntry]. It
// holds slices, so it is not comparable with ==.
type FeatureEntry struct {
	id      FeatureID
	name    string
	level   int
	text    string
	facts   []FeatureFact
	options []FeatureOption
	tables  []FeatureTable
}

// NewFeatureEntry returns the feature id called name with the members of f,
// copied. Whether the id carries its owner's prefix is checked when the entry
// is added to a class ([NewClassDefinition]).
//
// It returns the *[IDError] of [ParseID] for an invalid id, then a
// *[BlockError] wrapping srderr.ErrInvalidInput for the first rule broken, in
// the order of the members: BlockInvalidName; BlockOutOfRange for a level
// outside 1 to [ClassLevels]; BlockBlankText or BlockInvalidCharacter for the
// text; BlockInvalidMember for a fact, option or table that is not valid;
// BlockDuplicate for a repeated fact or option id; then the rules between the
// parts: BlockInvalidMember for a spell grant starting below the feature's
// level and BlockConstraint for a prerequisite that names an option the
// feature does not have, its own option, or closes a cycle. It never panics.
func NewFeatureEntry(id FeatureID, name string, f FeatureEntryFields) (FeatureEntry, error) {
	const block = "FeatureEntry"
	if _, err := ParseID(string(id)); err != nil {
		return FeatureEntry{}, err
	}
	if err := checkName(block, name); err != nil {
		return FeatureEntry{}, err
	}
	if f.Level < 1 || f.Level > ClassLevels {
		return FeatureEntry{}, blockError(block, BlockOutOfRange)
	}
	if err := checkText(block, f.Text); err != nil {
		return FeatureEntry{}, err
	}
	facts := make(map[string]struct{}, len(f.Facts))
	for _, fact := range f.Facts {
		if !validFeatureFact(fact) {
			return FeatureEntry{}, blockError(block, BlockInvalidMember)
		}
		k := factKey(fact)
		if _, dup := facts[k]; dup {
			return FeatureEntry{}, blockError(block, BlockDuplicate)
		}
		facts[k] = struct{}{}
	}
	ids := make(map[FeatureOptionID]struct{}, len(f.Options))
	for _, o := range f.Options {
		if !o.valid() {
			return FeatureEntry{}, blockError(block, BlockInvalidMember)
		}
		if _, dup := ids[o.id]; dup {
			return FeatureEntry{}, blockError(block, BlockDuplicate)
		}
		ids[o.id] = struct{}{}
	}
	if slices.ContainsFunc(f.Tables, func(t FeatureTable) bool { return !t.valid() }) {
		return FeatureEntry{}, blockError(block, BlockInvalidMember)
	}
	e := FeatureEntry{
		id: id, name: name, level: f.Level, text: f.Text, facts: cloneOrNil(f.Facts),
		options: cloneOrNil(f.Options), tables: cloneOrNil(f.Tables),
	}
	if ps := e.problems(); len(ps) > 0 { // equivalent mutant: a length is never negative, so > and != agree
		return FeatureEntry{}, ps[0].blockError(block)
	}
	return e, nil
}

// ID returns the feature id.
func (e FeatureEntry) ID() FeatureID { return e.id }

// Name returns the printed feature name, exactly as the SRD prints it.
func (e FeatureEntry) Name() string { return e.name }

// Level returns the class level the feature is gained at.
func (e FeatureEntry) Level() int { return e.level }

// Text returns the faithful SRD text.
func (e FeatureEntry) Text() string { return e.text }

// Facts returns the structured facts, as a new slice.
func (e FeatureEntry) Facts() []FeatureFact { return slices.Clone(e.facts) }

// Options returns the named options, as a new slice.
func (e FeatureEntry) Options() []FeatureOption { return slices.Clone(e.options) }

// Tables returns the embedded tables, as a new slice.
func (e FeatureEntry) Tables() []FeatureTable { return slices.Clone(e.tables) }

// valid reports whether e was built by [NewFeatureEntry]: the fields are
// unexported and every constructor path validates them, so only the zero value
// is invalid. It is constant time, so a class does not re-run the rules of
// each of its entries.
func (e FeatureEntry) valid() bool { return e.id != "" }

// issue is a rule broken between the parts of a value: where, relative to the
// value, and the problem kind the decoder reports for it. The constructor
// returns the first as a *BlockError and the decoder reports them all, so
// each cross rule lives once.
type issue struct {
	path string
	kind ProblemKind
}

// blockError converts the issue to the constructor error of block.
func (i issue) blockError(block string) error {
	p := BlockConstraint
	switch i.kind { //nolint:exhaustive // the other kinds all map to BlockConstraint.
	case ProblemMissingField:
		p = BlockMissing
	case ProblemOutOfRange:
		p = BlockOutOfRange
	case ProblemInvalidValue:
		p = BlockInvalidMember
	case ProblemDuplicate:
		p = BlockDuplicate
	}
	return blockError(block, p)
}

func idx(parts ...any) string {
	var s string
	for _, p := range parts {
		switch v := p.(type) {
		case int:
			s += "/" + strconv.Itoa(v)
		case string:
			s += "/" + v
		}
	}
	return s[1:]
}

// problems returns the rules broken between the parts of the entry, with paths
// relative to it, in the order: a spell grant starting below the feature's
// level, on the feature or on an option (invalidValue), and the option
// prerequisites (an id that names no option is unresolvedReference, one that
// names its own option or closes a cycle constraintViolation).
func (e FeatureEntry) problems() []issue {
	var out []issue
	for i, fact := range e.facts {
		out = append(out, grantProblems(idx("facts", i), fact, e.level)...)
	}
	byID := make(map[FeatureOptionID]int, len(e.options))
	for i, o := range e.options {
		byID[o.id] = i
	}
	comp := optionComponents(e.options, byID)
	for i, o := range e.options {
		for j, fact := range o.facts {
			out = append(out, grantProblems(idx("options", i, "facts", j), fact, e.level)...)
		}
	}
	for i, o := range e.options {
		for j, req := range o.prereq.requires {
			p := idx("options", i, "prerequisite", "requiresOptionIds", j)
			switch k, ok := byID[req]; {
			case !ok:
				out = append(out, issue{p, ProblemUnresolvedReference})
			case comp[k] == comp[i]: // a self edge is its own component
				out = append(out, issue{p, ProblemConstraintViolation})
			}
		}
	}
	return out
}

// grantProblems reports each grant of an AlwaysPreparedSpells fact that starts
// below level (FR-058).
func grantProblems(path string, fact FeatureFact, level int) []issue {
	a, ok := fact.(AlwaysPreparedSpells)
	if !ok {
		return nil
	}
	var out []issue
	for k, g := range a.grants {
		if g.level < level {
			out = append(out, issue{path + "/" + idx("grants", k, "level"), ProblemInvalidValue})
		}
	}
	return out
}

// optionComponents returns, for each option, the number of its strongly
// connected component over the "requires" edges (Tarjan, one pass, O(options +
// edges)). An edge from i to k closes a cycle exactly when both share a
// component, which includes k == i.
func optionComponents(options []FeatureOption, byID map[FeatureOptionID]int) []int {
	t := tarjan{options: options, byID: byID, index: make([]int, len(options)), low: make([]int, len(options)), on: make([]bool, len(options)), comp: make([]int, len(options))}
	for i := range options {
		if t.index[i] == 0 {
			t.visit(i)
		}
	}
	return t.comp
}

type tarjan struct {
	options    []FeatureOption
	byID       map[FeatureOptionID]int
	index, low []int // index is 1-based; 0 is unvisited
	on         []bool
	stack      []int
	comp       []int
	n, ncomp   int
}

func (t *tarjan) visit(v int) {
	t.n++
	t.index[v], t.low[v] = t.n, t.n
	t.stack = append(t.stack, v)
	t.on[v] = true
	for _, req := range t.options[v].prereq.requires {
		w, ok := t.byID[req]
		switch {
		case !ok:
		case t.index[w] == 0:
			t.visit(w)
			t.low[v] = min(t.low[v], t.low[w])
		case t.on[w]:
			t.low[v] = min(t.low[v], t.index[w])
		}
	}
	if t.low[v] != t.index[v] {
		return
	}
	for {
		w := t.stack[len(t.stack)-1]
		t.stack = t.stack[:len(t.stack)-1]
		t.on[w] = false
		t.comp[w] = t.ncomp
		if w == v {
			break
		}
	}
	t.ncomp++
}
