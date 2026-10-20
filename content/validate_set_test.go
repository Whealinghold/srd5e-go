package content_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// owner is a synthetic entry with reference sites.
type owner struct {
	key  content.Key
	refs []content.Reference
}

func (o owner) Key() content.Key                { return o.key }
func (o owner) References() []content.Reference { return slices.Clone(o.refs) }

// categorised is a synthetic entry that has a feat category.
type categorised struct {
	key content.Key
	cat content.FeatCategory
}

func (c categorised) Key() content.Key                   { return c.key }
func (c categorised) FeatCategory() content.FeatCategory { return c.cat }

func key(kind content.Kind, id string) content.Key {
	return content.NewKey(kind, content.MustID(id))
}

func originOnly() content.ReferenceConstraint {
	return content.TargetHasFeatCategory{Category: content.FeatOrigin}
}

func ref(field string, target content.Key, c content.ReferenceConstraint) content.Reference {
	return content.MustReference(field, target, c)
}

func newOwner(kind content.Kind, id string, refs ...content.Reference) owner {
	return owner{key: key(kind, id), refs: refs}
}

func validate(t *testing.T, sets ...content.Set) (content.Set, []content.ReferenceProblem) {
	t.Helper()
	merged, err := content.ValidateSet(sets...)
	if err != nil && (!merged.Equal(content.Set{}) || !errors.Is(err, srderr.ErrInvalidInput) || len(content.ReferenceProblems(err)) == 0) {
		t.Fatalf("a rejection must be the zero Set and an error of problems, got (%v, %v)", merged, err)
	}
	return merged, content.ReferenceProblems(err)
}

