package content

import (
	"cmp"
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// ReferenceProblem is one problem [ValidateSet] found: the entry Owner,
// holding at Field an id that names Target, which failed with Kind. For
// ProblemDuplicate (the same key in two sets) Field is empty and Target equals
// Owner. It wraps srderr.ErrInvalidInput. Construct it with a keyed literal.
//
//nolint:errname // the name is fixed by schema FR-125.
type ReferenceProblem struct {
	Owner  Key
	Field  string
	Target Key
	Kind   ProblemKind
}

// Error renders the kind, owner and field. The text is not a contract.
func (e *ReferenceProblem) Error() string {
	return "content: " + e.Kind.String() + " reference from " + e.Owner.Kind().String() + " " + e.Owner.ID().String() + " at " + e.Field
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ReferenceProblem) Unwrap() error { return srderr.ErrInvalidInput }

// ReferenceProblems returns every *[ReferenceProblem] held in err, flattened
// through errors.Join and wrapping, in order, as new values. It returns nil
// for a nil error and for an error holding none.
func ReferenceProblems(err error) []ReferenceProblem {
	var out []ReferenceProblem
	collectReferenceProblems(err, &out)
	return out
}

func collectReferenceProblems(err error, out *[]ReferenceProblem) {
	//nolint:errorlint // flattening must visit every branch of a join, which errors.As cannot do.
	switch e := err.(type) {
	case nil:
	case *ReferenceProblem:
		*out = append(*out, *e)
	case interface{ Unwrap() []error }:
		for _, inner := range e.Unwrap() {
			collectReferenceProblems(inner, out)
		}
	default:
		collectReferenceProblems(errors.Unwrap(err), out)
	}
}

// ValidateSet merges sets and checks every reference between their entries
// (schema FR-125 to FR-127, FR-140). It is a pure function that records no
// signal and is not a use case.
//
// For valid input it returns the merged immutable Set and a nil error.
// Otherwise it returns the zero Set and an errors.Join of one or more
// *[ReferenceProblem], read with [ReferenceProblems]: ProblemDuplicate when a
// key is in two sets (at the later set's entry), ProblemUnresolvedReference
// when a reference's target is in no set, ProblemConstraintViolation when the
// target exists but fails the constraint of its site. Every problem is
// reported, ordered by owner kind, owner id, then the order the entry lists
// its references, so a problem on one entry never hides another; the first
// entry holding a key is the one a reference resolves to. Entries are read
// through [References]; an entry that does not implement it has no sites. It
// never panics for content, and works in time linear in the entries and
// references.
func ValidateSet(sets ...Set) (Set, error) {
	var all []Entry
	for _, s := range sets {
		all = append(all, s.entries...)
	}
	slices.SortStableFunc(all, compareKeys)
	first := make(map[Key]Entry, len(all))
	var problems []*ReferenceProblem
	for _, e := range all {
		k := e.Key()
		if _, repeated := first[k]; repeated {
			problems = append(problems, &ReferenceProblem{Owner: k, Target: k, Kind: ProblemDuplicate})
			continue
		}
		first[k] = e
	}
	for _, e := range all {
		refs, ok := e.(References)
		if !ok {
			continue
		}
		for _, r := range refs.References() {
			if kind, bad := checkReference(first, r); bad {
				problems = append(problems, &ReferenceProblem{Owner: e.Key(), Field: r.field, Target: r.target, Kind: kind})
			}
		}
	}
	if len(problems) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return Set{entries: all}, nil
	}
	slices.SortStableFunc(problems, func(a, b *ReferenceProblem) int {
		return cmp.Or(cmp.Compare(a.Owner.Kind(), b.Owner.Kind()), cmp.Compare(a.Owner.ID().String(), b.Owner.ID().String()))
	})
	errs := make([]error, len(problems))
	for i, p := range problems {
		errs[i] = p
	}
	return Set{}, errors.Join(errs...)
}

// checkReference resolves one reference against the first entry of each key
// and reports the problem kind, or false when it holds.
func checkReference(first map[Key]Entry, r Reference) (ProblemKind, bool) {
	target, ok := first[r.target]
	if !ok {
		return ProblemUnresolvedReference, true
	}
	switch c := r.constraint.(type) {
	case nil:
		return 0, false // equivalent mutant: replaces the literal 0 with 0
	case TargetHasFeatCategory:
		if c.satisfiedBy(target) {
			return 0, false // equivalent mutant: replaces the literal 0 with 0
		}
	}
	return ProblemConstraintViolation, true
}