func rp(owner content.Key, field string, target content.Key, kind content.ProblemKind) content.ReferenceProblem {
	return content.ReferenceProblem{Owner: owner, Field: field, Target: target, Kind: kind}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-125, FR-126, FR-127, FR-140 - ValidateSet resolves
// every reference over the merged sets, reports every problem in owner order,
// and returns the merged Set when there is none.
// Path: per case.
func TestValidateSet(t *testing.T) {
	t.Parallel()
	f1 := key(content.KindFeat, "f1")
	b1 := key(content.KindBackground, "b1")
	site := func(target content.Key) content.Reference { return ref("feat/featId", target, originOnly()) }
	feat := func(id string, cat content.FeatCategory) content.Entry {
		return categorised{key: key(content.KindFeat, id), cat: cat}
	}
	cases := []struct {
		name, requirement, path string
		sets                    []content.Set
		want                    []content.ReferenceProblem // nil: valid
	}{
		{"reference resolves across sets", "FR-036, FR-125", "happy",
			[]content.Set{content.MustSet(feat("f1", content.FeatOrigin)), content.MustSet(newOwner(content.KindBackground, "b1", site(f1)))}, nil},
		{"order of the sets does not matter", "FR-125", "happy",
			[]content.Set{content.MustSet(newOwner(content.KindBackground, "b1", site(f1))), content.MustSet(feat("f1", content.FeatOrigin))}, nil},
		{"unresolved: the target is in no set", "FR-125", "unhappy (missing content)",
			[]content.Set{content.MustSet(feat("f1", content.FeatOrigin)), content.MustSet(newOwner(content.KindBackground, "b1", site(key(content.KindFeat, "nothing"))))},
			[]content.ReferenceProblem{rp(b1, "feat/featId", key(content.KindFeat, "nothing"), content.ProblemUnresolvedReference)}},
		{"unresolved: the target set is absent", "FR-125", "unhappy (missing content)",
			[]content.Set{content.MustSet(newOwner(content.KindBackground, "b1", site(f1)))},
			[]content.ReferenceProblem{rp(b1, "feat/featId", f1, content.ProblemUnresolvedReference)}},
		{"unresolved: the id exists under another kind", "FR-145", "unhappy (rule interaction)",
			[]content.Set{content.MustSet(plain(content.KindSpell, "f1"), newOwner(content.KindBackground, "b1", site(f1)))},
			[]content.ReferenceProblem{rp(b1, "feat/featId", f1, content.ProblemUnresolvedReference)}},
		{"wrong category", "FR-036, FR-126", "unhappy (rule interaction)",
			[]content.Set{content.MustSet(feat("f1", content.FeatGeneral)), content.MustSet(newOwner(content.KindBackground, "b1", site(f1)))},
			[]content.ReferenceProblem{rp(b1, "feat/featId", f1, content.ProblemConstraintViolation)}},
		{"target category below the constraint's", "FR-126", "unhappy (rule interaction)",
			[]content.Set{content.MustSet(feat("f1", content.FeatOrigin), newOwner(content.KindBackground, "b1", ref("feat/featId", f1, content.TargetHasFeatCategory{Category: content.FeatGeneral})))},
			[]content.ReferenceProblem{rp(b1, "feat/featId", f1, content.ProblemConstraintViolation)}},
		{"every category satisfies its own constraint", "FR-126", "happy",
			[]content.Set{content.MustSet(feat("f1", content.FeatEpicBoon), feat("f2", content.FeatFightingStyle),
				newOwner(content.KindBackground, "b1", ref("a", f1, content.TargetHasFeatCategory{Category: content.FeatEpicBoon}), ref("b", key(content.KindFeat, "f2"), content.TargetHasFeatCategory{Category: content.FeatFightingStyle})))}, nil},
		{"target is not a HasFeatCategory", "FR-126", "unhappy (rule interaction)",
			[]content.Set{content.MustSet(plain(content.KindFeat, "f1"), newOwner(content.KindBackground, "b1", site(f1)))},
			[]content.ReferenceProblem{rp(b1, "feat/featId", f1, content.ProblemConstraintViolation)}},
		{"a constraint that no category satisfies", "FR-126", "unhappy (boundary)",
			[]content.Set{content.MustSet(feat("f1", content.FeatOrigin), owner{key: b1, refs: []content.Reference{{}}})},
			[]content.ReferenceProblem{rp(b1, "", content.Key{}, content.ProblemUnresolvedReference)}},
		{"no constraint: existence only", "FR-126", "happy",
			[]content.Set{content.MustSet(plain(content.KindSpell, "s1"), newOwner(content.KindSpecies, "x", ref("a", key(content.KindSpell, "s1"), nil)))}, nil},
		{"same key in two sets", "FR-125", "unhappy (invalid input)",
			[]content.Set{content.MustSet(plain(content.KindSpell, "a")), content.MustSet(plain(content.KindSpell, "a"))},
			[]content.ReferenceProblem{rp(key(content.KindSpell, "a"), "", key(content.KindSpell, "a"), content.ProblemDuplicate)}},
		{"same key in three sets", "FR-125", "unhappy (invalid input)",
			[]content.Set{content.MustSet(plain(content.KindSpell, "a")), content.MustSet(plain(content.KindSpell, "a")), content.MustSet(plain(content.KindSpell, "a"))},
			[]content.ReferenceProblem{rp(key(content.KindSpell, "a"), "", key(content.KindSpell, "a"), content.ProblemDuplicate), rp(key(content.KindSpell, "a"), "", key(content.KindSpell, "a"), content.ProblemDuplicate)}},
		{"owner without references", "FR-125", "unhappy (empty case)",
			[]content.Set{content.MustSet(plain(content.KindBackground, "b1"), plain(content.KindFeat, "f1"))}, nil},
		{"owner with an empty reference list", "FR-125", "unhappy (empty case)",
			[]content.Set{content.MustSet(newOwner(content.KindBackground, "b1"))}, nil},
		{"no site opts out: one resolves, one does not", "FR-126, FR-140", "unhappy (dangling id)",
			[]content.Set{content.MustSet(plain(content.KindSpell, "s1"), newOwner(content.KindSpecies, "x",
				ref("good", key(content.KindSpell, "s1"), nil), ref("bad", key(content.KindSpell, "s2"), nil)))},
			[]content.ReferenceProblem{rp(key(content.KindSpecies, "x"), "bad", key(content.KindSpell, "s2"), content.ProblemUnresolvedReference)}},
		{"same-kind reference, target absent", "FR-126, FR-140", "unhappy (dangling id)",
			[]content.Set{content.MustSet(newOwner(content.KindEquipment, "e2", ref("variantOf", key(content.KindEquipment, "e1"), nil)))},
			[]content.ReferenceProblem{rp(key(content.KindEquipment, "e2"), "variantOf", key(content.KindEquipment, "e1"), content.ProblemUnresolvedReference)}},
		{"same-kind reference, target present", "FR-126, FR-140", "unhappy (dangling id)",
			[]content.Set{content.MustSet(plain(content.KindEquipment, "e1"), newOwner(content.KindEquipment, "e2", ref("variantOf", key(content.KindEquipment, "e1"), nil)))}, nil},
		{"ordering across kinds, then ids, then site order", "FR-127", "happy",
			[]content.Set{content.MustSet(
				newOwner(content.KindBackground, "b2", ref("f", f1, nil)),
				newOwner(content.KindSpecies, "s9", ref("f", f1, nil)),
				newOwner(content.KindCharacterClass, "c1", ref("z", f1, nil), ref("a", f1, nil)),
				newOwner(content.KindBackground, "b1", ref("f", f1, nil)),
			)},
			[]content.ReferenceProblem{
				rp(key(content.KindSpecies, "s9"), "f", f1, content.ProblemUnresolvedReference),
				rp(key(content.KindBackground, "b1"), "f", f1, content.ProblemUnresolvedReference),
				rp(key(content.KindBackground, "b2"), "f", f1, content.ProblemUnresolvedReference),
				rp(key(content.KindCharacterClass, "c1"), "z", f1, content.ProblemUnresolvedReference),
				rp(key(content.KindCharacterClass, "c1"), "a", f1, content.ProblemUnresolvedReference),
			}},
		{"duplicate and unresolved on one owner: duplicate first", "FR-127", "unhappy (rule interaction)",
			[]content.Set{content.MustSet(newOwner(content.KindBackground, "b1", ref("f", f1, nil))), content.MustSet(plain(content.KindBackground, "b1"))},
			[]content.ReferenceProblem{
				rp(b1, "", b1, content.ProblemDuplicate),
				rp(b1, "f", f1, content.ProblemUnresolvedReference),
			}},
		{"a duplicate of a later id follows the earlier id's unresolved reference", "FR-127", "unhappy (rule interaction)",
			[]content.Set{content.MustSet(newOwner(content.KindBackground, "b1", ref("f", f1, nil)), plain(content.KindBackground, "b2")), content.MustSet(plain(content.KindBackground, "b2"))},
			[]content.ReferenceProblem{
				rp(b1, "f", f1, content.ProblemUnresolvedReference),
				rp(key(content.KindBackground, "b2"), "", key(content.KindBackground, "b2"), content.ProblemDuplicate),
			}},
		{"one owner with many unresolved references keeps site order", "FR-127", "unhappy (boundary)",
			[]content.Set{content.MustSet(newOwner(content.KindBackground, "b1", manyRefs(f1, 40)...))},
			manyProblems(b1, f1, 40)},
		{"one owner: the duplicate stays ahead of many references", "FR-127", "unhappy (boundary)",
			[]content.Set{content.MustSet(newOwner(content.KindBackground, "b1", manyRefs(f1, 40)...)), content.MustSet(plain(content.KindBackground, "b1"))},
			append([]content.ReferenceProblem{rp(b1, "", b1, content.ProblemDuplicate)}, manyProblems(b1, f1, 40)...)},
		{"no sets", "FR-125", "unhappy (empty case)", nil, nil},
		{"zero sets", "FR-125", "unhappy (empty case)", []content.Set{{}, {}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			merged, got := validate(t, c.sets...)
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %+v, want %+v [%s]", got, c.want, c.requirement)
			}
			if c.want == nil {
				var all []content.Entry
				for _, s := range c.sets {
					all = append(all, s.Entries()...)
				}
				if !merged.Equal(content.MustSet(all...)) {
					t.Errorf("merged set = %v, want the union in FR-071 order", keys(merged))
				}
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-093 - a Set checks only keys: a background whose
// feat is absent is a valid Set (ValidateSet reports it).
// Path: unhappy (rule interaction).
func TestNewSetDoesNotCheckReferences(t *testing.T) {
	t.Parallel()
	s, err := content.NewSet(newOwner(content.KindBackground, "b1", ref("feat/featId", key(content.KindFeat, "f1"), originOnly())))
	if err != nil || len(s.Entries()) != 1 {
		t.Fatalf("NewSet = (%v, %v), want accepted", s, err)
	}
	if _, got := validate(t, s); len(got) != 1 {
		t.Errorf("ValidateSet found %d problems, want 1", len(got))
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-125, FR-127 - the result is deterministic and the
// errors are typed, comparable values that wrap ErrInvalidInput.
// Path: unhappy (rule interaction).
func TestValidateSetIsDeterministicAndTyped(t *testing.T) {
	t.Parallel()
	s := content.MustSet(newOwner(content.KindBackground, "b1", ref("f", key(content.KindFeat, "x"), nil)), plain(content.KindFeat, "y"))
	_, err1 := content.ValidateSet(s)
	_, err2 := content.ValidateSet(s)
	if !slices.Equal(content.ReferenceProblems(err1), content.ReferenceProblems(err2)) {
		t.Errorf("two runs differ")
	}
	var p *content.ReferenceProblem
	if !errors.As(err1, &p) || !errors.Is(err1, srderr.ErrInvalidInput) || p.Error() == "" {
		t.Errorf("error = %v, want a *ReferenceProblem wrapping ErrInvalidInput", err1)
	}
	if got := content.ReferenceProblems(nil); got != nil {
		t.Errorf("ReferenceProblems(nil) = %v", got)
	}
	if got := content.ReferenceProblems(errors.New("other")); got != nil {
		t.Errorf("ReferenceProblems(foreign) = %v", got)
	}
	wrapped := errors.Join(errors.New("x"), err1)
	if got := content.ReferenceProblems(wrapped); len(got) != 1 {
		t.Errorf("ReferenceProblems through a join = %v", got)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-125 - ValidateSet is linear: a hundred thousand
// entries with a reference each resolve without quadratic work.
// Path: unhappy (boundary).
func TestValidateSetScales(t *testing.T) {
	t.Parallel()
	const n = 100_000
	entries := make([]content.Entry, 0, 2*n)
	for i := range n {
		id := "e" + string(rune('a'+i%26)) + itoa(i)
		entries = append(entries, plain(content.KindSpell, id))
		entries = append(entries, newOwner(content.KindSpecies, id, ref("s", key(content.KindSpell, id), nil)))
	}
	merged, got := validate(t, content.MustSet(entries...))
	if len(got) != 0 || len(merged.Entries()) != 2*n {
		t.Errorf("got %d problems, %d entries", len(got), len(merged.Entries()))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for ; i > 0; i /= 10 {
		b = append([]byte{byte('0' + i%10)}, b...)
	}
	return string(b)
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-125, FR-126 - NewReference rejects an invalid field,
// target or constraint with a typed error, in that order, and MustReference
// panics with it.
// Path: per case.
func TestNewReference(t *testing.T) {
	t.Parallel()
	valid := key(content.KindFeat, "f1")
	cases := []struct {
		name  string
		field string
		key   content.Key
		c     content.ReferenceConstraint
		want  content.ReferenceProblemKind // 0: accepted
	}{
		{"with a constraint", "feat/featId", valid, originOnly(), 0},
		{"without a constraint", "feat/featId", valid, nil, 0},
		{"blank field", " ", valid, nil, content.ReferenceBlankField},
		{"empty field", "", valid, nil, content.ReferenceBlankField},
		{"zero target", "f", content.Key{}, nil, content.ReferenceInvalidTarget},
		{"target with an invalid kind", "f", content.NewKey(content.Kind(99), content.MustID("a")), nil, content.ReferenceInvalidTarget},
		{"target with the zero id", "f", content.NewKey(content.KindFeat, content.ID{}), nil, content.ReferenceInvalidTarget},
		{"zero category", "f", valid, content.TargetHasFeatCategory{}, content.ReferenceInvalidConstraint},
		{"category not a member", "f", valid, content.TargetHasFeatCategory{Category: content.FeatCategory(99)}, content.ReferenceInvalidConstraint},
		{"pointer constraint", "f", valid, &content.TargetHasFeatCategory{Category: content.FeatOrigin}, content.ReferenceInvalidConstraint},
		{"field outranks target", " ", content.Key{}, nil, content.ReferenceBlankField},
		{"target outranks constraint", "f", content.Key{}, content.TargetHasFeatCategory{}, content.ReferenceInvalidTarget},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := content.NewReference(c.field, c.key, c.c)
			panicked := panicValue(func() { content.MustReference(c.field, c.key, c.c) })
			if c.want == 0 {
				if err != nil || panicked != nil || r.Field() != c.field || r.Target() != c.key || r.Constraint() != c.c {
					t.Fatalf("got (%+v, %v), panic %v; want accepted", r, err, panicked)
				}
				return
			}
			var re *content.ReferenceError
			if !errors.As(err, &re) || re.Problem != c.want || !errors.Is(err, srderr.ErrInvalidInput) || re.Error() == "" {
				t.Fatalf("error = %v, want *ReferenceError %v", err, c.want)
			}
			if !errors.Is(err, &content.ReferenceError{Problem: c.want}) || errors.Is(err, &content.ReferenceError{Problem: c.want + 1}) || errors.Is(err, errors.New("x")) {
				t.Errorf("ReferenceError.Is must match by Problem only")
			}
			if pe, ok := panicked.(error); !ok || !errors.Is(pe, &content.ReferenceError{Problem: c.want}) || !r.Target().IsZero() {
				t.Errorf("MustReference panic = %v, want the constructor error", panicked)
			}
		})
	}
	for p, want := range map[content.ReferenceProblemKind]string{
		content.ReferenceBlankField: "blankField", content.ReferenceInvalidTarget: "invalidTarget",
		content.ReferenceInvalidConstraint: "invalidConstraint", 0: "unknown", 4: "unknown",
	} {
		if p.String() != want {
			t.Errorf("ReferenceProblemKind(%d).String() = %q, want %q", int(p), p.String(), want)
		}
	}
	if (&content.ReferenceError{}).Is(nil) || (&content.ReferenceError{Problem: 1}).Is((*content.ReferenceError)(nil)) {
		t.Errorf("Is(nil) must be false")
	}
	for _, p := range []content.ReferenceProblemKind{content.ReferenceBlankField, content.ReferenceInvalidTarget, content.ReferenceInvalidConstraint} {
		e := &content.ReferenceError{Problem: p}
		for _, other := range []content.ReferenceProblemKind{p - 1, p + 1} {
			if e.Is(&content.ReferenceError{Problem: other}) {
				t.Errorf("ReferenceError %v matches %v", p, other)
			}
		}
		if !e.Is(&content.ReferenceError{Problem: p}) {
			t.Errorf("ReferenceError %v does not match itself", p)
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-126 - a Reference is an immutable comparable value
// and TargetHasFeatCategory compares by category.
// Path: unhappy (rule interaction).
func TestReferenceEquality(t *testing.T) {
	t.Parallel()
	a := ref("f", key(content.KindFeat, "x"), originOnly())
	if a != ref("f", key(content.KindFeat, "x"), originOnly()) || a == ref("g", key(content.KindFeat, "x"), originOnly()) ||
		a == ref("f", key(content.KindFeat, "y"), originOnly()) || a == ref("f", key(content.KindFeat, "x"), nil) ||
		a == ref("f", key(content.KindFeat, "x"), content.TargetHasFeatCategory{Category: content.FeatGeneral}) {
		t.Errorf("Reference must compare by field, target and constraint")
	}
}

// manyRefs returns n unresolved references to target, fields "r0" to "r<n-1>".
func manyRefs(target content.Key, n int) []content.Reference {
	refs := make([]content.Reference, n)
	for i := range refs {
		refs[i] = ref("r"+itoa(i), target, nil)
	}
	return refs
}

// manyProblems returns the problems manyRefs gives for owner, in site order.
func manyProblems(owner, target content.Key, n int) []content.ReferenceProblem {
	out := make([]content.ReferenceProblem, n)
	for i := range out {
		out[i] = rp(owner, "r"+itoa(i), target, content.ProblemUnresolvedReference)
	}
	return out
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-127 - problems are ordered by owner, and on one owner the
// duplicate precedes its references in site order, however many owners interleave. A stable sort is required: an
// unstable one reorders the ties once the input is large enough to be partitioned.
// Path: unhappy (rule interaction).
func TestValidateSetProblemOrderIsStableAcrossOwners(t *testing.T) {
	t.Parallel()
	const owners, sites = 30, 3
	f1 := key(content.KindFeat, "f1")
	var first, second []content.Entry
	var want []content.ReferenceProblem
	for i := range owners {
		id := "o" + string(rune('a'+i/10)) + string(rune('a'+i%10))
		first = append(first, newOwner(content.KindBackground, id, manyRefs(f1, sites)...))
		second = append(second, plain(content.KindBackground, id))
		k := key(content.KindBackground, id)
		want = append(want, rp(k, "", k, content.ProblemDuplicate))
		want = append(want, manyProblems(k, f1, sites)...)
	}
	_, got := validate(t, content.MustSet(first...), content.MustSet(second...))
	if !slices.Equal(got, want) {
		t.Errorf("problems = %+v, want %+v", got, want)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-125, FR-127 - the first entry holding a key, in set order,
// is the one a reference resolves to and the one whose sites are checked, however many sets repeat the key.
// Path: unhappy (rule interaction).
func TestValidateSetFirstSetWinsAcrossManySets(t *testing.T) {
	t.Parallel()
	const feats, sets = 20, 4
	built := make([]content.Set, sets+1)
	var refs []content.Reference
	var want []content.ReferenceProblem
	for s := range sets {
		cat := content.FeatGeneral
		if s == 0 {
			cat = content.FeatOrigin
		}
		var es []content.Entry
		for i := range feats {
			es = append(es, categorised{key: key(content.KindFeat, "f"+string(rune('a'+i/10))+string(rune('a'+i%10))), cat: cat})
		}
		built[s] = content.MustSet(es...)
	}
	for i := range feats {
		k := key(content.KindFeat, "f"+string(rune('a'+i/10))+string(rune('a'+i%10)))
		refs = append(refs, ref("feat"+itoa(i), k, originOnly()))
	}
	built[sets] = content.MustSet(newOwner(content.KindBackground, "b1", refs...))
	for i := range feats {
		k := key(content.KindFeat, "f"+string(rune('a'+i/10))+string(rune('a'+i%10)))
		for range sets - 1 {
			want = append(want, rp(k, "", k, content.ProblemDuplicate))
		}
	}
	_, got := validate(t, built...)
	if !slices.Equal(got, want) {
		t.Errorf("problems = %+v, want %+v", got, want)
	}
}
